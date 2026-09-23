package nes

import (
	"fmt"

	"github.com/misawa/go-nes-by-pstack/internal/bus"
	"github.com/misawa/go-nes-by-pstack/internal/cartridge"
	"github.com/misawa/go-nes-by-pstack/internal/cpu"
	"github.com/misawa/go-nes-by-pstack/internal/ppu"
)

// Console is the public NES. Callers load a ROM, reset or start nestest, then step.
type Console struct {
	CPU  *cpu.CPU
	PPU  *ppu.PPU
	Bus  *bus.Bus
	Cart *cartridge.Cartridge
	dot  uint64
}

func Open(path string) (*Console, error) {
	cart, err := cartridge.LoadCartridge(path)
	if err != nil {
		return nil, err
	}
	c := &Console{
		CPU:  cpu.NewCPU(),
		PPU:  ppu.NewPPU(),
		Bus:  bus.NewBus(),
		Cart: cart,
	}
	c.CPU.ConnectBus(c.Bus)
	c.Bus.AttachPPU(c.PPU)
	c.Bus.AttachCartridge(cart)
	c.PPU.ConnectCartridge(cart)
	return c, nil
}

func (c *Console) Reset() {
	c.CPU.Reset()
	c.dot = 0
}

// PowerOnNestest matches the official nestest automation start state.
func (c *Console) PowerOnNestest() {
	c.CPU.PC = 0xC000
	c.CPU.Status = 0x24
	c.CPU.SetTotalCycles(7)
	c.PPU.SetTiming(0, 21)
}

func (c *Console) Clock() {
	c.PPU.Clock()
	if c.dot%3 == 0 {
		if c.PPU.NMI() {
			c.CPU.NMI()
		}
		c.CPU.Clock()
	}
	c.dot++
}

func (c *Console) StepInstruction() {
	before := c.CPU.GetTotalCycles()
	c.CPU.Clock()
	for !c.CPU.Complete() {
		c.CPU.Clock()
	}
	used := int(c.CPU.GetTotalCycles() - before)
	for i := 0; i < used*3; i++ {
		c.PPU.Clock()
	}
}

func (c *Console) StepFrame() {
	for i := 0; i < 89342; i++ {
		c.Clock()
	}
}

func (c *Console) Frame() *[240][256]uint8 {
	return c.PPU.GetScreen()
}

func (c *Console) TraceRegs() string {
	scan := int(c.PPU.GetScanline())
	dot := int(c.PPU.GetCycle())
	if scan == -1 {
		scan = 261
	}
	return fmt.Sprintf("%04X A:%02X X:%02X Y:%02X P:%02X SP:%02X PPU:%3d,%3d CYC:%d",
		c.CPU.PC, c.CPU.A, c.CPU.X, c.CPU.Y, c.CPU.Status, c.CPU.SP, scan, dot, c.CPU.GetTotalCycles())
}
