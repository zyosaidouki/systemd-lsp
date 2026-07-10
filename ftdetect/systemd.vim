if exists('g:did_systemd_lsp_filetype')
  finish
endif
let g:did_systemd_lsp_filetype = 1

augroup systemd_lsp_filetype
  autocmd!
  autocmd BufRead,BufNewFile *.service,*.socket,*.timer,*.path,*.mount,*.automount,*.swap,*.target,*.slice,*.scope setfiletype systemd
  autocmd BufRead,BufNewFile *.service.d/*.conf,*.socket.d/*.conf,*.timer.d/*.conf,*.path.d/*.conf,*.mount.d/*.conf,*.automount.d/*.conf,*.swap.d/*.conf,*.target.d/*.conf,*.slice.d/*.conf,*.scope.d/*.conf setfiletype systemd
augroup END
