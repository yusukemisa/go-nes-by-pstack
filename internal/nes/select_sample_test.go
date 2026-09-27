package nes

import (
	"testing"

	"github.com/misawa/go-nes-by-pstack/internal/controller"
)

// SMB title-screen reads $4016 with lda / lsr / ora / lsr / rol, which keeps
// the button when bit 0 or bit 1 is set. The other common form is
// and #$03 / cmp #$01 / rol (carry set iff the low two bits are nonzero).
// https://www.nesdev.org/wiki/Standard_controller
// https://www.nesdev.org/wiki/Controller_reading_code
//
// After eight rols the software byte is A B Select Start Up Down Left Right.
// ReadJoypads then drops Select/Start when JoypadBitMask already had them,
// so a held button is one edge.
const (
	smbSelect      = 0x20
	smbStart       = 0x10
	smbSelectStart = 0x30
)

func TestSMBSelectSample(t *testing.T) {
	t.Run("one CPU poll", func(t *testing.T) {
		c := newSelectProbe(t)
		pad := c.Bus.GetController1()
		var prev uint8

		pad.SetButton(controller.BUTTON_SELECT, true)
		raw1, bits1 := strobeBits(c)
		vis1, prev := smbPresent(assembleTwoBit(bits1), prev)
		pad.SetButton(controller.BUTTON_SELECT, false)
		raw2, bits2 := strobeBits(c)
		vis2, _ := smbPresent(assembleTwoBit(bits2), prev)

		if assembleBit0(bits1) != assembleTwoBit(bits1) || assembleAND3(bits1) != assembleTwoBit(bits1) {
			t.Fatalf("SELECT poll bits %v: bit0=%02X two-bit=%02X and3=%02X", bits1, assembleBit0(bits1), assembleTwoBit(bits1), assembleAND3(bits1))
		}
		if raw1 != smbSelect || vis1 != smbSelect {
			t.Fatalf("SELECT high for one strobe: raw %02X visible %02X, want %02X", raw1, vis1, smbSelect)
		}
		if raw2 != 0 || vis2 != 0 {
			t.Fatalf("second strobe after release: raw %02X visible %02X, want 0", raw2, vis2)
		}
		t.Logf("one poll: first read %02X is a new press, second read %02X", vis1, vis2)
	})

	t.Run("held across frames", func(t *testing.T) {
		c := newSelectProbe(t)
		keys := controller.NewFrameKeys(c.Bus.GetController1())
		keys.Key(1, controller.BUTTON_SELECT, true)

		rawA, rawB, vis, prev := sampleFrame(t, c, keys, 0)
		if rawA != smbSelect || rawB != smbSelect || vis != smbSelect {
			t.Fatalf("frame 1 held: reads %02X %02X visible %02X, want %02X", rawA, rawB, vis, smbSelect)
		}

		// SDL repeats key-down while Shift is held. That must not stick the button.
		keys.Key(1, controller.BUTTON_SELECT, true)
		rawA, rawB, vis, prev = sampleFrame(t, c, keys, prev)
		if rawA != smbSelect || rawB != smbSelect || vis != 0 {
			t.Fatalf("frame 2 still held: reads %02X %02X visible %02X, want raw %02X and no new edge", rawA, rawB, vis, smbSelect)
		}

		keys.Key(1, controller.BUTTON_SELECT, false)
		for i := 0; i < 2; i++ {
			rawA, rawB, vis, prev = sampleFrame(t, c, keys, prev)
			if vis&smbSelect != 0 {
				t.Fatalf("release frame %d produced another SELECT edge (%02X %02X)", i, rawA, rawB)
			}
		}
		if rawA != 0 || rawB != 0 {
			t.Fatalf("after release the pad stayed high: %02X %02X", rawA, rawB)
		}
		t.Logf("one frame / hold: exactly one SELECT edge")
	})

	t.Run("press and release inside StepFrame", func(t *testing.T) {
		c := newSelectProbe(t)
		keys := controller.NewFrameKeys(c.Bus.GetController1())
		var prev uint8
		edges := 0
		for i := 0; i < 10; i++ {
			keys.Key(1, controller.BUTTON_SELECT, true)
			keys.Key(1, controller.BUTTON_SELECT, false)
			rawA, rawB, vis, next := sampleFrame(t, c, keys, prev)
			if vis&smbSelect == 0 || rawA != smbSelect || rawB != smbSelect {
				t.Fatalf("tap %d dropped: reads %02X %02X visible %02X", i, rawA, rawB, vis)
			}
			edges++
			// SMB keeps JoypadBitMask until a later poll sees Select released.
			_, _, vis, next = sampleFrame(t, c, keys, next)
			if vis&smbSelect != 0 {
				t.Fatalf("tap %d release frame edged again: %02X", i, vis)
			}
			prev = next
		}
		if edges != 10 {
			t.Fatalf("10 Shift taps inside a frame produced %d SELECT edges, want 10", edges)
		}

		// A second frame with the key up must not repeat the edge.
		_, _, vis, _ := sampleFrame(t, c, keys, prev)
		if vis&smbSelect != 0 {
			t.Fatalf("idle frame repeated SELECT: %02X", vis)
		}
		t.Logf("press-and-release inside StepFrame: each tap is one edge")
	})

	t.Run("back to back taps are not a press counter", func(t *testing.T) {
		c := newSelectProbe(t)
		keys := controller.NewFrameKeys(c.Bus.GetController1())
		var prev uint8
		edges := 0
		for i := 0; i < 10; i++ {
			keys.Key(1, controller.BUTTON_SELECT, true)
			keys.Key(1, controller.BUTTON_SELECT, false)
			_, _, vis, next := sampleFrame(t, c, keys, prev)
			prev = next
			if vis&smbSelect != 0 {
				edges++
			}
		}
		if edges != 1 {
			t.Fatalf("10 taps on consecutive frames produced %d edges, want 1", edges)
		}
		t.Logf("back-to-back taps: one edge, not every 10th press")
	})

	t.Run("start still edges once", func(t *testing.T) {
		c := newSelectProbe(t)
		keys := controller.NewFrameKeys(c.Bus.GetController1())
		keys.Key(2, controller.BUTTON_START, true)
		_, _, vis, prev := sampleFrame(t, c, keys, 0)
		if vis != smbStart {
			t.Fatalf("START press visible %02X, want %02X", vis, smbStart)
		}
		rawA, rawB, vis, prev := sampleFrame(t, c, keys, prev)
		if rawA != smbStart || rawB != smbStart || vis&smbStart != 0 {
			t.Fatalf("held START streamed: reads %02X %02X visible %02X", rawA, rawB, vis)
		}
		keys.Key(2, controller.BUTTON_START, false)
		_, _, _, prev = sampleFrame(t, c, keys, prev)
		rawA, rawB, vis, _ = sampleFrame(t, c, keys, prev)
		if rawA != 0 || rawB != 0 || vis != 0 {
			t.Fatalf("START stuck after release: %02X %02X visible %02X", rawA, rawB, vis)
		}
	})

	t.Run("either shift keeps select", func(t *testing.T) {
		c := newSelectProbe(t)
		keys := controller.NewFrameKeys(c.Bus.GetController1())
		const left, right = 1, 2
		keys.Key(left, controller.BUTTON_SELECT, true)
		keys.Key(right, controller.BUTTON_SELECT, true)
		keys.Key(left, controller.BUTTON_SELECT, false)
		rawA, rawB, vis, prev := sampleFrame(t, c, keys, 0)
		if rawA != smbSelect || rawB != smbSelect || vis != smbSelect {
			t.Fatalf("right shift did not hold SELECT: %02X %02X visible %02X", rawA, rawB, vis)
		}
		keys.Key(right, controller.BUTTON_SELECT, false)
		_, _, _, prev = sampleFrame(t, c, keys, prev)
		rawA, rawB, _, _ = sampleFrame(t, c, keys, prev)
		if rawA != 0 || rawB != 0 {
			t.Fatalf("SELECT stuck after both shifts released: %02X %02X", rawA, rawB)
		}
	})
}

func newSelectProbe(t *testing.T) *Console {
	t.Helper()
	c := openTiming(t, selectProbeROM())
	c.Reset()
	c.PPU.SetTiming(0, 0)
	return c
}

func sampleFrame(t *testing.T, c *Console, keys *controller.FrameKeys, prev uint8) (rawA, rawB, visible, next uint8) {
	t.Helper()
	before := c.CPU.Read(0x0002)
	c.StepFrame()
	keys.EndFrame()
	after := c.CPU.Read(0x0002)
	if after != before+1 {
		t.Fatalf("NMI sampled %d times, want 1 (counter %d -> %d)", after-before, before, after)
	}
	rawA = c.CPU.Read(0x0300)
	rawB = c.CPU.Read(0x0301)
	visible, next = smbPresent(rawA, prev)
	return rawA, rawB, visible, next
}

// smbPresent is the Select/Start part of SMB ReadJoypads. prev is JoypadBitMask.
func smbPresent(raw, prev uint8) (visible, next uint8) {
	if raw&smbSelectStart&prev == 0 {
		return raw, raw
	}
	return raw &^ smbSelectStart, prev
}

func strobeBits(c *Console) (uint8, [8]uint8) {
	c.CPU.Write(0x4016, 1)
	c.CPU.Write(0x4016, 0)
	var bits [8]uint8
	var a uint8
	for i := 0; i < 8; i++ {
		bits[i] = c.CPU.Read(0x4016)
		a = (a << 1) | (((bits[i] >> 1) | bits[i]) & 1)
	}
	return a, bits
}

func assembleTwoBit(bits [8]uint8) uint8 {
	var a uint8
	for _, v := range bits {
		a = (a << 1) | (((v >> 1) | v) & 1)
	}
	return a
}

func assembleBit0(bits [8]uint8) uint8 {
	var a uint8
	for _, v := range bits {
		a = (a << 1) | (v & 1)
	}
	return a
}

func assembleAND3(bits [8]uint8) uint8 {
	var a uint8
	for _, v := range bits {
		var carry uint8
		if v&0x03 >= 1 {
			carry = 1
		}
		a = (a << 1) | carry
	}
	return a
}

// selectProbeROM enables NMI, then on each NMI strobes $4016 and does the
// SMB lsr/ora/lsr/rol read twice, storing the bytes at $0300 and $0301.
func selectProbeROM() []byte {
	prg := make([]byte, 16384)
	copy(prg[0:], []byte{
		0xA9, 0x80, // LDA #$80
		0x8D, 0x00, 0x20, // STA $2000
		0x4C, 0x05, 0xC0, // JMP $C005
	})
	copy(prg[0x008:], []byte{
		0x20, 0x20, 0xC0, // JSR $C020
		0x8D, 0x00, 0x03, // STA $0300
		0x20, 0x20, 0xC0, // JSR $C020
		0x8D, 0x01, 0x03, // STA $0301
		0xE6, 0x02, // INC $02
		0x40, // RTI
	})
	copy(prg[0x020:], []byte{
		0xA9, 0x01, // LDA #$01
		0x8D, 0x16, 0x40, // STA $4016
		0x4A,             // LSR A
		0xAA,             // TAX
		0x8D, 0x16, 0x40, // STA $4016
		0xA0, 0x08, // LDY #$08
		0x48,             // PHA
		0xAD, 0x16, 0x40, // LDA $4016
		0x85, 0x00, // STA $00
		0x4A,       // LSR A
		0x05, 0x00, // ORA $00
		0x4A,       // LSR A
		0x68,       // PLA
		0x2A,       // ROL A
		0x88,       // DEY
		0xD0, 0xF1, // BNE loop
		0x60, // RTS
	})
	prg[0x3FFA] = 0x08
	prg[0x3FFB] = 0xC0
	prg[0x3FFC] = 0x00
	prg[0x3FFD] = 0xC0
	prg[0x3FFE] = 0x05
	prg[0x3FFF] = 0xC0
	return prg
}
