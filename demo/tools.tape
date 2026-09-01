# vhs demo/tools.tape
#
# The full tool surface. `-tools` starts the server against an in-process
# client and lists what it actually registered, so this is the real list.

Output demo/tools.gif

Set Shell "bash"
Set FontFamily "JetBrainsMono Nerd Font"
Set FontSize 16
Set Width 1500
Set Height 840
Set Padding 30
Set Margin 20
Set MarginFill "#674EFF"
Set BorderRadius 10
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms

Hide
Type "export ICONS8_MCP_HOME=/tmp/i8demo PS1='$ ' && clear"
Enter
Sleep 1s
Show

Type "icons8-mcp -tools"
Sleep 500ms
Enter
Sleep 8s
