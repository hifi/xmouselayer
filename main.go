package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"github.com/jezek/xgb/xtest"
	"gopkg.in/yaml.v3"
)

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

func MotionEngine(ctx context.Context, X *xgb.Conn, config *Config, ix int16, iy int16) {
	x := float32(ix)
	y := float32(iy)
	setup := xproto.Setup(X)
	screen := setup.DefaultScreen(X)

	xproto.GrabKeyboard(X, true, screen.Root, xproto.TimeCurrentTime, xproto.GrabModeAsync, xproto.GrabModeAsync)
	defer xproto.UngrabKeyboard(X, xproto.TimeCurrentTime)

	speed := config.Speed
	accel := float32(0.0)

	button1 := false
	button2 := false
	button3 := false
	nextScroll := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Millisecond * time.Duration(1000/config.Rate)):
		}

		now := time.Now()
		ox := x
		oy := y
		obutton1 := button1
		obutton2 := button2
		obutton3 := button3

		config.Keymap.lock.RLock()
		if config.Keymap.Up.Down {
			y -= speed + accel
		}

		if config.Keymap.Down.Down {
			y += speed + accel
		}

		if config.Keymap.Left.Down {
			x -= speed + accel
		}

		if config.Keymap.Right.Down {
			x += speed + accel
		}

		button1 = config.Keymap.Button1.Down
		button2 = config.Keymap.Button2.Down
		button3 = config.Keymap.Button3.Down
		scrollUp := config.Keymap.ScrollUp.Down
		scrollDn := config.Keymap.ScrollDown.Down
		scrollLe := config.Keymap.ScrollLeft.Down
		scrollRi := config.Keymap.ScrollRight.Down
		decelerate := config.Keymap.Decelerate.Down

		config.Keymap.lock.RUnlock()

		if decelerate {
			accel = 0
			speed = config.MinSpeed
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
			if accel < config.MaxAcceleration {
				accel += config.Acceleration
			}
		} else {
			if accel > 0 {
				accel -= config.Deceleration
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
			nextScroll = now.Add(time.Millisecond * time.Duration(1000/config.ScrollRate))
		} else if scrollDn && now.After(nextScroll) {
			XTestFakeButtonEvent(X, xproto.ButtonIndex5, true, 0)
			XTestFakeButtonEvent(X, xproto.ButtonIndex5, false, 0)
			nextScroll = now.Add(time.Millisecond * time.Duration(1000/config.ScrollRate))
		}

		if scrollLe && now.After(nextScroll) {
			XTestFakeButtonEvent(X, 6, true, 0)
			XTestFakeButtonEvent(X, 6, false, 0)
			nextScroll = now.Add(time.Millisecond * time.Duration(1000/config.ScrollRate))
		} else if scrollRi && now.After(nextScroll) {
			XTestFakeButtonEvent(X, 7, true, 0)
			XTestFakeButtonEvent(X, 7, false, 0)
			nextScroll = now.Add(time.Millisecond * time.Duration(1000/config.ScrollRate))
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

	var config Config
	println("Loading config.yaml")

	// TODO this is pretty ugly
	if _, err := os.Stat("config.yaml"); err != nil {
		println("No config found, writing out default config.yaml")
		config = DefaultConfigWithKeymap()
		data, _ := yaml.Marshal(&config)
		ioutil.WriteFile("config.yaml", data, 0o644)
	}

	config, err = LoadConfig("config.yaml")
	if err != nil {
		panic(err)
	}

	println("Installing passive grabs")
	GrabKey(X, config.Keymap.Up)
	GrabKey(X, config.Keymap.Down)
	GrabKey(X, config.Keymap.Left)
	GrabKey(X, config.Keymap.Right)
	GrabKey(X, config.Keymap.Button1)
	GrabKey(X, config.Keymap.Button2)
	GrabKey(X, config.Keymap.Button3)
	GrabKey(X, config.Keymap.Button1Alt)
	GrabKey(X, config.Keymap.Button2Alt)
	GrabKey(X, config.Keymap.Button3Alt)
	GrabKey(X, config.Keymap.ScrollUp)
	GrabKey(X, config.Keymap.ScrollDown)
	GrabKey(X, config.Keymap.ScrollLeft)
	GrabKey(X, config.Keymap.ScrollRight)
	GrabKey(X, config.Keymap.Decelerate)
	GrabKey(X, config.Keymap.MLock)

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
			case config.Keymap.Up.Keycode:
				key = &config.Keymap.Up
			case config.Keymap.Down.Keycode:
				key = &config.Keymap.Down
			case config.Keymap.Left.Keycode:
				key = &config.Keymap.Left
			case config.Keymap.Right.Keycode:
				key = &config.Keymap.Right
			case config.Keymap.Button1.Keycode:
				key = &config.Keymap.Button1
			case config.Keymap.Button2.Keycode:
				key = &config.Keymap.Button2
			case config.Keymap.Button3.Keycode:
				key = &config.Keymap.Button3
			case config.Keymap.Button1Alt.Keycode:
				key = &config.Keymap.Button1
			case config.Keymap.Button2Alt.Keycode:
				key = &config.Keymap.Button2
			case config.Keymap.Button3Alt.Keycode:
				key = &config.Keymap.Button3
			case config.Keymap.ScrollUp.Keycode:
				key = &config.Keymap.ScrollUp
			case config.Keymap.ScrollDown.Keycode:
				key = &config.Keymap.ScrollDown
			case config.Keymap.ScrollLeft.Keycode:
				key = &config.Keymap.ScrollLeft
			case config.Keymap.ScrollRight.Keycode:
				key = &config.Keymap.ScrollRight
			case config.Keymap.Decelerate.Keycode:
				key = &config.Keymap.Decelerate
			case config.Keymap.MLock.Keycode:
			case modmap.Keycodes[24]:
				if mlock {
					mlock = false
				}
			}

			if key != nil {
				config.Keymap.lock.Lock()
				key.Down = true
				config.Keymap.lock.Unlock()
			}

		case xproto.KeyReleaseEvent:
			x = ev.RootX
			y = ev.RootY

			switch ev.Detail {
			case config.Keymap.Up.Keycode:
				key = &config.Keymap.Up
			case config.Keymap.Down.Keycode:
				key = &config.Keymap.Down
			case config.Keymap.Left.Keycode:
				key = &config.Keymap.Left
			case config.Keymap.Right.Keycode:
				key = &config.Keymap.Right
			case config.Keymap.Button1.Keycode:
				key = &config.Keymap.Button1
			case config.Keymap.Button2.Keycode:
				key = &config.Keymap.Button2
			case config.Keymap.Button3.Keycode:
				key = &config.Keymap.Button3
			case config.Keymap.Button1Alt.Keycode:
				key = &config.Keymap.Button1
			case config.Keymap.Button2Alt.Keycode:
				key = &config.Keymap.Button2
			case config.Keymap.Button3Alt.Keycode:
				key = &config.Keymap.Button3
			case config.Keymap.ScrollUp.Keycode:
				key = &config.Keymap.ScrollUp
			case config.Keymap.ScrollDown.Keycode:
				key = &config.Keymap.ScrollDown
			case config.Keymap.ScrollLeft.Keycode:
				key = &config.Keymap.ScrollLeft
			case config.Keymap.ScrollRight.Keycode:
				key = &config.Keymap.ScrollRight
			case config.Keymap.Decelerate.Keycode:
				key = &config.Keymap.Decelerate
			case config.Keymap.MLock.Keycode:
				if !mlock {
					println("Mlock enabled")
					mlock = true
					key = &config.Keymap.MLock // to always enable motion engine
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
				config.Keymap.lock.Lock()
				key.Down = false
				config.Keymap.lock.Unlock()
			}
		default:
			fmt.Printf("Unhandled event: %s\n", ev.String())
		}

		if key != nil && cancel == nil {
			var ctx context.Context
			ctx, cancel = context.WithCancel(context.Background())
			mod = true
			go MotionEngine(ctx, X, &config, x, y)
		}
	}
}
