"""Check update notifications and restart decisions in isolated editor sessions."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]


class EditorUpdateTest(unittest.TestCase):
    def test_vim(self):
        self.check_editor('vim')

    def test_neovim(self):
        self.check_editor('nvim')

    def check_editor(self, editor):
        if not shutil.which(editor):
            self.skipTest(f'{editor} is not installed')
        for scenario, message in (
            ('updated', 'systemd-lsp updated: build-41-1 -> build-42-1'),
            ('already', 'systemd-lsp already: build-42-1'),
            ('failed', 'checksum mismatch'),
        ):
            with self.subTest(scenario=scenario), tempfile.TemporaryDirectory() as temp:
                root = Path(temp) / 'plugin with spaces'
                shutil.copytree(ROOT / 'plugin', root / 'plugin')
                shutil.copytree(ROOT / 'lua', root / 'lua')
                (root / 'bin').mkdir()
                binary = root / 'bin/systemd-lsp'
                binary.write_text('#!/bin/sh\nexit 0\n')
                binary.chmod(0o755)
                (root / 'scripts').mkdir()
                (root / 'scripts/install-release.sh').write_text(
                    '#!/bin/sh\nprintf "%s\\n" "asset: OK" "$TEST_UPDATE_MESSAGE"\n'
                    'exit "$TEST_UPDATE_STATUS"\n')
                result_file = Path(temp) / 'result.json'
                env = dict(os.environ, TEST_PLUGIN_ROOT=str(root), TEST_RESULT=str(result_file),
                           TEST_UPDATE_MESSAGE=message, TEST_UPDATE_STATUS='1' if scenario == 'failed' else '0')
                if editor == 'nvim':
                    script = Path(temp) / 'test.lua'
                    script.write_text('''
local messages, stops = {}, 0
vim.notify = function(message) table.insert(messages, message) end
vim.lsp.config = function() end
vim.lsp.enable = function() end
vim.lsp.get_clients = function()
  return {{ stop = function() stops = stops + 1 end, is_stopped = function() return true end }}
end
vim.opt.runtimepath:prepend(vim.env.TEST_PLUGIN_ROOT)
require('systemd-lsp').setup()
vim.cmd('SystemdLspUpdate')
assert(vim.wait(3000, function() return #messages >= 2 end), 'update timed out')
vim.fn.writefile({vim.json.encode({messages = table.concat(messages, '\\n'), stops = stops})}, vim.env.TEST_RESULT)
vim.cmd('qa!')
''')
                    args = ['nvim', '--headless', '-u', 'NONE', '-i', 'NONE', '-l', str(script)]
                else:
                    (root / 'autoload').mkdir()
                    (root / 'autoload/lsp.vim').write_text('''
function! lsp#stop_server(name) abort
  let g:stops += 1
endfunction
function! lsp#get_server_status(name) abort
  return 'stopped'
endfunction
''')
                    script = Path(temp) / 'test.vim'
                    script.write_text(r'''
set nocompatible
let g:stops = 0
execute 'source ' . fnameescape($TEST_PLUGIN_ROOT . '/autoload/lsp.vim')
execute 'source ' . fnameescape($TEST_PLUGIN_ROOT . '/plugin/systemd-lsp.vim')
SystemdLspUpdate
let started = reltime()
while execute('messages') !~# 'systemd-lsp \(updated\|already:\|installation failed\)' && reltimefloat(reltime(started)) < 3
  sleep 10m
endwhile
call writefile([json_encode({'messages': execute('messages'), 'stops': g:stops})], $TEST_RESULT)
qa!
''')
                    args = ['vim', '-Nu', 'NONE', '-i', 'NONE', '-n', '-es', '-S', str(script)]
                completed = subprocess.run(args, env=env, capture_output=True, timeout=10)
                self.assertEqual(completed.returncode, 0, completed.stderr)
                result = json.loads(result_file.read_text())
                self.assertIn(message, result['messages'])
                self.assertEqual(result['stops'], 1 if scenario == 'updated' else 0)
                if scenario == 'already':
                    self.assertNotIn('restart', result['messages'])
                if scenario == 'failed':
                    self.assertIn('installation failed', result['messages'])


if __name__ == '__main__':
    unittest.main()
