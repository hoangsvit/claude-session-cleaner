Output demo/cancel.gif

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

# Select two items
Space
Down
Sleep 300ms
Space
Sleep 500ms

# Proceed to confirm screen
Enter
Sleep 800ms

# Default is No (safe default) — press Enter to cancel back to list
Enter
Sleep 500ms

# Quit
Type "q"
