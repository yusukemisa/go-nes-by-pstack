package cpu

import "fmt"

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

	// Instruction lookup table
	lookup []Instruction
}

type Bus interface {
	CPURead(addr uint16) uint8
	CPUWrite(addr uint16, data uint8)
	GetPPUCycles() (int, int16, uint16)
}

type Instruction struct {
	Name     string
	Operate  func(*CPU) uint8
	AddrMode func(*CPU) uint8
	Cycles   uint8
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
	cpu.setupInstructionTable()
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

func (cpu *CPU) NMI() {
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

	// Read NMI vector
	addrAbs := uint16(0xFFFA)
	lo := uint16(cpu.Read(addrAbs))
	hi := uint16(cpu.Read(addrAbs + 1))
	cpu.PC = (hi << 8) | lo

	cpu.cycles = 8
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

func (cpu *CPU) Disassemble(nStart, nStop uint16) map[uint16]string {
	mapLines := make(map[uint16]string)
	addr := nStart

	for addr <= nStop {
		lineAddr := addr
		sInst := fmt.Sprintf("$%04X: ", addr)
		opcode := cpu.bus.CPURead(addr)
		addr++
		sInst += cpu.lookup[opcode].Name
		mapLines[lineAddr] = sInst
		
		// Skip operand bytes (simplified)
		if opcode == 0x20 || opcode == 0x4C { // JSR, JMP absolute
			addr += 2
		} else if opcode >= 0x10 && opcode <= 0x70 && (opcode&0x0F) == 0x00 { // Branch instructions
			addr += 1
		}
	}

	return mapLines
}

func (cpu *CPU) SetTotalCycles(cycles uint64) {
	cpu.totalCycles = cycles
}