package cpu

import (
	"fmt"
	"os"
)

// Simplified CPU implementation focusing on basic functionality

var logFile *os.File

// EnableLogging enables instruction logging to a file
func EnableLogging(filename string) error {
	var err error
	logFile, err = os.Create(filename)
	if err != nil {
		return err
	}
	return nil
}

// DisableLogging closes the log file
func DisableLogging() {
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}

// logInstruction logs the current instruction in nestest.log format
func (cpu *CPU) logInstruction(opcode uint8, ppuCycle, cpuCycle int) {
	if logFile == nil {
		return
	}
	
	// Read instruction bytes
	byte1 := opcode
	byte2 := uint8(0)
	byte3 := uint8(0)
	instrLen := 1
	
	// Determine instruction length based on opcode
	if cpu.isMultiByteInstruction(opcode) {
		byte2 = cpu.Read(cpu.PC + 1)
		instrLen = 2
		if cpu.isThreeByteInstruction(opcode) {
			byte3 = cpu.Read(cpu.PC + 2)
			instrLen = 3
		}
	}
	
	// Disassemble instruction
	disasm := cpu.disassembleInstruction(opcode, byte2, byte3)
	
	
	// Format the log line (matching nestest.log format exactly)
	var logLine string
	if instrLen == 1 {
		logLine = fmt.Sprintf("%04X  %02X        %s A:%02X X:%02X Y:%02X P:%02X SP:%02X PPU:%3d,%3d CYC:%d\n",
			cpu.PC, byte1, fmt.Sprintf("%-32s", disasm),
			cpu.A, cpu.X, cpu.Y, cpu.Status, cpu.SP, ppuCycle/341, ppuCycle%341, cpuCycle)
	} else if instrLen == 2 {
		logLine = fmt.Sprintf("%04X  %02X %02X    %s A:%02X X:%02X Y:%02X P:%02X SP:%02X PPU:%3d,%3d CYC:%d\n",
			cpu.PC, byte1, byte2, fmt.Sprintf("%-32s", disasm),
			cpu.A, cpu.X, cpu.Y, cpu.Status, cpu.SP, ppuCycle/341, ppuCycle%341, cpuCycle)
	} else {
		logLine = fmt.Sprintf("%04X  %02X %02X %02X  %s A:%02X X:%02X Y:%02X P:%02X SP:%02X PPU:%3d,%3d CYC:%d\n",
			cpu.PC, byte1, byte2, byte3, fmt.Sprintf("%-31s", disasm),
			cpu.A, cpu.X, cpu.Y, cpu.Status, cpu.SP, ppuCycle/341, ppuCycle%341, cpuCycle)
	}
	
	logFile.WriteString(logLine)
}

// isMultiByteInstruction checks if instruction has operands
func (cpu *CPU) isMultiByteInstruction(opcode uint8) bool {
	singleByteInstructions := []uint8{
		0x00, 0x08, 0x18, 0x28, 0x38, 0x40, 0x48, 0x58, 0x60, 0x68, 0x78, // Single byte control
		0x88, 0x8A, 0x98, 0x9A, 0xA8, 0xAA, 0xB8, 0xBA, 0xCA, 0xC8, 0xD8, // Single byte register ops
		0xE8, 0xEA, 0xF8, // Single byte misc
	}
	
	for _, single := range singleByteInstructions {
		if opcode == single {
			return false
		}
	}
	return true
}

// isThreeByteInstruction checks if instruction is 3 bytes
func (cpu *CPU) isThreeByteInstruction(opcode uint8) bool {
	threeByteInstructions := []uint8{
		0x0C, // *NOP abs - Unofficial instruction (3 bytes)
		0x20, 0x4C, 0x6C, // JSR, JMP abs, JMP ind
		0xAD, 0xBD, 0xB9, // LDA abs variants
		0xAE, 0xBE, // LDX abs variants  
		0xAC, 0xBC, // LDY abs variants
		0x8D, 0x9D, 0x99, // STA abs variants
		0x8E, // STX abs
		0x8C, // STY abs
		0x0D, 0x1D, 0x19, // ORA abs variants
		0x2D, 0x3D, 0x39, // AND abs variants
		0x4D, 0x5D, 0x59, // EOR abs variants
		0x6D, 0x7D, 0x79, // ADC abs variants
		0xED, 0xFD, 0xF9, // SBC abs variants
		0xCD, 0xDD, 0xD9, // CMP abs variants
		0xEC, // CPX abs
		0xCC, // CPY abs
		// Unofficial 3-byte instructions
		0x1C, 0x3C, 0x5C, 0x7C, 0xDC, 0xFC, // 3-byte unofficial NOP abs,X instructions
		0xAF, 0xBF, // LAX $abs, LAX $abs,Y
		0x8F, // SAX $abs
		0xCF, 0xDB, 0xDF, // DCP $abs variants
		0xEF, 0xFB, 0xFF, // ISB $abs variants
		0x0F, 0x1B, 0x1F, // SLO $abs variants
		0x2F, 0x3B, 0x3F, // RLA $abs variants
		0x4F, 0x5B, 0x5F, // SRE $abs variants
		0x6F, // RRA $abs variants
	}
	
	for _, three := range threeByteInstructions {
		if opcode == three {
			return true
		}
	}
	return false
}

// disassembleInstruction returns the disassembled instruction string
func (cpu *CPU) disassembleInstruction(opcode, byte2, byte3 uint8) string {
	switch opcode {
	case 0x00: return "BRK"
	case 0x04: 
		memVal := cpu.Read(uint16(byte2))
		return fmt.Sprintf("*NOP $%02X = %02X", byte2, memVal)
	case 0x44: 
		memVal := cpu.Read(uint16(byte2))
		return fmt.Sprintf("*NOP $%02X = %02X", byte2, memVal)
	case 0x64: 
		memVal := cpu.Read(uint16(byte2))
		return fmt.Sprintf("*NOP $%02X = %02X", byte2, memVal)
	case 0x0C:
		addr := uint16(byte3)<<8 | uint16(byte2)
		memVal := cpu.Read(addr)
		return fmt.Sprintf("*NOP $%04X = %02X", addr, memVal)
	// 1-byte unofficial NOP instructions
	case 0x1A: return "*NOP"
	case 0x3A: return "*NOP"
	case 0x5A: return "*NOP"
	case 0x7A: return "*NOP"
	case 0xDA: return "*NOP"
	case 0xFA: return "*NOP"
	// 2-byte unofficial NOP instruction
	case 0x80: return fmt.Sprintf("*NOP #$%02X", byte2)
	// 3-byte unofficial NOP abs,X instructions
	case 0x1C: 
		addr := uint16(byte3)<<8 | uint16(byte2)
		effectiveAddr := addr + uint16(cpu.X)
		memVal := cpu.Read(effectiveAddr)
		return fmt.Sprintf("*NOP $%04X,X @ %04X = %02X", addr, effectiveAddr, memVal)
	case 0x3C: 
		addr := uint16(byte3)<<8 | uint16(byte2)
		effectiveAddr := addr + uint16(cpu.X)
		memVal := cpu.Read(effectiveAddr)
		return fmt.Sprintf("*NOP $%04X,X @ %04X = %02X", addr, effectiveAddr, memVal)
	case 0x5C: 
		addr := uint16(byte3)<<8 | uint16(byte2)
		effectiveAddr := addr + uint16(cpu.X)
		memVal := cpu.Read(effectiveAddr)
		return fmt.Sprintf("*NOP $%04X,X @ %04X = %02X", addr, effectiveAddr, memVal)
	case 0x7C: 
		addr := uint16(byte3)<<8 | uint16(byte2)
		effectiveAddr := addr + uint16(cpu.X)
		memVal := cpu.Read(effectiveAddr)
		return fmt.Sprintf("*NOP $%04X,X @ %04X = %02X", addr, effectiveAddr, memVal)
	case 0xDC: 
		addr := uint16(byte3)<<8 | uint16(byte2)
		effectiveAddr := addr + uint16(cpu.X)
		memVal := cpu.Read(effectiveAddr)
		return fmt.Sprintf("*NOP $%04X,X @ %04X = %02X", addr, effectiveAddr, memVal)
	case 0xFC: 
		addr := uint16(byte3)<<8 | uint16(byte2)
		effectiveAddr := addr + uint16(cpu.X)
		memVal := cpu.Read(effectiveAddr)
		return fmt.Sprintf("*NOP $%04X,X @ %04X = %02X", addr, effectiveAddr, memVal)
	case 0x08: return "PHP"
	case 0x10: return fmt.Sprintf("BPL $%04X", cpu.PC+2+uint16(int8(byte2)))
	case 0x18: return "CLC"
	case 0x20: return fmt.Sprintf("JSR $%04X", uint16(byte3)<<8|uint16(byte2))
	case 0x24: 
		memVal := cpu.Read(uint16(byte2))
		return fmt.Sprintf("BIT $%02X = %02X", byte2, memVal)
	case 0x2C:
		addr := uint16(byte3)<<8 | uint16(byte2)
		memVal := cpu.Read(addr)
		return fmt.Sprintf("BIT $%04X = %02X", addr, memVal)
	case 0x28: return "PLP"
	case 0x25: 
		memVal := cpu.Read(uint16(byte2))
		return fmt.Sprintf("AND $%02X = %02X", byte2, memVal)
	case 0x29: return fmt.Sprintf("AND #$%02X", byte2)
	case 0x05: 
		memVal := cpu.Read(uint16(byte2))
		return fmt.Sprintf("ORA $%02X = %02X", byte2, memVal)
	case 0x09: return fmt.Sprintf("ORA #$%02X", byte2)
	case 0x0D:
		addr := uint16(byte3)<<8 | uint16(byte2)
		memVal := cpu.Read(addr)
		return fmt.Sprintf("ORA $%04X = %02X", addr, memVal)
	case 0x19:
		addr := uint16(byte3)<<8 | uint16(byte2)
		effectiveAddr := addr + uint16(cpu.Y)
		memVal := cpu.Read(effectiveAddr)
		return fmt.Sprintf("ORA $%04X,Y @ %04X = %02X", addr, effectiveAddr, memVal)
	case 0x2D:
		addr := uint16(byte3)<<8 | uint16(byte2)
		memVal := cpu.Read(addr)
		return fmt.Sprintf("AND $%04X = %02X", addr, memVal)
	case 0x39:
		addr := uint16(byte3)<<8 | uint16(byte2)
		effectiveAddr := addr + uint16(cpu.Y)
		memVal := cpu.Read(effectiveAddr)
		return fmt.Sprintf("AND $%04X,Y @ %04X = %02X", addr, effectiveAddr, memVal)
	case 0x3D:
		addr := uint16(byte3)<<8 | uint16(byte2)
		effectiveAddr := addr + uint16(cpu.X)
		memVal := cpu.Read(effectiveAddr)
		return fmt.Sprintf("AND $%04X,X @ %04X = %02X", addr, effectiveAddr, memVal)
	case 0x59:
		addr := uint16(byte3)<<8 | uint16(byte2)
		effectiveAddr := addr + uint16(cpu.Y)
		memVal := cpu.Read(effectiveAddr)
		return fmt.Sprintf("EOR $%04X,Y @ %04X = %02X", addr, effectiveAddr, memVal)
	case 0x5D:
		addr := uint16(byte3)<<8 | uint16(byte2)
		effectiveAddr := addr + uint16(cpu.X)
		memVal := cpu.Read(effectiveAddr)
		return fmt.Sprintf("EOR $%04X,X @ %04X = %02X", addr, effectiveAddr, memVal)
	case 0x79:
		addr := uint16(byte3)<<8 | uint16(byte2)
		effectiveAddr := addr + uint16(cpu.Y)
		memVal := cpu.Read(effectiveAddr)
		return fmt.Sprintf("ADC $%04X,Y @ %04X = %02X", addr, effectiveAddr, memVal)
	case 0x7D:
		addr := uint16(byte3)<<8 | uint16(byte2)
		effectiveAddr := addr + uint16(cpu.X)
		memVal := cpu.Read(effectiveAddr)
		return fmt.Sprintf("ADC $%04X,X @ %04X = %02X", addr, effectiveAddr, memVal)
	case 0x30: return fmt.Sprintf("BMI $%04X", cpu.PC+2+uint16(int8(byte2)))
	case 0x33:
		// RLA ($zp),Y - unofficial instruction
		zp := uint16(byte2)
		lo := cpu.Read(zp)
		hi := cpu.Read((zp + 1) & 0xFF)
		baseAddr := uint16(hi)<<8 | uint16(lo)
		effectiveAddr := baseAddr + uint16(cpu.Y)
		memVal := cpu.Read(effectiveAddr)
		return fmt.Sprintf("*RLA ($%02X),Y = %04X @ %04X = %02X", byte2, baseAddr, effectiveAddr, memVal)
	case 0x38: return "SEC"
	case 0x40: return "RTI"
	case 0x45:
		memVal := cpu.Read(uint16(byte2))
		return fmt.Sprintf("EOR $%02X = %02X", byte2, memVal)
	case 0x48: return "PHA"
	case 0x49: return fmt.Sprintf("EOR #$%02X", byte2)
	case 0x4C: return fmt.Sprintf("JMP $%04X", uint16(byte3)<<8|uint16(byte2))
	case 0x50: return fmt.Sprintf("BVC $%04X", cpu.PC+2+uint16(int8(byte2)))
	case 0x58: return "CLI"
	case 0x60: return "RTS"
	case 0x65:
		memVal := cpu.Read(uint16(byte2))
		return fmt.Sprintf("ADC $%02X = %02X", byte2, memVal)
	case 0x68: return "PLA"
	case 0x69: return fmt.Sprintf("ADC #$%02X", byte2)
	case 0x6C:
		// JMP ($addr) - indirect addressing with page boundary bug
		indirectAddr := uint16(byte3)<<8 | uint16(byte2)
		lo := cpu.Read(indirectAddr)
		var hi uint8
		// Simulate same page boundary bug as in execution
		if (indirectAddr & 0xFF) == 0xFF {
			hi = cpu.Read(indirectAddr & 0xFF00)
		} else {
			hi = cpu.Read(indirectAddr + 1)
		}
		jumpAddr := uint16(hi)<<8 | uint16(lo)
		return fmt.Sprintf("JMP ($%04X) = %04X", indirectAddr, jumpAddr)
	case 0x70: return fmt.Sprintf("BVS $%04X", cpu.PC+2+uint16(int8(byte2)))
	case 0x78: return "SEI"
	case 0x84: return fmt.Sprintf("STY $%02X", byte2)
	case 0x81: return fmt.Sprintf("STA ($%02X,X)", byte2)
	case 0x85: 
		memVal := cpu.Read(uint16(byte2))
		return fmt.Sprintf("STA $%02X = %02X", byte2, memVal)
	case 0x91: return fmt.Sprintf("STA ($%02X),Y", byte2)
	case 0x86: 
		memVal := cpu.Read(uint16(byte2))
		return fmt.Sprintf("STX $%02X = %02X", byte2, memVal)
	case 0x88: return "DEY"
	case 0x8A: return "TXA"
	case 0x8C: return fmt.Sprintf("STY $%04X", uint16(byte3)<<8|uint16(byte2))
	case 0x8D: 
		addr := uint16(byte3)<<8 | uint16(byte2)
		memVal := cpu.Read(addr)
		return fmt.Sprintf("STA $%04X = %02X", addr, memVal)
	case 0x8E: return fmt.Sprintf("STX $%04X", uint16(byte3)<<8|uint16(byte2))
	case 0x90: return fmt.Sprintf("BCC $%04X", cpu.PC+2+uint16(int8(byte2)))
	case 0x94: return fmt.Sprintf("STY $%02X,X", byte2)
	case 0x95: return fmt.Sprintf("STA $%02X,X", byte2)
	case 0x96: return fmt.Sprintf("STX $%02X,Y", byte2)
	case 0x98: return "TYA"
	case 0x99: return fmt.Sprintf("STA $%04X,Y", uint16(byte3)<<8|uint16(byte2))
	case 0x9A: return "TXS"
	case 0x9D: return fmt.Sprintf("STA $%04X,X", uint16(byte3)<<8|uint16(byte2))
	case 0xA0: return fmt.Sprintf("LDY #$%02X", byte2)
	case 0xA1: return fmt.Sprintf("LDA ($%02X,X)", byte2)
	case 0xA2: return fmt.Sprintf("LDX #$%02X", byte2)
	case 0xA4: return fmt.Sprintf("LDY $%02X", byte2)
	case 0xA5: return fmt.Sprintf("LDA $%02X", byte2)
	case 0xA6: return fmt.Sprintf("LDX $%02X", byte2)
	case 0xA8: return "TAY"
	case 0xA9: return fmt.Sprintf("LDA #$%02X", byte2)
	case 0xAA: return "TAX"
	case 0xAC: return fmt.Sprintf("LDY $%04X", uint16(byte3)<<8|uint16(byte2))
	case 0xAD: return fmt.Sprintf("LDA $%04X", uint16(byte3)<<8|uint16(byte2))
	case 0xAE: return fmt.Sprintf("LDX $%04X", uint16(byte3)<<8|uint16(byte2))
	case 0xB0: return fmt.Sprintf("BCS $%04X", cpu.PC+2+uint16(int8(byte2)))
	case 0xB1: return fmt.Sprintf("LDA ($%02X),Y", byte2)
	case 0xB4: return fmt.Sprintf("LDY $%02X,X", byte2)
	case 0xB5: return fmt.Sprintf("LDA $%02X,X", byte2)
	case 0xB6: return fmt.Sprintf("LDX $%02X,Y", byte2)
	case 0xB8: return "CLV"
	case 0xB9: return fmt.Sprintf("LDA $%04X,Y", uint16(byte3)<<8|uint16(byte2))
	case 0xBA: return "TSX"
	case 0xBC: return fmt.Sprintf("LDY $%04X,X", uint16(byte3)<<8|uint16(byte2))
	case 0xBD: return fmt.Sprintf("LDA $%04X,X", uint16(byte3)<<8|uint16(byte2))
	case 0xBE: return fmt.Sprintf("LDX $%04X,Y", uint16(byte3)<<8|uint16(byte2))
	case 0xC0: return fmt.Sprintf("CPY #$%02X", byte2)
	case 0xC8: return "INY"
	case 0xC9: return fmt.Sprintf("CMP #$%02X", byte2)
	case 0xCA: return "DEX"
	case 0xD0: return fmt.Sprintf("BNE $%04X", cpu.PC+2+uint16(int8(byte2)))
	case 0xD1:
		// CMP ($zp),Y
		zp := uint16(byte2)
		lo := cpu.Read(zp)
		hi := cpu.Read((zp + 1) & 0xFF)
		baseAddr := uint16(hi)<<8 | uint16(lo)
		effectiveAddr := baseAddr + uint16(cpu.Y)
		memVal := cpu.Read(effectiveAddr)
		return fmt.Sprintf("CMP ($%02X),Y = %04X @ %04X = %02X", byte2, baseAddr, effectiveAddr, memVal)
	case 0xD8: return "CLD"
	case 0xD9:
		addr := uint16(byte3)<<8 | uint16(byte2)
		effectiveAddr := addr + uint16(cpu.Y)
		memVal := cpu.Read(effectiveAddr)
		return fmt.Sprintf("CMP $%04X,Y @ %04X = %02X", addr, effectiveAddr, memVal)
	case 0xDD:
		addr := uint16(byte3)<<8 | uint16(byte2)
		effectiveAddr := addr + uint16(cpu.X)
		memVal := cpu.Read(effectiveAddr)
		return fmt.Sprintf("CMP $%04X,X @ %04X = %02X", addr, effectiveAddr, memVal)
	case 0xE0: return fmt.Sprintf("CPX #$%02X", byte2)
	case 0xE8: return "INX"
	case 0xE9: return fmt.Sprintf("SBC #$%02X", byte2)
	case 0xEA: return "NOP"
	case 0xF0: return fmt.Sprintf("BEQ $%04X", cpu.PC+2+uint16(int8(byte2)))
	case 0xF1:
		// SBC ($zp),Y
		zp := uint16(byte2)
		lo := cpu.Read(zp)
		hi := cpu.Read((zp + 1) & 0xFF)
		baseAddr := uint16(hi)<<8 | uint16(lo)
		effectiveAddr := baseAddr + uint16(cpu.Y)
		memVal := cpu.Read(effectiveAddr)
		return fmt.Sprintf("SBC ($%02X),Y = %04X @ %04X = %02X", byte2, baseAddr, effectiveAddr, memVal)
	case 0xF9:
		addr := uint16(byte3)<<8 | uint16(byte2)
		effectiveAddr := addr + uint16(cpu.Y)
		memVal := cpu.Read(effectiveAddr)
		return fmt.Sprintf("SBC $%04X,Y @ %04X = %02X", addr, effectiveAddr, memVal)
	case 0xFD:
		addr := uint16(byte3)<<8 | uint16(byte2)
		effectiveAddr := addr + uint16(cpu.X)
		memVal := cpu.Read(effectiveAddr)
		return fmt.Sprintf("SBC $%04X,X @ %04X = %02X", addr, effectiveAddr, memVal)
	case 0xF8: return "SED"
	// LAX (LDA + LDX) - Unofficial instructions
	case 0xA3:
		// LAX ($zp,X)
		zp := uint16(byte2) + uint16(cpu.X)
		zp &= 0xFF // Keep in zero page
		memVal := cpu.Read(zp)
		return fmt.Sprintf("*LAX ($%02X,X) @ %02X = %02X", byte2, uint8(zp), memVal)
	case 0xA7:
		// LAX $zp
		memVal := cpu.Read(uint16(byte2))
		return fmt.Sprintf("*LAX $%02X = %02X", byte2, memVal)
	case 0xAF:
		// LAX $abs
		addr := uint16(byte3)<<8 | uint16(byte2)
		memVal := cpu.Read(addr)
		return fmt.Sprintf("*LAX $%04X = %02X", addr, memVal)
	case 0xB3:
		// LAX ($zp),Y
		zp := uint16(byte2)
		lo := cpu.Read(zp)
		hi := cpu.Read((zp + 1) & 0xFF)
		baseAddr := uint16(hi)<<8 | uint16(lo)
		effectiveAddr := baseAddr + uint16(cpu.Y)
		memVal := cpu.Read(effectiveAddr)
		return fmt.Sprintf("*LAX ($%02X),Y = %04X @ %04X = %02X", byte2, baseAddr, effectiveAddr, memVal)
	case 0xB7:
		// LAX $zp,Y
		addr := uint16(byte2) + uint16(cpu.Y)
		addr &= 0xFF // Keep in zero page
		memVal := cpu.Read(addr)
		return fmt.Sprintf("*LAX $%02X,Y @ %02X = %02X", byte2, uint8(addr), memVal)
	case 0xBF:
		// LAX $abs,Y
		addr := uint16(byte3)<<8 | uint16(byte2)
		effectiveAddr := addr + uint16(cpu.Y)
		memVal := cpu.Read(effectiveAddr)
		return fmt.Sprintf("*LAX $%04X,Y @ %04X = %02X", addr, effectiveAddr, memVal)
	case 0x8F: // SAX $abs
		addr := uint16(byte3)<<8 | uint16(byte2)
		memVal := cpu.Read(addr)
		return fmt.Sprintf("*SAX $%04X = %02X", addr, memVal)
	// SAX (STA and STX) instructions
	case 0x83: // SAX ($zp,X)
		zpBase := uint16(byte2)
		zpAddr := (zpBase + uint16(cpu.X)) & 0xFF
		lo := cpu.Read(zpAddr)
		hi := cpu.Read((zpAddr + 1) & 0xFF)
		effectiveAddr := (uint16(hi) << 8) | uint16(lo)
		memVal := cpu.Read(effectiveAddr)
		return fmt.Sprintf("*SAX ($%02X,X) @ %02X = %04X = %02X", byte2, uint8(zpAddr), effectiveAddr, memVal)
	case 0x87: // SAX $zp
		addr := uint16(byte2)
		memVal := cpu.Read(addr)
		return fmt.Sprintf("*SAX $%02X = %02X", byte2, memVal)
	case 0x97: // SAX $zp,Y
		baseAddr := uint16(byte2)
		addr := (baseAddr + uint16(cpu.Y)) & 0xFF
		memVal := cpu.Read(addr)
		return fmt.Sprintf("*SAX $%02X,Y @ %02X = %02X", byte2, uint8(addr), memVal)
	default: return fmt.Sprintf("UNK $%02X", opcode)
	}
}

// Override the Clock method to use enhanced execution
func (cpu *CPU) Clock() {
	if cpu.cycles == 0 {
		opcode := cpu.Read(cpu.PC)
		
		// Set accurate cycle count for each instruction first
		instructionCycles := cpu.getInstructionCycles(opcode)
		
		// Add extra cycle for branch instructions if branch will be taken
		if cpu.isBranchInstruction(opcode) {
			// For branch instructions, we need to predict if branch will be taken
			// This is a simplified prediction based on the current processor state
			if cpu.willBranchBeTaken(opcode) {
				instructionCycles++ // +1 cycle for successful branch
				
				// Also check for page crossing
				// For branch instructions, we need to predict the target address
				if cpu.willBranchCrossPage(opcode) {
					instructionCycles++ // +1 cycle for page crossing
				}
			}
		}
		
		// Log instruction before execution with nestest timing (including the cycles for this instruction)
		// nestest.log format requires the cycle count AFTER instruction execution
		// Calculate cycle count for logging with PLA special timing
		cpuCycle := int(cpu.totalCycles) + int(instructionCycles)
		if opcode == 104 { // PLA instruction (0x68 = 104) has extra internal timing cycle
			cpuCycle = cpuCycle + 1
		}
		
		// nestest.log shows cycles at the START of instruction execution
		// So use current totalCycles, not totalCycles + instructionCycles
		ppuCycleForLog := int(cpu.totalCycles) * 3
		cpuCycleAfterInstruction := int(cpu.totalCycles)
		
		cpu.logInstruction(opcode, ppuCycleForLog, cpuCycleAfterInstruction)

		// Base length is committed before execution so readIndexedAddr can add
		// a page-cross cycle to this instruction's countdown.
		cpu.cycles = instructionCycles
		cpu.executeInstructionEnhanced(opcode)
		
		// PLA timing adjustment is handled in logging phase
	}
	cpu.cycles--
	cpu.totalCycles++
}

// getInstructionCycles returns the base cycle count for opcode.
// Indexed-read page crosses are not included; readIndexedAddr adds those.
func (cpu *CPU) getInstructionCycles(opcode uint8) uint8 {
	return baseCycles[opcode]
}

// readIndexedAddr is the only place an indexed read adds a page-cross cycle.
// abs,X / abs,Y / (zp),Y reads (LDA, LDX, LDY, EOR, AND, ORA, ADC, SBC, CMP,
// LAX, and NOP abs,X) take one extra cycle when base and base+index are in
// different pages. The extra tick is added to the instruction countdown here;
// Clock records that same tick in totalCycles, so the two counters stay equal.
// Stores and RMW already include the indexed fixup in baseCycles and must not
// call this. A cross does not add a further cycle.
// https://www.nesdev.org/wiki/Instruction_reference
// https://www.nesdev.org/wiki/CPU_addressing_modes
func (cpu *CPU) readIndexedAddr(base uint16, index uint8) uint16 {
	addr := base + uint16(index)
	if base&0xFF00 != addr&0xFF00 {
		cpu.cycles++
	}
	return addr
}

// baseCycles is the cycle count per opcode before page-crossing extras.
var baseCycles = [256]uint8{
	7, 6, 2, 8, 3, 3, 5, 5, 3, 2, 2, 2, 4, 4, 6, 6, // 0x00
	2, 5, 2, 8, 4, 4, 6, 6, 2, 4, 2, 7, 4, 4, 7, 7, // 0x10
	6, 6, 2, 8, 3, 3, 5, 5, 4, 2, 2, 2, 4, 4, 6, 6, // 0x20
	2, 5, 2, 8, 4, 4, 6, 6, 2, 4, 2, 7, 4, 4, 7, 7, // 0x30
	6, 6, 2, 8, 3, 3, 5, 5, 3, 2, 2, 2, 3, 4, 6, 6, // 0x40
	2, 5, 2, 8, 4, 4, 6, 6, 2, 4, 2, 7, 4, 4, 7, 7, // 0x50
	6, 6, 2, 8, 3, 3, 5, 5, 4, 2, 2, 2, 5, 4, 6, 6, // 0x60
	2, 5, 2, 8, 4, 4, 6, 6, 2, 4, 2, 7, 4, 4, 7, 7, // 0x70
	2, 6, 2, 6, 3, 3, 3, 3, 2, 2, 2, 2, 4, 4, 4, 4, // 0x80
	2, 6, 2, 2, 4, 4, 4, 4, 2, 5, 2, 2, 2, 5, 2, 2, // 0x90
	2, 6, 2, 6, 3, 3, 3, 3, 2, 2, 2, 2, 4, 4, 4, 4, // 0xA0
	2, 5, 2, 5, 4, 4, 4, 4, 2, 4, 2, 2, 4, 4, 4, 4, // 0xB0
	2, 6, 2, 8, 3, 3, 5, 5, 2, 2, 2, 2, 4, 4, 6, 6, // 0xC0
	2, 5, 2, 8, 4, 4, 6, 6, 2, 4, 2, 7, 4, 4, 7, 7, // 0xD0
	2, 6, 2, 8, 3, 3, 5, 5, 2, 2, 2, 2, 4, 4, 6, 6, // 0xE0
	2, 5, 2, 8, 4, 4, 6, 6, 2, 4, 2, 7, 4, 4, 7, 7, // 0xF0
}

// Check if instruction is a branch instruction
func (cpu *CPU) isBranchInstruction(opcode uint8) bool {
	branchInstructions := []uint8{
		0x10, // BPL
		0x30, // BMI
		0x50, // BVC
		0x70, // BVS
		0x90, // BCC
		0xB0, // BCS
		0xD0, // BNE
		0xF0, // BEQ
	}
	
	for _, branch := range branchInstructions {
		if opcode == branch {
			return true
		}
	}
	return false
}

// Check if branch was taken by comparing PC values
func (cpu *CPU) branchTaken(opcode uint8, oldPC uint16) bool {
	// For branch instructions, if PC changed more than the instruction length, branch was taken
	expectedPC := oldPC + 2 // All branch instructions are 2 bytes
	return cpu.PC != expectedPC
}

// Predict if branch will be taken based on current processor state
func (cpu *CPU) willBranchBeTaken(opcode uint8) bool {
	switch opcode {
	case 0x10: return cpu.GetFlag(N) == 0 // BPL
	case 0x30: return cpu.GetFlag(N) == 1 // BMI
	case 0x50: return cpu.GetFlag(V) == 0 // BVC
	case 0x70: return cpu.GetFlag(V) == 1 // BVS
	case 0x90: return cpu.GetFlag(C) == 0 // BCC
	case 0xB0: return cpu.GetFlag(C) == 1 // BCS
	case 0xD0: return cpu.GetFlag(Z) == 0 // BNE
	case 0xF0: return cpu.GetFlag(Z) == 1 // BEQ
	}
	return false
}

// Check if branch instruction will cross page boundary
func (cpu *CPU) willBranchCrossPage(opcode uint8) bool {
	// Read the branch offset (relative addressing)
	offset := int8(cpu.Read(cpu.PC + 1))
	
	// Calculate target address
	// Branch PC is PC + 2 (after reading opcode + offset)
	branchPC := cpu.PC + 2
	targetPC := uint16(int32(branchPC) + int32(offset))
	
	// Check if page boundary is crossed
	return (branchPC & 0xFF00) != (targetPC & 0xFF00)
}

