Output demo/features.gif

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

# Skip update prompt
Type "n"
Sleep 300ms

# Pause to show initial list (sort: recent)
Sleep 600ms

# --- Sort ---
# Cycle sort: recent → size → tokens → name
Type "s"
Sleep 500ms
Type "s"
Sleep 500ms
Type "s"
Sleep 500ms
Type "s"
Sleep 400ms

# --- Filter ---
# Cycle filter: all → has-data → orphaned → all
Type "f"
Sleep 500ms
Type "f"
Sleep 500ms
Type "f"
Sleep 400ms

# --- Search ---
# Press / to open search prompt
Type "/"
Sleep 300ms
Type "api"
Sleep 600ms

# Clear search with Esc
Escape
Sleep 400ms

# --- Expiry threshold ---
# Press e to cycle expiry (show only older sessions)
Type "e"
Sleep 500ms
Type "e"
Sleep 400ms
# Reset back to no expiry
Type "d"
Sleep 400ms

# --- Select orphaned with o ---
Type "o"
Sleep 400ms
# Deselect all
Type "n"
Sleep 300ms

# --- Category cleanup screen ---
Type "c"
Sleep 800ms

# Navigate down a couple of items
Down
Sleep 200ms
Down
Sleep 200ms

# Select current item
Space
Sleep 300ms

# Select all
Type "a"
Sleep 400ms

# Deselect all with n
Type "n"
Sleep 300ms

# Back to list with Esc
Escape
Sleep 400ms

# Quit
Type "q"
