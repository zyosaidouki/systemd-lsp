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

## Installation and updates

Install the editor plugin using the [Neovim](#neovim) or
[Vim/gVim](#vim-and-gvim) instructions below. On first load, the plugin
asynchronously downloads the latest GitHub Release into its own
`bin/systemd-lsp` directory and starts the LSP client. There is no separate
binary installation, PATH configuration, Go toolchain, or `~/go` directory.
The executable is not installed in `~/.local/bin`.

Prebuilt releases support Linux x86-64 and macOS Apple Silicon (arm64).
Installation requires `sh`, `curl`, `tar`, and either `sha256sum` or `shasum`,
network access to GitHub, and write access to the plugin directory. GitHub
sign-in is not required. Checksums are verified before the executable is
replaced. A failed download or verification leaves the existing binary intact.

To update from Vim or Neovim:

```vim
:SystemdLspUpdate
```

The same command also retries a failed first installation. After a successful
installation or update, the plugin starts or restarts its LSP client. You do
not need to type an executable path or shell command. Ordinary editor starts
reuse the installed binary and do not check for updates. Removing the plugin
also removes its managed executable; if a plugin manager cleans the ignored
`bin/` directory, the next load downloads it again.

CI publishes Linux and Mac archives plus `SHA256SUMS` to
[GitHub Releases](https://github.com/zyosaidouki/systemd-lsp/releases/latest)
after tests and builds succeed on the current `main` commit. Each release has
a `build-<run number>-<attempt>` tag. Pull requests do not publish releases.

### Migrating an existing setup

Remove custom `SystemdLspUpdate` command definitions, `go install` build hooks,
and hard-coded binary paths from your editor configuration. For Neovim, also
remove the old `lsp/systemd.lua` configuration and `vim.lsp.enable("systemd")`
call; the plugin now registers and enables its own `systemd-lsp` client.
Use the minimal plugin specification below. Existing locale and catalog
preferences can be set using the documented global options.

Old standalone binaries can be removed after confirming the managed client
works. The plugin does not delete or modify other installations or change
Go's global configuration.

## Setup verification

Configuration is needed only once per machine. After installing the plugin,
open a `.service` file and wait for the first download to finish. Check:

```vim
:set filetype?
```

The result should be `systemd`. In Neovim, inspect the managed client:

```vim
:lua print(vim.lsp.get_clients({ name = "systemd-lsp" }))
```

In Vim/gVim, use `:LspStatus`. Opening a new, empty standalone `.service` file
should insert a template. Existing files and drop-ins are not populated.
If installation fails, inspect `:messages` and retry `:SystemdLspUpdate`.

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

Requires Neovim 0.11 or newer. With lazy.nvim, add this plugin specification
(for configurations importing `lua/plugins`, save it as
`~/.config/nvim/lua/plugins/systemd.lua`):

```lua
return {
  {
    "zyosaidouki/systemd-lsp",
    lazy = false,
    init = function()
      vim.g.systemd_lsp_locale = "ja" -- optional; default is English
    end,
  },
}
```

Run `:Lazy sync` and restart Neovim. No `build` hook, custom update command,
`lsp/systemd.lua`, or `vim.lsp.enable` call is needed. The plugin configures
filetype detection and its built-in LSP client automatically.

Without a plugin manager, clone into Neovim's native package directory:

```sh
git clone https://github.com/zyosaidouki/systemd-lsp \
  ~/.config/nvim/pack/lsp/start/systemd-lsp
```

Restart Neovim. To select a locale or an optional external directive catalog,
set these in `init.lua` before plugins load (or in lazy.nvim's `init` callback):

```lua
vim.g.systemd_lsp_locale = "ja" -- use "en" for English
vim.g.systemd_lsp_catalog_path = "/path/to/catalog.json" -- optional
```

## Vim and gVim

Requires Vim/gVim 8.0 or newer with `+job`, `+channel`, timers, lambdas, and JSON
support, and the `prabirshrestha/vim-lsp` client plugin. With vim-plug, put this
in `~/.vimrc`:

```vim
filetype plugin on
let g:systemd_lsp_locale = 'ja' " optional; default is English
call plug#begin()
Plug 'prabirshrestha/vim-lsp'
Plug 'zyosaidouki/systemd-lsp'
call plug#end()
```

Run `:PlugInstall`, then restart Vim/gVim. Do not add a `go install` hook or set
`g:systemd_lsp_command`: the plugin uses its own managed binary.

Without a plugin manager, install both native packages:

```sh
mkdir -p ~/.vim/pack/lsp/start
git clone https://github.com/prabirshrestha/vim-lsp \
  ~/.vim/pack/lsp/start/vim-lsp
git clone https://github.com/zyosaidouki/systemd-lsp \
  ~/.vim/pack/lsp/start/systemd-lsp
```

Add `filetype plugin on` to `~/.vimrc` and restart. Both installation methods
provide `:SystemdLspUpdate`. Completion uses `Ctrl-X Ctrl-O`; hover and symbols
are available through `:LspHover` and `:LspDocumentSymbol`.

For an optional external catalog, set this before plugins load:

```vim
let g:systemd_lsp_catalog_path = '/path/to/catalog.json'
```

## Development

Building from source is optional and requires Go 1.26 or newer. To build the
managed executable locally:

```sh
mkdir -p bin
go build -o bin/systemd-lsp ./cmd/systemd-lsp
```

Other LSP clients can use a manually downloaded release or a source build.
The standalone executable still supports `systemd-lsp update`; editor users
use `:SystemdLspUpdate` instead.

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
