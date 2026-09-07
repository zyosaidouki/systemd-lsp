local M = {}
local source = debug.getinfo(1, "S").source:sub(2)
local root = vim.fn.fnamemodify(source, ":p:h:h:h")
local binary = root .. "/bin/systemd-lsp"
local updating = false

local function enable()
  vim.lsp.config("systemd-lsp", {
    cmd = { binary },
    filetypes = { "systemd" },
    root_markers = { ".git" },
    workspace_required = false,
    init_options = {
      locale = vim.g.systemd_lsp_locale or "en",
      catalogPath = vim.g.systemd_lsp_catalog_path,
    },
  })
  vim.lsp.enable("systemd-lsp")
end

local function restart()
  local clients = vim.lsp.get_clients({ name = "systemd-lsp" })
  vim.lsp.enable("systemd-lsp", false)
  for _, client in ipairs(clients) do client:stop(true) end
  local attempts = 0
  local function finish()
    for _, client in ipairs(clients) do
      if not client:is_stopped() then
        attempts = attempts + 1
        if attempts < 50 then vim.defer_fn(finish, 100); return end
        vim.notify("systemd-lsp updated; restart the editor to reconnect LSP", vim.log.levels.WARN)
        return
      end
    end
    enable()
    vim.notify("systemd-lsp updated and LSP restarted")
  end
  finish()
end

function M.update()
  if updating then vim.notify("systemd-lsp update is already running"); return end
  updating = true
  vim.notify("Downloading systemd-lsp release…")
  local ok, err = pcall(vim.system, { "sh", root .. "/scripts/install-release.sh" }, { text = true }, function(result)
    vim.schedule(function()
      updating = false
      if result.code ~= 0 then
        vim.notify("systemd-lsp installation failed:\n" .. (result.stderr or "") .. (result.stdout or ""), vim.log.levels.ERROR)
        return
      end
      restart()
    end)
  end)
  if not ok then
    updating = false
    vim.notify(tostring(err), vim.log.levels.ERROR)
  end
end

function M.setup()
  if M.initialized then return end
  M.initialized = true
  vim.api.nvim_create_user_command("SystemdLspUpdate", M.update, { desc = "Install/update systemd-lsp and restart LSP" })
  if vim.fn.executable(binary) == 1 then enable() else M.update() end
end

return M
