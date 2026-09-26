package cartridge

import (
	"fmt"
	"os"
)

const (
	INESHeaderSize = 16
	PRGROMPageSize = 16384 // 16KB
	CHRROMPageSize = 8192  // 8KB
	trainerSize    = 512
)

type Cartridge struct {
	PRGROM []byte
	CHRROM []byte
	Mapper uint8
	// Mirror is iNES Flags 6 bit 0 stored as-is. Nametable layout is the PPU's job.
	// https://www.nesdev.org/wiki/INES
	Mirror uint8

	prgBank int
	chrBank int
	chrRAM  []byte
}

func LoadCartridge(filename string) (*Cartridge, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read ROM file: %w", err)
	}

	if len(data) < INESHeaderSize {
		return nil, fmt.Errorf("invalid ROM file: too small")
	}

	// Check iNES header signature
	if string(data[0:4]) != "NES\x1a" {
		return nil, fmt.Errorf("invalid iNES header")
	}

	prgPages := int(data[4])
	chrPages := int(data[5])
	flags6 := data[6]
	flags7 := data[7]

	// Mapper number: high nibble from Flags 7, low nibble from Flags 6.
	// https://www.nesdev.org/wiki/INES
	mapper := (flags7 & 0xF0) | (flags6 >> 4)

	// Flags 6 bit 0, unchanged. Do not interpret horizontal vs vertical here.
	mirror := flags6 & 0x01

	switch mapper {
	case 0, 2, 3:
	default:
		return nil, fmt.Errorf("unsupported mapper %d", mapper)
	}

	prgSize := prgPages * PRGROMPageSize
	chrSize := chrPages * CHRROMPageSize

	// NROM and CNROM fix PRG at 16KB (mirrored) or 32KB.
	// https://www.nesdev.org/wiki/NROM
	// https://www.nesdev.org/wiki/CNROM
	if mapper == 0 || mapper == 3 {
		if prgSize != PRGROMPageSize && prgSize != 2*PRGROMPageSize {
			return nil, fmt.Errorf("mapper %d PRG size must be 16KB or 32KB, got %d bytes", mapper, prgSize)
		}
	}
	// UxROM needs at least one 16KB bank so $C000 can fix the last bank.
	// https://www.nesdev.org/wiki/UxROM
	if mapper == 2 && prgPages < 1 {
		return nil, fmt.Errorf("UxROM PRG size must be at least 16KB, got %d bytes", prgSize)
	}

	// Flags 6 bit 2: 512-byte trainer stored before PRG. Skip it; do not map it.
	// https://www.nesdev.org/wiki/INES
	trainer := 0
	if flags6&0x04 != 0 {
		trainer = trainerSize
	}

	expectedSize := INESHeaderSize + trainer + prgSize + chrSize
	if len(data) < expectedSize {
		return nil, fmt.Errorf("ROM file too small: expected %d bytes, got %d", expectedSize, len(data))
	}

	prgStart := INESHeaderSize + trainer
	prgEnd := prgStart + prgSize
	prgROM := make([]byte, prgSize)
	copy(prgROM, data[prgStart:prgEnd])

	var chrROM []byte
	var chrRAM []byte
	if chrSize > 0 {
		chrROM = make([]byte, chrSize)
		copy(chrROM, data[prgEnd:prgEnd+chrSize])
	} else {
		// Byte 5 == 0 means CHR-RAM. NROM and UxROM homebrew use 8KB.
		// https://www.nesdev.org/wiki/INES
		// https://www.nesdev.org/wiki/CHR-ROM_vs_CHR-RAM
		chrRAM = make([]byte, CHRROMPageSize)
	}

	return &Cartridge{
		PRGROM: prgROM,
		CHRROM: chrROM,
		Mapper: mapper,
		Mirror: mirror,
		chrRAM: chrRAM,
	}, nil
}

func (c *Cartridge) CPURead(addr uint16) uint8 {
	if addr < 0x8000 {
		return 0
	}
	if c.Mapper == 2 {
		return c.readUxROM(addr)
	}
	// NROM and CNROM: PRG is not banked.
	return c.readFixedPRG(addr)
}

func (c *Cartridge) CPUWrite(addr uint16, data uint8) {
	if addr < 0x8000 {
		return
	}
	switch c.Mapper {
	case 2:
		// Full 8-bit bank number, no bus conflict. High bits past the
		// file's bank count are ignored (equivalent to unconnected ROM
		// address lines when the count is a power of two).
		// https://www.nesdev.org/wiki/UxROM
		n := c.prgBankCount()
		if n == 0 {
			return
		}
		c.prgBank = int(data) % n
	case 3:
		// Plain iNES has no submapper. Submapper 0 is "bus conflict unknown"
		// (1: none, 2: AND with the PRG byte). Latch the written byte with
		// no bus conflict. Security diodes are not emulated.
		// https://www.nesdev.org/wiki/CNROM
		n := c.chrBankCount()
		if n == 0 {
			return
		}
		c.chrBank = int(data) % n
	}
}

func (c *Cartridge) PPURead(addr uint16) uint8 {
	if addr > 0x1FFF {
		return 0
	}
	if c.chrIsRAM() {
		c.ensureCHRRAM()
		return c.chrRAM[addr]
	}
	off := c.chrOffset(addr)
	if off >= 0 && off < len(c.CHRROM) {
		return c.CHRROM[off]
	}
	return 0
}

func (c *Cartridge) PPUWrite(addr uint16, data uint8) {
	if addr > 0x1FFF {
		return
	}
	// CHR-ROM is read-only. CHR size 0 is 8KB CHR-RAM.
	if !c.chrIsRAM() {
		return
	}
	c.ensureCHRRAM()
	c.chrRAM[addr] = data
}

func (c *Cartridge) readFixedPRG(addr uint16) uint8 {
	index := int(addr - 0x8000)
	switch len(c.PRGROM) {
	case PRGROMPageSize:
		// 16KB PRG-ROM, mirrored at $8000 and $C000 (NROM-128).
		index %= PRGROMPageSize
	case 2 * PRGROMPageSize:
		// 32KB PRG-ROM, direct map (NROM-256).
	}
	if index >= 0 && index < len(c.PRGROM) {
		return c.PRGROM[index]
	}
	return 0
}

func (c *Cartridge) readUxROM(addr uint16) uint8 {
	n := c.prgBankCount()
	if n == 0 {
		return 0
	}
	bank := c.prgBank
	if addr >= 0xC000 {
		// $C000-$FFFF is fixed to the last 16KB bank.
		bank = n - 1
	}
	if bank < 0 || bank >= n {
		return 0
	}
	off := bank*PRGROMPageSize + int(addr&0x3FFF)
	if off < len(c.PRGROM) {
		return c.PRGROM[off]
	}
	return 0
}

func (c *Cartridge) chrOffset(addr uint16) int {
	bank := 0
	if c.Mapper == 3 {
		bank = c.chrBank
	}
	return bank*CHRROMPageSize + int(addr)
}

func (c *Cartridge) prgBankCount() int {
	return len(c.PRGROM) / PRGROMPageSize
}

func (c *Cartridge) chrBankCount() int {
	return len(c.CHRROM) / CHRROMPageSize
}

func (c *Cartridge) chrIsRAM() bool {
	return len(c.CHRROM) == 0
}

func (c *Cartridge) ensureCHRRAM() {
	if c.chrRAM == nil {
		c.chrRAM = make([]byte, CHRROMPageSize)
	}
}
