package ppu

import (
	"testing"

	"github.com/misawa/go-nes-by-pstack/internal/cartridge"
)

func setPPUAddr(p *PPU, addr uint16) {
	p.CPUWrite(PPUADDR, uint8(addr>>8))
	p.CPUWrite(PPUADDR, uint8(addr))
}

func TestPPUDataReadAdvancesByOne(t *testing.T) {
	p := NewPPU()
	setPPUAddr(p, 0x3F00)
	p.CPUWrite(PPUDATA, 0x11)
	p.CPUWrite(PPUDATA, 0x22)
	setPPUAddr(p, 0x3F00)

	first := p.CPURead(PPUDATA)
	second := p.CPURead(PPUDATA)
	if first != 0x11 || second != 0x22 {
		t.Fatalf("got %02X %02X, want 11 22", first, second)
	}
}

func TestPPUDataReadAdvancesByThirtyTwo(t *testing.T) {
	p := NewPPU()
	p.ConnectCartridge(&cartridge.Cartridge{})
	setPPUAddr(p, 0x2000)
	p.CPUWrite(PPUDATA, 0xAA)
	setPPUAddr(p, 0x2020)
	p.CPUWrite(PPUDATA, 0xCC)
	p.CPUWrite(PPUCTRL, CTRL_INCREMENT_MODE)
	setPPUAddr(p, 0x2000)

	_ = p.CPURead(PPUDATA)
	second := p.CPURead(PPUDATA)
	third := p.CPURead(PPUDATA)
	if second != 0xAA || third != 0xCC {
		t.Fatalf("got %02X %02X, want AA CC", second, third)
	}
}
