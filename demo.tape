# demo.tape — VHS recording script for Wharf
# Generate with: vhs demo.tape
# Requires: https://github.com/charmbracelet/vhs

Output demo.gif
Set Width 1200
Set Height 600
Set FontSize 14
Set Theme "Dracula"

Type "wharf"
Enter
Sleep 2s

# Navigate projects
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms

# Enter project
Type "l"
Sleep 1s

# View logs
Type "L"
Sleep 2s

# Back
Type "h"
Sleep 500ms

# Exec
Type "e"
Sleep 1s
Type "exit"
Enter
Sleep 500ms

# Quit
Type "q"
