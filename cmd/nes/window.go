package main

import (
	"runtime"
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"

	"github.com/misawa/go-nes-by-pstack/internal/controller"
	"github.com/misawa/go-nes-by-pstack/internal/nes"
)

const (
	screenW = 256
	screenH = 240
	scale   = 3
)

// macOS requires SDL video and event calls on the main OS thread.
func init() {
	runtime.LockOSThread()
}

var keyButtons = map[sdl.Keycode]uint8{
	sdl.K_z:      controller.BUTTON_A,
	sdl.K_x:      controller.BUTTON_B,
	sdl.K_RETURN: controller.BUTTON_START,
	sdl.K_LSHIFT: controller.BUTTON_SELECT,
	sdl.K_RSHIFT: controller.BUTTON_SELECT,
	sdl.K_UP:     controller.BUTTON_UP,
	sdl.K_DOWN:   controller.BUTTON_DOWN,
	sdl.K_LEFT:   controller.BUTTON_LEFT,
	sdl.K_RIGHT:  controller.BUTTON_RIGHT,
}

func runWindow(c *nes.Console) error {
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		return err
	}
	defer sdl.Quit()

	window, err := sdl.CreateWindow("nes", sdl.WINDOWPOS_CENTERED, sdl.WINDOWPOS_CENTERED,
		screenW*scale, screenH*scale, sdl.WINDOW_SHOWN)
	if err != nil {
		return err
	}
	defer window.Destroy()

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED|sdl.RENDERER_PRESENTVSYNC)
	if err != nil {
		return err
	}
	defer renderer.Destroy()

	texture, err := renderer.CreateTexture(sdl.PIXELFORMAT_RGB24, sdl.TEXTUREACCESS_STREAMING, screenW, screenH)
	if err != nil {
		return err
	}
	defer texture.Destroy()

	pad := c.Bus.GetController1()
	pixels := make([]byte, screenW*screenH*3)
	for {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch e := event.(type) {
			case *sdl.QuitEvent:
				return nil
			case *sdl.KeyboardEvent:
				key := e.Keysym.Sym
				if key == sdl.K_ESCAPE {
					return nil
				}
				if button, ok := keyButtons[key]; ok {
					pad.SetButton(button, e.State == sdl.PRESSED)
				}
			}
		}

		c.StepFrame()
		frameToRGB(c.Frame(), pixels)
		if err := texture.Update(nil, unsafe.Pointer(&pixels[0]), screenW*3); err != nil {
			return err
		}
		if err := renderer.Copy(texture, nil, nil); err != nil {
			return err
		}
		renderer.Present()
	}
}
