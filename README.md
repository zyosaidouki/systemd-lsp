# systemd-lsp

English | [日本語](README.ja.md)

A Vim/Neovim plugin providing completion, diagnostics, and hover documentation for systemd unit files.

[Linux x86-64 setup](#linux) | [macOS Apple Silicon setup](#macos)

<a id="linux"></a>

## Linux x86-64 setup

### 1. Check your editor and tools

- Neovim: 0.11 or newer
- Vim/gVim: 8.0 or newer with jobs, channels, timers, lambdas, and JSON support

Run these commands in a terminal:

```sh
uname -m
command -v git curl tar sh sha256sum
```

Check that `uname -m` prints `x86_64` and every listed command has a path.
Install any missing tools using your OS package manager. The first launch needs network access to GitHub.

### 2. Install the editor plugin

Follow the instructions for your editor. If you already use lazy.nvim or vim-plug,
use the [plugin manager configuration](#plugin-managers) instead of the clone commands below.

#### Neovim

Run in a terminal:

```sh
mkdir -p ~/.config/nvim/pack/lsp/start
git clone https://github.com/zyosaidouki/systemd-lsp \
  ~/.config/nvim/pack/lsp/start/systemd-lsp
```

To select Japanese documentation, add this to `~/.config/nvim/init.lua` (create the file if needed):

```lua
vim.g.systemd_lsp_locale = "ja"
```

#### Vim/gVim

Install the plugin and its LSP client in a terminal:

```sh
mkdir -p ~/.vim/pack/lsp/start
git clone https://github.com/prabirshrestha/vim-lsp \
  ~/.vim/pack/lsp/start/vim-lsp
git clone https://github.com/zyosaidouki/systemd-lsp \
  ~/.vim/pack/lsp/start/systemd-lsp
```

Add this to `~/.vimrc` (create the file if needed):

```vim
filetype plugin on
let g:systemd_lsp_locale = 'ja'
```

### 3. Start the editor and verify setup

Restart your editor and run this inside it:

```vim
:edit example.service
```

Wait for the first automatic binary download to complete, then run:

```vim
:set filetype?
```

The result should be `filetype=systemd`. In Neovim, check the LSP connection with:

```vim
:lua print(vim.lsp.get_clients({ name = "systemd-lsp" }))
```

A listed client confirms the connection. In Vim/gVim, run `:LspStatus` and
check that `systemd-lsp` is `running`.

<a id="macos"></a>

## macOS Apple Silicon setup

### 1. Check your editor and tools

- Neovim: 0.11 or newer
- Vim/gVim: 8.0 or newer with jobs, channels, timers, lambdas, and JSON support

Run these commands in a terminal:

```sh
uname -m
command -v git curl tar sh shasum
```

Check that `uname -m` prints `arm64` and every listed command has a path.
Install any missing tools using your OS package manager. The first launch needs network access to GitHub.

### 2. Install the editor plugin

Follow the instructions for your editor. If you already use lazy.nvim or vim-plug,
use the [plugin manager configuration](#plugin-managers) instead of the clone commands below.

#### Neovim

Run in a terminal:

```sh
mkdir -p ~/.config/nvim/pack/lsp/start
git clone https://github.com/zyosaidouki/systemd-lsp \
  ~/.config/nvim/pack/lsp/start/systemd-lsp
```

To select Japanese documentation, add this to `~/.config/nvim/init.lua` (create the file if needed):

```lua
vim.g.systemd_lsp_locale = "ja"
```

#### Vim/gVim

Install the plugin and its LSP client in a terminal:

```sh
mkdir -p ~/.vim/pack/lsp/start
git clone https://github.com/prabirshrestha/vim-lsp \
  ~/.vim/pack/lsp/start/vim-lsp
git clone https://github.com/zyosaidouki/systemd-lsp \
  ~/.vim/pack/lsp/start/systemd-lsp
```

Add this to `~/.vimrc` (create the file if needed):

```vim
filetype plugin on
let g:systemd_lsp_locale = 'ja'
```

### 3. Start the editor and verify setup

Restart your editor and run this inside it:

```vim
:edit example.service
```

Wait for the first automatic binary download to complete, then run:

```vim
:set filetype?
```

The result should be `filetype=systemd`. In Neovim, check the LSP connection with:

```vim
:lua print(vim.lsp.get_clients({ name = "systemd-lsp" }))
```

A listed client confirms the connection. In Vim/gVim, run `:LspStatus` and
check that `systemd-lsp` is `running`.

<a id="plugin-managers"></a>

## Using an existing plugin manager

These settings apply to both Linux and macOS. Add the configuration for the manager you already use.

### lazy.nvim

If your configuration imports `lua/plugins`, save this as
`~/.config/nvim/lua/plugins/systemd.lua`. Otherwise, add the inner plugin specification to your existing plugin list.

```lua
return {
  {
    "zyosaidouki/systemd-lsp",
    lazy = false,
    init = function()
      vim.g.systemd_lsp_locale = "ja"
    end,
  },
}
```

Run `:Lazy sync`, then follow step 3 above to restart and verify setup.

### vim-plug

Add these two lines between the existing `plug#begin()` and `plug#end()` calls in `~/.vimrc`:

```vim
Plug 'prabirshrestha/vim-lsp'
Plug 'zyosaidouki/systemd-lsp'
```

Also add these settings to the same file:

```vim
filetype plugin on
let g:systemd_lsp_locale = 'ja'
```

Run `:PlugInstall`, then follow step 3 above to restart and verify setup.

## Updating

Run inside your editor:

```vim
:SystemdLspUpdate
```

This downloads the latest release and restarts the LSP client. The binary is stored at `bin/systemd-lsp` inside the plugin.

## Troubleshooting

- Download fails: inspect `:messages`, then retry with `:SystemdLspUpdate`.
- Two LSP clients start: remove the old `lsp/systemd.lua`, `vim.lsp.enable("systemd")`, and other custom startup configuration.
- Migrating an older installation: remove `go install` hooks, fixed binary paths, and custom `SystemdLspUpdate` definitions, then use the configuration above.

## Language and external catalogs

The setup examples select Japanese documentation. Change `ja` to `en` for English.
To use an external catalog, configure it before the plugin loads:

Neovim (`init.lua` or lazy.nvim `init`):

```lua
vim.g.systemd_lsp_catalog_path = "/path/to/catalog.json"
```

Vim/gVim (`~/.vimrc`):

```vim
let g:systemd_lsp_catalog_path = '/path/to/catalog.json'
```

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

## Supported file extensions

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
