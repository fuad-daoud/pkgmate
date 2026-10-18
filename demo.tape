Output demo.gif

Require pkgmate

Set FontSize 13
Set Width 1400
Set Height 1000
Set Theme "Catppuccin Mocha"
Set WindowBar Colorful
Set PlaybackSpeed 1.3

# Record on a machine where at least one app is installed through two
# managers (for example pacman and Flatpak), or the Duplicates tab is empty.

# Launch; duplicates are computed once every backend has finished loading
Type "./pkgmate" Sleep 300ms Enter
Sleep 4s

# Direct packages
Down@150ms 5 Sleep 400ms
Up@150ms 2 Sleep 400ms

# Walk the tabs: Dependency, All, Orphan, then Duplicates
Tab Sleep 1s
Tab Sleep 1s
Ctrl+d Sleep 150ms Ctrl+d Sleep 150ms Ctrl+d Sleep 400ms
Tab Sleep 1s
Tab Sleep 2.5s
Down@300ms 3 Sleep 1s

# Back to Direct packages and filter the Package Manager column for the
# "(+flatpak)"-style marker that flags a package installed more than once
Tab Sleep 1s
Type "/" Sleep 300ms
Tab Sleep 300ms
Tab Sleep 300ms
Type "(+" Sleep 1.5s
Enter Sleep 300ms
Type j Sleep 150ms Type j Sleep 150ms Type k Sleep 1s
Escape Sleep 500ms

# Command palette - quit
Ctrl+k Sleep 1s
Down Sleep 300ms Down Sleep 300ms
Enter Sleep 1s
