# Render settings
Output demo.gif
Set FontSize 16
Set Width 1200
Set Height 700
Set Padding 20
Set Theme "Catppuccin Mocha"

# Launch and let the list load
Type "./essm"  Enter
Sleep 3s

# Search by name
Type "/"       Sleep 500ms
Type "web"     Sleep 2s
Escape         Sleep 1s

# Open the info panel, walk down a couple instances (panel follows)
Type "i"       Sleep 2s
Down           Sleep 1500ms
Down           Sleep 2s
Type "i"       Sleep 1s

# Quit
Type "q"