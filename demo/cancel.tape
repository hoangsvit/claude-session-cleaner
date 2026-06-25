Output demo/cancel.gif

Set Theme "Dracula"
Set FontSize 14
Set Width 130
Set Height 120
Set PlaybackSpeed 0.8

Exec "node demo/seed.js /tmp/claude-demo"

Type "go run . --claude-dir /tmp/claude-demo"
Enter
Sleep 5s

Space
Down
Down
Space
Sleep 500ms

Enter
Sleep 1s

Type "nope"
Enter
Sleep 1s

Escape
Sleep 1s

Type "q"
