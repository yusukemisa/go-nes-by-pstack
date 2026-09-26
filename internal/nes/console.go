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

	// OAM DMA halt requested through Bus.TakeOAMDMA.
	dmaLeft   int
	dmaQueued bool
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
	c.dmaLeft = 0
	c.dmaQueued = false
	c.Bus.TakeOAMDMA()
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
	if c.PPU.NMI() {
		c.CPU.RequestNMI()
	}
	if c.dot%3 == 0 {
		c.cpuClock()
	}
	c.dot++
}

// cpuClock advances one CPU cycle. NMI is taken only when the previous
// instruction has finished, and OAM DMA halts the CPU without dropping that
// instruction. https://www.nesdev.org/wiki/CPU_interrupts
func (c *Console) cpuClock() {
	if c.dmaLeft > 0 {
		c.dmaLeft--
		c.CPU.AddCycle()
		return
	}
	if _, ok := c.Bus.TakeOAMDMA(); ok {
		c.dmaQueued = true
	}
	if c.CPU.Complete() && c.dmaQueued {
		c.startOAMDMA()
		return
	}
	c.CPU.ServiceNMI()
	c.CPU.Clock()
	if _, ok := c.Bus.TakeOAMDMA(); ok {
		c.dmaQueued = true
	}
}

// startOAMDMA stalls for 513 or 514 CPU cycles after the $4014 write.
// The write is the store's last cycle. Even GetTotalCycles at the start of
// that cycle is treated as an APU get cycle (513); odd is a put cycle and
// needs an alignment cycle (514). Parity is only an approximation of the
// APU get/put phase, which is random at power-on.
// https://www.nesdev.org/wiki/DMA
func (c *Console) startOAMDMA() {
	writeCycle := c.CPU.GetTotalCycles() - 1
	n := 513
	if writeCycle&1 == 1 {
		n = 514
	}
	c.dmaQueued = false
	c.dmaLeft = n - 1
	c.CPU.AddCycle()
}

func (c *Console) StepInstruction() {
	before := c.CPU.GetTotalCycles()
	c.cpuClock()
	for !c.CPU.Complete() || c.dmaLeft > 0 || c.dmaQueued {
		c.cpuClock()
	}
	used := int(c.CPU.GetTotalCycles() - before)
	for i := 0; i < used*3; i++ {
		c.PPU.Clock()
		if c.PPU.NMI() {
			c.CPU.RequestNMI()
		}
	}
	c.dot += uint64(used) * 3
}

// StepFrame runs until the PPU signals the end of a frame, from whatever
// dot the caller is on. A fixed 89342-dot loop would drift once a frame
// skips a dot. https://www.nesdev.org/wiki/PPU_frame_timing
func (c *Console) StepFrame() {
	for {
		c.Clock()
		if c.PPU.FrameComplete() {
			return
		}
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
