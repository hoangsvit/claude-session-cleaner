Output demo/update.gif

Set Theme "Dracula"
Set FontSize 16
Set Framerate 12
Set Margin 0
Set PlaybackSpeed 1

Env COLORTERM truecolor
Env TERM xterm-256color

Type `PATH="$PWD/demo/mock-bin:$PATH" ./claude-cleaner --claude-dir /tmp/claude-demo --mock-update`
Enter
Sleep 1500ms

# Update prompt shown — Yes is highlighted by default
Sleep 500ms

# Confirm update (Enter with Yes selected)
Enter
Sleep 1500ms

# Updating spinner plays, then app restarts / exits
Sleep 500ms
