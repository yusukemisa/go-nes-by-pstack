package cpu

import "testing"

// interruptBus is a flat memory used only by interrupt tests.
type interruptBus struct {
	mem [65536]byte
}

func (b *interruptBus) CPURead(addr uint16) uint8 { return b.mem[addr] }
func (b *interruptBus) CPUWrite(addr uint16, data uint8) {
	b.mem[addr] = data
}
func (b *interruptBus) GetPPUCycles() (int, int16, uint16) { return 0, 0, 0 }

func newInterruptCPU(pc uint16, status uint8) (*CPU, *interruptBus) {
	bus := &interruptBus{}
	bus.mem[0xFFFA] = 0x00
	bus.mem[0xFFFB] = 0xC1
	bus.mem[0xC100] = 0xEA // NOP at the handler
	cpu := NewCPU()
	cpu.ConnectBus(bus)
	cpu.PC = pc
	cpu.SP = 0xFD
	cpu.Status = status
	return cpu, bus
}

func TestNMISequenceIsSevenCycles(t *testing.T) {
	cpu, bus := newInterruptCPU(0x1234, C|Z) // I clear, bit5 clear until the push
	if cpu.GetCycles() != 0 {
		t.Fatalf("cycles %d", cpu.GetCycles())
	}
	cpu.NMI()
	if cpu.GetCycles() != 7 {
		t.Fatalf("NMI cycles = %d, want 7", cpu.GetCycles())
	}
	if cpu.PC != 0xC100 {
		t.Fatalf("PC %04X", cpu.PC)
	}
	if cpu.GetFlag(I) != 1 {
		t.Fatal("I not set")
	}
	if cpu.SP != 0xFA {
		t.Fatalf("SP %02X", cpu.SP)
	}
	if bus.mem[0x01FD] != 0x12 || bus.mem[0x01FC] != 0x34 {
		t.Fatalf("pushed PC %02X %02X", bus.mem[0x01FD], bus.mem[0x01FC])
	}
	pushed := bus.mem[0x01FB]
	if pushed&B != 0 {
		t.Fatalf("pushed B set: %02X", pushed)
	}
	if pushed&U == 0 {
		t.Fatalf("pushed bit5 clear: %02X", pushed)
	}
	if pushed&I != 0 {
		t.Fatalf("I is set on the vector fetch, after P is pushed: %02X", pushed)
	}
	if pushed&(C|Z) != (C | Z) {
		t.Fatalf("flags not preserved: %02X", pushed)
	}
}

func TestNMIIgnoresInterruptDisable(t *testing.T) {
	cpu, _ := newInterruptCPU(0x8000, I|U)
	cpu.RequestNMI()
	if !cpu.ServiceNMI() {
		t.Fatal("NMI is not masked by I")
	}
	if cpu.PC != 0xC100 || cpu.GetCycles() != 7 {
		t.Fatalf("PC %04X cycles %d", cpu.PC, cpu.GetCycles())
	}
}

func TestNMILatchesUntilInstructionBoundary(t *testing.T) {
	cpu, bus := newInterruptCPU(0x4000, U)
	bus.mem[0x4000] = 0xEA // NOP, 2 cycles
	cpu.Clock()
	left := cpu.GetCycles()
	if left == 0 {
		t.Fatal("NOP should still be in progress")
	}
	cpu.RequestNMI()
	if cpu.ServiceNMI() {
		t.Fatal("serviced mid-instruction")
	}
	if cpu.GetCycles() != left {
		t.Fatalf("remaining cycles %d, was %d", cpu.GetCycles(), left)
	}
	if cpu.PC == 0xC100 {
		t.Fatal("vector taken early")
	}
	for !cpu.Complete() {
		cpu.Clock()
	}
	if !cpu.ServiceNMI() {
		t.Fatal("expected NMI at the boundary")
	}
	if cpu.GetCycles() != 7 || cpu.PC != 0xC100 {
		t.Fatalf("PC %04X cycles %d", cpu.PC, cpu.GetCycles())
	}
}

func TestNMISequenceRunsOneHandlerInstruction(t *testing.T) {
	cpu, _ := newInterruptCPU(0x4000, U)
	cpu.RequestNMI()
	if !cpu.ServiceNMI() {
		t.Fatal("service")
	}
	cpu.RequestNMI()
	for !cpu.Complete() {
		cpu.Clock()
	}
	if cpu.ServiceNMI() {
		t.Fatal("second NMI must wait for one handler instruction")
	}
	cpu.Clock()
	for !cpu.Complete() {
		cpu.Clock()
	}
	if !cpu.ServiceNMI() || cpu.GetCycles() != 7 {
		t.Fatalf("second NMI cycles %d", cpu.GetCycles())
	}
}
