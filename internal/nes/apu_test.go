package nes

import (
	"testing"

	"github.com/misawa/go-nes-by-pstack/internal/controller"
	"github.com/misawa/go-nes-by-pstack/internal/cpu"
)

func apuStepCPU(c *Console) {
	before := c.CPU.GetTotalCycles()
	for c.CPU.GetTotalCycles() == before {
		c.Clock()
	}
}

func advanceAPUCPU(c *Console, n int, step bool) {
	target := c.CPU.GetTotalCycles() + uint64(n)
	for c.CPU.GetTotalCycles() < target {
		if step {
			c.StepInstruction()
		} else {
			apuStepCPU(c)
		}
	}
}

// apuROM spins at $C000. $C010 is ASL $0200, $C013 is STA $4014, and the IRQ
// vector points at LDA #$AB at $C100. The spin keeps long frame-counter waits
// out of PPU register space.
func apuROM() []byte {
	prg := make([]byte, 16384)
	prg[0] = 0x4C // JMP $C000
	prg[1] = 0x00
	prg[2] = 0xC0
	prg[0x10] = 0x0E // ASL $0200
	prg[0x11] = 0x00
	prg[0x12] = 0x02
	prg[0x13] = 0x8D // STA $4014
	prg[0x14] = 0x14
	prg[0x15] = 0x40
	prg[0x16] = 0xEA
	prg[0x100] = 0xA9 // LDA #$AB
	prg[0x101] = 0xAB
	prg[0x3FFA] = 0x00
	prg[0x3FFB] = 0xC1
	prg[0x3FFE] = 0x00
	prg[0x3FFF] = 0xC1
	return prg
}

func prepareAPU(t *testing.T) *Console {
	t.Helper()
	c := openTiming(t, apuROM())
	c.CPU.PC = 0xC000
	c.CPU.Status = cpu.I | cpu.U
	return c
}

func TestAPUReadsAndControllerPort(t *testing.T) {
	c := prepareAPU(t)
	for addr := uint16(0x4000); addr <= 0x4013; addr++ {
		c.CPU.Write(addr, 0x5A)
		if got := c.CPU.Read(addr); got != 0 {
			t.Fatalf("$%04X read %02X", addr, got)
		}
	}
	c.CPU.Write(0x4015, 0x0F)
	if got := c.CPU.Read(0x4015); got != 0 {
		t.Fatalf("status echoed the enable write: %02X", got)
	}

	c.Bus.GetController2().SetButton(controller.BUTTON_A, true)
	c.CPU.Write(0x4016, 1)
	c.CPU.Write(0x4017, 0x80)
	if got := c.CPU.Read(0x4017); got != 1 {
		t.Fatalf("$4017 read %02X, want controller 2", got)
	}
}

func TestStatusLengthHaltAndModes(t *testing.T) {
	load := func(t *testing.T, frame uint8) *Console {
		t.Helper()
		c := prepareAPU(t)
		c.CPU.Write(0x4015, 0x0F)
		for _, addr := range []uint16{0x4003, 0x4007, 0x400B, 0x400F} {
			c.CPU.Write(addr, 0x18)
		}
		c.CPU.Write(0x4017, frame)
		return c
	}

	c := load(t, 0x00)
	if got := c.CPU.Read(0x4015); got != 0x0F {
		t.Fatalf("lengths not visible: %02X", got)
	}
	c.CPU.Write(0x4015, 0x00)
	if got := c.CPU.Read(0x4015); got != 0 {
		t.Fatalf("disabled channels still set: %02X", got)
	}

	// $18 loads 2. 4-step: still audible after the first half frame, silent
	// after the second, which is also when the frame IRQ flag is up.
	four := load(t, 0x00)
	advanceAPUCPU(four, 16000, false)
	if got := four.CPU.Read(0x4015); got&0x0F != 0x0F || got&0x40 != 0 {
		t.Fatalf("after first half: %02X", got)
	}
	advanceAPUCPU(four, 16000, false)
	if got := four.CPU.Read(0x4015); got&0x0F != 0 || got&0x40 == 0 || got&0x90 != 0 {
		t.Fatalf("after second half: %02X", got)
	}

	halted := load(t, 0x00)
	halted.CPU.Write(0x4000, 0x20)
	halted.CPU.Write(0x4004, 0x20)
	halted.CPU.Write(0x4008, 0x80)
	halted.CPU.Write(0x400C, 0x20)
	advanceAPUCPU(halted, 32000, false)
	if got := halted.CPU.Read(0x4015); got&0x0F != 0x0F {
		t.Fatalf("halted lengths cleared: %02X", got)
	}

	// 5-step clocks one half frame within a few cycles of the write, so the
	// counter of 2 expires on the following half frame (~14913), and bit6 stays
	// clear. https://www.nesdev.org/wiki/APU_Frame_Counter
	five := load(t, 0x80)
	advanceAPUCPU(five, 100, false)
	if got := five.CPU.Read(0x4015); got&0x0F != 0x0F || got&0x40 != 0 {
		t.Fatalf("just after 5-step write: %02X", got)
	}
	advanceAPUCPU(five, 16000, false)
	if got := five.CPU.Read(0x4015); got&0x0F != 0 || got&0x40 != 0 {
		t.Fatalf("5-step after the next half: %02X", got)
	}

	inhibited := load(t, 0x40)
	advanceAPUCPU(inhibited, 36000, false)
	if got := inhibited.CPU.Read(0x4015); got&0x40 != 0 {
		t.Fatalf("inhibit raised bit6: %02X", got)
	}
}

func TestFrameIRQWindowBothPaths(t *testing.T) {
	for _, step := range []bool{false, true} {
		c := prepareAPU(t)
		c.CPU.Write(0x4017, 0x00)
		seen := -1
		for i := 1; i <= 30000; i++ {
			if step {
				c.StepInstruction()
				i = int(c.CPU.GetTotalCycles())
			} else {
				apuStepCPU(c)
			}
			if c.Bus.APU().FrameIRQ() {
				seen = int(c.CPU.GetTotalCycles())
				break
			}
			if step && c.CPU.GetTotalCycles() > 30000 {
				break
			}
		}
		if seen < 29830-4 || seen > 29830+4 {
			t.Fatalf("step %v frame IRQ at CPU cycle %d, want 29830±4", step, seen)
		}
	}
}

func consoleAtFrameIRQ(t *testing.T) *Console {
	t.Helper()
	c := prepareAPU(t)
	c.CPU.Write(0x4017, 0x00)
	for !c.Bus.APU().FrameIRQ() {
		apuStepCPU(c)
		if c.CPU.GetTotalCycles() > 40000 {
			t.Fatal("frame IRQ never asserted")
		}
	}
	for !c.CPU.Complete() {
		apuStepCPU(c)
	}
	if !c.Bus.APU().FrameIRQ() {
		t.Fatal("reached the boundary with the flag clear")
	}
	if c.CPU.PC != 0xC000 {
		t.Fatalf("spin PC %04X", c.CPU.PC)
	}
	return c
}

func TestFrameIRQTakenOnInstructionBoundary(t *testing.T) {
	step := consoleAtFrameIRQ(t)
	clk := consoleAtFrameIRQ(t)
	step.CPU.SetFlag(cpu.I, false)
	clk.CPU.SetFlag(cpu.I, false)

	boundary := step.CPU.GetTotalCycles()
	pushedPC := step.CPU.PC
	dots := ppuDots(step)
	step.StepInstruction()
	if step.CPU.PC != 0xC100 {
		t.Fatalf("PC %04X", step.CPU.PC)
	}
	if step.CPU.GetTotalCycles() != boundary+7 {
		t.Fatalf("CYC %d, boundary %d", step.CPU.GetTotalCycles(), boundary)
	}
	if ppuDots(step)-dots != 21 {
		t.Fatalf("PPU dots %d", ppuDots(step)-dots)
	}
	if step.CPU.A == 0xAB {
		t.Fatal("handler instruction ran inside the IRQ sequence")
	}
	if step.CPU.GetFlag(cpu.I) != 1 {
		t.Fatal("I not set")
	}
	if step.CPU.Read(0x01FD) != uint8(pushedPC>>8) || step.CPU.Read(0x01FC) != uint8(pushedPC) {
		t.Fatalf("pushed PC %02X%02X, want %04X", step.CPU.Read(0x01FD), step.CPU.Read(0x01FC), pushedPC)
	}
	pushed := step.CPU.Read(0x01FB)
	if pushed&cpu.B != 0 || pushed&cpu.U == 0 {
		t.Fatalf("pushed P %02X", pushed)
	}

	runCPUCycles(clk, 7)
	if clk.CPU.PC != step.CPU.PC || clk.CPU.GetTotalCycles() != step.CPU.GetTotalCycles() ||
		clk.CPU.SP != step.CPU.SP || clk.CPU.Status != step.CPU.Status {
		t.Fatalf("clock path PC %04X CYC %d SP %02X P %02X",
			clk.CPU.PC, clk.CPU.GetTotalCycles(), clk.CPU.SP, clk.CPU.Status)
	}

	masked := consoleAtFrameIRQ(t)
	masked.StepInstruction()
	if masked.CPU.PC != 0xC000 {
		t.Fatalf("I=1 PC %04X", masked.CPU.PC)
	}

	// The flag is already set. Starting an instruction with I=1 must finish
	// it; clearing I mid-instruction must not vector until the boundary.
	mid := consoleAtFrameIRQ(t)
	mid.CPU.PC = 0xC010 // ASL $0200
	mid.CPU.Write(0x0200, 0x80)
	apuStepCPU(mid)
	mid.CPU.SetFlag(cpu.I, false)
	for mid.CPU.GetCycles() != 0 {
		apuStepCPU(mid)
	}
	if mid.CPU.PC == 0xC100 {
		t.Fatal("IRQ aborted ASL")
	}
	if mid.CPU.PC != 0xC013 {
		t.Fatalf("ASL PC %04X", mid.CPU.PC)
	}
	if mid.CPU.Read(0x0200) != 0x00 {
		t.Fatalf("ASL result %02X", mid.CPU.Read(0x0200))
	}
	boundary = mid.CPU.GetTotalCycles()
	mid.StepInstruction()
	if mid.CPU.PC != 0xC100 || mid.CPU.GetTotalCycles() != boundary+7 {
		t.Fatalf("late IRQ PC %04X CYC %d boundary %d", mid.CPU.PC, mid.CPU.GetTotalCycles(), boundary)
	}

	// Reading $4015 clears the flag, so the next boundary runs the instruction.
	cleared := consoleAtFrameIRQ(t)
	if cleared.CPU.Read(0x4015)&0x40 == 0 || cleared.Bus.APU().FrameIRQ() {
		t.Fatal("status read did not observe and clear bit6")
	}
	cleared.CPU.SetFlag(cpu.I, false)
	cleared.StepInstruction()
	if cleared.CPU.PC != 0xC000 {
		t.Fatalf("IRQ after the flag was cleared, PC %04X", cleared.CPU.PC)
	}
	if cleared.CPU.Read(0x4017) == 0x40 {
		t.Fatal("$4017 read returned the frame register")
	}
}

func TestAPUClockedDuringOAMDMA(t *testing.T) {
	c := prepareAPU(t)
	c.CPU.Write(0x4017, 0x00)
	advanceAPUCPU(c, 29700, true)
	if c.Bus.APU().FrameIRQ() {
		t.Fatal("frame IRQ before the DMA window")
	}
	c.CPU.PC = 0xC013 // STA $4014
	c.CPU.A = 0x02
	for i := 0; i < 256; i++ {
		c.CPU.Write(uint16(0x0200+i), uint8(i)^0x5A)
	}
	start := c.CPU.GetTotalCycles()
	c.CPU.SetFlag(cpu.I, false)
	c.StepInstruction()
	delta := int(c.CPU.GetTotalCycles() - start)
	// STA's write is its last cycle. Even start-of-cycle count → 513.
	// https://www.nesdev.org/wiki/DMA
	writeCycle := start + 3
	want := 4 + 513
	if writeCycle&1 == 1 {
		want = 4 + 514
	}
	if delta != want {
		t.Fatalf("STA $4014 delta %d, want %d (start %d)", delta, want, start)
	}
	if c.CPU.PC != 0xC016 {
		t.Fatalf("PC %04X, DMA interrupted", c.CPU.PC)
	}
	if c.PPU.OAM[0] != 0x5A || c.PPU.OAM[255] != uint8(255)^0x5A || !c.Bus.APU().FrameIRQ() {
		t.Fatalf("OAM %02X %02X frame IRQ %v", c.PPU.OAM[0], c.PPU.OAM[255], c.Bus.APU().FrameIRQ())
	}
	boundary := c.CPU.GetTotalCycles()
	c.StepInstruction()
	if c.CPU.PC != 0xC100 || c.CPU.GetTotalCycles() != boundary+7 || c.CPU.GetFlag(cpu.I) != 1 {
		t.Fatalf("IRQ after DMA PC %04X CYC %d boundary %d P %02X",
			c.CPU.PC, c.CPU.GetTotalCycles(), boundary, c.CPU.Status)
	}
}
