package bus

import (
	"github.com/misawa/go-nes-by-pstack/internal/apu"
	"github.com/misawa/go-nes-by-pstack/internal/cartridge"
	"github.com/misawa/go-nes-by-pstack/internal/controller"
)

type Bus struct {
	cpu_ram     [2048]uint8 // 2KB internal RAM
	cart        *cartridge.Cartridge
	ppu         PPU
	apu         *apu.APU
	controller1 *controller.Controller
	controller2 *controller.Controller
}

type PPU interface {
	CPURead(addr uint16) uint8
	CPUWrite(addr uint16, data uint8)
	Clock()
	OAMDMA(page uint8, ram []uint8)
	GetTotalCycles() int
	GetScanline() int16
	GetCycle() uint16
}

func NewBus() *Bus {
	return &Bus{
		apu:         apu.New(),
		controller1: controller.NewController(),
		controller2: controller.NewController(),
	}
}

func (b *Bus) AttachCartridge(cart *cartridge.Cartridge) {
	b.cart = cart
}

func (b *Bus) AttachPPU(ppu PPU) {
	b.ppu = ppu
}

func (b *Bus) CPURead(addr uint16) uint8 {
	switch {
	case addr >= 0x0000 && addr <= 0x1FFF:
		// 2KB internal RAM, mirrored every 2KB up to 0x1FFF
		return b.cpu_ram[addr&0x07FF]
	case addr >= 0x2000 && addr <= 0x3FFF:
		// PPU registers, mirrored every 8 bytes
		if b.ppu != nil {
			return b.ppu.CPURead(0x2000 + (addr&0x0007))
		}
		return 0
	case addr >= 0x4000 && addr <= 0x4013:
		// APU registers
		return b.apu.Read(addr)
	case addr == 0x4015:
		// APU status register
		return b.apu.Read(addr)
	case addr == 0x4016:
		// Controller 1
		return b.controller1.Read()
	case addr == 0x4017:
		// Controller 2 or APU frame counter
		if addr == 0x4017 {
			return b.controller2.Read()
		}
		return b.apu.Read(addr)
	case addr >= 0x8000 && addr <= 0xFFFF:
		// Cartridge space
		if b.cart != nil {
			return b.cart.CPURead(addr)
		}
		return 0
	}
	return 0
}

func (b *Bus) CPUWrite(addr uint16, data uint8) {
	switch {
	case addr >= 0x0000 && addr <= 0x1FFF:
		// 2KB internal RAM, mirrored every 2KB up to 0x1FFF
		b.cpu_ram[addr&0x07FF] = data
	case addr >= 0x2000 && addr <= 0x3FFF:
		// PPU registers, mirrored every 8 bytes
		if b.ppu != nil {
			b.ppu.CPUWrite(0x2000+(addr&0x0007), data)
		}
	case addr >= 0x4000 && addr <= 0x4013:
		// APU registers
		b.apu.Write(addr, data)
	case addr == 0x4014:
		// OAM DMA - Transfer 256 bytes from page $XX00-$XXFF to OAM
		if b.ppu != nil {
			b.ppu.OAMDMA(data, b.cpu_ram[:])
		}
	case addr == 0x4015:
		// APU status register
		b.apu.Write(addr, data)
	case addr == 0x4016:
		// Controller strobe
		b.controller1.Write(data)
		b.controller2.Write(data)
	case addr == 0x4017:
		// APU frame counter
		b.apu.Write(addr, data)
	case addr >= 0x8000 && addr <= 0xFFFF:
		// Cartridge space
		if b.cart != nil {
			b.cart.CPUWrite(addr, data)
		}
	}
}

func (b *Bus) GetController1() *controller.Controller {
	return b.controller1
}

func (b *Bus) GetController2() *controller.Controller {
	return b.controller2
}

// GetPPUCycles returns the current PPU cycle information
func (b *Bus) GetPPUCycles() (int, int16, uint16) {
	if b.ppu != nil {
		return b.ppu.GetTotalCycles(), b.ppu.GetScanline(), b.ppu.GetCycle()
	}
	return 0, 0, 0
}