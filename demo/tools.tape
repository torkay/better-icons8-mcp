# vhs demo/tools.tape
#
# The full tool surface. `-tools` starts the server against an in-process
# client and lists what it registered, so this is the real list.

Output demo/tools.gif

Set Shell "bash"
Set FontFamily "DejaVu Sans Mono"
Set FontSize 15
Set Width 1500
Set Height 500
Set Padding 26
Set Margin 20
Set MarginFill "#674EFF"
Set BorderRadius 10
Set Theme "Catppuccin Mocha"
Set TypingSpeed 45ms

Hide
Type "export ICONS8_MCP_HOME=/tmp/i8demo PATH=$HOME/go/bin:$PATH PS1='$ ' && clear"
Enter
Sleep 1s
Show

Type "icons8-mcp -tools"
Sleep 500ms
Enter
Sleep 8s
