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
- Template insertion for empty `.service` files

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

## Install

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

## Neovim

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
Plug 'zyosaidouki/systemd-lsp', { 'do': 'go install ./cmd/systemd-lsp' }
call plug#end()
```

Run `:PlugInstall`, then restart Vim or gVim.

### Without a plugin manager

Vim's built-in package support can load both repositories directly:

```sh
go install github.com/zyosaidouki/systemd-lsp/cmd/systemd-lsp@latest

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
```

Use `en` instead of `ja` for English documentation. Completion is available
through Vim's standard omni-completion with `Ctrl-X Ctrl-O`. Hover and document
symbols are available through `:LspHover` and `:LspDocumentSymbol`.

When gVim is started from a desktop menu, it may not inherit the shell's
`PATH`. In that case, set the executable explicitly before the plugins load:

```vim
let g:systemd_lsp_command = expand('~/go/bin/systemd-lsp')
```

An external catalog can also be selected in `.vimrc`:

```vim
let g:systemd_lsp_catalog_path = '/path/to/systemd-v258-catalog.json'
```

## Development

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
