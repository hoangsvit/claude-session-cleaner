Output demo/screenshot.png

Set Theme "Dracula"
Set FontSize 16
Set Width 180
Set Margin 20
Set MarginFill "#282a36"

Env COLORTERM truecolor
Env TERM xterm-256color

Type `PATH="$PWD/demo/mock-bin:$PATH" ./claude-cleaner --claude-dir /tmp/claude-demo`
Enter
Sleep 1500ms

# Skip update prompt
Type "n"
Sleep 200ms

# Select a couple of items to show the UI in action
Down
Sleep 100ms
Space
Down
Sleep 100ms
Space
Sleep 300ms
