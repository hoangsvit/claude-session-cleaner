Output demo/full.gif

Set Theme "Dracula"
Set FontSize 13
Set Framerate 15
Set Margin 0
Set PlaybackSpeed 0.8

Type "PATH=\"$PWD/demo/mock-bin:$PATH\" ./claude-cleaner --claude-dir /tmp/claude-demo"
Enter
Sleep 3s

# Skip update prompt if present (n = skip; harmless in list state)
Type "n"
Sleep 500ms

# Pause to show list with token usage and status columns
Sleep 1s

# Navigate and select 2 items
Down
Sleep 300ms
Space
Down
Sleep 300ms
Space
Sleep 500ms

# Proceed to confirm screen
Enter
Sleep 800ms

# Default is No — move Right to highlight Yes
Right
Sleep 600ms

# Confirm deletion — watch live progress bar
Enter
Sleep 3s

# Done screen — rescan to show r key
Type "r"
Sleep 2500ms

# Back at list — quit
Type "q"
