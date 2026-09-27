package ppu

import (
	"testing"

	"github.com/misawa/go-nes-by-pstack/internal/cartridge"
)

// TestNametableColumnAppearsOnScreenColumn checks the background prefetch
// window (dots 321–336). Each nametable column is a distinct tile, and
// screen tile-column N must show that tile, including columns 0 and 31.
// Dots 321–336 must shift the BG registers every dot and call IncrementX
// at 328 and 336. Otherwise column 0 shows the previous line's column 30,
// columns 1 and 2 both show column 0, and column 31 never appears.
// https://www.nesdev.org/wiki/PPU_rendering
// https://www.nesdev.org/wiki/PPU_scrolling
func TestNametableColumnAppearsOnScreenColumn(t *testing.T) {
	const (
		bgColor = 0x0E
		fgColor = 0x30
	)

	chr := make([]byte, cartridge.CHRROMPageSize)
	for col := 0; col < 32; col++ {
		// Plane 0 of every row is the column number. Bit 7 is the left pixel.
		// Plane 1 stays 0, so each pixel is palette index 0 or 1.
		tile := col * 16
		for y := 0; y < 8; y++ {
			chr[tile+y] = byte(col)
		}
	}
	p := NewPPU()
	p.ConnectCartridge(&cartridge.Cartridge{
		PRGROM: make([]byte, cartridge.PRGROMPageSize),
		CHRROM: chr,
		Mapper: 0,
		Mirror: 0,
	})

	for y := 0; y < 30; y++ {
		for x := 0; x < 32; x++ {
			writeVRAM(p, uint16(0x2000+y*32+x), byte(x))
		}
	}
	writeVRAM(p, 0x3F00, bgColor)
	writeVRAM(p, 0x3F01, fgColor)

	// Rendering stays off through the first vblank so the fetcher phase is
	// still 0 when the pre-render prefetch starts. Scroll is set after the
	// $2006 writes, which share t with $2005.
	clockUntil(t, p, 241, 1)
	p.CPUWrite(PPUCTRL, 0)
	p.CPUWrite(PPUSCROLL, 0)
	p.CPUWrite(PPUSCROLL, 0)
	p.CPUWrite(PPUMASK, MASK_RENDER_BG|MASK_RENDER_BG_LEFT)

	// Next post-render line is after scanlines 0–239 of the frame that
	// was fetched with rendering enabled.
	clockUntil(t, p, 240, 0)

	screen := p.GetScreen()
	// y=0 is filled by the pre-render prefetch (dots 321–336 of clockPreRender).
	// A later row is filled by the same window in clockVisible.
	for _, y := range []int{0, 16} {
		for col := 0; col < 32; col++ {
			got := decodeBGColumn(t, screen, y, col, bgColor, fgColor)
			if got != byte(col) {
				t.Errorf("y=%d screen column %d = nametable column %d, want %d", y, col, got, col)
			}
		}
	}
}

func decodeBGColumn(t *testing.T, screen *[240][256]uint8, y, col int, bgColor, fgColor byte) byte {
	t.Helper()
	var got byte
	for px := 0; px < 8; px++ {
		color := screen[y][col*8+px]
		got <<= 1
		switch color {
		case bgColor:
		case fgColor:
			got |= 1
		default:
			t.Fatalf("y=%d screen column %d pixel %d color %02X, want %02X or %02X", y, col, px, color, bgColor, fgColor)
		}
	}
	return got
}
