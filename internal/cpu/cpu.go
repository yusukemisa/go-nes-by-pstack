package cpu

type CPU struct {
	// Registers
	A      uint8  // Accumulator
	X      uint8  // X register
	Y      uint8  // Y register
	PC     uint16 // Program Counter
	SP     uint8  // Stack Pointer
	Status uint8  // Status flags

	// Bus connection
	bus Bus

	// Cycle tracking
	cycles uint8
	totalCycles uint64

	// nmiPending is latched until the next instruction boundary.
	// nmiDefer forces one instruction to run after an NMI sequence so the
	// sequence itself is not interrupted.
	// https://www.nesdev.org/wiki/CPU_interrupts
	nmiPending bool
	nmiDefer   bool
}

type Bus interface {
	CPURead(addr uint16) uint8
	CPUWrite(addr uint16, data uint8)
	GetPPUCycles() (int, int16, uint16)
}

// Status flags
const (
	C = 1 << 0 // Carry
	Z = 1 << 1 // Zero
	I = 1 << 2 // Interrupt Disable
	D = 1 << 3 // Decimal Mode
	B = 1 << 4 // Break
	U = 1 << 5 // Unused
	V = 1 << 6 // Overflow
	N = 1 << 7 // Negative
)

func NewCPU() *CPU {
	cpu := &CPU{
		A:      0x00,
		X:      0x00,
		Y:      0x00,
		PC:     0x0000,
		SP:     0xFD,
		Status: 0x00 | U,
	}
	return cpu
}

func (cpu *CPU) ConnectBus(bus Bus) {
	cpu.bus = bus
}

func (cpu *CPU) Read(addr uint16) uint8 {
	return cpu.bus.CPURead(addr)
}

func (cpu *CPU) Write(addr uint16, data uint8) {
	cpu.bus.CPUWrite(addr, data)
}

func (cpu *CPU) GetFlag(flag uint8) uint8 {
	if (cpu.Status & flag) > 0 {
		return 1
	}
	return 0
}

func (cpu *CPU) SetFlag(flag uint8, value bool) {
	if value {
		cpu.Status |= flag
	} else {
		cpu.Status &= ^flag
	}
}

func (cpu *CPU) Reset() {
	// Read reset vector from $FFFC-$FFFD as per NES specification
	addrAbs := uint16(0xFFFC)
	lo := uint16(cpu.Read(addrAbs))
	hi := uint16(cpu.Read(addrAbs + 1))
	cpu.PC = (hi << 8) | lo

	// Reset internal registers
	cpu.A = 0x00
	cpu.X = 0x00
	cpu.Y = 0x00
	cpu.SP = 0xFD
	cpu.Status = 0x24  // U=1, I=1

	// Reset takes time
	cpu.cycles = 8  
	cpu.totalCycles = 7  // nestest.log baseline
	cpu.nmiPending = false
	cpu.nmiDefer = false
}

func (cpu *CPU) IRQ() {
	if cpu.GetFlag(I) == 0 {
		// Push PC and status to stack
		cpu.Write(0x0100+uint16(cpu.SP), uint8((cpu.PC>>8)&0x00FF))
		cpu.SP--
		cpu.Write(0x0100+uint16(cpu.SP), uint8(cpu.PC&0x00FF))
		cpu.SP--

		cpu.SetFlag(B, false)
		cpu.SetFlag(U, true)
		cpu.SetFlag(I, true)
		cpu.Write(0x0100+uint16(cpu.SP), cpu.Status)
		cpu.SP--

		// Read IRQ vector
		addrAbs := uint16(0xFFFE)
		lo := uint16(cpu.Read(addrAbs))
		hi := uint16(cpu.Read(addrAbs + 1))
		cpu.PC = (hi << 8) | lo

		cpu.cycles = 7
	}
}

// RequestNMI latches NMI. It is serviced at an instruction boundary.
func (cpu *CPU) RequestNMI() {
	cpu.nmiPending = true
}

func (cpu *CPU) NMIPending() bool {
	return cpu.nmiPending
}

// ServiceNMI starts the 7-cycle NMI sequence when one is pending and the CPU
// is at an instruction boundary. A sequence in progress is left alone, and the
// following boundary runs one instruction before another NMI.
// https://www.nesdev.org/wiki/CPU_interrupts
func (cpu *CPU) ServiceNMI() bool {
	if cpu.cycles != 0 {
		return false
	}
	if cpu.nmiDefer {
		cpu.nmiDefer = false
		return false
	}
	if !cpu.nmiPending {
		return false
	}
	cpu.NMI()
	return true
}

func (cpu *CPU) NMI() {
	cpu.nmiPending = false
	cpu.nmiDefer = true

	// Push PC and status to stack. I is set on the vector fetch, after P is
	// pushed (B clear, unused/bit5 set). https://www.nesdev.org/wiki/CPU_interrupts
	cpu.Write(0x0100+uint16(cpu.SP), uint8((cpu.PC>>8)&0x00FF))
	cpu.SP--
	cpu.Write(0x0100+uint16(cpu.SP), uint8(cpu.PC&0x00FF))
	cpu.SP--

	cpu.SetFlag(B, false)
	cpu.SetFlag(U, true)
	cpu.Write(0x0100+uint16(cpu.SP), cpu.Status)
	cpu.SP--
	cpu.SetFlag(I, true)

	// Read NMI vector
	addrAbs := uint16(0xFFFA)
	lo := uint16(cpu.Read(addrAbs))
	hi := uint16(cpu.Read(addrAbs + 1))
	cpu.PC = (hi << 8) | lo

	cpu.cycles = 7
}

// AddCycle counts one CPU cycle without fetching. Used while the CPU is halted.
func (cpu *CPU) AddCycle() {
	cpu.totalCycles++
}

// Clock method moved to simple_cpu.go

func (cpu *CPU) Complete() bool {
	return cpu.cycles == 0
}

// Debug methods
func (cpu *CPU) GetTotalCycles() uint64 {
	return cpu.totalCycles
}

func (cpu *CPU) GetCycles() uint8 {
	return cpu.cycles
}

func (cpu *CPU) SetTotalCycles(cycles uint64) {
	cpu.totalCycles = cycles
}