package main

import (
	"io/ioutil"
	"sync"

	"github.com/jezek/xgb/xproto"
	"gopkg.in/yaml.v3"
)

type Keymap struct {
	lock sync.RWMutex `yaml:"-"`

	Up    Keystate `yaml:"up"`
	Down  Keystate `yaml:"down"`
	Left  Keystate `yaml:"left"`
	Right Keystate `yaml:"right"`

	Button1 Keystate `yaml:"button1"`
	Button2 Keystate `yaml:"button2"`
	Button3 Keystate `yaml:"button3"`

	Button1Alt Keystate `yaml:"button1_alt"`
	Button2Alt Keystate `yaml:"button2_alt"`
	Button3Alt Keystate `yaml:"button3_alt"`

	ScrollUp    Keystate `yaml:"scroll_up"`
	ScrollDown  Keystate `yaml:"scroll_down"`
	ScrollLeft  Keystate `yaml:"scroll_left"`
	ScrollRight Keystate `yaml:"scroll_right"`

	Decelerate Keystate `yaml:"decelerate"`
	MLock      Keystate `yaml:"mlock"`
}

type Keystate struct {
	Keycode xproto.Keycode `yaml:"keycode,omitempty"`
	NoGrab  bool           `yaml:"no_grab,omitempty"`
	Down    bool           `yaml:"-"`
}

type Config struct {
	Speed           float32 `yaml:"speed"`
	MinSpeed        float32 `yaml:"min_speed"`
	Rate            int     `yaml:"rate"`
	Acceleration    float32 `yaml:"acceleration"`
	MaxAcceleration float32 `yaml:"max_acceleration"`
	Deceleration    float32 `yaml:"deceleration"`
	ScrollRate      int     `yaml:"scroll_rate"`

	Keymap Keymap `yaml:"keymap"`
}

func DefaultConfig() Config {
	return Config{
		Speed:           0.5,
		MinSpeed:        0.25,
		Rate:            1000,
		Acceleration:    0.005,
		MaxAcceleration: 6.0,
		Deceleration:    0.2,
		ScrollRate:      20,
	}
}

func DefaultConfigWithKeymap() Config {
	config := DefaultConfig()
	config.Keymap = DefaultKeymap()
	return config
}

func DefaultKeymap() Keymap {
	return Keymap{
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
}

func LoadConfig(path string) (config Config, err error) {
	config = DefaultConfig()

	var data []byte
	data, err = ioutil.ReadFile(path)
	if err != nil {
		return
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return
	}

	return
}
