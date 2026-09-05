# Browse TUI: filter, category tab, details, close
# Esc on the details dialog closes it. A second Esc would clear the search
# box rather than quit; hold the filtered catalog as the last frame.
Output vhs/exports/browse.gif
Output vhs/exports/browse.mp4
Output vhs/exports/browse.webm

Require fontget

Set Width 1400
Set Height 900
Source vhs/tapes/settings.tape

Hide
Type `clear`
Enter
Show

Sleep 400ms
Type `fontget browse --scope user`
Enter
Wait+Screen@30s /Font Browser/
Sleep 1s

Type@80ms `roboto`
Wait+Screen@20s /google.roboto/
Sleep 1s

Tab
Sleep 1.2s

Down
Sleep 400ms
Down
Sleep 600ms

Enter
Sleep 2s

Escape
Sleep 2s
