Output demo/update.gif

Set Theme "Dracula"
Set FontSize 16
Set Framerate 15
Set Margin 0
Set PlaybackSpeed 0.8

Type `PATH="$PWD/demo/mock-bin:$PATH" ./claude-cleaner --claude-dir /tmp/claude-demo --mock-update`
Enter
Sleep 3s

# Update prompt shown — Yes is highlighted by default
Sleep 1s

# Confirm update (Enter with Yes selected)
Enter
Sleep 3s

# Updating spinner plays, then app restarts / exits
Sleep 1s
