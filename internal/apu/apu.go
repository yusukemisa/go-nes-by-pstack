package apu

type APU struct {
	registers [0x18]uint8
}

func New() *APU {
	apu := &APU{}
	// Initialize registers to 0xFF for nestest compatibility
	for i := range apu.registers {
		apu.registers[i] = 0xFF
	}
	return apu
}

func (apu *APU) Read(addr uint16) uint8 {
	if addr >= 0x4000 && addr <= 0x4017 {
		return apu.registers[addr-0x4000]
	}
	return 0x00
}

func (apu *APU) Write(addr uint16, data uint8) {
	if addr >= 0x4000 && addr <= 0x4017 {
		apu.registers[addr-0x4000] = data
	}
}