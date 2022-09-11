package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"github.com/jezek/xgb/xtest"
)

type Keymap struct {
	lock sync.RWMutex

	Up    Keystate
	Down  Keystate
	Left  Keystate
	Right Keystate

	Button1 Keystate
	Button2 Keystate
	Button3 Keystate

	Button1Alt Keystate
	Button2Alt Keystate
	Button3Alt Keystate

	ScrollUp    Keystate
	ScrollDown  Keystate
	ScrollLeft  Keystate
	ScrollRight Keystate

	Decelerate Keystate
	MLock      Keystate
}

type Keystate struct {
	Keycode xproto.Keycode
	NoGrab  bool
	Down    bool
}

func GrabKey(X *xgb.Conn, key Keystate) (err error) {
	if key.Keycode == 0 || key.NoGrab {
		return
	}

	setup := xproto.Setup(X)
	root := setup.DefaultScreen(X).Root

	err = xproto.GrabKeyChecked(X, true, root, xproto.ModMask4, key.Keycode, xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
	if err != nil {
		return
	}

	err = xproto.GrabKeyChecked(X, true, root, xproto.ModMask4|xproto.ModMask2, key.Keycode, xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
	return
}

func XTestFakeMotionEvent(X *xgb.Conn, screen_number int, x int16, y int16, delay uint32) {
	xtest.FakeInput(X, xproto.MotionNotify, 0, delay, 0, x, y, 0)
}

func XTestFakeButtonEvent(X *xgb.Conn, button xproto.Button, down bool, delay uint32) {
	if down {
		xtest.FakeInput(X, xproto.ButtonPress, byte(button), delay, 0, 0, 0, 0)
	} else {
		xtest.FakeInput(X, xproto.ButtonRelease, byte(button), delay, 0, 0, 0, 0)
	}
}

func MotionEngine(ctx context.Context, X *xgb.Conn, keymap *Keymap, ix int16, iy int16) {
	x := float32(ix)
	y := float32(iy)
	setup := xproto.Setup(X)
	screen := setup.DefaultScreen(X)

	xproto.GrabKeyboard(X, true, screen.Root, xproto.TimeCurrentTime, xproto.GrabModeAsync, xproto.GrabModeAsync)
	defer xproto.UngrabKeyboard(X, xproto.TimeCurrentTime)

	speed := float32(0.5)
	accel := float32(0.0)
	maxAccel := float32(6.0)

	button1 := false
	button2 := false
	button3 := false
	nextScroll := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Millisecond * 1):
		}

		now := time.Now()
		ox := x
		oy := y
		obutton1 := button1
		obutton2 := button2
		obutton3 := button3

		keymap.lock.RLock()
		if keymap.Up.Down {
			y -= speed + accel
		}

		if keymap.Down.Down {
			y += speed + accel
		}

		if keymap.Left.Down {
			x -= speed + accel
		}

		if keymap.Right.Down {
			x += speed + accel
		}

		button1 = keymap.Button1.Down
		button2 = keymap.Button2.Down
		button3 = keymap.Button3.Down
		scrollUp := keymap.ScrollUp.Down
		scrollDn := keymap.ScrollDown.Down
		scrollLe := keymap.ScrollLeft.Down
		scrollRi := keymap.ScrollRight.Down
		decelerate := keymap.Decelerate.Down

		keymap.lock.RUnlock()

		if decelerate {
			accel = 0
			speed = 0.25
		}

		if x < 0 {
			x = 0
		}
		if x > float32(screen.WidthInPixels) {
			x = float32(screen.WidthInPixels)
		}
		if y < 0 {
			y = 0
		}
		if y > float32(screen.HeightInPixels) {
			y = float32(screen.HeightInPixels)
		}

		if x != ox || y != oy {
			XTestFakeMotionEvent(X, 0, int16(ox), int16(oy), 0)
			if accel < maxAccel {
				accel += 0.005
			}
		} else {
			if accel > 0 {
				accel -= 0.2
			}
		}

		if button1 != obutton1 {
			XTestFakeButtonEvent(X, xproto.ButtonIndex1, button1, 0)
		}

		if button2 != obutton2 {
			XTestFakeButtonEvent(X, xproto.ButtonIndex2, button2, 0)
		}

		if button3 != obutton3 {
			XTestFakeButtonEvent(X, xproto.ButtonIndex3, button3, 0)
		}

		if scrollUp && now.After(nextScroll) {
			XTestFakeButtonEvent(X, xproto.ButtonIndex4, true, 0)
			XTestFakeButtonEvent(X, xproto.ButtonIndex4, false, 0)
			nextScroll = now.Add(time.Millisecond * 50)
		} else if scrollDn && now.After(nextScroll) {
			XTestFakeButtonEvent(X, xproto.ButtonIndex5, true, 0)
			XTestFakeButtonEvent(X, xproto.ButtonIndex5, false, 0)
			nextScroll = now.Add(time.Millisecond * 50)
		}

		if scrollLe && now.After(nextScroll) {
			XTestFakeButtonEvent(X, 6, true, 0)
			XTestFakeButtonEvent(X, 6, false, 0)
			nextScroll = now.Add(time.Millisecond * 50)
		} else if scrollRi && now.After(nextScroll) {
			XTestFakeButtonEvent(X, 7, true, 0)
			XTestFakeButtonEvent(X, 7, false, 0)
			nextScroll = now.Add(time.Millisecond * 50)
		}
	}
}

func main() {
	println("xmouselayer")

	X, err := xgb.NewConn()
	if err != nil {
		panic(err)
	}

	err = xtest.Init(X)
	if err != nil {
		panic(err)
	}

	modmap, err := xproto.GetModifierMapping(X).Reply()
	if err != nil {
		panic(err)
	}

	if len(modmap.Keycodes) < 32 {
		panic("Not enough keycodes in modmap")
	}

	keymap := Keymap{
		Up:    Keystate{Keycode: 31}, // I
		Down:  Keystate{Keycode: 45}, // K
		Left:  Keystate{Keycode: 44}, // J
		Right: Keystate{Keycode: 46}, // L

		Button1: Keystate{Keycode: 41, NoGrab: true}, // S
		Button2: Keystate{Keycode: 40, NoGrab: true}, // D
		Button3: Keystate{Keycode: 39, NoGrab: true}, // F

		Button1Alt: Keystate{Keycode: 58, NoGrab: true}, // M
		Button2Alt: Keystate{Keycode: 59, NoGrab: true}, // ,
		Button3Alt: Keystate{Keycode: 60, NoGrab: true}, // .

		ScrollUp:    Keystate{Keycode: 43}, // H
		ScrollDown:  Keystate{Keycode: 57}, // N
		ScrollLeft:  Keystate{Keycode: 30}, // U
		ScrollRight: Keystate{Keycode: 32}, // O

		Decelerate: Keystate{Keycode: 65}, // Space
		MLock:      Keystate{Keycode: 33}, // P
	}

	println("Installing passive grabs")
	GrabKey(X, keymap.Up)
	GrabKey(X, keymap.Down)
	GrabKey(X, keymap.Left)
	GrabKey(X, keymap.Right)
	GrabKey(X, keymap.Button1)
	GrabKey(X, keymap.Button2)
	GrabKey(X, keymap.Button3)
	GrabKey(X, keymap.Button1Alt)
	GrabKey(X, keymap.Button2Alt)
	GrabKey(X, keymap.Button3Alt)
	GrabKey(X, keymap.ScrollUp)
	GrabKey(X, keymap.ScrollDown)
	GrabKey(X, keymap.ScrollLeft)
	GrabKey(X, keymap.ScrollRight)
	GrabKey(X, keymap.Decelerate)
	GrabKey(X, keymap.MLock)

	println("Handling input")
	var x int16
	var y int16
	var cancel context.CancelFunc
	mlock := false
	mod := false

	for {
		ev, err := X.WaitForEvent()
		if err != nil {
			panic(err)
		}

		var key *Keystate

		switch ev := ev.(type) {
		case xproto.KeyPressEvent:
			x = ev.RootX
			y = ev.RootY

			switch ev.Detail {
			case keymap.Up.Keycode:
				key = &keymap.Up
			case keymap.Down.Keycode:
				key = &keymap.Down
			case keymap.Left.Keycode:
				key = &keymap.Left
			case keymap.Right.Keycode:
				key = &keymap.Right
			case keymap.Button1.Keycode:
				key = &keymap.Button1
			case keymap.Button2.Keycode:
				key = &keymap.Button2
			case keymap.Button3.Keycode:
				key = &keymap.Button3
			case keymap.Button1Alt.Keycode:
				key = &keymap.Button1
			case keymap.Button2Alt.Keycode:
				key = &keymap.Button2
			case keymap.Button3Alt.Keycode:
				key = &keymap.Button3
			case keymap.ScrollUp.Keycode:
				key = &keymap.ScrollUp
			case keymap.ScrollDown.Keycode:
				key = &keymap.ScrollDown
			case keymap.ScrollLeft.Keycode:
				key = &keymap.ScrollLeft
			case keymap.ScrollRight.Keycode:
				key = &keymap.ScrollRight
			case keymap.Decelerate.Keycode:
				key = &keymap.Decelerate
			case keymap.MLock.Keycode:
			case modmap.Keycodes[24]:
				if mlock {
					mlock = false
				}
			}

			if key != nil {
				keymap.lock.Lock()
				key.Down = true
				keymap.lock.Unlock()
			}

		case xproto.KeyReleaseEvent:
			x = ev.RootX
			y = ev.RootY

			switch ev.Detail {
			case keymap.Up.Keycode:
				key = &keymap.Up
			case keymap.Down.Keycode:
				key = &keymap.Down
			case keymap.Left.Keycode:
				key = &keymap.Left
			case keymap.Right.Keycode:
				key = &keymap.Right
			case keymap.Button1.Keycode:
				key = &keymap.Button1
			case keymap.Button2.Keycode:
				key = &keymap.Button2
			case keymap.Button3.Keycode:
				key = &keymap.Button3
			case keymap.Button1Alt.Keycode:
				key = &keymap.Button1
			case keymap.Button2Alt.Keycode:
				key = &keymap.Button2
			case keymap.Button3Alt.Keycode:
				key = &keymap.Button3
			case keymap.ScrollUp.Keycode:
				key = &keymap.ScrollUp
			case keymap.ScrollDown.Keycode:
				key = &keymap.ScrollDown
			case keymap.ScrollLeft.Keycode:
				key = &keymap.ScrollLeft
			case keymap.ScrollRight.Keycode:
				key = &keymap.ScrollRight
			case keymap.Decelerate.Keycode:
				key = &keymap.Decelerate
			case keymap.MLock.Keycode:
				if !mlock {
					println("Mlock enabled")
					mlock = true
					key = &keymap.MLock // to always enable motion engine
				} else {
					if cancel != nil && !mod {
						println("Mlock released")
						cancel()
						cancel = nil
					}
					mlock = false
				}
			case modmap.Keycodes[24]:
				// TODO this is weird
				// cancel motion if we release modifier
				if cancel != nil {
					if mlock {
						mod = false
					} else {
						cancel()
						cancel = nil
					}
				}
			}

			if key != nil {
				keymap.lock.Lock()
				key.Down = false
				keymap.lock.Unlock()
			}
		default:
			fmt.Printf("Unhandled event: %s\n", ev.String())
		}

		if key != nil && cancel == nil {
			var ctx context.Context
			ctx, cancel = context.WithCancel(context.Background())
			mod = true
			go MotionEngine(ctx, X, &keymap, x, y)
		}
	}
}
