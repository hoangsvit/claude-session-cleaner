Output demo/cancel.gif

Set Theme "Dracula"
Set FontSize 14
Set Width 130
Set Height 44
Set PlaybackSpeed 0.8

Exec "node demo/seed.js /tmp/claude-demo"

Type "node bin/claude-session-cleaner.js --claude-dir /tmp/claude-demo"
Enter
Sleep 3s

Type "1,3"
Enter
Sleep 1s

Type "nope"
Enter
Sleep 2s
