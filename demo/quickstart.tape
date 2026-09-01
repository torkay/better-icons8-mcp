# vhs demo/quickstart.tape
#
# Install, point it at a browser session, confirm the tools are wired up.
# The cookie dump used here is demo/cookies.example.json, which carries a
# fabricated JWT for you@example.com, so no real session is recorded.

Output demo/quickstart.gif

Set Shell "bash"
Set FontFamily "JetBrainsMono Nerd Font"
Set FontSize 20
Set Width 1280
Set Height 700
Set Padding 30
Set Margin 20
Set MarginFill "#674EFF"
Set BorderRadius 10
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms

Hide
Type "export ICONS8_MCP_HOME=/tmp/i8demo PS1='$ ' && rm -rf /tmp/i8demo && clear"
Enter
Sleep 1s
Show

Type "go install github.com/torkay/icons8-mcp-server/cmd/icons8-mcp@latest"
Sleep 500ms
Enter
Sleep 12s

Type "icons8-mcp -import cookies.json"
Sleep 500ms
Enter
Sleep 3s

Type "icons8-mcp -tools | tail -4"
Sleep 500ms
Enter
Sleep 4s

Type "claude mcp add icons8 -s user -- icons8-mcp"
Sleep 3s
