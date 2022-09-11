xmouselayer
===========
Like [UHK mouse layer](https://ultimatehackingkeyboard.com/manuals/uhk60/mouse) but for X11.

Only Super/Mod4 is supported as the trigger key to enable the layer.
I suggest making your Caps Lock an additional Super key so it is as ergonomic as with UHK.
Default keys are slightly different for ergonomic reasons without a split keyboard, namely scrolling up and down are `[H]`/`[N]` instead of `[Y]`/`[H]`.

Mouse layer can be locked with `[P]` and deceleration happens by holding `[Space]`.

This is at proof-of-concept state so use at your own risk.

How it works
------------
At startup all keys that don't have grab disabled in config are registered as passive hotkeys that will enable the layer once they are triggered.
After the layer is activated by any passive hotkey all keys on the keyboard are grabbed until you release the Super key or the layer can be locked with the lock key until released with the same key.

Why?
----
I wanted to try if I'd like the mouse emulation of UHK without buying one 🤷.
Also got a reason to hack something small together.

TODO
----
- Configurability
- Better logging
- Daemonizing
- XKB key repeat hint (works fine without, though)
- Status icon for docks