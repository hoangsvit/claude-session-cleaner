Output demo/cancel.gif

Set Theme "Dracula"
Set FontSize 16
Set Width 140
Set Height 30
Set Framerate 12
Set Margin 0
Set PlaybackSpeed 1

Env COLORTERM truecolor
Env TERM xterm-256color

Type `PATH="$PWD/demo/mock-bin:$PATH" ./claude-cleaner --claude-dir /tmp/claude-demo`
Enter
Sleep 1500ms

# Skip update prompt if present (n = skip; harmless in list state)
Type "n"
Sleep 200ms

# Select two items
Space
Down
Sleep 150ms
Space
Sleep 250ms

# Proceed to confirm screen
Enter
Sleep 400ms

# Default is No (safe default) — press Enter to cancel back to list
Enter
Sleep 250ms

# Quit
Type "q"
