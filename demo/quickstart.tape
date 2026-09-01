# vhs demo/quickstart.tape
#
# The whole setup: add the marketplace, install the plugin, then ask Claude
# Code to connect the account. The plugin fetches the server binary for the
# platform on first run, so nothing is installed beforehand.

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
Set TypingSpeed 40ms

Hide
Type "cd /tmp/i8demo-project && export PS1='$ '"
Enter
Type "unset $(env | grep -o '^CLAUDE[A-Z_]*' | tr '\n' ' ') && clear"
Enter
Sleep 1s
Show

Type "claude plugin marketplace add torkay/better-icons8-mcp"
Sleep 400ms
Enter
Sleep 14s

Type "claude plugin install icons8@better-icons8-mcp"
Sleep 400ms
Enter
Sleep 12s

Type "claude"
Sleep 400ms
Enter
Sleep 12s

Type "connect my icons8 account and tell me which asset types the licence covers. Don't print my email address."
Sleep 400ms
Enter
Sleep 45s
