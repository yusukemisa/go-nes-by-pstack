package ppu

import (
	"testing"

	"github.com/misawa/go-nes-by-pstack/internal/cartridge"
)

func writeVRAM(p *PPU, addr uint16, data uint8) {
	setPPUAddr(p, addr)
	p.CPUWrite(PPUDATA, data)
}

func readVRAM(p *PPU, addr uint16) uint8 {
	setPPUAddr(p, addr)
	_ = p.CPURead(PPUDATA)
	return p.CPURead(PPUDATA)
}

func TestNametableMirroring(t *testing.T) {
	cases := []struct {
		name   string
		mirror uint8
		pairs  [2][2]uint16
	}{
		{
			name:   "horizontal",
			mirror: 0,
			pairs:  [2][2]uint16{{0x2000, 0x2400}, {0x2800, 0x2C00}},
		},
		{
			name:   "vertical",
			mirror: 1,
			pairs:  [2][2]uint16{{0x2000, 0x2800}, {0x2400, 0x2C00}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewPPU()
			p.ConnectCartridge(&cartridge.Cartridge{Mirror: tc.mirror})

			for _, pair := range tc.pairs {
				writeVRAM(p, pair[0], 0)
				writeVRAM(p, pair[1], 0)
			}

			for i, pair := range tc.pairs {
				value := uint8(0xA1 + i)
				writeVRAM(p, pair[0], value)
				if got := readVRAM(p, pair[0]); got != value {
					t.Fatalf("read %04X = %02X, want %02X", pair[0], got, value)
				}
				if got := readVRAM(p, pair[1]); got != value {
					t.Fatalf("read %04X = %02X, want %02X (alias of %04X)", pair[1], got, value, pair[0])
				}
				for j, other := range tc.pairs {
					if j == i {
						continue
					}
					if got := readVRAM(p, other[0]); got == value {
						t.Fatalf("read %04X = %02X, same as %04X", other[0], got, pair[0])
					}
				}

				value = uint8(0xB1 + i)
				writeVRAM(p, pair[1]+0x10, value)
				if got := readVRAM(p, pair[0]+0x10); got != value {
					t.Fatalf("read %04X = %02X, want %02X (alias of write to %04X)", pair[0]+0x10, got, value, pair[1]+0x10)
				}
			}
		})
	}
}

func TestPPUDataWriteIsImmediateWithRenderingOff(t *testing.T) {
	p := NewPPU()
	p.ConnectCartridge(&cartridge.Cartridge{Mirror: 0})
	p.SetTiming(100, 0)
	if p.mask != 0 {
		t.Fatalf("mask = %02X, want rendering off", p.mask)
	}

	writeVRAM(p, 0x2000, 0x77)
	if p.GetScanline() != 100 {
		t.Fatalf("scanline = %d, want 100", p.GetScanline())
	}
	if got := p.ppuRead(0x2000); got != 0x77 {
		t.Fatalf("ppuRead($2000) = %02X, want 77", got)
	}
	if got := readVRAM(p, 0x2400); got != 0x77 {
		t.Fatalf("read $2400 = %02X, want 77", got)
	}
	if got := p.ppuRead(0x2800); got != 0 {
		t.Fatalf("ppuRead($2800) = %02X, want 00", got)
	}
}

func TestOAMDataWriteIncrementsAddr(t *testing.T) {
	p := NewPPU()
	p.CPUWrite(OAMADDR, 0)
	p.CPUWrite(OAMDATA, 0x11)
	p.CPUWrite(OAMDATA, 0x22)
	if p.OAM[0] != 0x11 || p.OAM[1] != 0x22 {
		t.Fatalf("OAM[0]=%02X OAM[1]=%02X, want 11 22", p.OAM[0], p.OAM[1])
	}
	if p.oamAddr != 2 {
		t.Fatalf("oamAddr = %d, want 2", p.oamAddr)
	}
}

func clockUntil(t *testing.T, p *PPU, scanline int16, dot uint16) {
	t.Helper()
	for i := 0; i < 341*300; i++ {
		if p.GetScanline() == scanline && p.GetDot() == dot {
			return
		}
		p.Clock()
	}
	t.Fatalf("never reached scanline %d dot %d (at %d,%d)", scanline, dot, p.GetScanline(), p.GetDot())
}

func hideSprites(p *PPU) {
	p.CPUWrite(OAMADDR, 0)
	for i := 0; i < 256; i++ {
		p.CPUWrite(OAMDATA, 0xFF)
	}
}

func TestSpriteYStartsOnNextScanline(t *testing.T) {
	chr := make([]byte, 16)
	chr[0] = 0x80 // tile 0, row 0, leftmost pixel

	p := NewPPU()
	p.ConnectCartridge(&cartridge.Cartridge{CHRROM: chr, Mirror: 0})
	hideSprites(p)
	p.CPUWrite(OAMADDR, 0)
	p.CPUWrite(OAMDATA, 10) // Y
	p.CPUWrite(OAMDATA, 0)  // tile
	p.CPUWrite(OAMDATA, 0)  // attributes
	p.CPUWrite(OAMDATA, 0)  // X
	setPPUAddr(p, 0x3F11)
	p.CPUWrite(PPUDATA, 0x21)
	p.CPUWrite(PPUMASK, MASK_RENDER_SPR|MASK_RENDER_SPR_LEFT)

	clockUntil(t, p, 10, 2)
	if got := p.sprScreen[10][0]; got != 0 {
		t.Fatalf("scanline 10 pixel = %02X, want 00", got)
	}
	clockUntil(t, p, 11, 2)
	if got := p.sprScreen[11][0]; got != 0x21 {
		t.Fatalf("scanline 11 pixel = %02X, want 21", got)
	}
	if got := p.sprScreen[11][1]; got != 0 {
		t.Fatalf("scanline 11 x=1 = %02X, want 00", got)
	}
	clockUntil(t, p, 12, 2)
	if got := p.sprScreen[12][0]; got != 0 {
		t.Fatalf("scanline 12 pixel = %02X, want 00 (pattern row 0 only)", got)
	}
}

func TestSpriteOverflowOnNinthSprite(t *testing.T) {
	for _, n := range []int{8, 9} {
		t.Run(string(rune('0'+n)), func(t *testing.T) {
			p := NewPPU()
			p.ConnectCartridge(&cartridge.Cartridge{Mirror: 0})
			hideSprites(p)
			p.CPUWrite(OAMADDR, 0)
			for i := 0; i < n; i++ {
				p.CPUWrite(OAMDATA, 10)
				p.CPUWrite(OAMDATA, 0)
				p.CPUWrite(OAMDATA, 0)
				p.CPUWrite(OAMDATA, uint8(i*8))
			}
			p.CPUWrite(PPUMASK, MASK_RENDER_SPR)

			clockUntil(t, p, 11, 0)
			overflow := p.CPURead(PPUSTATUS)&STATUS_SPRITE_OVERFLOW != 0
			if n == 8 && overflow {
				t.Fatal("8 sprites set overflow")
			}
			if n == 9 && !overflow {
				t.Fatal("9 sprites did not set overflow")
			}
		})
	}
}
