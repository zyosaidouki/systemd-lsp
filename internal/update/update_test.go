package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
)

func archiveFor(t *testing.T, name string, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0755, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestInstall(t *testing.T) {
	if !((runtime.GOOS == "linux" && runtime.GOARCH == "amd64") || (runtime.GOOS == "darwin" && runtime.GOARCH == "arm64")) {
		t.Skip("unsupported host")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	binary, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	good := archiveFor(t, "systemd-lsp", binary)
	for _, scenario := range []string{"success", "symlink", "already", "identical", "unknown", "checksum", "missing", "http", "traversal", "invalid-binary"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			target := filepath.Join(dir, "systemd-lsp")
			original := []byte("original")
			if scenario == "identical" {
				original = binary
			}
			if err := os.WriteFile(target, original, 0755); err != nil {
				t.Fatal(err)
			}
			before, err := os.Stat(target)
			if err != nil {
				t.Fatal(err)
			}
			invoked := target
			if scenario == "symlink" {
				invoked = filepath.Join(dir, "link")
				if err := os.Symlink(target, invoked); err != nil {
					t.Fatal(err)
				}
			}
			archive := good
			if scenario == "traversal" {
				archive = archiveFor(t, "../systemd-lsp", binary)
			}
			if scenario == "invalid-binary" {
				archive = archiveFor(t, "systemd-lsp", []byte("not executable"))
			}
			digest := fmt.Sprintf("%x", sha256.Sum256(archive))
			if scenario == "checksum" {
				digest = strings.Repeat("0", 64)
			}
			name := "systemd-lsp-" + runtime.GOOS + "-" + runtime.GOARCH + ".tar.gz"
			var downloads atomic.Int32
			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/latest":
					if scenario == "http" {
						http.Error(w, "rate limited", 403)
						return
					}
					assets := []map[string]string{{"name": name, "browser_download_url": server.URL + "/archive"}}
					if scenario != "missing" {
						assets = append(assets, map[string]string{"name": "SHA256SUMS", "browser_download_url": server.URL + "/sums"})
					}
					if err := json.MarshalWrite(w, map[string]any{"tag_name": "build-42-1", "assets": assets}); err != nil {
						t.Errorf("encode release metadata: %v", err)
					}
				case "/sums":
					downloads.Add(1)
					fmt.Fprintf(w, "%s  %s\n", digest, name)
				case "/archive":
					downloads.Add(1)
					w.Write(archive)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			current := "build-41-1"
			if scenario == "already" {
				current = "build-42-1"
			} else if scenario == "unknown" {
				current = ""
			}
			result, err := install(server.Client(), server.URL+"/latest", invoked, runtime.GOOS, runtime.GOARCH, current)
			unchanged := scenario == "already" || scenario == "identical"
			success := scenario == "success" || scenario == "symlink" || scenario == "unknown" || unchanged
			if success {
				previous := current
				if previous == "" {
					previous = "unknown"
				}
				if unchanged {
					previous = "build-42-1"
				}
				want := Result{Previous: previous, Current: "build-42-1", Already: unchanged}
				if err != nil || result != want {
					t.Fatalf("result=%+v want=%+v err=%v", result, want, err)
				}
			}
			if scenario == "already" && downloads.Load() != 0 {
				t.Fatalf("downloaded %d assets for current version", downloads.Load())
			}
			if !success && err == nil {
				t.Fatal("expected error")
			}
			got, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			want := original
			if success && !unchanged {
				want = binary
			}
			if !bytes.Equal(got, want) {
				t.Fatal("unexpected executable contents")
			}
			info, err := os.Stat(target)
			if err != nil || info.Mode().Perm() != 0755 {
				t.Fatalf("mode: %v, %v", info, err)
			}
			if unchanged && !os.SameFile(before, info) {
				t.Fatal("already-current executable was replaced")
			}
			leftovers, _ := filepath.Glob(filepath.Join(dir, ".systemd-lsp-update-*"))
			if len(leftovers) != 0 {
				t.Fatalf("temporary files left: %v", leftovers)
			}
			if scenario == "symlink" {
				info, err := os.Lstat(invoked)
				if err != nil || info.Mode()&os.ModeSymlink == 0 {
					t.Fatal("symlink was replaced")
				}
			}
		})
	}
}

func TestUnsupportedPlatform(t *testing.T) {
	if _, err := install(nil, "", "", "linux", "arm64", "dev"); err == nil {
		t.Fatal("expected unsupported platform error")
	}
}

func TestInstallRejectsInvalidJSONMetadata(t *testing.T) {
	for _, metadata := range []string{
		`{"tag_name":"first","tag_name":"second","assets":[]}`,
		"{\"tag_name\":\"\xff\",\"assets\":[]}",
		`{"tag_name":"build-42","assets":[]} {}`,
	} {
		target := filepath.Join(t.TempDir(), "systemd-lsp")
		if err := os.WriteFile(target, []byte("original"), 0755); err != nil {
			t.Fatal(err)
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, metadata) }))
		_, err := install(server.Client(), server.URL, target, "linux", "amd64", "dev")
		server.Close()
		if err == nil || !strings.Contains(err.Error(), "release metadata:") {
			t.Fatalf("expected metadata parse error, got %v", err)
		}
		data, readErr := os.ReadFile(target)
		if readErr != nil || string(data) != "original" {
			t.Fatalf("invalid metadata modified the executable: %q, %v", data, readErr)
		}
	}
}

func TestDownloadLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "12345") }))
	defer server.Close()
	if _, err := download(server.Client(), server.URL, 4); err == nil {
		t.Fatal("expected size limit error")
	}
}

func TestResultMessage(t *testing.T) {
	for _, tc := range []struct {
		result Result
		want   string
	}{
		{Result{Previous: "build-41-1", Current: "build-42-1"}, "systemd-lsp updated: build-41-1 -> build-42-1"},
		{Result{Previous: "build-42-1", Current: "build-42-1", Already: true}, "systemd-lsp already: build-42-1"},
	} {
		if got := tc.result.Message(); got != tc.want {
			t.Fatalf("message = %q, want %q", got, tc.want)
		}
	}
}
