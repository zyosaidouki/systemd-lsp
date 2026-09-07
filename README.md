# systemd-lsp

A small Language Server Protocol implementation for systemd unit files.

It runs over stdio and can be used from Neovim, Vim, gVim, or another LSP
client.

## Features

- Diagnostics for common systemd unit file mistakes
  - keys before any section
  - malformed section headers
  - unknown sections
  - unknown directives in known sections
  - duplicate singleton directives
- Completion for common sections and directives, including syntax and examples
- Embedded generated directive catalog from systemd's parser source
- Optional external catalog with man XML documentation for a specific systemd version
- Value completion for common enum-like directives
- Hover documentation for known directives
- Document symbols for sections and directives
- Template insertion for empty standalone `.service` files; drop-ins and other
  unit types are never populated automatically

## System requirements

- Neovim 0.11 or newer can use its built-in LSP client.
- Vim or gVim 8.0 or newer requires an LSP client plugin. The included
  integration uses `prabirshrestha/vim-lsp` and requires Vim features `+job`,
  `+channel`, timers, lambdas, and JSON support.
- Other editors need an LSP client that can start a server over standard input
  and standard output.
- Linux is the primary target because systemd unit files are normally used on
  Linux. The language server itself is written in pure Go and does not invoke
  `systemd`, `systemctl`, or `systemd-analyze` at runtime.
- A local systemd installation, systemd source tree, man pages, C compiler, and
  systemd development headers are not required. The default directive catalog
  is embedded in the executable.

## Installation requirements

The recommended installation uses the prebuilt [release binaries](#linux-x86-64).
It does not require Go or create a `~/go` directory. Update with
`systemd-lsp update` after installation.

### Optional installation from source

Installing with `go install` requires:

- Go 1.22 or newer
- Network access to download the module from GitHub or a configured Go module
  proxy
- The Go binary installation directory in `PATH`. This is `GOBIN` when set,
  otherwise it is usually `$(go env GOPATH)/bin`.

An operating-system package manager such as `apt`, `dnf`, or `pacman` is not
required by `systemd-lsp`. Go may be installed using a package manager or the
official Go archive. A Vim plugin manager is also optional. Vim/gVim does
require an LSP client plugin, but it can be installed with Vim's built-in
package support as shown below.

Generating an optional catalog for a specific systemd version additionally
requires a checkout of that systemd source tree. `git` and `curl` are used by
the examples in this README, but neither is a runtime dependency of the
language server.

## Install from source (optional)

```sh
go install github.com/zyosaidouki/systemd-lsp/cmd/systemd-lsp@latest
```

If the command installs successfully but your editor cannot find
`systemd-lsp`, add the Go binary installation directory to `PATH`, for example:

```sh
export PATH="$(go env GOPATH)/bin:$PATH"
```

For local development from this repository:

```sh
go install ./cmd/systemd-lsp
```

## Install from Releases (recommended)

Perform installation and editor configuration once per machine. Starting your
editor or updating the server does not require repeating the setup. These
instructions use `~/.local/bin/systemd-lsp` and do not require Go or create
`~/go`.

Choose the download commands for your platform below. To use `systemd-lsp`
from a terminal, add this line once to `~/.zshrc` (zsh) or `~/.bashrc` (bash),
then open a new terminal:

```sh
export PATH="$HOME/.local/bin:$PATH"
```

After installation, configure [Neovim](#neovim) or [Vim/gVim](#vim-and-gvim).
The explicit executable paths in the examples also work when a desktop editor
does not inherit your shell's `PATH`.

### Linux x86-64

The `CI` workflow builds both Linux x86-64 and Apple Silicon binaries in each
run. After CI succeeds on the current `main` commit, it automatically publishes
both archives and `SHA256SUMS` to [GitHub Releases](https://github.com/zyosaidouki/systemd-lsp/releases/latest).
Each release uses a `build-<run number>-<attempt>` tag pointing to the tested
commit. Pull requests do not publish releases. Archives also remain available
in the workflow run's Artifacts section.

Download and install the latest Linux build without GitHub authentication:

```sh
curl -fL --retry 3 -o systemd-lsp-linux-amd64.tar.gz \
  https://github.com/zyosaidouki/systemd-lsp/releases/latest/download/systemd-lsp-linux-amd64.tar.gz
tar -xzf systemd-lsp-linux-amd64.tar.gz
mkdir -p "$HOME/.local/bin"
install -m 755 systemd-lsp "$HOME/.local/bin/systemd-lsp"
```

Add `$HOME/.local/bin` to your `PATH`, or configure your editor to use that
absolute executable path. To build locally (including from macOS):

```sh
mkdir -p dist/linux-amd64
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
  -o dist/linux-amd64/systemd-lsp ./cmd/systemd-lsp
```

### Apple Silicon (macOS arm64)

To build the language server for Apple Silicon, including from Linux:

```sh
mkdir -p dist/darwin-arm64
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -trimpath \
  -o dist/darwin-arm64/systemd-lsp ./cmd/systemd-lsp
```

The `CI` workflow also cross-compiles this binary on the Linux self-hosted
runner. Download and install the latest release on your Mac:

```sh
curl -fL --retry 3 -o systemd-lsp-darwin-arm64.tar.gz \
  https://github.com/zyosaidouki/systemd-lsp/releases/latest/download/systemd-lsp-darwin-arm64.tar.gz
tar -xzf systemd-lsp-darwin-arm64.tar.gz
mkdir -p "$HOME/.local/bin"
install -m 755 systemd-lsp "$HOME/.local/bin/systemd-lsp"
```

Add `$HOME/.local/bin` to your `PATH`, or configure your editor to use that
absolute executable path. The archive preserves the executable permission.
CI tests run on Linux; the macOS binary is cross-compiled, not tested on macOS
by the workflow.

### Update from Releases

```sh
systemd-lsp update
```

This command downloads the latest release for Linux x86-64 or macOS arm64,
verifies its SHA-256 checksum and executable format, and atomically replaces
the invoked executable. Symbolic links are followed to their target. Go and
GitHub authentication are not required. The executable's directory must be
writable by your user; a failed download or validation leaves it unchanged.
Restart your editor's LSP client after updating.

Older binaries without the `update` command need a one-time installation using
the release download commands above. If multiple copies are installed, invoke
the full path used by your editor, for example
`$HOME/.local/bin/systemd-lsp update`. The command updates the server binary;
Vim plugin files are still managed by your plugin manager.

For checksum verification, download `SHA256SUMS` and the archive from the same
tagged release, then use `sha256sum --ignore-missing -c SHA256SUMS` on Linux or
compare `shasum -a 256` output on macOS.

## Setup

Complete these steps after installing the `systemd-lsp` executable:

1. Confirm that the executable is available in the current shell:

   ```sh
   command -v systemd-lsp
   ```

   The command must print the path to the executable. An editor started from a
   desktop menu may use a different `PATH`; use an absolute executable path in
   the editor configuration if necessary. Do not run `systemd-lsp` directly as
   a health check; it communicates over standard input and output and waits for
   an LSP client.

2. Configure one editor integration:

   - Use the [Neovim setup](#neovim) with Neovim's built-in LSP client.
   - Use the [Vim and gVim setup](#vim-and-gvim) with `vim-lsp`.

3. Restart the editor and open a systemd unit file such as
   `example.service`.

4. In the editor, check the detected filetype:

   ```vim
   :set filetype?
   ```

   The result must be `filetype=systemd`. If it is empty or different, add the
   filetype configuration shown in the relevant editor section below.

5. Confirm that the language server is running. In Neovim, run:

   ```vim
   :lua print(vim.inspect(vim.lsp.get_clients({ bufnr = 0 })))
   ```

   In Vim or gVim with `vim-lsp`, run:

   ```vim
   :LspStatus
   ```

   The output should include a running client named `systemd-lsp`. Opening a
   new, empty standalone `.service` file should then insert the default service
   template. Files containing non-whitespace content, drop-ins, and other unit
   types are left unchanged.

### Supported file extensions

The editor starts `systemd-lsp` when one of these extensions is detected as the
`systemd` filetype:

| Extension | Unit-specific section |
| --- | --- |
| `.service` | `[Service]` |
| `.socket` | `[Socket]` |
| `.timer` | `[Timer]` |
| `.path` | `[Path]` |
| `.mount` | `[Mount]` |
| `.automount` | `[Automount]` |
| `.swap` | `[Swap]` |
| `.target` | No unit-specific section |
| `.slice` | `[Slice]` |
| `.scope` | `[Scope]` |

Drop-in files are also supported when the parent directory identifies one of
these unit types. For example, `example.service.d/override.conf` and
`backup.timer.d/schedule.conf` activate the LSP. A general `.conf` file outside
a supported unit drop-in directory does not activate it. Drop-ins receive
completion and diagnostics but are never populated with a template.

Other systemd-related formats such as `.network`, `.netdev`, `.link`, and
`.nspawn` are not unit files and are not handled by this language server.
`.device` units are also not included in the current catalog or editor
filetype rules.

## Neovim

### Setup with lazy.nvim and release binaries

For Neovim 0.11 or newer with lazy.nvim, install the release binary first.
Save the following as `~/.config/nvim/lsp/systemd.lua`:

```lua
return {
  cmd = { vim.fn.expand("~/.local/bin/systemd-lsp") },
  filetypes = { "systemd" },
  root_markers = { ".git" },
  workspace_required = false,
  init_options = { locale = "ja" },
}
```

Add this plugin specification to your lazy.nvim configuration. If your setup
imports `lua/plugins`, save it as
`~/.config/nvim/lua/plugins/systemd.lua`:

```lua
return {
  {
    "zyosaidouki/systemd-lsp",
    lazy = false,
    build = function()
      local output = vim.fn.system({
        vim.fn.expand("~/.local/bin/systemd-lsp"), "update",
      })
      if vim.v.shell_error ~= 0 then
        error(output)
      end
    end,
    config = function()
      vim.lsp.enable("systemd")
    end,
  },
}
```

Run `:Lazy sync`, restart Neovim, and open a `.service` file. Use
`:checkhealth vim.lsp` to inspect the client. The build hook downloads the latest
release whenever lazy.nvim runs the plugin's build step; it does not run on
every editor startup. You can also run `systemd-lsp update` in a terminal at
any time and restart the LSP client afterward.

When migrating an existing configuration, replace its
`build = "go install ./cmd/systemd-lsp"` hook and any `~/go/bin/systemd-lsp`
path with the settings above. Use this configuration or the manual setup
below so that the server is not started twice.

### Manual setup

With Neovim 0.11 or newer:

```lua
vim.api.nvim_create_autocmd("FileType", {
  pattern = "systemd",
  callback = function()
    vim.lsp.start({
      name = "systemd-lsp",
      cmd = { "systemd-lsp" },
      root_dir = vim.fs.root(0, { ".git" }) or vim.fn.getcwd(),
    })
  end,
})
```

Completion and hover documentation is English by default. To show Japanese
documentation:

```lua
vim.api.nvim_create_autocmd("FileType", {
  pattern = "systemd",
  callback = function()
    vim.lsp.start({
      name = "systemd-lsp",
      cmd = { "systemd-lsp" },
      root_dir = vim.fs.root(0, { ".git" }) or vim.fn.getcwd(),
      initialization_options = {
        locale = "ja",
      },
    })
  end,
})
```

Use `locale = "en"` or omit `initialization_options` for English.

To use a generated catalog for a specific systemd version:

```lua
vim.api.nvim_create_autocmd("FileType", {
  pattern = "systemd",
  callback = function()
    vim.lsp.start({
      name = "systemd-lsp",
      cmd = { "systemd-lsp" },
      root_dir = vim.fs.root(0, { ".git" }) or vim.fn.getcwd(),
      initialization_options = {
        catalogPath = "/path/to/systemd-v258-catalog.json",
      },
    })
  end,
})
```

The language server already includes a generated parser catalog for broad
default completion. Use `catalogPath` when you want to replace or enrich it
with catalog data generated from a specific systemd version and its man XML.
You can also set `SYSTEMD_LSP_CATALOG=/path/to/catalog.json` before starting
the language server.

If your Neovim does not detect systemd files automatically, add:

```lua
vim.filetype.add({
  extension = {
    service = "systemd",
    socket = "systemd",
    timer = "systemd",
    path = "systemd",
    mount = "systemd",
    automount = "systemd",
    swap = "systemd",
    target = "systemd",
    slice = "systemd",
    scope = "systemd",
  },
  pattern = {
    [".*/.+%.service%.d/.+%.conf"] = "systemd",
    [".*/.+%.socket%.d/.+%.conf"] = "systemd",
    [".*/.+%.timer%.d/.+%.conf"] = "systemd",
    [".*/.+%.path%.d/.+%.conf"] = "systemd",
    [".*/.+%.mount%.d/.+%.conf"] = "systemd",
    [".*/.+%.automount%.d/.+%.conf"] = "systemd",
    [".*/.+%.swap%.d/.+%.conf"] = "systemd",
    [".*/.+%.target%.d/.+%.conf"] = "systemd",
    [".*/.+%.slice%.d/.+%.conf"] = "systemd",
    [".*/.+%.scope%.d/.+%.conf"] = "systemd",
  },
})
```

## Vim and gVim

Vim and gVim use the same Vimscript configuration. This repository includes
filetype detection and automatic registration for
[`prabirshrestha/vim-lsp`](https://github.com/prabirshrestha/vim-lsp).

Check that Vim has the features needed by `vim-lsp`:

```vim
:echo has('job') && has('channel') && has('timers') && has('lambda') && exists('*json_encode')
```

The result must be `1`.

### With a plugin manager

A plugin manager is optional. For example, with `vim-plug`:

```vim
call plug#begin()
Plug 'prabirshrestha/vim-lsp'
Plug 'zyosaidouki/systemd-lsp'
call plug#end()
```

Install the server from Releases first, then run `:PlugInstall` and restart
Vim or gVim. Update the server with `systemd-lsp update`; plugin updates do not
compile the server or require Go.

### Without a plugin manager

Vim's built-in package support can load both repositories directly:

Install the release binary first using the platform instructions above.

```sh
mkdir -p ~/.vim/pack/lsp/start
git clone --depth 1 https://github.com/prabirshrestha/vim-lsp \
  ~/.vim/pack/lsp/start/vim-lsp
git clone --depth 1 https://github.com/zyosaidouki/systemd-lsp \
  ~/.vim/pack/lsp/start/systemd-lsp
```

No separate package manager is used by this method. The `git` commands may be
replaced by downloading and extracting the two repositories into the same
directories.

### Configuration

Add this to `~/.vimrc` to enable filetype plugins and select Japanese
documentation:

```vim
filetype plugin on
let g:systemd_lsp_locale = 'ja'
let g:systemd_lsp_command = expand('~/.local/bin/systemd-lsp')
```

Use `en` instead of `ja` for English documentation. Completion is available
through Vim's standard omni-completion with `Ctrl-X Ctrl-O`. Hover and document
symbols are available through `:LspHover` and `:LspDocumentSymbol`.

When gVim is started from a desktop menu, it may not inherit the shell's
`PATH`. In that case, set the executable explicitly before the plugins load:

```vim
let g:systemd_lsp_command = expand('~/.local/bin/systemd-lsp')
```

An external catalog can also be selected in `.vimrc`:

```vim
let g:systemd_lsp_catalog_path = '/path/to/systemd-v258-catalog.json'
```

## Development

GitHub Actions runs tests, `go vet`, and a build on the Linux x64 self-hosted
runner for pushes to `main` and pull requests from this repository. Fork pull
requests are skipped on the self-hosted runner. You can also start the `CI`
workflow manually from the Actions tab. The runner must be online and have the
`self-hosted`, `Linux`, and `X64` labels; Go is installed by the workflow using
the version specified in `go.mod`.

```sh
go test ./...
go run ./cmd/systemd-lsp
```

## Generated catalog

systemd adds and removes unit directives between releases. For better coverage,
the language server embeds a generated parser catalog by default. To pin
completion and hover documentation to a specific systemd release, generate a
catalog from the target systemd source tag.

The generated catalog uses:

- `src/core/load-fragment-gperf.gperf.in` for accepted section/directive names
  and parser functions
- `man/*.xml` for completion and hover documentation
- inferred value kinds and enum values for common parser types

Generate a catalog from a checked-out systemd source tree:

```sh
git clone --depth 1 --branch v258 https://github.com/systemd/systemd /tmp/systemd-v258

go run ./cmd/systemd-lsp-generate-catalog \
  -version v258 \
  -man-dir /tmp/systemd-v258/man \
  /tmp/systemd-v258/src/core/load-fragment-gperf.gperf.in \
  > /tmp/systemd-v258-catalog.json

go run ./cmd/systemd-lsp-check-catalog \
  -min-directives 500 \
  /tmp/systemd-v258-catalog.json
```

For a single-file quick check without man-page documentation:

```sh
curl -fL \
  https://raw.githubusercontent.com/systemd/systemd/v258/src/core/load-fragment-gperf.gperf.in \
  -o /tmp/load-fragment-gperf.gperf.in

go run ./cmd/systemd-lsp-generate-catalog \
  -version v258 \
  /tmp/load-fragment-gperf.gperf.in \
  > /tmp/systemd-v258-catalog.json
```

Load the catalog in the language server:

```sh
SYSTEMD_LSP_CATALOG=/tmp/systemd-v258-catalog.json go run ./cmd/systemd-lsp
```

The generator expands the common `EXEC_CONTEXT_CONFIG_ITEMS`,
`CGROUP_CONTEXT_CONFIG_ITEMS`, and `KILL_CONTEXT_CONFIG_ITEMS` macro calls in
systemd's gperf template, then emits JSON containing section, directive, parser
function, inferred value kind, syntax, example, man page, enum values where
known, and whether repeated assignments are normally expected.

The checker prints catalog statistics and fails on obvious catalog problems
such as duplicate section/directive entries, empty names, or a directive count
below the requested minimum.
