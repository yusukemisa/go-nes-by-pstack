package nes

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTimingROM(t *testing.T, prg []byte) string {
	t.Helper()
	if len(prg) != 16384 {
		t.Fatalf("prg %d", len(prg))
	}
	rom := make([]byte, 16+16384)
	copy(rom, "NES\x1a")
	rom[4] = 1
	copy(rom[16:], prg)
	path := filepath.Join(t.TempDir(), "timing.nes")
	if err := os.WriteFile(path, rom, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func openTiming(t *testing.T, prg []byte) *Console {
	t.Helper()
	c, err := Open(writeTimingROM(t, prg))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func ppuDots(c *Console) int {
	scan := int(c.PPU.GetScanline())
	if scan < 0 {
		scan = 261
	}
	return scan*341 + int(c.PPU.GetCycle())
}

// runCPUCycles clocks until n CPU cycles have been counted, then finishes the
// two PPU dots that belong to the last CPU cycle.
func runCPUCycles(c *Console, n int) {
	target := c.CPU.GetTotalCycles() + uint64(n)
	for c.CPU.GetTotalCycles() < target {
		c.Clock()
	}
	for c.dot%3 != 0 {
		c.Clock()
	}
}

func nmiProgram() []byte {
	prg := make([]byte, 16384)
	// $C000: ASL $0200 / NOP. Handler at $C100 is LDA #$55.
	prg[0] = 0x0E
	prg[1] = 0x00
	prg[2] = 0x02
	prg[3] = 0xEA
	prg[0x100] = 0xA9
	prg[0x101] = 0x55
	prg[0x3FFA] = 0x00
	prg[0x3FFB] = 0xC1
	return prg
}

func prepNMI(t *testing.T) *Console {
	t.Helper()
	c := openTiming(t, nmiProgram())
	c.CPU.PC = 0xC000
	c.CPU.Status = 0x20 // bit5
	c.CPU.SetTotalCycles(1000)
	c.CPU.Write(0x0200, 0x80)
	c.PPU.CPUWrite(0x2000, 0x80) // NMI enabled
	c.PPU.SetTiming(241, 0)      // next dots raise vblank NMI
	return c
}

func TestNMIAtBoundaryBothPaths(t *testing.T) {
	step := prepNMI(t)
	clk := prepNMI(t)

	start := step.CPU.GetTotalCycles()
	step.StepInstruction() // ASL, 6 cycles; NMI stays pending
	if got := step.CPU.GetTotalCycles() - start; got != 6 {
		t.Fatalf("ASL cycles %d, want 6", got)
	}
	if step.CPU.PC != 0xC003 {
		t.Fatalf("PC %04X, instruction aborted", step.CPU.PC)
	}
	if v := step.CPU.Read(0x0200); v != 0x00 {
		t.Fatalf("ASL result %02X", v)
	}
	if !step.CPU.NMIPending() {
		t.Fatal("NMI should be pending after the instruction")
	}
	boundary := step.CPU.GetTotalCycles()
	dotsBeforeNMI := ppuDots(step)
	step.StepInstruction() // NMI sequence
	if step.CPU.PC != 0xC100 {
		t.Fatalf("handler PC %04X", step.CPU.PC)
	}
	if step.CPU.GetTotalCycles() != boundary+7 {
		t.Fatalf("handler start CYC %d, boundary %d", step.CPU.GetTotalCycles(), boundary)
	}
	if ppuDots(step)-dotsBeforeNMI != 21 {
		t.Fatalf("PPU dots during NMI %d", ppuDots(step)-dotsBeforeNMI)
	}
	if step.CPU.A == 0x55 {
		t.Fatal("handler instruction ran inside the NMI sequence")
	}
	assertNMIStack(t, step)

	runCPUCycles(clk, 6+7)
	if clk.CPU.PC != step.CPU.PC || clk.CPU.GetTotalCycles() != step.CPU.GetTotalCycles() ||
		clk.CPU.SP != step.CPU.SP || clk.CPU.Status != step.CPU.Status ||
		clk.CPU.Read(0x0200) != step.CPU.Read(0x0200) ||
		clk.CPU.Read(0x01FB) != step.CPU.Read(0x01FB) ||
		ppuDots(clk) != ppuDots(step) {
		t.Fatalf("clock path\n got PC %04X CYC %d SP %02X P %02X PPU %d\nwant PC %04X CYC %d SP %02X P %02X PPU %d",
			clk.CPU.PC, clk.CPU.GetTotalCycles(), clk.CPU.SP, clk.CPU.Status, ppuDots(clk),
			step.CPU.PC, step.CPU.GetTotalCycles(), step.CPU.SP, step.CPU.Status, ppuDots(step))
	}
}

func assertNMIStack(t *testing.T, c *Console) {
	t.Helper()
	if c.CPU.Read(0x01FD) != 0xC0 || c.CPU.Read(0x01FC) != 0x03 {
		t.Fatalf("pushed PC %02X%02X", c.CPU.Read(0x01FD), c.CPU.Read(0x01FC))
	}
	pushed := c.CPU.Read(0x01FB)
	// ASL $80 -> $00 sets Z and C. I is not set in the pushed byte.
	if pushed&0x10 != 0 || pushed&0x20 == 0 || pushed&0x04 != 0 {
		t.Fatalf("pushed P %02X", pushed)
	}
	if c.CPU.GetFlag(0x04) != 1 { // I
		t.Fatal("I not set after NMI")
	}
}

func TestSTA2000NMIKeepsStoreCycles(t *testing.T) {
	prg := make([]byte, 16384)
	prg[0] = 0x8D // STA $2000
	prg[1] = 0x00
	prg[2] = 0x20
	prg[3] = 0xEA
	prg[0x3FFA] = 0x00
	prg[0x3FFB] = 0xC1
	prg[0x100] = 0xEA

	setup := func(t *testing.T) *Console {
		t.Helper()
		c := openTiming(t, prg)
		c.CPU.PC = 0xC000
		c.CPU.A = 0x80
		c.CPU.Status = 0x20
		c.CPU.SetTotalCycles(50)
		c.PPU.SetTiming(241, 0)
		c.PPU.Clock()
		c.PPU.Clock() // vblank flag, NMI still disabled
		return c
	}
	step := setup(t)
	clk := setup(t)
	start := step.CPU.GetTotalCycles()
	step.StepInstruction()
	if step.CPU.GetTotalCycles()-start != 4 || step.CPU.PC != 0xC003 {
		t.Fatalf("STA took %d PC %04X", step.CPU.GetTotalCycles()-start, step.CPU.PC)
	}
	boundary := step.CPU.GetTotalCycles()
	step.StepInstruction()
	if step.CPU.PC != 0xC100 || step.CPU.GetTotalCycles() != boundary+7 {
		t.Fatalf("PC %04X CYC %d boundary %d", step.CPU.PC, step.CPU.GetTotalCycles(), boundary)
	}
	runCPUCycles(clk, 4+7)
	if clk.CPU.PC != step.CPU.PC || clk.CPU.GetTotalCycles() != step.CPU.GetTotalCycles() ||
		clk.PPU.GetCtrl() != 0x80 || ppuDots(clk) != ppuDots(step) {
		t.Fatalf("clock PC %04X CYC %d ctrl %02X ppu %d; step PC %04X CYC %d ppu %d",
			clk.CPU.PC, clk.CPU.GetTotalCycles(), clk.PPU.GetCtrl(), ppuDots(clk),
			step.CPU.PC, step.CPU.GetTotalCycles(), ppuDots(step))
	}
}

func TestOAMDMAStall(t *testing.T) {
	prg := make([]byte, 16384)
	prg[0] = 0x8D // STA $4014
	prg[1] = 0x14
	prg[2] = 0x40
	prg[3] = 0xEA

	// Even start cycle: write falls on an odd (put) cycle -> 514.
	// Odd start cycle: write falls on an even (get) cycle -> 513.
	// https://www.nesdev.org/wiki/DMA
	cases := []struct {
		start uint64
		stall int
	}{
		{100, 514},
		{101, 513},
	}
	for _, tc := range cases {
		step := openTiming(t, prg)
		clk := openTiming(t, prg)
		for _, c := range []*Console{step, clk} {
			c.CPU.PC = 0xC000
			c.CPU.A = 0x02
			c.CPU.SetTotalCycles(tc.start)
			c.PPU.SetTiming(0, 0)
			for i := 0; i < 256; i++ {
				c.CPU.Write(uint16(0x0200+i), uint8(i)^0xA5)
			}
			c.CPU.Write(0x0300, 0xFF)
		}
		dots0 := ppuDots(step)
		step.StepInstruction()
		delta := int(step.CPU.GetTotalCycles() - tc.start)
		if delta != 4+tc.stall {
			t.Fatalf("start %d delta %d, want %d", tc.start, delta, 4+tc.stall)
		}
		if step.CPU.PC != 0xC003 {
			t.Fatalf("PC %04X", step.CPU.PC)
		}
		if ppuDots(step)-dots0 != delta*3 {
			t.Fatalf("PPU dots %d, want %d", ppuDots(step)-dots0, delta*3)
		}
		if step.PPU.OAM[0] != 0xA5 || step.PPU.OAM[255] != uint8(255)^0xA5 {
			t.Fatalf("OAM %02X %02X", step.PPU.OAM[0], step.PPU.OAM[255])
		}
		runCPUCycles(clk, delta)
		if clk.CPU.GetTotalCycles() != step.CPU.GetTotalCycles() || clk.CPU.PC != step.CPU.PC ||
			ppuDots(clk) != ppuDots(step) || clk.PPU.OAM[0] != step.PPU.OAM[0] {
			t.Fatalf("clock CYC %d PC %04X PPU %d OAM %02X",
				clk.CPU.GetTotalCycles(), clk.CPU.PC, ppuDots(clk), clk.PPU.OAM[0])
		}
	}
}

func TestStepFrameStopsAtFrameComplete(t *testing.T) {
	prg := make([]byte, 16384)
	phases := [][2]int{{0, 0}, {0, 21}, {80, 10}, {241, 1}, {261, 0}}
	counts := make([]int, 0, len(phases))
	for _, ph := range phases {
		ref := openTiming(t, prg)
		got := openTiming(t, prg)
		ref.PPU.SetTiming(int16(ph[0]), uint16(ph[1]))
		got.PPU.SetTiming(int16(ph[0]), uint16(ph[1]))
		n := 0
		for {
			ref.Clock()
			n++
			if ref.PPU.FrameComplete() {
				break
			}
			if n > 100000 {
				t.Fatal("frame never completed")
			}
		}
		got.StepFrame()
		if ref.PPU.GetScanline() != got.PPU.GetScanline() || ref.PPU.GetCycle() != got.PPU.GetCycle() {
			t.Fatalf("phase %v ref PPU %d,%d step %d,%d after %d clocks",
				ph, ref.PPU.GetScanline(), ref.PPU.GetCycle(), got.PPU.GetScanline(), got.PPU.GetCycle(), n)
		}
		counts = append(counts, n)
	}
	if counts[0] == counts[2] {
		t.Fatalf("frame length did not depend on phase: %v", counts)
	}
	for _, n := range counts[1:] {
		if n == 89342 {
			t.Fatalf("mid-frame StepFrame still used 89342 dots: %v", counts)
		}
	}
}
