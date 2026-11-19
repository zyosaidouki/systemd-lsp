package config

import (
	"os"
	"path/filepath"
	"testing"
)

func env(values map[string]string) func(string) string {
	return func(name string) string { return values[name] }
}

func TestHome(t *testing.T) {
	for _, tt := range []struct {
		name string
		env  map[string]string
		want string
	}{
		{"XDG_CONFIG_HOME wins", map[string]string{"XDG_CONFIG_HOME": "/xdg/config", "HOME": "/home/u"}, "/xdg/config"},
		{"trailing slash", map[string]string{"XDG_CONFIG_HOME": "/xdg/config/", "HOME": "/home/u"}, "/xdg/config"},
		{"unset", map[string]string{"HOME": "/home/u"}, "/home/u/.config"},
		{"empty", map[string]string{"XDG_CONFIG_HOME": "", "HOME": "/home/u"}, "/home/u/.config"},
		{"relative is ignored", map[string]string{"XDG_CONFIG_HOME": "config", "HOME": "/home/u"}, "/home/u/.config"},
		{"nothing usable", map[string]string{"XDG_CONFIG_HOME": "config"}, ""},
	} {
		if got := Home(env(tt.env)); got != tt.want {
			t.Errorf("%s: Home() = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func writeConfig(t *testing.T, home, content string) {
	t.Helper()
	path := Path(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadReadsFileBelowXDGConfigHome(t *testing.T) {
	xdg, home := t.TempDir(), t.TempDir()
	writeConfig(t, xdg, `{"locale": "ja", "catalogPath": "/opt/catalog.json", "future": true}`)
	writeConfig(t, filepath.Join(home, ".config"), `{"locale": "en"}`)

	got, err := Load(env(map[string]string{"XDG_CONFIG_HOME": xdg, "HOME": home}))
	if err != nil {
		t.Fatal(err)
	}
	want := Settings{Locale: "ja", CatalogPath: "/opt/catalog.json", UserUnitDir: filepath.Join(xdg, "systemd", "user")}
	if got != want {
		t.Fatalf("Load() = %#v, want %#v", got, want)
	}
}

func TestLoadFallsBackToHomeConfig(t *testing.T) {
	home := t.TempDir()
	base := filepath.Join(home, ".config")
	writeConfig(t, base, `{"catalogPath": "catalogs/v257.json"}`)

	got, err := Load(env(map[string]string{"HOME": home}))
	if err != nil {
		t.Fatal(err)
	}
	want := Settings{
		CatalogPath: filepath.Join(base, "systemd-lsp", "catalogs", "v257.json"),
		UserUnitDir: filepath.Join(base, "systemd", "user"),
	}
	if got != want {
		t.Fatalf("Load() = %#v, want %#v", got, want)
	}
}

func TestLoadWithoutFile(t *testing.T) {
	xdg := t.TempDir()
	got, err := Load(env(map[string]string{"XDG_CONFIG_HOME": xdg}))
	if err != nil {
		t.Fatal(err)
	}
	if want := (Settings{UserUnitDir: filepath.Join(xdg, "systemd", "user")}); got != want {
		t.Fatalf("Load() = %#v, want %#v", got, want)
	}
}

func TestLoadReportsInvalidFile(t *testing.T) {
	xdg := t.TempDir()
	writeConfig(t, xdg, `{"locale": `)
	got, err := Load(env(map[string]string{"XDG_CONFIG_HOME": xdg}))
	if err == nil {
		t.Fatal("Load() returned no error for malformed JSON")
	}
	if want := (Settings{UserUnitDir: filepath.Join(xdg, "systemd", "user")}); got != want {
		t.Fatalf("Load() = %#v, want %#v", got, want)
	}
}

func TestLoadWithoutEnvironment(t *testing.T) {
	got, err := Load(env(nil))
	if err != nil || got != (Settings{}) {
		t.Fatalf("Load() = %#v, %v; want zero settings", got, err)
	}
}
