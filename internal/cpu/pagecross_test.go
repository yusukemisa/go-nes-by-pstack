package cpu

import "testing"

// pageCrossBus is a flat memory image for page-cross cycle tests.
type pageCrossBus struct {
	mem [1 << 16]uint8
}

func (b *pageCrossBus) CPURead(addr uint16) uint8 { return b.mem[addr] }

func (b *pageCrossBus) CPUWrite(addr uint16, data uint8) { b.mem[addr] = data }

func (b *pageCrossBus) GetPPUCycles() (int, int16, uint16) { return 0, 0, 0 }

// Indexed reads use base $10F0 + index $20, which is the LDA $10F0,Y case.
const (
	pageCrossSameBase = 0x1000
	pageCrossNextBase = 0x10F0
	pageCrossIndex    = 0x20
	pageCrossZP       = 0x10
	pageCrossProg     = 0x8000
	pageCrossMemValue = 0x42
	pageCrossStoreVal = 0x99
)

type pageCrossOp struct {
	name    string
	opcode  uint8
	mode    string // absX, absY, or izy
	same    int
	crossed int
	store   bool
}

func TestPageCrossCycles(t *testing.T) {
	// Cycle counts: indexed reads add 1 on a page cross; stores and RMW do not.
	// https://www.nesdev.org/wiki/Instruction_reference
	ops := []pageCrossOp{
		{"LDA abs,X", 0xBD, "absX", 4, 5, false},
		{"LDA abs,Y", 0xB9, "absY", 4, 5, false},
		{"LDA (zp),Y", 0xB1, "izy", 5, 6, false},
		{"LDX abs,Y", 0xBE, "absY", 4, 5, false},
		{"LDY abs,X", 0xBC, "absX", 4, 5, false},
		{"EOR abs,X", 0x5D, "absX", 4, 5, false},
		{"EOR abs,Y", 0x59, "absY", 4, 5, false},
		{"EOR (zp),Y", 0x51, "izy", 5, 6, false},
		{"AND abs,X", 0x3D, "absX", 4, 5, false},
		{"AND abs,Y", 0x39, "absY", 4, 5, false},
		{"AND (zp),Y", 0x31, "izy", 5, 6, false},
		{"ORA abs,X", 0x1D, "absX", 4, 5, false},
		{"ORA abs,Y", 0x19, "absY", 4, 5, false},
		{"ORA (zp),Y", 0x11, "izy", 5, 6, false},
		{"ADC abs,X", 0x7D, "absX", 4, 5, false},
		{"ADC abs,Y", 0x79, "absY", 4, 5, false},
		{"ADC (zp),Y", 0x71, "izy", 5, 6, false},
		{"SBC abs,X", 0xFD, "absX", 4, 5, false},
		{"SBC abs,Y", 0xF9, "absY", 4, 5, false},
		{"SBC (zp),Y", 0xF1, "izy", 5, 6, false},
		{"CMP abs,X", 0xDD, "absX", 4, 5, false},
		{"CMP abs,Y", 0xD9, "absY", 4, 5, false},
		{"CMP (zp),Y", 0xD1, "izy", 5, 6, false},
		{"LAX abs,Y", 0xBF, "absY", 4, 5, false},
		{"LAX (zp),Y", 0xB3, "izy", 5, 6, false},
		{"NOP abs,X $1C", 0x1C, "absX", 4, 5, false},
		{"NOP abs,X $3C", 0x3C, "absX", 4, 5, false},
		{"NOP abs,X $5C", 0x5C, "absX", 4, 5, false},
		{"NOP abs,X $7C", 0x7C, "absX", 4, 5, false},
		{"NOP abs,X $DC", 0xDC, "absX", 4, 5, false},
		{"NOP abs,X $FC", 0xFC, "absX", 4, 5, false},

		{"STA abs,X", 0x9D, "absX", 5, 5, true},
		{"STA abs,Y", 0x99, "absY", 5, 5, true},
		{"STA (zp),Y", 0x91, "izy", 6, 6, true},

		{"ASL abs,X", 0x1E, "absX", 7, 7, false},
		{"LSR abs,X", 0x5E, "absX", 7, 7, false},
		{"ROL abs,X", 0x3E, "absX", 7, 7, false},
		{"ROR abs,X", 0x7E, "absX", 7, 7, false},
		{"INC abs,X", 0xFE, "absX", 7, 7, false},
		{"DEC abs,X", 0xDE, "absX", 7, 7, false},

		{"SLO abs,X", 0x1F, "absX", 7, 7, false},
		{"SLO abs,Y", 0x1B, "absY", 7, 7, false},
		{"SLO (zp),Y", 0x13, "izy", 8, 8, false},
		{"RLA abs,X", 0x3F, "absX", 7, 7, false},
		{"RLA abs,Y", 0x3B, "absY", 7, 7, false},
		{"RLA (zp),Y", 0x33, "izy", 8, 8, false},
		{"SRE abs,X", 0x5F, "absX", 7, 7, false},
		{"SRE abs,Y", 0x5B, "absY", 7, 7, false},
		{"SRE (zp),Y", 0x53, "izy", 8, 8, false},
		{"RRA abs,X", 0x7F, "absX", 7, 7, false},
		{"RRA abs,Y", 0x7B, "absY", 7, 7, false},
		{"RRA (zp),Y", 0x73, "izy", 8, 8, false},
		{"DCP abs,X", 0xDF, "absX", 7, 7, false},
		{"DCP abs,Y", 0xDB, "absY", 7, 7, false},
		{"DCP (zp),Y", 0xD3, "izy", 8, 8, false},
		{"ISB abs,X", 0xFF, "absX", 7, 7, false},
		{"ISB abs,Y", 0xFB, "absY", 7, 7, false},
		{"ISB (zp),Y", 0xF3, "izy", 8, 8, false},
	}

	for _, op := range ops {
		for _, crossed := range []bool{false, true} {
			want := op.same
			label := "same-page"
			if crossed {
				want = op.crossed
				label = "crossed"
			}
			t.Run(op.name+"/"+label, func(t *testing.T) {
				got := runPageCrossOp(t, op.opcode, op.mode, crossed)
				if got.clocks != want || int(got.total) != want {
					t.Fatalf("Clock()=%d GetTotalCycles()=%d, want both %d", got.clocks, got.total, want)
				}
				if op.opcode == 0xB9 && crossed && got.cpu.A != pageCrossMemValue {
					t.Fatalf("LDA $10F0,Y loaded $%02X, want $%02X", got.cpu.A, pageCrossMemValue)
				}
				if op.opcode == 0xBF && crossed && (got.cpu.A != pageCrossMemValue || got.cpu.X != pageCrossMemValue) {
					t.Fatalf("LAX abs,Y A=$%02X X=$%02X, want $%02X", got.cpu.A, got.cpu.X, pageCrossMemValue)
				}
				if op.store && got.bus.mem[got.eff] != pageCrossStoreVal {
					t.Fatalf("stored $%02X at $%04X, want $%02X", got.bus.mem[got.eff], got.eff, pageCrossStoreVal)
				}
			})
		}
	}
}

type pageCrossRun struct {
	clocks int
	total  uint64
	cpu    *CPU
	bus    *pageCrossBus
	eff    uint16
}

func runPageCrossOp(t *testing.T, opcode uint8, mode string, crossed bool) pageCrossRun {
	t.Helper()
	bus := &pageCrossBus{}
	c := NewCPU()
	c.ConnectBus(bus)
	c.PC = pageCrossProg
	c.A = pageCrossStoreVal
	c.X = pageCrossIndex
	c.Y = pageCrossIndex

	base := uint16(pageCrossSameBase)
	if crossed {
		base = pageCrossNextBase
	}
	eff := base + pageCrossIndex
	bus.mem[eff] = pageCrossMemValue

	switch mode {
	case "absX", "absY":
		bus.mem[pageCrossProg] = opcode
		bus.mem[pageCrossProg+1] = uint8(base)
		bus.mem[pageCrossProg+2] = uint8(base >> 8)
	case "izy":
		bus.mem[pageCrossProg] = opcode
		bus.mem[pageCrossProg+1] = pageCrossZP
		bus.mem[pageCrossZP] = uint8(base)
		bus.mem[pageCrossZP+1] = uint8(base >> 8)
	default:
		t.Fatalf("unknown mode %q", mode)
	}

	start := c.GetTotalCycles()
	clocks := 0
	for {
		c.Clock()
		clocks++
		if c.Complete() || clocks > 20 {
			break
		}
	}
	if !c.Complete() {
		t.Fatalf("opcode $%02X did not complete", opcode)
	}
	return pageCrossRun{
		clocks: clocks,
		total:  c.GetTotalCycles() - start,
		cpu:    c,
		bus:    bus,
		eff:    eff,
	}
}
