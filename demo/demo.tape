# Record with: docker run --rm -v "$PWD":/vhs ghcr.io/charmbracelet/vhs demo/demo.tape
# (needs a linux else.sh binary at demo/bin/else.sh — see `make demo`)
Output demo/demo.gif

Set Shell "bash"
Set FontSize 18
Set Width 1100
Set Height 680
Set Padding 24
Set TypingSpeed 55ms
Set Theme "Catppuccin Mocha"
Set WindowBar Colorful

Hide
Type "export PATH=$PWD/demo/bin:$PATH HOME=/tmp/demo-home PS1='\[\e[38;5;117m\]~/code/api\[\e[0m\] \[\e[38;5;141m\]$\[\e[0m\] ' ELSE_SH_SHELL=bash && mkdir -p $HOME && clear"
Enter
Show

Type "else.sh install"
Sleep 300ms
Enter
Sleep 2.2s
Type "source ~/.bashrc && clear"
Enter
Sleep 500ms

Type "# no agent session open, just type what you want:"
Enter
Sleep 600ms
Type "fix the flaky token expiry test"
Sleep 500ms
Enter
Sleep 5.5s

Type "# typos and real commands still behave normally:"
Enter
Sleep 400ms
Type "gti status"
Enter
Sleep 1.8s
Type "git status"
Enter
Sleep 2.5s
