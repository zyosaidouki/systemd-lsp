if exists('g:loaded_systemd_lsp_vim')
  finish
endif
let g:loaded_systemd_lsp_vim = 1

if has('nvim')
  if has('nvim-0.11')
    lua require('systemd-lsp').setup()
  else
    echoerr 'systemd-lsp requires Neovim 0.11 or newer'
  endif
  finish
endif

if !has('job') || !has('channel') || !has('timers') || !has('lambda') || !exists('*json_encode')
  finish
endif

let s:root = expand('<sfile>:p:h:h')
let s:binary = s:root . '/bin/systemd-lsp'
let s:updating = 0

function! s:initialization_options() abort
  let options = {'locale': get(g:, 'systemd_lsp_locale', 'en')}
  let catalog_path = get(g:, 'systemd_lsp_catalog_path', '')
  if !empty(catalog_path)
    let options.catalogPath = catalog_path
  endif
  return options
endfunction

function! s:register_server() abort
  let command = s:binary
  if !exists('*lsp#register_server') || !executable(command)
    return
  endif
  if exists('*lsp#is_valid_server_name') && lsp#is_valid_server_name('systemd-lsp')
    return
  endif
  call lsp#register_server({
        \ 'name': 'systemd-lsp',
        \ 'cmd': {server_info -> [command]},
        \ 'allowlist': ['systemd'],
        \ 'initialization_options': s:initialization_options(),
        \ })
endfunction

function! s:on_lsp_buffer_enabled() abort
  if &l:filetype ==# 'systemd'
    setlocal omnifunc=lsp#complete
  endif
endfunction

augroup systemd_lsp_vim
  autocmd!
  autocmd User lsp_setup call s:register_server()
  autocmd User lsp_buffer_enabled call s:on_lsp_buffer_enabled()
augroup END

function! s:reconnect(attempt, timer) abort
  if exists('*lsp#get_server_status') && lsp#get_server_status('systemd-lsp') ==# 'running'
    if a:attempt < 50
      call timer_start(100, function('s:reconnect', [a:attempt + 1]))
    else
      echom 'systemd-lsp updated; restart Vim to reconnect LSP'
    endif
    return
  endif
  call s:register_server()
  for window in getwininfo()
    if getbufvar(window.bufnr, '&filetype') ==# 'systemd'
      if exists('*lsp#activate')
        call win_execute(window.winid, 'call lsp#activate()')
      else
        call win_execute(window.winid, 'doautocmd <nomodeline> lsp BufReadPost')
      endif
    endif
  endfor
  echom 'systemd-lsp updated and LSP restart requested'
endfunction

function! s:installed(job, status) abort
  let s:updating = 0
  let output = filereadable(s:log) ? join(readfile(s:log), "\n") : ''
  call delete(s:log)
  if a:status != 0
    echohl ErrorMsg
    echom 'systemd-lsp installation failed: ' . output
    echohl None
    return
  endif
  if exists('*lsp#stop_server')
    call lsp#stop_server('systemd-lsp')
  endif
  call timer_start(100, function('s:reconnect', [0]))
endfunction

function! s:update() abort
  if s:updating
    echom 'systemd-lsp update is already running'
    return
  endif
  let s:updating = 1
  let s:log = tempname()
  echom 'Downloading systemd-lsp release...'
  let s:job = job_start(['sh', s:root . '/scripts/install-release.sh'], {
        \ 'out_io': 'file', 'out_name': s:log, 'err_io': 'out',
        \ 'exit_cb': function('s:installed'),
        \ })
  if job_status(s:job) ==# 'fail'
    let s:updating = 0
    call delete(s:log)
    echoerr 'Unable to start systemd-lsp installer'
  endif
endfunction

command! SystemdLspUpdate call s:update()
if !executable(s:binary)
  call timer_start(0, {timer -> s:update()})
endif
