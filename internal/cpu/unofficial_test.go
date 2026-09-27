package cpu_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/misawa/go-nes-by-pstack/internal/cpu"
	"github.com/misawa/go-nes-by-pstack/internal/nes"
)

// unofficialBus is a flat 64KB image for unofficial-opcode tests.
type unofficialBus struct {
	mem   [65536]byte
	reads int
}

func (b *unofficialBus) CPURead(addr uint16) uint8 {
	b.reads++
	return b.mem[addr]
}

func (b *unofficialBus) CPUWrite(addr uint16, data uint8) {
	b.mem[addr] = data
}

func (b *unofficialBus) GetPPUCycles() (int, int16, uint16) { return 0, 0, 0 }

func newUnofficialCPU() (*cpu.CPU, *unofficialBus) {
	bus := &unofficialBus{}
	c := cpu.NewCPU()
	c.ConnectBus(bus)
	c.PC = 0x8000
	c.SP = 0xFD
	c.Status = cpu.U
	return c, bus
}

func runUnofficial(t *testing.T, c *cpu.CPU) int {
	t.Helper()
	start := c.GetTotalCycles()
	clocks := 0
	for {
		c.Clock()
		clocks++
		if c.Complete() || clocks > 16 {
			break
		}
	}
	if !c.Complete() {
		t.Fatalf("instruction at $%04X did not complete", c.PC)
	}
	return int(c.GetTotalCycles() - start)
}

func putImm(bus *unofficialBus, pc uint16, opcode, imm uint8) {
	bus.mem[pc] = opcode
	bus.mem[pc+1] = imm
}

func putAbs(bus *unofficialBus, pc uint16, opcode uint8, addr uint16) {
	bus.mem[pc] = opcode
	bus.mem[pc+1] = uint8(addr)
	bus.mem[pc+2] = uint8(addr >> 8)
}

func TestAllOpcodesDoNotPanic(t *testing.T) {
	for opcode := 0; opcode < 256; opcode++ {
		c, bus := newUnofficialCPU()
		bus.mem[0x8000] = byte(opcode)
		runUnofficial(t, c)
	}
}

func TestNOPImmediate(t *testing.T) {
	for _, opcode := range []uint8{0x82, 0x89, 0xC2, 0xE2} {
		c, bus := newUnofficialCPU()
		c.A, c.X, c.Y = 0x55, 0x66, 0x77
		c.Status = cpu.U | cpu.C | cpu.V
		putImm(bus, c.PC, opcode, 0xAB)
		if got := runUnofficial(t, c); got != 2 {
			t.Fatalf("$%02X cycles %d, want 2", opcode, got)
		}
		if c.PC != 0x8002 || c.A != 0x55 || c.X != 0x66 || c.Y != 0x77 || c.Status != cpu.U|cpu.C|cpu.V {
			t.Fatalf("$%02X changed state PC=%04X A=%02X X=%02X Y=%02X P=%02X", opcode, c.PC, c.A, c.X, c.Y, c.Status)
		}
	}
}

func TestANCFlags(t *testing.T) {
	cases := []struct {
		a, imm uint8
		wantA  uint8
		c, z   bool
	}{
		{0xF0, 0x80, 0x80, true, false},
		{0x0F, 0x0F, 0x0F, false, false},
		{0xFF, 0x00, 0x00, false, true},
		{0x80, 0xFF, 0x80, true, false},
	}
	for _, opcode := range []uint8{0x0B, 0x2B} {
		for _, tc := range cases {
			c, bus := newUnofficialCPU()
			c.A = tc.a
			c.SetFlag(cpu.V, true)
			c.SetFlag(cpu.I, true)
			putImm(bus, c.PC, opcode, tc.imm)
			if got := runUnofficial(t, c); got != 2 {
				t.Fatalf("$%02X cycles %d", opcode, got)
			}
			if c.A != tc.wantA {
				t.Fatalf("$%02X A=%02X want %02X", opcode, c.A, tc.wantA)
			}
			if (c.GetFlag(cpu.C) == 1) != tc.c || (c.GetFlag(cpu.N) == 1) != tc.c {
				t.Fatalf("$%02X C=%d N=%d, want both %v (C=N)", opcode, c.GetFlag(cpu.C), c.GetFlag(cpu.N), tc.c)
			}
			if (c.GetFlag(cpu.Z) == 1) != tc.z {
				t.Fatalf("$%02X Z=%d want %v", opcode, c.GetFlag(cpu.Z), tc.z)
			}
			if c.GetFlag(cpu.V) != 1 || c.GetFlag(cpu.I) != 1 {
				t.Fatalf("$%02X dropped V or I: P=%02X", opcode, c.Status)
			}
		}
	}
}

func TestALRAndThenLSR(t *testing.T) {
	cases := []struct{ a, imm uint8 }{
		{0xFF, 0xFF},
		{0x01, 0x01},
		{0x02, 0x03},
		{0xFF, 0xFE}, // AND #$FE then LSR leaves C clear
		{0x00, 0xFF},
	}
	for _, tc := range cases {
		c, bus := newUnofficialCPU()
		c.A = tc.a
		c.SetFlag(cpu.C, true)
		c.SetFlag(cpu.V, true)
		putImm(bus, c.PC, 0x4B, tc.imm)
		if got := runUnofficial(t, c); got != 2 {
			t.Fatalf("cycles %d", got)
		}
		and := tc.a & tc.imm
		want := and >> 1
		if c.A != want {
			t.Fatalf("A=%02X imm=%02X result %02X want %02X", tc.a, tc.imm, c.A, want)
		}
		if (c.GetFlag(cpu.C) == 1) != (and&0x01 != 0) {
			t.Fatalf("A=%02X imm=%02X C=%d", tc.a, tc.imm, c.GetFlag(cpu.C))
		}
		if c.GetFlag(cpu.N) != 0 {
			t.Fatal("LSR left N set")
		}
		if (c.GetFlag(cpu.Z) == 1) != (want == 0) {
			t.Fatalf("Z=%d", c.GetFlag(cpu.Z))
		}
		if c.GetFlag(cpu.V) != 1 {
			t.Fatal("ALR changed V")
		}
	}
}

func TestARRFlags(t *testing.T) {
	// Result is ROR(A&imm). C is bit 6, V is bit 6 xor bit 5.
	cases := []struct {
		cIn        bool
		a, imm     uint8
		wantA      uint8
		c, v, n, z bool
	}{
		{false, 0xFF, 0xFF, 0x7F, true, false, false, false},
		{true, 0x00, 0xFF, 0x80, false, false, true, false},
		{false, 0x80, 0xFF, 0x40, true, true, false, false},
		{false, 0x40, 0xFF, 0x20, false, true, false, false},
		{false, 0xFF, 0x00, 0x00, false, false, false, true},
		{true, 0x01, 0xFF, 0x80, false, false, true, false},
	}
	for _, tc := range cases {
		c, bus := newUnofficialCPU()
		c.A = tc.a
		c.SetFlag(cpu.C, tc.cIn)
		c.SetFlag(cpu.I, true)
		putImm(bus, c.PC, 0x6B, tc.imm)
		if got := runUnofficial(t, c); got != 2 {
			t.Fatalf("cycles %d", got)
		}
		if c.A != tc.wantA {
			t.Fatalf("A in %02X C in %v -> %02X want %02X", tc.a, tc.cIn, c.A, tc.wantA)
		}
		if (c.GetFlag(cpu.C) == 1) != tc.c || (c.GetFlag(cpu.V) == 1) != tc.v {
			t.Fatalf("A=%02X C=%d V=%d, want C=%v V=%v", c.A, c.GetFlag(cpu.C), c.GetFlag(cpu.V), tc.c, tc.v)
		}
		if (c.GetFlag(cpu.N) == 1) != tc.n || (c.GetFlag(cpu.Z) == 1) != tc.z {
			t.Fatalf("N=%d Z=%d", c.GetFlag(cpu.N), c.GetFlag(cpu.Z))
		}
		if c.GetFlag(cpu.I) != 1 {
			t.Fatal("ARR cleared I")
		}
	}
}

func TestAXSCompareWithoutBorrow(t *testing.T) {
	cases := []struct {
		a, x, imm uint8
		cIn       bool
	}{
		{0x0F, 0xFF, 0x01, false},
		{0x10, 0x10, 0x10, true},
		{0x01, 0xFF, 0x02, false},
		{0x80, 0xF0, 0x01, true},
		{0x05, 0xFF, 0x01, false},
		{0x05, 0xFF, 0x01, true},
	}
	for _, tc := range cases {
		c, bus := newUnofficialCPU()
		c.A = tc.a
		c.X = tc.x
		c.SetFlag(cpu.C, tc.cIn)
		c.SetFlag(cpu.V, true)
		putImm(bus, c.PC, 0xCB, tc.imm)
		if got := runUnofficial(t, c); got != 2 {
			t.Fatalf("cycles %d", got)
		}
		left := tc.a & tc.x
		wantX := left - tc.imm
		if c.X != wantX || c.A != tc.a {
			t.Fatalf("A=%02X X=%02X imm=%02X -> X=%02X A=%02X, want X=%02X A unchanged", tc.a, tc.x, tc.imm, c.X, c.A, wantX)
		}
		if (c.GetFlag(cpu.C) == 1) != (left >= tc.imm) {
			t.Fatalf("C=%d, CMP-style want %v (left %02X imm %02X, Cin %v)", c.GetFlag(cpu.C), left >= tc.imm, left, tc.imm, tc.cIn)
		}
		if (c.GetFlag(cpu.Z) == 1) != (wantX == 0) || (c.GetFlag(cpu.N) == 1) != (wantX&0x80 != 0) {
			t.Fatalf("Z=%d N=%d", c.GetFlag(cpu.Z), c.GetFlag(cpu.N))
		}
		if c.GetFlag(cpu.V) != 1 {
			t.Fatal("AXS changed V")
		}
	}
}

func TestLXAAndXAAMagic(t *testing.T) {
	// magic $EE: (A | $EE) keeps A's bits 0 and 4, forces the rest on.
	const magic uint8 = 0xEE
	lxaCases := []struct{ a, imm, want uint8 }{
		{0x00, 0xFF, 0xEE},
		{0x11, 0xFF, 0xFF},
		{0x01, 0x0F, 0x0F},
		{0x00, 0x00, 0x00},
	}
	for _, tc := range lxaCases {
		c, bus := newUnofficialCPU()
		c.A = tc.a
		c.X = 0x5A
		c.SetFlag(cpu.C, true)
		putImm(bus, c.PC, 0xAB, tc.imm)
		runUnofficial(t, c)
		want := (tc.a | magic) & tc.imm
		if want != tc.want || c.A != want || c.X != want {
			t.Fatalf("LXA A=%02X imm=%02X -> A=%02X X=%02X want %02X", tc.a, tc.imm, c.A, c.X, want)
		}
		if c.GetFlag(cpu.C) != 1 {
			t.Fatal("LXA changed C")
		}
		if (c.GetFlag(cpu.Z) == 1) != (want == 0) || (c.GetFlag(cpu.N) == 1) != (want&0x80 != 0) {
			t.Fatalf("LXA flags Z=%d N=%d", c.GetFlag(cpu.Z), c.GetFlag(cpu.N))
		}
	}

	xaaCases := []struct{ a, x, imm, want uint8 }{
		{0x00, 0xFF, 0xFF, 0xEE},
		{0x11, 0xFF, 0xFF, 0xFF},
		{0xFF, 0x0F, 0xFF, 0x0F},
		{0xFF, 0xFF, 0x00, 0x00},
	}
	for _, tc := range xaaCases {
		c, bus := newUnofficialCPU()
		c.A = tc.a
		c.X = tc.x
		putImm(bus, c.PC, 0x8B, tc.imm)
		runUnofficial(t, c)
		want := (tc.a | magic) & tc.x & tc.imm
		if want != tc.want || c.A != want || c.X != tc.x {
			t.Fatalf("XAA A=%02X X=%02X imm=%02X -> A=%02X X=%02X want A=%02X", tc.a, tc.x, tc.imm, c.A, c.X, want)
		}
	}
}

func TestLAS(t *testing.T) {
	for _, crossed := range []bool{false, true} {
		c, bus := newUnofficialCPU()
		c.Y = 0x20
		c.SP = 0xF0
		c.SetFlag(cpu.C, true)
		c.SetFlag(cpu.V, true)
		base := uint16(0x1000)
		if crossed {
			base = 0x10F0
		}
		eff := base + uint16(c.Y)
		bus.mem[eff] = 0x3C
		putAbs(bus, c.PC, 0xBB, base)
		got := runUnofficial(t, c)
		wantCycles := 4
		if crossed {
			wantCycles = 5
		}
		if got != wantCycles {
			t.Fatalf("crossed=%v cycles %d want %d", crossed, got, wantCycles)
		}
		const want uint8 = 0x30 // 0x3C & 0xF0
		if c.A != want || c.X != want || c.SP != want {
			t.Fatalf("A=%02X X=%02X SP=%02X want %02X", c.A, c.X, c.SP, want)
		}
		if c.GetFlag(cpu.C) != 1 || c.GetFlag(cpu.V) != 1 || c.GetFlag(cpu.N) != 0 || c.GetFlag(cpu.Z) != 0 {
			t.Fatalf("flags P=%02X", c.Status)
		}
	}

	c, bus := newUnofficialCPU()
	c.Y = 0x00
	c.SP = 0x0F
	putAbs(bus, c.PC, 0xBB, 0x0400)
	bus.mem[0x0400] = 0xF0
	runUnofficial(t, c)
	if c.A != 0 || c.X != 0 || c.SP != 0 || c.GetFlag(cpu.Z) != 1 || c.GetFlag(cpu.N) != 0 {
		t.Fatalf("zero result A=%02X X=%02X SP=%02X P=%02X", c.A, c.X, c.SP, c.Status)
	}
}

func TestUnstableStores(t *testing.T) {
	// No page cross: data is reg & (H+1), address is base+index, no extra cycle.
	t.Run("shx-same", func(t *testing.T) {
		c, bus := newUnofficialCPU()
		c.X = 0xFF
		c.Y = 0x10
		c.Status = cpu.U | cpu.C | cpu.V
		putAbs(bus, c.PC, 0x9E, 0x1200)
		if got := runUnofficial(t, c); got != 5 {
			t.Fatalf("cycles %d", got)
		}
		if bus.mem[0x1210] != 0x13 { // X & (0x12+1)
			t.Fatalf("stored %02X", bus.mem[0x1210])
		}
		if c.Status != cpu.U|cpu.C|cpu.V {
			t.Fatalf("flags changed %02X", c.Status)
		}
	})

	// Page cross: high byte of the computed address is ANDed with X.
	// X=$0F, H+1=$13 → value $03, address high $13&$0F=$03.
	t.Run("shx-cross", func(t *testing.T) {
		c, bus := newUnofficialCPU()
		c.X = 0x0F
		c.Y = 0x20
		putAbs(bus, c.PC, 0x9E, 0x12F0)
		if got := runUnofficial(t, c); got != 5 {
			t.Fatalf("cycles %d, page cross must not add one", got)
		}
		if bus.mem[0x0310] != 0x03 || bus.mem[0x1310] != 0 {
			t.Fatalf("at $0310=%02X $1310=%02X", bus.mem[0x0310], bus.mem[0x1310])
		}
	})

	t.Run("shy-cross", func(t *testing.T) {
		c, bus := newUnofficialCPU()
		c.Y = 0x0F
		c.X = 0x20
		putAbs(bus, c.PC, 0x9C, 0x12F0)
		if got := runUnofficial(t, c); got != 5 {
			t.Fatalf("cycles %d", got)
		}
		if bus.mem[0x0310] != 0x03 || bus.mem[0x1310] != 0 {
			t.Fatalf("at $0310=%02X $1310=%02X", bus.mem[0x0310], bus.mem[0x1310])
		}
	})

	// SHA page cross ANDs the address high byte with X, not with the stored
	// value. A=$0F X=$FF → value $03, but address high stays $13 & $FF.
	t.Run("sha-abs-cross", func(t *testing.T) {
		c, bus := newUnofficialCPU()
		c.A = 0x0F
		c.X = 0xFF
		c.Y = 0x20
		putAbs(bus, c.PC, 0x9F, 0x12F0)
		if got := runUnofficial(t, c); got != 5 {
			t.Fatalf("cycles %d", got)
		}
		if bus.mem[0x1310] != 0x03 || bus.mem[0x0310] != 0 {
			t.Fatalf("$1310=%02X $0310=%02X", bus.mem[0x1310], bus.mem[0x0310])
		}
	})

	t.Run("sha-izy", func(t *testing.T) {
		c, bus := newUnofficialCPU()
		c.A = 0xFF
		c.X = 0x0F
		c.Y = 0x20
		bus.mem[0x8000] = 0x93
		bus.mem[0x8001] = 0x10
		bus.mem[0x10] = 0xF0
		bus.mem[0x11] = 0x12
		if got := runUnofficial(t, c); got != 6 {
			t.Fatalf("cycles %d", got)
		}
		if bus.mem[0x0310] != 0x03 || bus.mem[0x1310] != 0 {
			t.Fatalf("$0310=%02X $1310=%02X", bus.mem[0x0310], bus.mem[0x1310])
		}
	})

	t.Run("sha-izy-same", func(t *testing.T) {
		c, bus := newUnofficialCPU()
		c.A = 0xFF
		c.X = 0xFF
		c.Y = 0x01
		bus.mem[0x8000] = 0x93
		bus.mem[0x8001] = 0x20
		bus.mem[0x20] = 0x00
		bus.mem[0x21] = 0x40
		if got := runUnofficial(t, c); got != 6 {
			t.Fatalf("cycles %d", got)
		}
		if bus.mem[0x4001] != 0x41 { // A & X & (0x40+1)
			t.Fatalf("stored %02X", bus.mem[0x4001])
		}
	})

	t.Run("tas", func(t *testing.T) {
		c, bus := newUnofficialCPU()
		c.A = 0xFF
		c.X = 0xFF
		c.Y = 0x01
		c.SP = 0xFD
		putAbs(bus, c.PC, 0x9B, 0x2000)
		if got := runUnofficial(t, c); got != 5 {
			t.Fatalf("cycles %d", got)
		}
		if c.SP != 0xFF {
			t.Fatalf("SP %02X", c.SP)
		}
		if bus.mem[0x2001] != 0x21 {
			t.Fatalf("stored %02X", bus.mem[0x2001])
		}
	})

	t.Run("tas-cross", func(t *testing.T) {
		c, bus := newUnofficialCPU()
		c.A = 0xF0
		c.X = 0x0F
		c.Y = 0x20
		bus.mem[0x0310] = 0xFF
		putAbs(bus, c.PC, 0x9B, 0x12F0)
		if got := runUnofficial(t, c); got != 5 {
			t.Fatalf("cycles %d", got)
		}
		if c.SP != 0x00 {
			t.Fatalf("SP %02X", c.SP)
		}
		// value = (A&X) & (H+1) = 0. Address high = $13 & X = $03.
		if bus.mem[0x0310] != 0x00 || bus.mem[0x1310] != 0 {
			t.Fatalf("$0310=%02X $1310=%02X", bus.mem[0x0310], bus.mem[0x1310])
		}
	})
}

func TestSTPIgnoresFetchAndInterrupts(t *testing.T) {
	jams := []uint8{0x02, 0x12, 0x22, 0x32, 0x42, 0x52, 0x62, 0x72, 0x92, 0xB2, 0xD2, 0xF2}
	for _, opcode := range jams {
		t.Run(fmt.Sprintf("%02X", opcode), func(t *testing.T) {
			c, bus := newUnofficialCPU()
			bus.mem[0x8000] = opcode
			bus.mem[0x8001] = 0xA9
			bus.mem[0x8002] = 0x42
			bus.mem[0xFFFE] = 0x00
			bus.mem[0xFFFF] = 0xC0
			bus.mem[0xFFFA] = 0x00
			bus.mem[0xFFFB] = 0xC0
			bus.mem[0xC000] = 0xA9
			bus.mem[0xC001] = 0x99
			if got := runUnofficial(t, c); got != 2 {
				t.Fatalf("cycles %d", got)
			}
			if c.PC != 0x8000 || c.A != 0 {
				t.Fatalf("PC=%04X A=%02X", c.PC, c.A)
			}
			reads := bus.reads
			if !c.Complete() {
				t.Fatal("halted CPU is not Complete")
			}
			c.Clock()
			if !c.Complete() || bus.reads != reads || c.PC != 0x8000 {
				t.Fatalf("halted Clock fetched or left cycles=%d reads %d→%d PC=%04X", c.GetCycles(), reads, bus.reads, c.PC)
			}
			c.RequestNMI()
			if c.ServiceNMI() || c.NMIPending() == false {
				t.Fatal("NMI was taken or dropped")
			}
			c.NMI()
			if c.PC != 0x8000 || c.GetCycles() != 0 {
				t.Fatalf("direct NMI escaped the halt PC=%04X cycles=%d", c.PC, c.GetCycles())
			}
			c.Status = cpu.U // I clear
			sp := c.SP
			c.IRQ()
			if c.PC != 0x8000 || c.SP != sp || c.GetCycles() != 0 {
				t.Fatalf("IRQ escaped the halt PC=%04X SP=%02X cycles=%d", c.PC, c.SP, c.GetCycles())
			}

			bus.mem[0xFFFC] = 0x00
			bus.mem[0xFFFD] = 0x90
			bus.mem[0x9000] = 0xEA
			c.Reset()
			if c.PC != 0x9000 {
				t.Fatalf("reset PC %04X", c.PC)
			}
			for !c.Complete() {
				c.Clock()
			}
			runUnofficial(t, c)
			if c.PC != 0x9001 {
				t.Fatalf("NOP after reset PC=%04X", c.PC)
			}
		})
	}
}

func writeUnofficialROM(t *testing.T, prg []byte) string {
	t.Helper()
	if len(prg) != 16384 {
		t.Fatalf("prg %d", len(prg))
	}
	rom := make([]byte, 16+len(prg))
	copy(rom[:4], "NES\x1a")
	rom[4] = 1
	copy(rom[16:], prg)
	path := filepath.Join(t.TempDir(), "unofficial.nes")
	if err := os.WriteFile(path, rom, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestJamDoesNotHangStepInstruction(t *testing.T) {
	jams := []uint8{0x02, 0x12, 0x22, 0x32, 0x42, 0x52, 0x62, 0x72, 0x92, 0xB2, 0xD2, 0xF2}
	for _, opcode := range jams {
		t.Run(fmt.Sprintf("%02X", opcode), func(t *testing.T) {
			prg := make([]byte, 16384)
			prg[0] = opcode
			prg[1] = 0xA9
			prg[2] = 0x42
			// Reset vector $8100: LDA #$99
			prg[0x100] = 0xA9
			prg[0x101] = 0x99
			// 16KB PRG is mirrored, so $FFFC lives at offset $3FFC.
			prg[0x3FFC] = 0x00
			prg[0x3FFD] = 0x81
			c, err := nes.Open(writeUnofficialROM(t, prg))
			if err != nil {
				t.Fatal(err)
			}
			c.CPU.PC = 0x8000
			c.StepInstruction()
			c.StepInstruction()
			c.CPU.RequestNMI()
			c.StepInstruction()
			if c.CPU.PC != 0x8000 || c.CPU.A != 0 {
				t.Fatalf("PC=%04X A=%02X", c.CPU.PC, c.CPU.A)
			}
			c.Reset()
			c.StepInstruction() // reset's 8-cycle stall
			c.StepInstruction() // LDA #$99
			if c.CPU.A != 0x99 || c.CPU.PC != 0x8102 {
				t.Fatalf("after reset A=%02X PC=%04X", c.CPU.A, c.CPU.PC)
			}
		})
	}
}
