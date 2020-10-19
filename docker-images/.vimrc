" An example for a vimrc file.
"
" Maintainer:	Bram Moolenaar <Bram@vim.org>
" Last change:	2004 May 18
"
" To use it, copy it to
"     for Unix and OS/2:  ~/.vimrc
"	      for Amiga:  s:.vimrc
"  for MS-DOS and Win32:  $VIM\_vimrc
"	    for OpenVMS:  sys$login:.vimrc

" When started as "evim", evim.vim will already have done these settings.
if v:progname =~? "evim"
  finish
endif
" Use Vim settings, rather then Vi settings (much better!).
" This must be first, because it changes other options as a side effect.
set nocompatible
colorscheme zellner

" For ruby
set expandtab
set tabstop=2 shiftwidth=2 softtabstop=2
set smartindent
set autoindent
filetype indent on

" allow backspacing over everything in insert mode
set backspace=indent,eol,start

set autoindent		" always set autoindenting on
if has("vms")
  set nobackup		" do not keep a backup file, use versions instead
else			" should be nobackup 
  set backup		" keep a backup file
endif
set history=1000	" keep 50 lines of command line history
set shiftwidth=4	" wciecia
set ruler		" show the cursor position all the time
set showcmd		" display incomplete commands
set incsearch		" do incremental searching
"set syntax=on
set nobackup
"set number
set nowrap
"set sidescroll=10
set tags=./tags,tags,/data/cprogs/tags 
"Morgoth added Shortcuts

map Zc :!ctags %<CR>
map Za :set wrap<CR>
map ZA :set nowrap\|set sidescroll=10<CR>
map ZC :!ctags *.[cChH]*<CR>
map ZR :!ctags -R *.[cChHsSpP]*<CR>
map Z+ :!mkalltags<CR>
map Z9 :rew<CR>
map Zb :first<CR>
map ZB :last<CR>
map Zq :prev<CR>
map ZQ :next<CR>
map Ze :cc<CR>
map Zp :cprev<CR>
map Zn :cnext<CR>
map Zm :w\|mak<CR>
map Zo :w\|!make fast<CR>
map Zg :!gdb<CR>
map Zl :!ls -al<CR>
map Zt :!cat tags\|more<CR>
map Zy {"yy}
map ZK {$ZLi/*<Esc>j}k$ZLi*/<Esc>
map ZJ {$2j$ZLi/*<Esc>}2k$ZLi*/<Esc>
map Zv "yp
map Zd {"yd}
map Zi 0"yy$
map Zj 0"yd$
map Z0 :%s/[ 	][ 	]*$//<CR>
map Z1 :g/^[ 	]*$/d<CR>
map Zk 0i/*<Esc>$a*/<Esc>
map ZL $o<Esc>
map ZN :se nu<CR>
map ZU :se nonu<CR>
map ZS :syntax on<CR>
map ZY :syntax off<CR>
map ZM i#include <mh.h><Esc>ZLZLiint main(int lb, char** par)<Esc>ZLi{<Esc>ZLi return 0;<Esc>ZLi}<Esc>ZLZL
map ZG :/ main(/<CR>
map ZW :g/.*/mo0<CR>
map Zh :%!xxd<CR>
map ZH :%!xxd -r<CR>
map Zr :/morgothjhfat53452msssaq<CR>
map Z2 100j
map Z3 100k
map ZF /FIXME<CR>
map Zf :%s/\(^}$\)/\1\r/g<CR>:%s/\(\_^.*\n\)\(\_^{\n\)/\r\1\2/g<CR>
map Zs :%!sort<CR>
map ZT :1000<CR>
map Zu 100u
map ZD :r !date<CR>
map ZE :e ./Makefile<CR>
map ZI i <Esc>
map Z? :!cat ~/vim/infoall\|more<CR>
map ZP :set syntax=cpp<CR>
map Z4 <C-]>
map Z5 :tfirst<CR>
map Z6 :tprev<CR>
map Z7 :tnext<CR>
map Z8 :tlast<CR>
map ZO Z1ZfZ0
map ZX ZNZSZPZAZCZOZr
map Zx ZNZSZA

ab ZZmain int main(int lb, char**par)<CR>
ab ZZclass class<CR>{<CR><Esc>i public:<CR><CR><Esc>i private:<CR><CR>};<CR>
ab ZZroot <Esc>ZMi
ab ZZinfo <Esc>ZD:r ~/vim/copyright<CR>i
ab ZZstruct struct<CR>{<CR>};<CR>
ab ZZdate <Esc>ZDi
ab ZZhelp <Esc>Z?i
ab ZZasm  <Esc>:r ~/vim/asm<CR>i
ab ZZif  <Esc>:r ~/vim/if<CR>i
ab ZZdo  <Esc>:r ~/vim/do<CR>i
ab ZZwhile  <Esc>:r ~/vim/while<CR>i
ab ZZvfunc  <Esc>:r ~/vim/vfunc<CR>i
ab ZZifunc  <Esc>:r ~/vim/ifunc<CR>i
ab ZZcase  <Esc>:r ~/vim/case<CR>i
ab ZZdebug  <Esc>:r ~/vim/debug<CR>i
ab ZZdbgp  <Esc>:r ~/vim/dbgp<CR>i
ab ZZpri  <Esc>:r ~/vim/pri<CR>i
ab ZZifel  <Esc>:r ~/vim/ifel<CR>i
ab ZZfor  <Esc>:r ~/vim/for<CR>i
ab ZZputs  <Esc>:r ~/vim/puts<CR>i
ab ZZxlib  <Esc>:r ~/vim/xlib<CR>i
ab ZZmake  <Esc>:r ~/vim/make<CR>i



" For Win32 GUI: remove 't' flag from 'guioptions': no tearoff menu entries
" let &guioptions = substitute(&guioptions, "t", "", "g")

" Don't use Ex mode, use Q for formatting
map Q gq

" Make p in Visual mode replace the selected text with the "" register.
vnoremap p <Esc>:let current_reg = @"<CR>gvs<C-R>=current_reg<CR><Esc>

" This is an alternative that also works in block mode, but the deleted
" text is lost and it only works for putting the current register.
"vnoremap p "_dp

" Switch syntax highlighting on, when the terminal has colors
" Also switch on highlighting the last used search pattern.
if &t_Co > 2 || has("gui_running")
  syntax on
  set hlsearch
endif

" Only do this part when compiled with support for autocommands.
if has("autocmd")

  " Enable file type detection.
  " Use the default filetype settings, so that mail gets 'tw' set to 72,
  " 'cindent' is on in C files, etc.
  " Also load indent files, to automatically do language-dependent indenting.
  filetype plugin indent on

  " For all text files set 'textwidth' to 78 characters.
  autocmd FileType text setlocal textwidth=0

  " When editing a file, always jump to the last known cursor position.
  " Don't do it when the position is invalid or when inside an event handler
  " (happens when dropping a file on gvim).
  autocmd BufReadPost *
    \ if line("'\"") > 0 && line("'\"") <= line("$") |
    \   exe "normal g`\"" |
    \ endif

endif " has("autocmd")
set mouse=
syntax on
