Output demo.gif

Require pkgmate

Set FontSize 13
Set Width 1400
Set Height 1000
Set Theme "Catppuccin Mocha"
Set WindowBar Colorful
Set PlaybackSpeed 1.3

# Launch
Type "pkgmate" Sleep 300ms Enter
Sleep 1s

# Search
Type "/" Sleep 300ms
Type "python" Sleep 1.5s
Enter Sleep 300ms
Type j Sleep 100ms Type j Sleep 100ms Type j Sleep 100ms
Type k Sleep 100ms Type k Sleep 100ms Type j Sleep 100ms
Escape Sleep 500ms

# Navigate tabs
Tab Sleep 1s
Tab Sleep 1s

# Keyboard navigation
Down@150ms 5 Sleep 400ms
Up@150ms 2 Sleep 400ms
Ctrl+d Sleep 150ms Ctrl+d Sleep 150ms Ctrl+d Sleep 150ms Ctrl+d Sleep 150ms
Type "ctrl+u" Sleep 150ms Type "ctrl+u" Sleep 150ms

# Command palette - switch to debug
Ctrl+k Sleep 1s
Down Sleep 300ms Down Sleep 300ms
Enter Sleep 2s

# Command palette - back to packages
Ctrl+k Sleep 1s
Enter Sleep 1.5s

# Command palette - quit
Ctrl+k Sleep 1s
Down Sleep 300ms Down Sleep 300ms Down Sleep 300ms
Enter Sleep 1s
