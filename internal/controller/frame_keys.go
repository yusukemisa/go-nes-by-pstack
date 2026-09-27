package controller

// FrameKeys feeds host keyboard state to the pad.
//
// The NES latches the button level when the game strobes $4016. Super Mario
// Bros. turns a new Select or Start in that latch into one edge, and ignores
// the button while it stays down. A key-up applied in the same poll as the
// key-down clears the level before that strobe, so the press never happens.
// EndFrame applies key-up after the frame has sampled.
// https://www.nesdev.org/wiki/Standard_controller
// https://www.nesdev.org/wiki/Controller_reading_code
type FrameKeys struct {
	pad  *Controller
	held map[int]uint8
	uses [8]int
}

func NewFrameKeys(pad *Controller) *FrameKeys {
	return &FrameKeys{
		pad:  pad,
		held: make(map[int]uint8),
	}
}

// Key records one host key. pressed is true on key-down, including repeats.
// key distinguishes keys that share a button, such as the two Shift keys.
func (f *FrameKeys) Key(key int, button uint8, pressed bool) {
	if button == 0 || button&(button-1) != 0 {
		return
	}
	if pressed {
		if prev, ok := f.held[key]; ok {
			if prev == button {
				return
			}
			f.drop(key, prev)
		}
		f.held[key] = button
		idx := bitIndex(button)
		if f.uses[idx] == 0 {
			f.pad.SetButton(button, true)
		}
		f.uses[idx]++
		return
	}
	prev, ok := f.held[key]
	if !ok {
		return
	}
	f.drop(key, prev)
}

func (f *FrameKeys) drop(key int, button uint8) {
	delete(f.held, key)
	idx := bitIndex(button)
	if f.uses[idx] > 0 {
		f.uses[idx]--
	}
}

// EndFrame releases buttons whose keys are already up. Call it after the
// console step that strobes the pad.
func (f *FrameKeys) EndFrame() {
	for i := 0; i < 8; i++ {
		if f.uses[i] == 0 {
			f.pad.SetButton(1<<uint(i), false)
		}
	}
}

func bitIndex(button uint8) int {
	n := 0
	for button > 1 {
		button >>= 1
		n++
	}
	return n
}
