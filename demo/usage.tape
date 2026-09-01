# vhs demo/usage.tape
#
# The tools in use. One request, one style choice, real files on disk.

Output demo/usage.gif

Set Shell "bash"
Set FontFamily "DejaVu Sans Mono"
Set FontSize 17
Set Width 1500
Set Height 820
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

Type "claude"
Sleep 400ms
Enter
Sleep 10s

Type "I'm building a landing page for a rocket-launch SaaS. Pick one icon style and one illustration style, then download a hero illustration, three feature icons as SVG, and a favicon set. Report the paths."
Sleep 400ms
Enter
Sleep 150s
