package cartridge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeINES(t *testing.T, prgPages, chrPages, flags6, flags7 byte, prg, chr, trainer []byte) string {
	t.Helper()
	prg = fitROM(prg, int(prgPages)*PRGROMPageSize)
	chr = fitROM(chr, int(chrPages)*CHRROMPageSize)
	buf := make([]byte, 0, INESHeaderSize+len(trainer)+len(prg)+len(chr))
	hdr := make([]byte, INESHeaderSize)
	copy(hdr[0:4], "NES\x1a")
	hdr[4] = prgPages
	hdr[5] = chrPages
	hdr[6] = flags6
	hdr[7] = flags7
	buf = append(buf, hdr...)
	if flags6&0x04 != 0 {
		if trainer == nil {
			trainer = make([]byte, trainerSize)
		}
		buf = append(buf, trainer...)
	}
	buf = append(buf, prg...)
	buf = append(buf, chr...)

	path := filepath.Join(t.TempDir(), "rom.nes")
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func fitROM(data []byte, size int) []byte {
	if len(data) == size {
		return data
	}
	out := make([]byte, size)
	copy(out, data)
	return out
}

func TestLoadRejectsUnsupportedMapper(t *testing.T) {
	cases := []struct {
		flags6 byte
		flags7 byte
		number string
	}{
		{flags6: 0x10, number: "unsupported mapper 1"},
		{flags6: 0x40, number: "unsupported mapper 4"},
		{flags6: 0x70, number: "unsupported mapper 7"},
		{flags7: 0xF0, flags6: 0xF0, number: "unsupported mapper 255"},
	}
	for _, tc := range cases {
		path := writeINES(t, 1, 1, tc.flags6, tc.flags7, nil, nil, nil)
		_, err := LoadCartridge(path)
		if err == nil || !strings.Contains(err.Error(), tc.number) {
			t.Fatalf("flags6=%02X flags7=%02X: got %v, want %q", tc.flags6, tc.flags7, err, tc.number)
		}
	}
}

func TestNROMMapping(t *testing.T) {
	prg16 := make([]byte, PRGROMPageSize)
	prg16[0] = 0x11
	prg16[0x100] = 0x22
	prg16[PRGROMPageSize-1] = 0x33
	c, err := LoadCartridge(writeINES(t, 1, 1, 0, 0, prg16, []byte{0xAB}, nil))
	if err != nil {
		t.Fatal(err)
	}
	if c.Mapper != 0 || c.Mirror != 0 {
		t.Fatalf("mapper/mirror = %d/%d", c.Mapper, c.Mirror)
	}
	for _, addr := range []uint16{0x8000, 0xC000} {
		if got := c.CPURead(addr); got != 0x11 {
			t.Fatalf("$%04X = %02X, want 11", addr, got)
		}
	}
	if c.CPURead(0x8100) != 0x22 || c.CPURead(0xC100) != 0x22 {
		t.Fatal("16KB mirror mismatch at +$100")
	}
	if c.CPURead(0xBFFF) != 0x33 || c.CPURead(0xFFFF) != 0x33 {
		t.Fatal("16KB mirror mismatch at the last byte")
	}

	prg32 := make([]byte, 2*PRGROMPageSize)
	prg32[0] = 0x44
	prg32[PRGROMPageSize] = 0x55
	prg32[2*PRGROMPageSize-1] = 0x66
	c, err = LoadCartridge(writeINES(t, 2, 1, 0x01, 0, prg32, nil, nil))
	if err != nil {
		t.Fatal(err)
	}
	if c.Mirror != 1 {
		t.Fatalf("Mirror = %d, want 1 (Flags 6 bit0)", c.Mirror)
	}
	if c.CPURead(0x8000) != 0x44 || c.CPURead(0xC000) != 0x55 || c.CPURead(0xFFFF) != 0x66 {
		t.Fatalf("32KB map got %02X %02X %02X", c.CPURead(0x8000), c.CPURead(0xC000), c.CPURead(0xFFFF))
	}
	c.CPUWrite(0x8000, 0x99)
	if c.CPURead(0x8000) != 0x44 {
		t.Fatal("NROM PRG accepted a write")
	}
}

func TestNROMRejectsOtherPRGSizes(t *testing.T) {
	for _, pages := range []byte{0, 3, 4} {
		path := writeINES(t, pages, 0, 0, 0, nil, nil, nil)
		_, err := LoadCartridge(path)
		if err == nil {
			t.Fatalf("NROM with %d PRG pages loaded", pages)
		}
	}
}

func TestUxROMBankSwitch(t *testing.T) {
	const banks = 4
	prg := make([]byte, banks*PRGROMPageSize)
	for b := 0; b < banks; b++ {
		base := b * PRGROMPageSize
		for i := range PRGROMPageSize {
			prg[base+i] = byte(0x10 + b)
		}
	}
	// Byte at $8000 in bank 0 would AND away a write of 0x02.
	prg[0] = 0x01
	c, err := LoadCartridge(writeINES(t, banks, 0, 0x20, 0, prg, nil, nil))
	if err != nil {
		t.Fatal(err)
	}
	if c.CPURead(0x8000) != 0x01 || c.CPURead(0x8001) != 0x10 {
		t.Fatalf("initial window got %02X %02X", c.CPURead(0x8000), c.CPURead(0x8001))
	}
	if c.CPURead(0xC000) != 0x13 || c.CPURead(0xFFFF) != 0x13 {
		t.Fatal("fixed bank is not the last bank")
	}

	// ROM at $8000 is 0x01. AND-type conflict would turn 0x02 into 0x00.
	c.CPUWrite(0x8000, 0x02)
	if c.CPURead(0x8000) != 0x12 || c.CPURead(0xBFFF) != 0x12 {
		t.Fatalf("switch to bank 2 got %02X", c.CPURead(0x8000))
	}
	if c.CPURead(0xC000) != 0x13 {
		t.Fatal("fixed bank changed")
	}

	// $FFFF is in the write window too. 0x05 masks to bank 1 on a 4-bank ROM.
	c.CPUWrite(0xFFFF, 0x05)
	if c.CPURead(0x8000) != 0x11 {
		t.Fatalf("8-bit value 0x05 selected %02X, want bank 1", c.CPURead(0x8000))
	}
	if c.CPURead(0xC000) != 0x13 {
		t.Fatal("fixed bank changed after write to $FFFF")
	}

	c.CPUWrite(0x7FFF, 0x03)
	if c.CPURead(0x8000) != 0x11 {
		t.Fatal("write below $8000 switched the bank")
	}

	c.CPUWrite(0x8000, 0)
	if c.CPURead(0x8001) != 0x10 || c.CPURead(0xC000) != 0x13 {
		t.Fatal("bank 0 / fixed last bank mismatch")
	}
}

func TestUxROMSixteenBanks(t *testing.T) {
	const banks = 16
	prg := make([]byte, banks*PRGROMPageSize)
	for b := 0; b < banks; b++ {
		prg[b*PRGROMPageSize] = byte(b)
	}
	c, err := LoadCartridge(writeINES(t, banks, 0, 0x20, 0, prg, nil, nil))
	if err != nil {
		t.Fatal(err)
	}
	c.CPUWrite(0x8000, 0x0A)
	if c.CPURead(0x8000) != 0x0A {
		t.Fatalf("UOROM-sized bank = %02X, want 0A", c.CPURead(0x8000))
	}
	if c.CPURead(0xC000) != byte(banks-1) {
		t.Fatal("last bank is not fixed")
	}
}

func TestUxROMRejectsEmptyPRG(t *testing.T) {
	path := writeINES(t, 0, 0, 0x20, 0, nil, nil, nil)
	if _, err := LoadCartridge(path); err == nil {
		t.Fatal("UxROM with no PRG loaded")
	}
}

func TestCNROMBankSwitch(t *testing.T) {
	prg := make([]byte, PRGROMPageSize)
	prg[0] = 0x01
	prg[0x10] = 0x55
	chr := make([]byte, 4*CHRROMPageSize)
	for b := 0; b < 4; b++ {
		base := b * CHRROMPageSize
		chr[base] = byte(0xA0 + b)
		chr[base+CHRROMPageSize-1] = byte(0xB0 + b)
	}
	c, err := LoadCartridge(writeINES(t, 1, 4, 0x30, 0, prg, chr, nil))
	if err != nil {
		t.Fatal(err)
	}
	if c.PPURead(0x0000) != 0xA0 || c.PPURead(0x1FFF) != 0xB0 {
		t.Fatalf("initial CHR got %02X %02X", c.PPURead(0x0000), c.PPURead(0x1FFF))
	}
	if c.CPURead(0x8000) != 0x01 || c.CPURead(0xC010) != 0x55 {
		t.Fatal("CNROM PRG is not the fixed 16KB mirror")
	}

	// $8000 holds 0x01. AND with 0x02 would yield 0 and stay on bank 0.
	c.CPUWrite(0x8000, 0x02)
	if c.PPURead(0x0000) != 0xA2 || c.PPURead(0x1FFF) != 0xB2 {
		t.Fatalf("CHR bank 2 got %02X %02X", c.PPURead(0x0000), c.PPURead(0x1FFF))
	}
	if c.CPURead(0x8010) != 0x55 {
		t.Fatal("CHR bank write changed PRG")
	}

	c.CPUWrite(0xFFFF, 0x05) // 0x05 % 4 == 1
	if c.PPURead(0x0000) != 0xA1 {
		t.Fatalf("masked CHR bank got %02X, want bank 1", c.PPURead(0x0000))
	}

	before := c.PPURead(0x0000)
	c.PPUWrite(0x0000, 0xFF)
	if c.PPURead(0x0000) != before {
		t.Fatal("CHR-ROM accepted a PPU write")
	}
}

func TestCNROMRejectsOtherPRGSizes(t *testing.T) {
	path := writeINES(t, 4, 1, 0x30, 0, nil, nil, nil)
	if _, err := LoadCartridge(path); err == nil {
		t.Fatal("CNROM with 64KB PRG loaded")
	}
}

func TestCHRRAMAndROM(t *testing.T) {
	prg := make([]byte, PRGROMPageSize)
	c, err := LoadCartridge(writeINES(t, 1, 0, 0, 0, prg, nil, nil))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.CHRROM) != 0 {
		t.Fatal("CHR size 0 should not populate CHRROM")
	}
	if c.PPURead(0x0000) != 0 || c.PPURead(0x1FFF) != 0 {
		t.Fatal("CHR-RAM should power up as zero")
	}
	c.PPUWrite(0x0000, 0x12)
	c.PPUWrite(0x1FFF, 0x34)
	c.PPUWrite(0x0100, 0x56)
	if c.PPURead(0x0000) != 0x12 || c.PPURead(0x1FFF) != 0x34 || c.PPURead(0x0100) != 0x56 {
		t.Fatal("CHR-RAM did not read back")
	}
	c.PPUWrite(0x2000, 0x99)
	if c.PPURead(0x0000) != 0x12 {
		t.Fatal("write above $1FFF changed CHR-RAM")
	}

	chr := make([]byte, CHRROMPageSize)
	chr[0] = 0x77
	c, err = LoadCartridge(writeINES(t, 1, 1, 0, 0, prg, chr, nil))
	if err != nil {
		t.Fatal(err)
	}
	c.PPUWrite(0x0000, 0x88)
	if c.PPURead(0x0000) != 0x77 {
		t.Fatalf("CHR-ROM read %02X after write", c.PPURead(0x0000))
	}
}

func TestTrainerIsSkipped(t *testing.T) {
	trainer := make([]byte, trainerSize)
	for i := range trainer {
		trainer[i] = 0xEE
	}
	prg := make([]byte, PRGROMPageSize)
	prg[0] = 0x42
	prg[PRGROMPageSize-1] = 0x43
	chr := []byte{0x44}
	// Flags 6: mapper 0, trainer, mirror bit 0. Mirror must stay 1.
	c, err := LoadCartridge(writeINES(t, 1, 1, 0x05, 0, prg, chr, trainer))
	if err != nil {
		t.Fatal(err)
	}
	if c.Mirror != 1 {
		t.Fatalf("Mirror = %d, want 1", c.Mirror)
	}
	if c.CPURead(0x8000) != 0x42 || c.CPURead(0xFFFF) != 0x43 {
		t.Fatalf("PRG after trainer got %02X %02X", c.CPURead(0x8000), c.CPURead(0xFFFF))
	}
	if c.PPURead(0x0000) != 0x44 {
		t.Fatalf("CHR after trainer got %02X", c.PPURead(0x0000))
	}
}

func TestZeroCartridgeLiteral(t *testing.T) {
	c := &Cartridge{}
	c.CPUWrite(0x8000, 0x7F)
	if got := c.CPURead(0x8000); got != 0 {
		t.Fatalf("empty NROM CPURead = %02X", got)
	}
	c.PPUWrite(0x0123, 0xAB)
	if got := c.PPURead(0x0123); got != 0xAB {
		t.Fatalf("empty cartridge PPURead = %02X, want AB", got)
	}
	if c.Mapper != 0 || c.Mirror != 0 {
		t.Fatalf("zero literal mapper/mirror = %d/%d", c.Mapper, c.Mirror)
	}
}
