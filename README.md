# systemd-lsp

A small Language Server Protocol implementation for systemd unit files.

It runs over stdio and is intended to be easy to use from Neovim.

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

- An editor or client that can start an LSP server over standard input and
  standard output. The configuration below requires Neovim 0.11 or newer.
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

Generating an optional catalog for a specific systemd version additionally
requires a checkout of that systemd source tree. `git` and `curl` are used by
the examples in this README, but neither is a runtime dependency of the
language server.

## Install

```sh
go install github.com/zyosaidouki/systemd-lsp/cmd/systemd-lsp@latest
```

If the command installs successfully but Neovim cannot find `systemd-lsp`, add
the Go binary installation directory to `PATH`, for example:

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
    [".*/systemd/.+%.d/.+%.conf"] = "systemd",
  },
})
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
