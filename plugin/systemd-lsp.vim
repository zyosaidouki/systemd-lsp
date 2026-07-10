if exists('g:loaded_systemd_lsp_vim')
  finish
endif
let g:loaded_systemd_lsp_vim = 1

if !has('job') || !has('channel') || !has('timers') || !has('lambda') || !exists('*json_encode')
  finish
endif

function! s:initialization_options() abort
  let options = {'locale': get(g:, 'systemd_lsp_locale', 'en')}
  let catalog_path = get(g:, 'systemd_lsp_catalog_path', '')
  if !empty(catalog_path)
    let options.catalogPath = catalog_path
  endif
  return options
endfunction

function! s:register_server() abort
  let command = get(g:, 'systemd_lsp_command', 'systemd-lsp')
  if !exists('*lsp#register_server') || !executable(command)
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
