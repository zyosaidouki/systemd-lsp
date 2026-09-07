// Package update installs verified binaries from GitHub Releases.
package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"debug/elf"
	"debug/macho"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const latestURL = "https://api.github.com/repos/zyosaidouki/systemd-lsp/releases/latest"
const maxBinary = 64 << 20

type release struct {
	Tag    string `json:"tag_name"`
	Assets []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

// Run replaces the executable that was invoked, following symbolic links.
func Run() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	return install(&http.Client{Timeout: 2 * time.Minute}, latestURL, executable, runtime.GOOS, runtime.GOARCH)
}

func install(client *http.Client, endpoint, target, goos, arch string) (string, error) {
	if !((goos == "linux" && arch == "amd64") || (goos == "darwin" && arch == "arm64")) {
		return "", fmt.Errorf("no release binary for %s/%s", goos, arch)
	}
	target, err := filepath.EvalSymlinks(target)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(target)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("not a regular executable: %s", target)
	}
	metadata, err := download(client, endpoint, 1<<20)
	if err != nil {
		return "", err
	}
	var rel release
	if err := json.Unmarshal(metadata, &rel); err != nil {
		return "", fmt.Errorf("release metadata: %w", err)
	}
	if rel.Tag == "" {
		return "", fmt.Errorf("release has no tag")
	}
	name := "systemd-lsp-" + goos + "-" + arch + ".tar.gz"
	urls := map[string]string{}
	for _, asset := range rel.Assets {
		urls[asset.Name] = asset.URL
	}
	if urls[name] == "" || urls["SHA256SUMS"] == "" {
		return "", fmt.Errorf("release %s is missing %s or SHA256SUMS", rel.Tag, name)
	}
	// Both URLs come from one release response, never separate 'latest' requests.
	sums, err := download(client, urls["SHA256SUMS"], 1<<20)
	if err != nil {
		return "", err
	}
	var expected []byte
	for _, line := range strings.Split(string(sums), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == name {
			if expected != nil {
				return "", fmt.Errorf("duplicate checksum for %s", name)
			}
			expected, err = hex.DecodeString(fields[0])
			if err != nil || len(expected) != sha256.Size {
				return "", fmt.Errorf("invalid checksum for %s", name)
			}
		}
	}
	if expected == nil {
		return "", fmt.Errorf("missing checksum for %s", name)
	}
	archive, err := download(client, urls[name], 32<<20)
	if err != nil {
		return "", err
	}
	actual := sha256.Sum256(archive)
	if !bytes.Equal(expected, actual[:]) {
		return "", fmt.Errorf("checksum mismatch for %s; executable unchanged", name)
	}
	binary, err := unpack(archive)
	if err != nil {
		return "", err
	}
	if err := validateBinary(binary, goos); err != nil {
		return "", err
	}
	// A new inode in the same directory permits atomic replacement of a running
	// Linux executable and avoids overwriting a running Mach-O image in place.
	f, err := os.CreateTemp(filepath.Dir(target), ".systemd-lsp-update-*")
	if err != nil {
		return "", fmt.Errorf("cannot write executable directory: %w", err)
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if _, err = f.Write(binary); err != nil {
		return "", err
	}
	if err = f.Chmod(info.Mode().Perm() | 0o111); err != nil {
		return "", err
	}
	if err = f.Sync(); err != nil {
		return "", err
	}
	if err = f.Close(); err != nil {
		return "", err
	}
	if err = os.Rename(f.Name(), target); err != nil {
		return "", fmt.Errorf("replace executable: %w", err)
	}
	return rel.Tag, nil
}

func download(client *http.Client, url string, limit int64) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "systemd-lsp-updater")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: HTTP %d (%s)", resp.StatusCode, url)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("download exceeds size limit")
	}
	return data, nil
}

func unpack(data []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	tr := tar.NewReader(io.LimitReader(gz, maxBinary+(1<<20)))
	header, err := tr.Next()
	if err != nil {
		return nil, err
	}
	if header.Name != "systemd-lsp" || header.Typeflag != tar.TypeReg || header.Size <= 0 || header.Size > maxBinary {
		return nil, fmt.Errorf("unexpected release archive entry")
	}
	binary, err := io.ReadAll(tr)
	if err != nil {
		return nil, err
	}
	if _, err = tr.Next(); err != io.EOF {
		return nil, fmt.Errorf("unexpected extra archive entry or invalid archive")
	}
	return binary, nil
}

func validateBinary(data []byte, goos string) error {
	if goos == "linux" {
		f, err := elf.NewFile(bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("invalid Linux executable: %w", err)
		}
		defer f.Close()
		if f.Machine != elf.EM_X86_64 || f.Class != elf.ELFCLASS64 {
			return fmt.Errorf("expected Linux x86-64 executable")
		}
	} else {
		f, err := macho.NewFile(bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("invalid macOS executable: %w", err)
		}
		defer f.Close()
		if f.Cpu != macho.CpuArm64 || f.Type != macho.TypeExec {
			return fmt.Errorf("expected macOS arm64 executable")
		}
	}
	return nil
}
