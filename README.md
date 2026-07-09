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
- Completion for common sections and directives
- Value completion for common enum-like directives
- Hover documentation for known directives
- Document symbols for sections and directives

## Install

```sh
go install github.com/zako/systemd-lsp/cmd/systemd-lsp@latest
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
