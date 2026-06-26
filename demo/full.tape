Output demo/full.gif

Set Theme "Dracula"
Set FontSize 16
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

# Pause to show list with token usage and status columns
Sleep 500ms

# Navigate and select 2 items
Down
Sleep 150ms
Space
Down
Sleep 150ms
Space
Sleep 250ms

# Proceed to confirm screen
Enter
Sleep 400ms

# Default is No — move Right to highlight Yes
Right
Sleep 300ms

# Confirm deletion — watch live progress bar
Enter
Sleep 1500ms

# Done screen — rescan to show r key
Type "r"
Sleep 1200ms

# Back at list — quit
Type "q"
