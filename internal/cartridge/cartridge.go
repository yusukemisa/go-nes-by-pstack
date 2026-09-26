package cartridge

import (
	"fmt"
	"os"
)

const (
	INESHeaderSize = 16
	PRGROMPageSize = 16384 // 16KB
	CHRROMPageSize = 8192  // 8KB
)

type Cartridge struct {
	PRGROM []byte
	CHRROM []byte
	Mapper uint8
	Mirror uint8
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

	// Extract mapper number
	mapper := ((flags7 & 0xF0) | (flags6 >> 4))
	
	// Extract mirroring type
	mirror := flags6 & 0x01

	// Calculate ROM sizes
	prgSize := prgPages * PRGROMPageSize
	chrSize := chrPages * CHRROMPageSize

	// Check if file has enough data
	expectedSize := INESHeaderSize + prgSize + chrSize
	if len(data) < expectedSize {
		return nil, fmt.Errorf("ROM file too small: expected %d bytes, got %d", expectedSize, len(data))
	}

	// Extract PRG-ROM
	prgStart := INESHeaderSize
	prgEnd := prgStart + prgSize
	prgROM := make([]byte, prgSize)
	copy(prgROM, data[prgStart:prgEnd])

	// Extract CHR-ROM
	chrROM := make([]byte, chrSize)
	if chrSize > 0 {
		chrStart := prgEnd
		chrEnd := chrStart + chrSize
		copy(chrROM, data[chrStart:chrEnd])
	}

	return &Cartridge{
		PRGROM:    prgROM,
		CHRROM:    chrROM,
		Mapper:    mapper,
		Mirror:    mirror,
	}, nil
}

func (c *Cartridge) CPURead(addr uint16) uint8 {
	if addr >= 0x8000 && addr <= 0xFFFF {
		// Map to PRG-ROM
		index := int(addr - 0x8000)
		if len(c.PRGROM) == PRGROMPageSize {
			// 16KB PRG-ROM, mirror in both halves
			index = index % PRGROMPageSize
		} else if len(c.PRGROM) == 2*PRGROMPageSize {
			// 32KB PRG-ROM, direct mapping
			// index stays as is
		}
		if index < len(c.PRGROM) {
			return c.PRGROM[index]
		}
	}
	return 0
}

func (c *Cartridge) CPUWrite(addr uint16, data uint8) {
	// Most mappers don't allow writes to ROM area
}

func (c *Cartridge) PPURead(addr uint16) uint8 {
	if addr >= 0x0000 && addr <= 0x1FFF {
		// CHR-ROM
		if int(addr) < len(c.CHRROM) {
			return c.CHRROM[addr]
		}
	}
	return 0
}

func (c *Cartridge) PPUWrite(addr uint16, data uint8) {
	// CHR-ROM is read-only
}
