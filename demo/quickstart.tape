# vhs demo/quickstart.tape
#
# The whole setup: add the marketplace, install the plugin, then ask Claude
# Code to connect the account. The plugin fetches the server binary for the
# platform on first run, so nothing is installed beforehand.
#
# Steps end on a Wait rather than a Sleep so the recording carries no dead time
# waiting for a network call that already finished.

Output demo/quickstart.gif

Set Shell "bash"
Set FontFamily "DejaVu Sans Mono"
Set FontSize 18
Set Width 1400
Set Height 700
Set Padding 28
Set Margin 20
Set MarginFill "#674EFF"
Set BorderRadius 10
Set Theme "Catppuccin Mocha"
Set TypingSpeed 30ms

Hide
Type "cd /tmp/i8demo-project && export PS1='$ '"
Enter
Type "unset $(env | grep -o '^CLAUDE[A-Z_]*' | tr '\n' ' ')"
Enter
# Start from nothing installed, so the install output in the recording is real.
Type "claude plugin uninstall icons8@better-icons8-mcp >/dev/null 2>&1; claude plugin marketplace remove better-icons8-mcp >/dev/null 2>&1; clear"
Enter
Sleep 3s
Show

# The owner/repo shorthand clones over SSH first and prints a fallback notice.
Type "claude plugin marketplace add https://github.com/torkay/better-icons8-mcp.git"
Sleep 150ms
Enter
Wait+Screen@120s /Successfully added marketplace/
Sleep 800ms

Type "claude plugin install icons8@better-icons8-mcp"
Sleep 150ms
Enter
Wait+Screen@120s /Successfully installed plugin/
Sleep 1200ms

# Claude Code takes the whole screen, so clear before it starts.
Hide
Type "clear"
Enter
Show

Type "claude"
Sleep 150ms
Enter
Wait+Screen@60s /to cycle/
Sleep 600ms

Type "Connect to my icons8 account using the MCP. Don't print my email address."
Sleep 150ms
Enter
Sleep 40s
