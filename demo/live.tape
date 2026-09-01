# vhs demo/live.tape
#
# The search half of the smoke suite, run against the live Icons8 API through
# the built binary over stdio, the same path an MCP host takes.

Output demo/live.gif

Set Shell "bash"
Set FontFamily "DejaVu Sans Mono"
Set FontSize 18
Set Width 1400
Set Height 420
Set Padding 30
Set Margin 20
Set MarginFill "#674EFF"
Set BorderRadius 10
Set Theme "Catppuccin Mocha"
Set TypingSpeed 45ms

Hide
Type "export PS1='$ ' && clear"
Enter
Sleep 1s
Show

Type "go run ./cmd/smoke -bin ./dist/icons8-mcp -only search"
Sleep 500ms
Enter
Sleep 14s
