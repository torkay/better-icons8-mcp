# vhs demo/quickstart.tape
#
# Install, point it at a browser session, confirm the tools are wired up.
# The cookie dump used here is demo/cookies.example.json, which carries a
# fabricated JWT for you@example.com, so no real session is recorded.

Output demo/quickstart.gif

Set Shell "bash"
Set FontFamily "DejaVu Sans Mono"
Set FontSize 20
Set Width 1280
Set Height 380
Set Padding 30
Set Margin 20
Set MarginFill "#674EFF"
Set BorderRadius 10
Set Theme "Catppuccin Mocha"
Set TypingSpeed 45ms

Hide
Type "mkdir -p /tmp/i8quick && cp demo/cookies.example.json /tmp/i8quick/cookies.json && cd /tmp/i8quick"
Enter
Type "export ICONS8_MCP_HOME=/tmp/i8demo PATH=$HOME/go/bin:$PATH PS1='$ ' && rm -rf /tmp/i8demo && clear"
Enter
Sleep 1s
Show

Type "go install github.com/torkay/better-icons8-mcp/cmd/icons8-mcp@latest"
Sleep 500ms
Enter
Sleep 6s

Type "icons8-mcp -import cookies.json"
Sleep 500ms
Enter
Sleep 3s

Type "icons8-mcp -tools | tail -3"
Sleep 500ms
Enter
Sleep 3s

Type "claude mcp add icons8 -s user -- icons8-mcp"
Sleep 4s
