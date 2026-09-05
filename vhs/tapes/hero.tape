# Hero: search → add → list → remove
Output vhs/exports/hero.gif
Output vhs/exports/hero.mp4
Output vhs/exports/hero.webm

Require fontget

Set Width 1200
Set Height 700
Source vhs/tapes/settings.tape

Hide
Type `clear`
Enter
Show

Sleep 500ms
Type `fontget search roboto`
Enter
Wait+Screen@45s /google.roboto/
Sleep 1.5s

Type `fontget add google.roboto --scope user`
Enter
Wait+Screen@60s /Installed/
Sleep 1.5s

Type `fontget list roboto`
Enter
Wait+Screen@30s /Roboto/
Sleep 1.5s

Type `fontget remove google.roboto --scope user`
Enter
Wait+Screen@45s /Removed/
Sleep 2s
