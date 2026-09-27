package cpu

import "fmt"

// Enhanced CPU implementation for nestest.nes compatibility

func (cpu *CPU) executeInstructionEnhanced(opcode uint8) {
	switch opcode {
	// Control Instructions
	case 0x00: // BRK
		cpu.PC += 2 // BRK is a 2-byte instruction  
		cpu.SetFlag(B, true)
		cpu.SetFlag(I, true)
		// Push PC and status to stack
		cpu.Write(0x0100+uint16(cpu.SP), uint8((cpu.PC>>8)&0x00FF))
		cpu.SP--
		cpu.Write(0x0100+uint16(cpu.SP), uint8(cpu.PC&0x00FF))
		cpu.SP--
		cpu.Write(0x0100+uint16(cpu.SP), cpu.Status)
		cpu.SP--
		// Read IRQ vector
		lo := uint16(cpu.Read(0xFFFE))
		hi := uint16(cpu.Read(0xFFFF))
		cpu.PC = (hi << 8) | lo
	
	case 0x04: // *NOP zp - Unofficial instruction (2 bytes)
		// This is an unofficial NOP instruction that reads a zero page address but does nothing
		cpu.PC++ // Skip operand byte
		cpu.PC++
	
	case 0x44: // *NOP zp - Unofficial instruction (2 bytes)
		// This is an unofficial NOP instruction that reads a zero page address but does nothing
		cpu.PC++ // Skip operand byte
		cpu.PC++
	
	case 0x64: // *NOP zp - Unofficial instruction (2 bytes)
		// This is an unofficial NOP instruction that reads a zero page address but does nothing
		cpu.PC++ // Skip operand byte
		cpu.PC++
	
	case 0x0C: // *NOP abs - Unofficial instruction (3 bytes)
		// This is an unofficial NOP instruction that reads an absolute address but does nothing
		cpu.PC++ // Skip low byte
		cpu.PC++ // Skip high byte
		cpu.PC++
	
	case 0x14: // *NOP zp,X - Unofficial instruction (2 bytes)
		// This is an unofficial NOP instruction that reads a zero page indexed address but does nothing
		cpu.PC++ // Skip operand byte
		cpu.PC++
	
	case 0x34: // *NOP zp,X - Unofficial instruction (2 bytes)
		// This is an unofficial NOP instruction that reads a zero page indexed address but does nothing
		cpu.PC++ // Skip operand byte
		cpu.PC++
	
	case 0x54: // *NOP zp,X - Unofficial instruction (2 bytes)
		// This is an unofficial NOP instruction that reads a zero page indexed address but does nothing
		cpu.PC++ // Skip operand byte
		cpu.PC++
	
	case 0x74: // *NOP zp,X - Unofficial instruction (2 bytes)
		// This is an unofficial NOP instruction that reads a zero page indexed address but does nothing
		cpu.PC++ // Skip operand byte
		cpu.PC++
	
	case 0xD4: // *NOP zp,X - Unofficial instruction (2 bytes)
		// This is an unofficial NOP instruction that reads a zero page indexed address but does nothing
		cpu.PC++ // Skip operand byte
		cpu.PC++
	
	case 0xF4: // *NOP zp,X - Unofficial instruction (2 bytes)
		// This is an unofficial NOP instruction that reads a zero page indexed address but does nothing
		cpu.PC++ // Skip operand byte
		cpu.PC++
	
	// 1-byte unofficial NOP instructions
	case 0x1A: // *NOP - Unofficial instruction (1 byte)
		// This is an unofficial NOP instruction that does nothing
		cpu.PC++

	case 0x3A: // *NOP - Unofficial instruction (1 byte)
		// This is an unofficial NOP instruction that does nothing
		cpu.PC++

	case 0x5A: // *NOP - Unofficial instruction (1 byte)
		// This is an unofficial NOP instruction that does nothing
		cpu.PC++

	case 0x7A: // *NOP - Unofficial instruction (1 byte)
		// This is an unofficial NOP instruction that does nothing
		cpu.PC++

	case 0xDA: // *NOP - Unofficial instruction (1 byte)
		// This is an unofficial NOP instruction that does nothing
		cpu.PC++

	case 0xFA: // *NOP - Unofficial instruction (1 byte)
		// This is an unofficial NOP instruction that does nothing
		cpu.PC++

	// 2-byte unofficial NOP instruction
	case 0x80: // *NOP #imm - Unofficial instruction (2 bytes)
		// This is an unofficial NOP instruction that reads an immediate value but does nothing
		cpu.PC++ // Skip operand byte
		cpu.PC++

	// 3-byte unofficial NOP abs,X. The indexed read crosses a page in readIndexedAddr.
	case 0x1C, 0x3C, 0x5C, 0x7C, 0xDC, 0xFC:
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		cpu.readIndexedAddr((hi<<8)|lo, cpu.X)
		cpu.PC++

	case 0x20: // JSR abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		target := (hi << 8) | lo
		returnAddr := cpu.PC
		cpu.Write(0x0100+uint16(cpu.SP), uint8(returnAddr>>8))
		cpu.SP--
		cpu.Write(0x0100+uint16(cpu.SP), uint8(returnAddr&0xFF))
		cpu.SP--
		cpu.PC = target
	
	case 0x24: // BIT zp
		cpu.PC++
		addr := uint16(cpu.Read(cpu.PC))
		value := cpu.Read(addr)
		result := cpu.A & value
		// BIT sets Z flag based on AND result, V and N from memory value
		cpu.SetFlag(Z, result == 0)
		cpu.SetFlag(V, (value & 0x40) != 0)
		cpu.SetFlag(N, (value & 0x80) != 0)
		cpu.PC++
	
	case 0x2C: // BIT abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := (hi << 8) | lo
		value := cpu.Read(addr)
		result := cpu.A & value
		// BIT sets Z flag based on AND result, V and N from memory value
		cpu.SetFlag(Z, result == 0)
		cpu.SetFlag(V, (value & 0x40) != 0)
		cpu.SetFlag(N, (value & 0x80) != 0)
		cpu.PC++
	
	case 0x4C: // JMP abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		cpu.PC = (hi << 8) | lo
	
	case 0x6C: // JMP ind
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		ptr := (hi << 8) | lo
		// Simulate page boundary bug: when ptr low byte is 0xFF, 
		// high byte is read from same page (ptr&0xFF00) instead of ptr+1
		if (ptr & 0xFF) == 0xFF {
			cpu.PC = (uint16(cpu.Read(ptr&0xFF00)) << 8) | uint16(cpu.Read(ptr))
		} else {
			cpu.PC = (uint16(cpu.Read(ptr+1)) << 8) | uint16(cpu.Read(ptr))
		}
	
	case 0x60: // RTS
		cpu.SP++
		lo := uint16(cpu.Read(0x0100 + uint16(cpu.SP)))
		cpu.SP++
		hi := uint16(cpu.Read(0x0100 + uint16(cpu.SP)))
		cpu.PC = (hi << 8) | lo
		cpu.PC++
	
	case 0x40: // RTI
		cpu.SP++
		cpu.Status = cpu.Read(0x0100 + uint16(cpu.SP))
		cpu.Status &= ^uint8(B)
		cpu.SetFlag(U, true)
		cpu.SP++
		lo := uint16(cpu.Read(0x0100 + uint16(cpu.SP)))
		cpu.SP++
		hi := uint16(cpu.Read(0x0100 + uint16(cpu.SP)))
		cpu.PC = (hi << 8) | lo

	// Load/Store Instructions
	case 0xA9: // LDA #
		cpu.PC++
		cpu.A = cpu.Read(cpu.PC)
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0xA5: // LDA zp
		cpu.PC++
		addr := uint16(cpu.Read(cpu.PC))
		cpu.A = cpu.Read(addr)
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0xB5: // LDA zp,X
		cpu.PC++
		addr := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		cpu.A = cpu.Read(addr)
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0xAD: // LDA abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := (hi << 8) | lo
		cpu.A = cpu.Read(addr)
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0xBD: // LDA abs,X
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := cpu.readIndexedAddr((hi<<8)|lo, cpu.X)
		cpu.A = cpu.Read(addr)
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0xB9: // LDA abs,Y
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		baseAddr := (hi << 8) | lo
		addr := cpu.readIndexedAddr(baseAddr, cpu.Y)
		cpu.A = cpu.Read(addr)
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0xA1: // LDA (zp,X)
		cpu.PC++
		zp := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		lo := uint16(cpu.Read(zp))
		hi := uint16(cpu.Read((zp + 1) & 0xFF))
		addr := (hi << 8) | lo
		cpu.A = cpu.Read(addr)
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0xB1: // LDA (zp),Y
		cpu.PC++
		zp := uint16(cpu.Read(cpu.PC))
		lo := uint16(cpu.Read(zp))
		hi := uint16(cpu.Read((zp + 1) & 0xFF))
		addr := cpu.readIndexedAddr((hi<<8)|lo, cpu.Y)
		cpu.A = cpu.Read(addr)
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++

	// Similar patterns for LDX and LDY
	case 0xA2: // LDX #
		cpu.PC++
		cpu.X = cpu.Read(cpu.PC)
		cpu.SetFlag(Z, cpu.X == 0x00)
		cpu.SetFlag(N, cpu.X&0x80 != 0)
		cpu.PC++
	
	case 0xA6: // LDX zp
		cpu.PC++
		addr := uint16(cpu.Read(cpu.PC))
		cpu.X = cpu.Read(addr)
		cpu.SetFlag(Z, cpu.X == 0x00)
		cpu.SetFlag(N, cpu.X&0x80 != 0)
		cpu.PC++
	
	case 0xB6: // LDX zp,Y
		cpu.PC++
		addr := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.Y)) & 0xFF
		cpu.X = cpu.Read(addr)
		cpu.SetFlag(Z, cpu.X == 0x00)
		cpu.SetFlag(N, cpu.X&0x80 != 0)
		cpu.PC++
	
	case 0xAE: // LDX abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := (hi << 8) | lo
		cpu.X = cpu.Read(addr)
		cpu.SetFlag(Z, cpu.X == 0x00)
		cpu.SetFlag(N, cpu.X&0x80 != 0)
		cpu.PC++
	
	case 0xBE: // LDX abs,Y
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := cpu.readIndexedAddr((hi<<8)|lo, cpu.Y)
		cpu.X = cpu.Read(addr)
		cpu.SetFlag(Z, cpu.X == 0x00)
		cpu.SetFlag(N, cpu.X&0x80 != 0)
		cpu.PC++

	case 0xA0: // LDY #
		cpu.PC++
		cpu.Y = cpu.Read(cpu.PC)
		cpu.SetFlag(Z, cpu.Y == 0x00)
		cpu.SetFlag(N, cpu.Y&0x80 != 0)
		cpu.PC++
	
	case 0xA4: // LDY zp
		cpu.PC++
		addr := uint16(cpu.Read(cpu.PC))
		cpu.Y = cpu.Read(addr)
		cpu.SetFlag(Z, cpu.Y == 0x00)
		cpu.SetFlag(N, cpu.Y&0x80 != 0)
		cpu.PC++
	
	case 0xB4: // LDY zp,X
		cpu.PC++
		addr := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		cpu.Y = cpu.Read(addr)
		cpu.SetFlag(Z, cpu.Y == 0x00)
		cpu.SetFlag(N, cpu.Y&0x80 != 0)
		cpu.PC++
	
	case 0xAC: // LDY abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := (hi << 8) | lo
		cpu.Y = cpu.Read(addr)
		cpu.SetFlag(Z, cpu.Y == 0x00)
		cpu.SetFlag(N, cpu.Y&0x80 != 0)
		cpu.PC++
	
	case 0xBC: // LDY abs,X
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := cpu.readIndexedAddr((hi<<8)|lo, cpu.X)
		cpu.Y = cpu.Read(addr)
		cpu.SetFlag(Z, cpu.Y == 0x00)
		cpu.SetFlag(N, cpu.Y&0x80 != 0)
		cpu.PC++

	// Status Flag Operations
	case 0x18: // CLC
		cpu.SetFlag(C, false)
		cpu.PC++
	case 0x38: // SEC
		cpu.SetFlag(C, true)
		cpu.PC++
	case 0x58: // CLI
		cpu.SetFlag(I, false)
		cpu.PC++
	case 0x78: // SEI
		cpu.SetFlag(I, true)
		cpu.PC++
	case 0xB8: // CLV
		cpu.SetFlag(V, false)
		cpu.PC++
	case 0xD8: // CLD
		cpu.SetFlag(D, false)
		cpu.PC++
	case 0xF8: // SED
		cpu.SetFlag(D, true)
		cpu.PC++

	// Register Transfer
	case 0xAA: // TAX
		cpu.X = cpu.A
		cpu.SetFlag(Z, cpu.X == 0x00)
		cpu.SetFlag(N, cpu.X&0x80 != 0)
		cpu.PC++
	case 0xA8: // TAY
		cpu.Y = cpu.A
		cpu.SetFlag(Z, cpu.Y == 0x00)
		cpu.SetFlag(N, cpu.Y&0x80 != 0)
		cpu.PC++
	case 0x8A: // TXA
		cpu.A = cpu.X
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	case 0x98: // TYA
		cpu.A = cpu.Y
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	case 0x9A: // TXS
		cpu.SP = cpu.X
		cpu.PC++
	case 0xBA: // TSX
		cpu.X = cpu.SP
		cpu.SetFlag(Z, cpu.X == 0x00)
		cpu.SetFlag(N, cpu.X&0x80 != 0)
		cpu.PC++

	// Stack Operations
	case 0x48: // PHA
		cpu.Write(0x0100+uint16(cpu.SP), cpu.A)
		cpu.SP--
		cpu.PC++
	case 0x68: // PLA
		cpu.SP++
		cpu.A = cpu.Read(0x0100 + uint16(cpu.SP))
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	case 0x08: // PHP
		cpu.Write(0x0100+uint16(cpu.SP), cpu.Status|B|U)
		cpu.SP--
		cpu.PC++
	case 0x28: // PLP
		cpu.SP++
		stackValue := cpu.Read(0x0100 + uint16(cpu.SP))
		// PLP ignores bits 4 and 5 (B and U), U is always 1, B is never set by PLP
		cpu.Status = (stackValue & ^uint8(B)) | uint8(U)
		cpu.PC++

	// Increment/Decrement
	case 0xE8: // INX
		cpu.X++
		cpu.SetFlag(Z, cpu.X == 0x00)
		cpu.SetFlag(N, cpu.X&0x80 != 0)
		cpu.PC++
	case 0xC8: // INY
		cpu.Y++
		cpu.SetFlag(Z, cpu.Y == 0x00)
		cpu.SetFlag(N, cpu.Y&0x80 != 0)
		cpu.PC++
	case 0xCA: // DEX
		cpu.X--
		cpu.SetFlag(Z, cpu.X == 0x00)
		cpu.SetFlag(N, cpu.X&0x80 != 0)
		cpu.PC++
	case 0x88: // DEY
		cpu.Y--
		cpu.SetFlag(Z, cpu.Y == 0x00)
		cpu.SetFlag(N, cpu.Y&0x80 != 0)
		cpu.PC++

	// Compare Operations  
	case 0xC9: // CMP #
		cpu.PC++
		value := cpu.Read(cpu.PC)
		temp := uint16(cpu.A) - uint16(value)
		cpu.SetFlag(C, cpu.A >= value)
		cpu.SetFlag(Z, (temp&0x00FF) == 0x0000)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.PC++
	
	case 0xC5: // CMP zp
		cpu.PC++
		addr := uint16(cpu.Read(cpu.PC))
		value := cpu.Read(addr)
		temp := uint16(cpu.A) - uint16(value)
		cpu.SetFlag(C, cpu.A >= value)
		cpu.SetFlag(Z, (temp&0x00FF) == 0x0000)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.PC++
	
	case 0xD5: // CMP zp,X
		cpu.PC++
		addr := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		value := cpu.Read(addr)
		temp := uint16(cpu.A) - uint16(value)
		cpu.SetFlag(C, cpu.A >= value)
		cpu.SetFlag(Z, (temp&0x00FF) == 0x0000)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.PC++
	
	case 0xCD: // CMP abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := (hi << 8) | lo
		value := cpu.Read(addr)
		temp := uint16(cpu.A) - uint16(value)
		cpu.SetFlag(C, cpu.A >= value)
		cpu.SetFlag(Z, (temp&0x00FF) == 0x0000)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.PC++
	
	case 0xE0: // CPX #
		cpu.PC++
		value := cpu.Read(cpu.PC)
		temp := uint16(cpu.X) - uint16(value)
		cpu.SetFlag(C, cpu.X >= value)
		cpu.SetFlag(Z, (temp&0x00FF) == 0x0000)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.PC++
	
	case 0xE4: // CPX zp
		cpu.PC++
		addr := uint16(cpu.Read(cpu.PC))
		value := cpu.Read(addr)
		temp := uint16(cpu.X) - uint16(value)
		cpu.SetFlag(C, cpu.X >= value)
		cpu.SetFlag(Z, (temp&0x00FF) == 0x0000)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.PC++
	
	case 0xEC: // CPX abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := (hi << 8) | lo
		value := cpu.Read(addr)
		temp := uint16(cpu.X) - uint16(value)
		cpu.SetFlag(C, cpu.X >= value)
		cpu.SetFlag(Z, (temp&0x00FF) == 0x0000)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.PC++
	
	case 0xC0: // CPY #
		cpu.PC++
		value := cpu.Read(cpu.PC)
		temp := uint16(cpu.Y) - uint16(value)
		cpu.SetFlag(C, cpu.Y >= value)
		cpu.SetFlag(Z, (temp&0x00FF) == 0x0000)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.PC++
	
	case 0xC4: // CPY zp
		cpu.PC++
		addr := uint16(cpu.Read(cpu.PC))
		value := cpu.Read(addr)
		temp := uint16(cpu.Y) - uint16(value)
		cpu.SetFlag(C, cpu.Y >= value)
		cpu.SetFlag(Z, (temp&0x00FF) == 0x0000)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.PC++

	case 0xCC: // CPY abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := (hi << 8) | lo
		value := cpu.Read(addr)
		temp := uint16(cpu.Y) - uint16(value)
		cpu.SetFlag(C, cpu.Y >= value)
		cpu.SetFlag(Z, (temp&0x00FF) == 0x0000)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.PC++

	case 0xDD: // CMP abs,X
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		baseAddr := (hi << 8) | lo
		addr := cpu.readIndexedAddr(baseAddr, cpu.X)
		value := cpu.Read(addr)
		temp := uint16(cpu.A) - uint16(value)
		cpu.SetFlag(C, cpu.A >= value)
		cpu.SetFlag(Z, (temp&0x00FF) == 0x0000)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.PC++

	case 0xD9: // CMP abs,Y
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		baseAddr := (hi << 8) | lo
		addr := cpu.readIndexedAddr(baseAddr, cpu.Y)
		value := cpu.Read(addr)
		temp := uint16(cpu.A) - uint16(value)
		cpu.SetFlag(C, cpu.A >= value)
		cpu.SetFlag(Z, (temp&0x00FF) == 0x0000)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.PC++

	// Logical Operations
	case 0x21: // AND ($zp,X)
		cpu.PC++
		zpAddr := uint16(cpu.Read(cpu.PC))
		indirectAddr := (zpAddr + uint16(cpu.X)) & 0xFF
		lo := uint16(cpu.Read(indirectAddr))
		hi := uint16(cpu.Read((indirectAddr + 1) & 0xFF))
		addr := (hi << 8) | lo
		value := cpu.Read(addr)
		cpu.A = cpu.A & value
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0x25: // AND zp
		cpu.PC++
		addr := uint16(cpu.Read(cpu.PC))
		value := cpu.Read(addr)
		cpu.A = cpu.A & value
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0x35: // AND zp,X
		cpu.PC++
		addr := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		value := cpu.Read(addr)
		cpu.A = cpu.A & value
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0x29: // AND #
		cpu.PC++
		value := cpu.Read(cpu.PC)
		cpu.A = cpu.A & value
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0x01: // ORA ($zp,X)
		cpu.PC++
		zpAddr := uint16(cpu.Read(cpu.PC))
		indirectAddr := (zpAddr + uint16(cpu.X)) & 0xFF
		lo := uint16(cpu.Read(indirectAddr))
		hi := uint16(cpu.Read((indirectAddr + 1) & 0xFF))
		addr := (hi << 8) | lo
		value := cpu.Read(addr)
		cpu.A = cpu.A | value
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0x11: // ORA ($zp),Y
		cpu.PC++
		zpAddr := uint16(cpu.Read(cpu.PC))
		lo := uint16(cpu.Read(zpAddr))
		hi := uint16(cpu.Read((zpAddr + 1) & 0xFF))
		effectiveAddr := cpu.readIndexedAddr((hi<<8)|lo, cpu.Y)
		value := cpu.Read(effectiveAddr)
		cpu.A = cpu.A | value
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0x31: // AND ($zp),Y
		cpu.PC++
		zpAddr := uint16(cpu.Read(cpu.PC))
		lo := uint16(cpu.Read(zpAddr))
		hi := uint16(cpu.Read((zpAddr + 1) & 0xFF))
		effectiveAddr := cpu.readIndexedAddr((hi<<8)|lo, cpu.Y)
		value := cpu.Read(effectiveAddr)
		cpu.A = cpu.A & value
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0x51: // EOR ($zp),Y
		cpu.PC++
		zpAddr := uint16(cpu.Read(cpu.PC))
		lo := uint16(cpu.Read(zpAddr))
		hi := uint16(cpu.Read((zpAddr + 1) & 0xFF))
		effectiveAddr := cpu.readIndexedAddr((hi<<8)|lo, cpu.Y)
		value := cpu.Read(effectiveAddr)
		cpu.A = cpu.A ^ value
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0x71: // ADC ($zp),Y
		cpu.PC++
		zpAddr := uint16(cpu.Read(cpu.PC))
		lo := uint16(cpu.Read(zpAddr))
		hi := uint16(cpu.Read((zpAddr + 1) & 0xFF))
		effectiveAddr := cpu.readIndexedAddr((hi<<8)|lo, cpu.Y)
		value := cpu.Read(effectiveAddr)
		oldA := cpu.A
		carry := uint16(0)
		if cpu.GetFlag(C) == 1 {
			carry = 1
		}
		result16 := uint16(cpu.A) + uint16(value) + carry
		cpu.A = uint8(result16 & 0xFF)
		cpu.SetFlag(Z, cpu.A == 0)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.SetFlag(C, result16 > 255)
		// Overflow: if sign of both operands is same and result sign is different
		cpu.SetFlag(V, ((oldA^value) == 0) && ((oldA^cpu.A)&0x80) != 0)
		cpu.PC++
	
	case 0x05: // ORA zp
		cpu.PC++
		addr := uint16(cpu.Read(cpu.PC))
		value := cpu.Read(addr)
		cpu.A = cpu.A | value
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0x15: // ORA zp,X
		cpu.PC++
		baseAddr := uint16(cpu.Read(cpu.PC))
		addr := (baseAddr + uint16(cpu.X)) & 0xFF
		value := cpu.Read(addr)
		cpu.A = cpu.A | value
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0x09: // ORA #
		cpu.PC++
		value := cpu.Read(cpu.PC)
		cpu.A = cpu.A | value
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0x0D: // ORA abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := (hi << 8) | lo
		value := cpu.Read(addr)
		cpu.A = cpu.A | value
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0x19: // ORA abs,Y
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		baseAddr := (hi << 8) | lo
		addr := cpu.readIndexedAddr(baseAddr, cpu.Y)
		value := cpu.Read(addr)
		cpu.A = cpu.A | value
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++

	case 0x1D: // ORA abs,X
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		baseAddr := (hi << 8) | lo
		addr := cpu.readIndexedAddr(baseAddr, cpu.X)
		value := cpu.Read(addr)
		cpu.A = cpu.A | value
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0x2D: // AND abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := (hi << 8) | lo
		value := cpu.Read(addr)
		cpu.A = cpu.A & value
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0x3D: // AND abs,X
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		baseAddr := (hi << 8) | lo
		addr := cpu.readIndexedAddr(baseAddr, cpu.X)
		value := cpu.Read(addr)
		cpu.A = cpu.A & value
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0x39: // AND abs,Y
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		baseAddr := (hi << 8) | lo
		addr := cpu.readIndexedAddr(baseAddr, cpu.Y)
		value := cpu.Read(addr)
		cpu.A = cpu.A & value
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0x45: // EOR zp
		cpu.PC++
		addr := uint16(cpu.Read(cpu.PC))
		value := cpu.Read(addr)
		cpu.A = cpu.A ^ value
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0x55: // EOR zp,X
		cpu.PC++
		baseAddr := uint16(cpu.Read(cpu.PC))
		addr := (baseAddr + uint16(cpu.X)) & 0xFF
		value := cpu.Read(addr)
		cpu.A = cpu.A ^ value
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0x49: // EOR #
		cpu.PC++
		value := cpu.Read(cpu.PC)
		cpu.A = cpu.A ^ value
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0x4D: // EOR abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := (hi << 8) | lo
		value := cpu.Read(addr)
		cpu.A = cpu.A ^ value
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0x5D: // EOR abs,X
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		baseAddr := (hi << 8) | lo
		addr := cpu.readIndexedAddr(baseAddr, cpu.X)
		value := cpu.Read(addr)
		cpu.A = cpu.A ^ value
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0x59: // EOR abs,Y
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		baseAddr := (hi << 8) | lo
		addr := cpu.readIndexedAddr(baseAddr, cpu.Y)
		value := cpu.Read(addr)
		cpu.A = cpu.A ^ value
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0x41: // EOR (zp,X)
		cpu.PC++
		zp := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		lo := uint16(cpu.Read(zp))
		hi := uint16(cpu.Read((zp + 1) & 0xFF))
		addr := (hi << 8) | lo
		value := cpu.Read(addr)
		cpu.A = cpu.A ^ value
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0x61: // ADC (zp,X)
		cpu.PC++
		zp := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		lo := uint16(cpu.Read(zp))
		hi := uint16(cpu.Read((zp + 1) & 0xFF))
		addr := (hi << 8) | lo
		value := uint16(cpu.Read(addr))
		temp := uint16(cpu.A) + value + uint16(cpu.GetFlag(C))
		cpu.SetFlag(C, temp > 255)
		cpu.SetFlag(Z, (temp&0x00FF) == 0)
		cpu.SetFlag(V, (temp^uint16(cpu.A))&(temp^value)&0x0080 != 0)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.A = uint8(temp & 0x00FF)
		cpu.PC++
	
	case 0xC1: // CMP (zp,X)
		cpu.PC++
		zp := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		lo := uint16(cpu.Read(zp))
		hi := uint16(cpu.Read((zp + 1) & 0xFF))
		addr := (hi << 8) | lo
		value := uint16(cpu.Read(addr))
		temp := uint16(cpu.A) - value
		cpu.SetFlag(C, cpu.A >= uint8(value))
		cpu.SetFlag(Z, (temp&0x00FF) == 0)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.PC++

	case 0xD1: // CMP ($zp),Y
		cpu.PC++
		zp := uint16(cpu.Read(cpu.PC))
		lo := uint16(cpu.Read(zp))
		hi := uint16(cpu.Read((zp + 1) & 0xFF))
		effectiveAddr := cpu.readIndexedAddr((hi<<8)|lo, cpu.Y)
		value := uint16(cpu.Read(effectiveAddr))
		temp := uint16(cpu.A) - value
		cpu.SetFlag(C, cpu.A >= uint8(value))
		cpu.SetFlag(Z, (temp&0x00FF) == 0)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.PC++

	// Store Operations
	case 0x85: // STA zp
		cpu.PC++
		addr := uint16(cpu.Read(cpu.PC))
		cpu.Write(addr, cpu.A)
		cpu.PC++
	
	case 0x95: // STA zp,X
		cpu.PC++
		addr := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		cpu.Write(addr, cpu.A)
		cpu.PC++
	
	case 0x8D: // STA abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := (hi << 8) | lo
		cpu.Write(addr, cpu.A)
		cpu.PC++
	
	case 0x9D: // STA abs,X
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := ((hi << 8) | lo) + uint16(cpu.X)
		cpu.Write(addr, cpu.A)
		cpu.PC++
	
	case 0x99: // STA abs,Y
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := ((hi << 8) | lo) + uint16(cpu.Y)
		cpu.Write(addr, cpu.A)
		cpu.PC++
	
	case 0x81: // STA (zp,X)
		cpu.PC++
		zp := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		lo := uint16(cpu.Read(zp))
		hi := uint16(cpu.Read((zp + 1) & 0xFF))
		addr := (hi << 8) | lo
		cpu.Write(addr, cpu.A)
		cpu.PC++
	
	case 0x91: // STA (zp),Y
		cpu.PC++
		zp := uint16(cpu.Read(cpu.PC))
		lo := uint16(cpu.Read(zp))
		hi := uint16(cpu.Read((zp + 1) & 0xFF))
		addr := ((hi << 8) | lo) + uint16(cpu.Y)
		cpu.Write(addr, cpu.A)
		cpu.PC++

	case 0x86: // STX zp
		cpu.PC++
		addr := uint16(cpu.Read(cpu.PC))
		cpu.Write(addr, cpu.X)
		cpu.PC++
	
	case 0x96: // STX zp,Y
		cpu.PC++
		addr := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.Y)) & 0xFF
		cpu.Write(addr, cpu.X)
		cpu.PC++
	
	case 0x8E: // STX abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := (hi << 8) | lo
		cpu.Write(addr, cpu.X)
		cpu.PC++

	case 0x84: // STY zp
		cpu.PC++
		addr := uint16(cpu.Read(cpu.PC))
		cpu.Write(addr, cpu.Y)
		cpu.PC++
	
	case 0x94: // STY zp,X
		cpu.PC++
		addr := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		cpu.Write(addr, cpu.Y)
		cpu.PC++
	
	case 0x8C: // STY abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := (hi << 8) | lo
		cpu.Write(addr, cpu.Y)
		cpu.PC++

	// Branch Operations
	case 0x10: // BPL
		cpu.PC++
		offset := int8(cpu.Read(cpu.PC))
		cpu.PC++
		if cpu.GetFlag(N) == 0 {
			cpu.PC = uint16(int32(cpu.PC) + int32(offset))
		}
	
	case 0x30: // BMI
		cpu.PC++
		offset := int8(cpu.Read(cpu.PC))
		cpu.PC++
		if cpu.GetFlag(N) == 1 {
			cpu.PC = uint16(int32(cpu.PC) + int32(offset))
		}
	
	case 0x50: // BVC
		cpu.PC++
		offset := int8(cpu.Read(cpu.PC))
		cpu.PC++
		if cpu.GetFlag(V) == 0 {
			cpu.PC = uint16(int32(cpu.PC) + int32(offset))
		}
	
	case 0x70: // BVS
		cpu.PC++
		offset := int8(cpu.Read(cpu.PC))
		cpu.PC++
		if cpu.GetFlag(V) == 1 {
			cpu.PC = uint16(int32(cpu.PC) + int32(offset))
		}
	
	case 0x90: // BCC
		cpu.PC++
		offset := int8(cpu.Read(cpu.PC))
		cpu.PC++
		if cpu.GetFlag(C) == 0 {
			cpu.PC = uint16(int32(cpu.PC) + int32(offset))
		}
	
	case 0xB0: // BCS
		cpu.PC++
		offset := int8(cpu.Read(cpu.PC))
		cpu.PC++
		if cpu.GetFlag(C) == 1 {
			cpu.PC = uint16(int32(cpu.PC) + int32(offset))
		}
	
	case 0xD0: // BNE
		cpu.PC++
		offset := int8(cpu.Read(cpu.PC))
		cpu.PC++
		if cpu.GetFlag(Z) == 0 {
			cpu.PC = uint16(int32(cpu.PC) + int32(offset))
		}
	
	case 0xF0: // BEQ
		cpu.PC++
		offset := int8(cpu.Read(cpu.PC))
		cpu.PC++
		if cpu.GetFlag(Z) == 1 {
			cpu.PC = uint16(int32(cpu.PC) + int32(offset))
		}

	// Shift/Rotate Operations
	case 0x0A: // ASL A
		carry := (cpu.A & 0x80) != 0
		cpu.A <<= 1
		cpu.SetFlag(C, carry)
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0x4A: // LSR A
		carry := (cpu.A & 0x01) != 0
		cpu.A >>= 1
		cpu.SetFlag(C, carry)
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, false) // LSR always clears N flag
		cpu.PC++
	
	case 0x2A: // ROL A
		carry := (cpu.A & 0x80) != 0
		cpu.A <<= 1
		if cpu.GetFlag(C) == 1 {
			cpu.A |= 0x01
		}
		cpu.SetFlag(C, carry)
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	case 0x6A: // ROR A
		carry := (cpu.A & 0x01) != 0
		cpu.A >>= 1
		if cpu.GetFlag(C) == 1 {
			cpu.A |= 0x80
		}
		cpu.SetFlag(C, carry)
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
	
	// Shift/Rotate Operations - Zero Page
	case 0x06: // ASL zp
		cpu.PC++
		addr := uint16(cpu.Read(cpu.PC))
		value := cpu.Read(addr)
		carry := (value & 0x80) != 0
		result := value << 1
		cpu.Write(addr, result)
		cpu.SetFlag(C, carry)
		cpu.SetFlag(Z, result == 0x00)
		cpu.SetFlag(N, result&0x80 != 0)
		cpu.PC++
	
	case 0x46: // LSR zp
		cpu.PC++
		addr := uint16(cpu.Read(cpu.PC))
		value := cpu.Read(addr)
		carry := (value & 0x01) != 0
		result := value >> 1
		cpu.Write(addr, result)
		cpu.SetFlag(C, carry)
		cpu.SetFlag(Z, result == 0x00)
		cpu.SetFlag(N, false) // LSR always clears N flag
		cpu.PC++
	
	case 0x26: // ROL zp
		cpu.PC++
		addr := uint16(cpu.Read(cpu.PC))
		value := cpu.Read(addr)
		carry := (value & 0x80) != 0
		result := value << 1
		if cpu.GetFlag(C) == 1 {
			result |= 0x01
		}
		cpu.Write(addr, result)
		cpu.SetFlag(C, carry)
		cpu.SetFlag(Z, result == 0x00)
		cpu.SetFlag(N, result&0x80 != 0)
		cpu.PC++
	
	case 0x66: // ROR zp
		cpu.PC++
		addr := uint16(cpu.Read(cpu.PC))
		value := cpu.Read(addr)
		carry := (value & 0x01) != 0
		result := value >> 1
		if cpu.GetFlag(C) == 1 {
			result |= 0x80
		}
		cpu.Write(addr, result)
		cpu.SetFlag(C, carry)
		cpu.SetFlag(Z, result == 0x00)
		cpu.SetFlag(N, result&0x80 != 0)
		cpu.PC++

	// Shift/Rotate Operations - Absolute addressing
	case 0x0E: // ASL abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := (hi << 8) | lo
		value := cpu.Read(addr)
		carry := (value & 0x80) != 0
		result := value << 1
		cpu.Write(addr, result)
		cpu.SetFlag(C, carry)
		cpu.SetFlag(Z, result == 0x00)
		cpu.SetFlag(N, result&0x80 != 0)
		cpu.PC++
	
	case 0x4E: // LSR abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := (hi << 8) | lo
		value := cpu.Read(addr)
		carry := (value & 0x01) != 0
		result := value >> 1
		cpu.Write(addr, result)
		cpu.SetFlag(C, carry)
		cpu.SetFlag(Z, result == 0x00)
		cpu.SetFlag(N, false) // LSR always clears N flag
		cpu.PC++
	
	case 0x2E: // ROL abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := (hi << 8) | lo
		value := cpu.Read(addr)
		carry := (value & 0x80) != 0
		result := value << 1
		if cpu.GetFlag(C) == 1 {
			result |= 0x01
		}
		cpu.Write(addr, result)
		cpu.SetFlag(C, carry)
		cpu.SetFlag(Z, result == 0x00)
		cpu.SetFlag(N, result&0x80 != 0)
		cpu.PC++
	
	case 0x6E: // ROR abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := (hi << 8) | lo
		value := cpu.Read(addr)
		carry := (value & 0x01) != 0
		result := value >> 1
		if cpu.GetFlag(C) == 1 {
			result |= 0x80
		}
		cpu.Write(addr, result)
		cpu.SetFlag(C, carry)
		cpu.SetFlag(Z, result == 0x00)
		cpu.SetFlag(N, result&0x80 != 0)
		cpu.PC++

	// Shift/Rotate abs,X Instructions
	case 0x1E: // ASL abs,X
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := ((hi << 8) | lo) + uint16(cpu.X)
		value := cpu.Read(addr)
		carry := (value & 0x80) != 0
		result := value << 1
		cpu.Write(addr, result)
		cpu.SetFlag(C, carry)
		cpu.SetFlag(Z, result == 0x00)
		cpu.SetFlag(N, result&0x80 != 0)
		cpu.PC++
	
	case 0x5E: // LSR abs,X
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := ((hi << 8) | lo) + uint16(cpu.X)
		value := cpu.Read(addr)
		carry := (value & 0x01) != 0
		result := value >> 1
		cpu.Write(addr, result)
		cpu.SetFlag(C, carry)
		cpu.SetFlag(Z, result == 0x00)
		cpu.SetFlag(N, false) // LSR always clears N flag
		cpu.PC++
	
	case 0x3E: // ROL abs,X
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := ((hi << 8) | lo) + uint16(cpu.X)
		value := cpu.Read(addr)
		carry := (value & 0x80) != 0
		result := value << 1
		if cpu.GetFlag(C) == 1 {
			result |= 0x01
		}
		cpu.Write(addr, result)
		cpu.SetFlag(C, carry)
		cpu.SetFlag(Z, result == 0x00)
		cpu.SetFlag(N, result&0x80 != 0)
		cpu.PC++
	
	case 0x7E: // ROR abs,X
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := ((hi << 8) | lo) + uint16(cpu.X)
		value := cpu.Read(addr)
		carry := (value & 0x01) != 0
		result := value >> 1
		if cpu.GetFlag(C) == 1 {
			result |= 0x80
		}
		cpu.Write(addr, result)
		cpu.SetFlag(C, carry)
		cpu.SetFlag(Z, result == 0x00)
		cpu.SetFlag(N, result&0x80 != 0)
		cpu.PC++

	// Increment/Decrement abs,X Instructions
	case 0xFE: // INC abs,X
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := ((hi << 8) | lo) + uint16(cpu.X)
		value := cpu.Read(addr)
		result := value + 1
		cpu.Write(addr, result)
		cpu.SetFlag(Z, result == 0x00)
		cpu.SetFlag(N, result&0x80 != 0)
		cpu.PC++
	
	case 0xDE: // DEC abs,X
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := ((hi << 8) | lo) + uint16(cpu.X)
		value := cpu.Read(addr)
		result := value - 1
		cpu.Write(addr, result)
		cpu.SetFlag(Z, result == 0x00)
		cpu.SetFlag(N, result&0x80 != 0)
		cpu.PC++

	// Arithmetic Operations
	case 0x65: // ADC zp
		cpu.PC++
		addr := uint16(cpu.Read(cpu.PC))
		value := cpu.Read(addr)
		temp := uint16(cpu.A) + uint16(value) + uint16(cpu.GetFlag(C))
		cpu.SetFlag(C, temp > 255)
		cpu.SetFlag(Z, (temp&0x00FF) == 0)
		cpu.SetFlag(V, (^(uint16(cpu.A)^uint16(value))&(uint16(cpu.A)^temp))&0x0080 != 0)
		cpu.SetFlag(N, temp&0x80 != 0)
		cpu.A = uint8(temp & 0x00FF)
		cpu.PC++

	case 0x69: // ADC #
		cpu.PC++
		value := cpu.Read(cpu.PC)
		temp := uint16(cpu.A) + uint16(value) + uint16(cpu.GetFlag(C))
		cpu.SetFlag(C, temp > 255)
		cpu.SetFlag(Z, (temp&0x00FF) == 0)
		cpu.SetFlag(V, ((cpu.A^uint8(temp))&(value^uint8(temp))&0x80) != 0)
		cpu.SetFlag(N, temp&0x80 != 0)
		cpu.A = uint8(temp & 0x00FF)
		cpu.PC++
	
	case 0x6D: // ADC abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := (hi << 8) | lo
		value := cpu.Read(addr)
		temp := uint16(cpu.A) + uint16(value) + uint16(cpu.GetFlag(C))
		cpu.SetFlag(C, temp > 255)
		cpu.SetFlag(Z, (temp&0x00FF) == 0)
		cpu.SetFlag(V, ((cpu.A^uint8(temp))&(value^uint8(temp))&0x80) != 0)
		cpu.SetFlag(N, temp&0x80 != 0)
		cpu.A = uint8(temp & 0x00FF)
		cpu.PC++
	
	case 0x7D: // ADC abs,X
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		baseAddr := (hi << 8) | lo
		addr := cpu.readIndexedAddr(baseAddr, cpu.X)
		value := cpu.Read(addr)
		temp := uint16(cpu.A) + uint16(value) + uint16(cpu.GetFlag(C))
		cpu.SetFlag(C, temp > 255)
		cpu.SetFlag(Z, (temp&0x00FF) == 0)
		cpu.SetFlag(V, ((cpu.A^uint8(temp))&(value^uint8(temp))&0x80) != 0)
		cpu.SetFlag(N, temp&0x80 != 0)
		cpu.A = uint8(temp & 0x00FF)
		cpu.PC++
	
	case 0x79: // ADC abs,Y
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		baseAddr := (hi << 8) | lo
		addr := cpu.readIndexedAddr(baseAddr, cpu.Y)
		value := cpu.Read(addr)
		temp := uint16(cpu.A) + uint16(value) + uint16(cpu.GetFlag(C))
		cpu.SetFlag(C, temp > 255)
		cpu.SetFlag(Z, (temp&0x00FF) == 0)
		cpu.SetFlag(V, ((cpu.A^uint8(temp))&(value^uint8(temp))&0x80) != 0)
		cpu.SetFlag(N, temp&0x80 != 0)
		cpu.A = uint8(temp & 0x00FF)
		cpu.PC++
	
	case 0xE9: // SBC #
		cpu.PC++
		value := cpu.Read(cpu.PC)
		// SBC is implemented as A = A + (~M) + C
		invValue := uint16(value) ^ 0x00FF
		temp := uint16(cpu.A) + invValue + uint16(cpu.GetFlag(C))
		cpu.SetFlag(C, temp&0xFF00 != 0)
		cpu.SetFlag(Z, (temp&0x00FF) == 0)
		// Correct V flag calculation for SBC: V = (A7 ^ R7) & (~M7 ^ R7)
		result := uint8(temp & 0x00FF)
		cpu.SetFlag(V, ((cpu.A^result)&((value^0xFF)^result)&0x80) != 0)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.A = result
		cpu.PC++
	
	case 0xEB: // *SBC # (unofficial)
		cpu.PC++
		value := cpu.Read(cpu.PC)
		// SBC is implemented as A = A + (~M) + C
		invValue := uint16(value) ^ 0x00FF
		temp := uint16(cpu.A) + invValue + uint16(cpu.GetFlag(C))
		cpu.SetFlag(C, temp&0xFF00 != 0)
		cpu.SetFlag(Z, (temp&0x00FF) == 0)
		// Correct V flag calculation for SBC: V = (A7 ^ R7) & (~M7 ^ R7)
		result := uint8(temp & 0x00FF)
		cpu.SetFlag(V, ((cpu.A^result)&((value^0xFF)^result)&0x80) != 0)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.A = result
		cpu.PC++
	
	case 0xE1: // SBC ($zp,X)
		cpu.PC++
		zpAddr := uint16(cpu.Read(cpu.PC))
		indirectAddr := (zpAddr + uint16(cpu.X)) & 0xFF
		lo := uint16(cpu.Read(indirectAddr))
		hi := uint16(cpu.Read((indirectAddr + 1) & 0xFF))
		addr := (hi << 8) | lo
		value := cpu.Read(addr)
		// SBC is implemented as A = A + (~M) + C
		invValue := uint16(value) ^ 0x00FF
		temp := uint16(cpu.A) + invValue + uint16(cpu.GetFlag(C))
		cpu.SetFlag(C, temp&0xFF00 != 0)
		cpu.SetFlag(Z, (temp&0x00FF) == 0)
		// Correct V flag calculation for SBC: V = (A7 ^ R7) & (~M7 ^ R7)
		result := uint8(temp & 0x00FF)
		cpu.SetFlag(V, ((cpu.A^result)&((value^0xFF)^result)&0x80) != 0)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.A = result
		cpu.PC++

	case 0xF1: // SBC ($zp),Y
		cpu.PC++
		zp := uint16(cpu.Read(cpu.PC))
		lo := uint16(cpu.Read(zp))
		hi := uint16(cpu.Read((zp + 1) & 0xFF))
		effectiveAddr := cpu.readIndexedAddr((hi<<8)|lo, cpu.Y)
		value := cpu.Read(effectiveAddr)
		// SBC is implemented as A = A + (~M) + C
		invValue := uint16(value) ^ 0x00FF
		temp := uint16(cpu.A) + invValue + uint16(cpu.GetFlag(C))
		cpu.SetFlag(C, temp&0xFF00 != 0)
		cpu.SetFlag(Z, (temp&0x00FF) == 0)
		// Correct V flag calculation for SBC: V = (A7 ^ R7) & (~M7 ^ R7)
		result := uint8(temp & 0x00FF)
		cpu.SetFlag(V, ((cpu.A^result)&((value^0xFF)^result)&0x80) != 0)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.A = result
		cpu.PC++
	
	case 0xE5: // SBC zp
		cpu.PC++
		addr := uint16(cpu.Read(cpu.PC))
		value := cpu.Read(addr)
		// SBC is implemented as A = A + (~M) + C
		invValue := uint16(value) ^ 0x00FF
		temp := uint16(cpu.A) + invValue + uint16(cpu.GetFlag(C))
		cpu.SetFlag(C, temp&0xFF00 != 0)
		cpu.SetFlag(Z, (temp&0x00FF) == 0)
		// Correct V flag calculation for SBC: V = (A7 ^ R7) & (~M7 ^ R7)
		result := uint8(temp & 0x00FF)
		cpu.SetFlag(V, ((cpu.A^result)&((value^0xFF)^result)&0x80) != 0)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.A = result
		cpu.PC++

	case 0x75: // ADC zp,X
		cpu.PC++
		addr := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		value := cpu.Read(addr)
		temp := uint16(cpu.A) + uint16(value) + uint16(cpu.GetFlag(C))
		cpu.SetFlag(C, temp > 255)
		cpu.SetFlag(Z, (temp&0x00FF) == 0)
		cpu.SetFlag(V, (^(uint16(cpu.A)^uint16(value))&(uint16(cpu.A)^temp))&0x0080 != 0)
		cpu.SetFlag(N, temp&0x80 != 0)
		cpu.A = uint8(temp & 0x00FF)
		cpu.PC++

	case 0xED: // SBC abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := (hi << 8) | lo
		value := cpu.Read(addr)
		// SBC is implemented as A = A + (~M) + C
		invValue := uint16(value) ^ 0x00FF
		temp := uint16(cpu.A) + invValue + uint16(cpu.GetFlag(C))
		cpu.SetFlag(C, temp&0xFF00 != 0)
		cpu.SetFlag(Z, (temp&0x00FF) == 0)
		// Correct V flag calculation for SBC: V = (A7 ^ R7) & (~M7 ^ R7)
		result := uint8(temp & 0x00FF)
		cpu.SetFlag(V, ((cpu.A^result)&((value^0xFF)^result)&0x80) != 0)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.A = result
		cpu.PC++
	
	case 0xF5: // SBC zp,X
		cpu.PC++
		addr := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		value := cpu.Read(addr)
		// SBC is implemented as A = A + (~M) + C
		invValue := uint16(value) ^ 0x00FF
		temp := uint16(cpu.A) + invValue + uint16(cpu.GetFlag(C))
		cpu.SetFlag(C, temp&0xFF00 != 0)
		cpu.SetFlag(Z, (temp&0x00FF) == 0)
		// Correct V flag calculation for SBC: V = (A7 ^ R7) & (~M7 ^ R7)
		result := uint8(temp & 0x00FF)
		cpu.SetFlag(V, ((cpu.A^result)&((value^0xFF)^result)&0x80) != 0)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.A = result
		cpu.PC++

	case 0xFD: // SBC abs,X
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		baseAddr := (hi << 8) | lo
		addr := cpu.readIndexedAddr(baseAddr, cpu.X)
		value := cpu.Read(addr)
		// SBC is implemented as A = A + (~M) + C
		invValue := uint16(value) ^ 0x00FF
		temp := uint16(cpu.A) + invValue + uint16(cpu.GetFlag(C))
		cpu.SetFlag(C, temp&0xFF00 != 0)
		cpu.SetFlag(Z, (temp&0x00FF) == 0)
		// Correct V flag calculation for SBC: V = (A7 ^ R7) & (~M7 ^ R7)
		result := uint8(temp & 0x00FF)
		cpu.SetFlag(V, ((cpu.A^result)&((value^0xFF)^result)&0x80) != 0)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.A = result
		cpu.PC++

	case 0xF9: // SBC abs,Y
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		baseAddr := (hi << 8) | lo
		addr := cpu.readIndexedAddr(baseAddr, cpu.Y)
		value := cpu.Read(addr)
		// SBC is implemented as A = A + (~M) + C
		invValue := uint16(value) ^ 0x00FF
		temp := uint16(cpu.A) + invValue + uint16(cpu.GetFlag(C))
		cpu.SetFlag(C, temp&0xFF00 != 0)
		cpu.SetFlag(Z, (temp&0x00FF) == 0)
		// Correct V flag calculation for SBC: V = (A7 ^ R7) & (~M7 ^ R7)
		result := uint8(temp & 0x00FF)
		cpu.SetFlag(V, ((cpu.A^result)&((value^0xFF)^result)&0x80) != 0)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.A = result
		cpu.PC++

	// Increment/Decrement Operations
	case 0xE6: // INC zp
		cpu.PC++
		addr := uint16(cpu.Read(cpu.PC))
		value := cpu.Read(addr)
		result := value + 1
		cpu.Write(addr, result)
		cpu.SetFlag(Z, result == 0x00)
		cpu.SetFlag(N, result&0x80 != 0)
		cpu.PC++
	
	case 0xC6: // DEC zp
		cpu.PC++
		addr := uint16(cpu.Read(cpu.PC))
		value := cpu.Read(addr)
		result := value - 1
		cpu.Write(addr, result)
		cpu.SetFlag(Z, result == 0x00)
		cpu.SetFlag(N, result&0x80 != 0)
		cpu.PC++

	// Increment/Decrement Operations - Absolute addressing
	case 0xEE: // INC abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := (hi << 8) | lo
		value := cpu.Read(addr)
		result := value + 1
		cpu.Write(addr, result)
		cpu.SetFlag(Z, result == 0x00)
		cpu.SetFlag(N, result&0x80 != 0)
		cpu.PC++
	
	case 0xCE: // DEC abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := (hi << 8) | lo
		value := cpu.Read(addr)
		result := value - 1
		cpu.Write(addr, result)
		cpu.SetFlag(Z, result == 0x00)
		cpu.SetFlag(N, result&0x80 != 0)
		cpu.PC++

	// Unofficial/Illegal Instructions
	case 0x33: // RLA ($zp),Y - Unofficial instruction: ROL memory then AND with A
		cpu.PC++
		zpAddr := uint16(cpu.Read(cpu.PC))
		lo := uint16(cpu.Read(zpAddr))
		hi := uint16(cpu.Read((zpAddr + 1) & 0xFF))
		baseAddr := (hi << 8) | lo
		effectiveAddr := baseAddr + uint16(cpu.Y)
		
		// First: ROL memory
		value := cpu.Read(effectiveAddr)
		carry := (value & 0x80) != 0
		result := value << 1
		if cpu.GetFlag(C) == 1 {
			result |= 0x01
		}
		cpu.Write(effectiveAddr, result)
		cpu.SetFlag(C, carry)
		
		// Second: AND result with accumulator
		cpu.A = cpu.A & result
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++

	// Shift Operations - zp,X versions
	case 0x16: // ASL zp,X
		cpu.PC++
		zpAddr := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		value := cpu.Read(zpAddr)
		carry := (value & 0x80) != 0
		result := value << 1
		cpu.Write(zpAddr, result)
		cpu.SetFlag(C, carry)
		cpu.SetFlag(Z, result == 0x00)
		cpu.SetFlag(N, result&0x80 != 0)
		cpu.PC++

	case 0x56: // LSR zp,X
		cpu.PC++
		zpAddr := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		value := cpu.Read(zpAddr)
		carry := (value & 0x01) != 0
		result := value >> 1
		cpu.Write(zpAddr, result)
		cpu.SetFlag(C, carry)
		cpu.SetFlag(Z, result == 0x00)
		cpu.SetFlag(N, false) // LSR always clears N flag
		cpu.PC++

	case 0x36: // ROL zp,X
		cpu.PC++
		zpAddr := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		value := cpu.Read(zpAddr)
		carry := (value & 0x80) != 0
		result := value << 1
		if cpu.GetFlag(C) == 1 {
			result |= 0x01
		}
		cpu.Write(zpAddr, result)
		cpu.SetFlag(C, carry)
		cpu.SetFlag(Z, result == 0x00)
		cpu.SetFlag(N, result&0x80 != 0)
		cpu.PC++

	case 0x76: // ROR zp,X
		cpu.PC++
		zpAddr := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		value := cpu.Read(zpAddr)
		carry := (value & 0x01) != 0
		result := value >> 1
		if cpu.GetFlag(C) == 1 {
			result |= 0x80
		}
		cpu.Write(zpAddr, result)
		cpu.SetFlag(C, carry)
		cpu.SetFlag(Z, result == 0x00)
		cpu.SetFlag(N, result&0x80 != 0)
		cpu.PC++

	// INC/DEC Operations - zp,X versions
	case 0xF6: // INC zp,X
		cpu.PC++
		zpAddr := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		value := cpu.Read(zpAddr)
		result := value + 1
		cpu.Write(zpAddr, result)
		cpu.SetFlag(Z, result == 0x00)
		cpu.SetFlag(N, result&0x80 != 0)
		// INC does not affect carry flag
		cpu.PC++

	case 0xD6: // DEC zp,X
		cpu.PC++
		zpAddr := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		value := cpu.Read(zpAddr)
		result := value - 1
		cpu.Write(zpAddr, result)
		cpu.SetFlag(Z, result == 0x00)
		cpu.SetFlag(N, result&0x80 != 0)
		// DEC does not affect carry flag
		cpu.PC++

	// SAX (STA and STX: A & X) - Unofficial instruction
	case 0x83: // SAX ($zp,X) - Store A AND X to memory (indirect indexed)
		cpu.PC++
		zpBase := uint16(cpu.Read(cpu.PC))
		zpAddr := (zpBase + uint16(cpu.X)) & 0xFF
		lo := uint16(cpu.Read(zpAddr))
		hi := uint16(cpu.Read((zpAddr + 1) & 0xFF))
		effectiveAddr := (hi << 8) | lo
		value := cpu.A & cpu.X  // SAX stores A AND X
		cpu.Write(effectiveAddr, value)
		cpu.PC++

	case 0x87: // SAX $zp - Store A AND X to memory (zero page)
		cpu.PC++
		addr := uint16(cpu.Read(cpu.PC))
		value := cpu.A & cpu.X  // SAX stores A AND X
		cpu.Write(addr, value)
		cpu.PC++

	case 0x8F: // SAX $abs - Store A AND X to memory (absolute)
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := (hi << 8) | lo
		value := cpu.A & cpu.X  // SAX stores A AND X
		cpu.Write(addr, value)
		cpu.PC++

	case 0x97: // SAX $zp,Y - Store A AND X to memory (zero page indexed by Y)
		cpu.PC++
		baseAddr := uint16(cpu.Read(cpu.PC))
		addr := (baseAddr + uint16(cpu.Y)) & 0xFF
		value := cpu.A & cpu.X  // SAX stores A AND X
		cpu.Write(addr, value)
		cpu.PC++

	// LAX (LDA + LDX) - Unofficial instruction variants
	case 0xA3: // LAX ($zp,X) - Load A and X with memory (indirect indexed)
		cpu.PC++
		zpBase := uint16(cpu.Read(cpu.PC))
		zpAddr := (zpBase + uint16(cpu.X)) & 0xFF
		lo := uint16(cpu.Read(zpAddr))
		hi := uint16(cpu.Read((zpAddr + 1) & 0xFF))
		effectiveAddr := (hi << 8) | lo
		value := cpu.Read(effectiveAddr)
		cpu.A = value
		cpu.X = value
		cpu.SetFlag(Z, value == 0x00)
		cpu.SetFlag(N, value&0x80 != 0)
		cpu.PC++

	case 0xA7: // LAX $zp - Load A and X with memory (zero page)
		cpu.PC++
		addr := uint16(cpu.Read(cpu.PC))
		value := cpu.Read(addr)
		cpu.A = value
		cpu.X = value
		cpu.SetFlag(Z, value == 0x00)
		cpu.SetFlag(N, value&0x80 != 0)
		cpu.PC++

	case 0xAF: // LAX $abs - Load A and X with memory (absolute)
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := (hi << 8) | lo
		value := cpu.Read(addr)
		cpu.A = value
		cpu.X = value
		cpu.SetFlag(Z, value == 0x00)
		cpu.SetFlag(N, value&0x80 != 0)
		cpu.PC++

	case 0xB3: // LAX ($zp),Y - Load A and X with memory (indirect indexed Y)
		cpu.PC++
		zpAddr := uint16(cpu.Read(cpu.PC))
		lo := uint16(cpu.Read(zpAddr))
		hi := uint16(cpu.Read((zpAddr + 1) & 0xFF))
		effectiveAddr := cpu.readIndexedAddr((hi<<8)|lo, cpu.Y)
		value := cpu.Read(effectiveAddr)
		cpu.A = value
		cpu.X = value
		cpu.SetFlag(Z, value == 0x00)
		cpu.SetFlag(N, value&0x80 != 0)
		cpu.PC++

	case 0xB7: // LAX $zp,Y - Load A and X with memory (zero page Y)
		cpu.PC++
		baseAddr := uint16(cpu.Read(cpu.PC))
		addr := (baseAddr + uint16(cpu.Y)) & 0xFF
		value := cpu.Read(addr)
		cpu.A = value
		cpu.X = value
		cpu.SetFlag(Z, value == 0x00)
		cpu.SetFlag(N, value&0x80 != 0)
		cpu.PC++

	case 0xBF: // LAX $abs,Y - Load A and X with memory (absolute Y)
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		effectiveAddr := cpu.readIndexedAddr((hi<<8)|lo, cpu.Y)
		value := cpu.Read(effectiveAddr)
		cpu.A = value
		cpu.X = value
		cpu.SetFlag(Z, value == 0x00)
		cpu.SetFlag(N, value&0x80 != 0)
		cpu.PC++

	// NOP
	case 0xEA: // NOP
		cpu.PC++

	// DCP (Decrement and Compare) - Unofficial instruction
	case 0xC3: // DCP ($zp,X)
		cpu.PC++
		zpAddr := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		effectiveAddr := uint16(cpu.Read(zpAddr)) | (uint16(cpu.Read((zpAddr+1)&0xFF)) << 8)
		value := cpu.Read(effectiveAddr)
		value--
		cpu.Write(effectiveAddr, value)
		// Compare A with decremented value
		result := cpu.A - value
		cpu.SetFlag(C, cpu.A >= value)
		cpu.SetFlag(Z, result == 0)
		cpu.SetFlag(N, result&0x80 != 0)
		cpu.PC++

	case 0xC7: // DCP $zp
		cpu.PC++
		addr := uint16(cpu.Read(cpu.PC))
		value := cpu.Read(addr)
		value--
		cpu.Write(addr, value)
		// Compare A with decremented value
		result := cpu.A - value
		cpu.SetFlag(C, cpu.A >= value)
		cpu.SetFlag(Z, result == 0)
		cpu.SetFlag(N, result&0x80 != 0)
		cpu.PC++

	case 0xCF: // DCP $abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := (hi << 8) | lo
		value := cpu.Read(addr)
		value--
		cpu.Write(addr, value)
		// Compare A with decremented value
		result := cpu.A - value
		cpu.SetFlag(C, cpu.A >= value)
		cpu.SetFlag(Z, result == 0)
		cpu.SetFlag(N, result&0x80 != 0)
		cpu.PC++

	case 0xD3: // DCP ($zp),Y
		cpu.PC++
		zpAddr := uint16(cpu.Read(cpu.PC))
		baseAddr := uint16(cpu.Read(zpAddr)) | (uint16(cpu.Read((zpAddr+1)&0xFF)) << 8)
		effectiveAddr := baseAddr + uint16(cpu.Y)
		value := cpu.Read(effectiveAddr)
		value--
		cpu.Write(effectiveAddr, value)
		// Compare A with decremented value
		result := cpu.A - value
		cpu.SetFlag(C, cpu.A >= value)
		cpu.SetFlag(Z, result == 0)
		cpu.SetFlag(N, result&0x80 != 0)
		cpu.PC++

	case 0xD7: // DCP $zp,X
		cpu.PC++
		addr := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		value := cpu.Read(addr)
		value--
		cpu.Write(addr, value)
		// Compare A with decremented value
		result := cpu.A - value
		cpu.SetFlag(C, cpu.A >= value)
		cpu.SetFlag(Z, result == 0)
		cpu.SetFlag(N, result&0x80 != 0)
		cpu.PC++

	case 0xDB: // DCP $abs,Y
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		baseAddr := (hi << 8) | lo
		effectiveAddr := baseAddr + uint16(cpu.Y)
		value := cpu.Read(effectiveAddr)
		value--
		cpu.Write(effectiveAddr, value)
		// Compare A with decremented value
		result := cpu.A - value
		cpu.SetFlag(C, cpu.A >= value)
		cpu.SetFlag(Z, result == 0)
		cpu.SetFlag(N, result&0x80 != 0)
		cpu.PC++

	case 0xDF: // DCP $abs,X
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		baseAddr := (hi << 8) | lo
		effectiveAddr := baseAddr + uint16(cpu.X)
		value := cpu.Read(effectiveAddr)
		value--
		cpu.Write(effectiveAddr, value)
		// Compare A with decremented value
		result := cpu.A - value
		cpu.SetFlag(C, cpu.A >= value)
		cpu.SetFlag(Z, result == 0)
		cpu.SetFlag(N, result&0x80 != 0)
		cpu.PC++

	// ISB (Increment and Subtract with Borrow) - Unofficial instruction
	case 0xE3: // ISB ($zp,X)
		cpu.PC++
		zpAddr := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		effectiveAddr := uint16(cpu.Read(zpAddr)) | (uint16(cpu.Read((zpAddr+1)&0xFF)) << 8)
		value := cpu.Read(effectiveAddr)
		value++ // First: Increment memory
		cpu.Write(effectiveAddr, value)
		// Second: Subtract with Borrow (SBC)
		invValue := uint16(value) ^ 0x00FF
		temp := uint16(cpu.A) + invValue + uint16(cpu.GetFlag(C))
		cpu.SetFlag(C, temp&0xFF00 != 0)
		cpu.SetFlag(Z, (temp&0x00FF) == 0)
		result := uint8(temp & 0x00FF)
		cpu.SetFlag(V, ((cpu.A^result)&((value^0xFF)^result)&0x80) != 0)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.A = result
		cpu.PC++

	case 0xE7: // ISB $zp
		cpu.PC++
		addr := uint16(cpu.Read(cpu.PC))
		value := cpu.Read(addr)
		value++ // First: Increment memory
		cpu.Write(addr, value)
		// Second: Subtract with Borrow (SBC)
		invValue := uint16(value) ^ 0x00FF
		temp := uint16(cpu.A) + invValue + uint16(cpu.GetFlag(C))
		cpu.SetFlag(C, temp&0xFF00 != 0)
		cpu.SetFlag(Z, (temp&0x00FF) == 0)
		result := uint8(temp & 0x00FF)
		cpu.SetFlag(V, ((cpu.A^result)&((value^0xFF)^result)&0x80) != 0)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.A = result
		cpu.PC++

	case 0xEF: // ISB $abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := (hi << 8) | lo
		value := cpu.Read(addr)
		value++ // First: Increment memory
		cpu.Write(addr, value)
		// Second: Subtract with Borrow (SBC)
		invValue := uint16(value) ^ 0x00FF
		temp := uint16(cpu.A) + invValue + uint16(cpu.GetFlag(C))
		cpu.SetFlag(C, temp&0xFF00 != 0)
		cpu.SetFlag(Z, (temp&0x00FF) == 0)
		result := uint8(temp & 0x00FF)
		cpu.SetFlag(V, ((cpu.A^result)&((value^0xFF)^result)&0x80) != 0)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.A = result
		cpu.PC++

	case 0xF3: // ISB ($zp),Y
		cpu.PC++
		zpAddr := uint16(cpu.Read(cpu.PC))
		baseAddr := uint16(cpu.Read(zpAddr)) | (uint16(cpu.Read((zpAddr+1)&0xFF)) << 8)
		effectiveAddr := baseAddr + uint16(cpu.Y)
		value := cpu.Read(effectiveAddr)
		value++ // First: Increment memory
		cpu.Write(effectiveAddr, value)
		// Second: Subtract with Borrow (SBC)
		invValue := uint16(value) ^ 0x00FF
		temp := uint16(cpu.A) + invValue + uint16(cpu.GetFlag(C))
		cpu.SetFlag(C, temp&0xFF00 != 0)
		cpu.SetFlag(Z, (temp&0x00FF) == 0)
		result := uint8(temp & 0x00FF)
		cpu.SetFlag(V, ((cpu.A^result)&((value^0xFF)^result)&0x80) != 0)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.A = result
		cpu.PC++

	case 0xF7: // ISB $zp,X
		cpu.PC++
		addr := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		value := cpu.Read(addr)
		value++ // First: Increment memory
		cpu.Write(addr, value)
		// Second: Subtract with Borrow (SBC)
		invValue := uint16(value) ^ 0x00FF
		temp := uint16(cpu.A) + invValue + uint16(cpu.GetFlag(C))
		cpu.SetFlag(C, temp&0xFF00 != 0)
		cpu.SetFlag(Z, (temp&0x00FF) == 0)
		result := uint8(temp & 0x00FF)
		cpu.SetFlag(V, ((cpu.A^result)&((value^0xFF)^result)&0x80) != 0)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.A = result
		cpu.PC++

	case 0xFB: // ISB $abs,Y
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		baseAddr := (hi << 8) | lo
		effectiveAddr := baseAddr + uint16(cpu.Y)
		value := cpu.Read(effectiveAddr)
		value++ // First: Increment memory
		cpu.Write(effectiveAddr, value)
		// Second: Subtract with Borrow (SBC)
		invValue := uint16(value) ^ 0x00FF
		temp := uint16(cpu.A) + invValue + uint16(cpu.GetFlag(C))
		cpu.SetFlag(C, temp&0xFF00 != 0)
		cpu.SetFlag(Z, (temp&0x00FF) == 0)
		result := uint8(temp & 0x00FF)
		cpu.SetFlag(V, ((cpu.A^result)&((value^0xFF)^result)&0x80) != 0)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.A = result
		cpu.PC++

	case 0xFF: // ISB $abs,X
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		baseAddr := (hi << 8) | lo
		effectiveAddr := baseAddr + uint16(cpu.X)
		value := cpu.Read(effectiveAddr)
		value++ // First: Increment memory
		cpu.Write(effectiveAddr, value)
		// Second: Subtract with Borrow (SBC)
		invValue := uint16(value) ^ 0x00FF
		temp := uint16(cpu.A) + invValue + uint16(cpu.GetFlag(C))
		cpu.SetFlag(C, temp&0xFF00 != 0)
		cpu.SetFlag(Z, (temp&0x00FF) == 0)
		result := uint8(temp & 0x00FF)
		cpu.SetFlag(V, ((cpu.A^result)&((value^0xFF)^result)&0x80) != 0)
		cpu.SetFlag(N, temp&0x0080 != 0)
		cpu.A = result
		cpu.PC++

	// SLO (Shift Left and OR) - Unofficial instruction
	case 0x03: // SLO ($zp,X)
		cpu.PC++
		zpAddr := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		effectiveAddr := uint16(cpu.Read(zpAddr)) | (uint16(cpu.Read((zpAddr+1)&0xFF)) << 8)
		value := cpu.Read(effectiveAddr)
		// First: Shift Left (ASL)
		carry := (value & 0x80) != 0
		result := value << 1
		cpu.Write(effectiveAddr, result)
		cpu.SetFlag(C, carry)
		// Second: OR with accumulator
		cpu.A = cpu.A | result
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++

	case 0x07: // SLO $zp
		cpu.PC++
		addr := uint16(cpu.Read(cpu.PC))
		value := cpu.Read(addr)
		// First: Shift Left (ASL)
		carry := (value & 0x80) != 0
		result := value << 1
		cpu.Write(addr, result)
		cpu.SetFlag(C, carry)
		// Second: OR with accumulator
		cpu.A = cpu.A | result
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++

	case 0x0F: // SLO $abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		addr := (hi << 8) | lo
		value := cpu.Read(addr)
		// First: Shift Left (ASL)
		carry := (value & 0x80) != 0
		result := value << 1
		cpu.Write(addr, result)
		cpu.SetFlag(C, carry)
		// Second: OR with accumulator
		cpu.A = cpu.A | result
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++

	case 0x13: // SLO ($zp),Y
		cpu.PC++
		zpAddr := uint16(cpu.Read(cpu.PC))
		baseAddr := uint16(cpu.Read(zpAddr)) | (uint16(cpu.Read((zpAddr+1)&0xFF)) << 8)
		effectiveAddr := baseAddr + uint16(cpu.Y)
		value := cpu.Read(effectiveAddr)
		// First: Shift Left (ASL)
		carry := (value & 0x80) != 0
		result := value << 1
		cpu.Write(effectiveAddr, result)
		cpu.SetFlag(C, carry)
		// Second: OR with accumulator
		cpu.A = cpu.A | result
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++

	case 0x17: // SLO $zp,X
		cpu.PC++
		addr := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		value := cpu.Read(addr)
		// First: Shift Left (ASL)
		carry := (value & 0x80) != 0
		result := value << 1
		cpu.Write(addr, result)
		cpu.SetFlag(C, carry)
		// Second: OR with accumulator
		cpu.A = cpu.A | result
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++

	case 0x1B: // SLO $abs,Y
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		baseAddr := (hi << 8) | lo
		effectiveAddr := baseAddr + uint16(cpu.Y)
		value := cpu.Read(effectiveAddr)
		// First: Shift Left (ASL)
		carry := (value & 0x80) != 0
		result := value << 1
		cpu.Write(effectiveAddr, result)
		cpu.SetFlag(C, carry)
		// Second: OR with accumulator
		cpu.A = cpu.A | result
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++

	case 0x1F: // SLO $abs,X
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		baseAddr := (hi << 8) | lo
		effectiveAddr := baseAddr + uint16(cpu.X)
		value := cpu.Read(effectiveAddr)
		// First: Shift Left (ASL)
		carry := (value & 0x80) != 0
		result := value << 1
		cpu.Write(effectiveAddr, result)
		cpu.SetFlag(C, carry)
		// Second: OR with accumulator
		cpu.A = cpu.A | result
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++

	// RLA (ROL + AND) - Unofficial instructions
	case 0x23: // RLA ($zp,X)
		zpAddr := (uint16(cpu.Read(cpu.PC+1)) + uint16(cpu.X)) & 0xFF
		lo := uint16(cpu.Read(zpAddr))
		hi := uint16(cpu.Read((zpAddr + 1) & 0xFF))
		effectiveAddr := (hi << 8) | lo
		value := cpu.Read(effectiveAddr)
		// First: Rotate Left (ROL)
		carry := (value & 0x80) != 0
		result := (value << 1) | (cpu.Status&0x01)
		cpu.Write(effectiveAddr, result)
		cpu.SetFlag(C, carry)
		// Second: AND with accumulator
		cpu.A = cpu.A & result
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++
		cpu.PC++

	case 0x27: // RLA $zp
		cpu.PC++
		zpAddr := uint16(cpu.Read(cpu.PC))
		value := cpu.Read(zpAddr)
		// First: Rotate Left (ROL)
		carry := (value & 0x80) != 0
		result := (value << 1) | (cpu.Status&0x01)
		cpu.Write(zpAddr, result)
		cpu.SetFlag(C, carry)
		// Second: AND with accumulator
		cpu.A = cpu.A & result
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++

	case 0x2F: // RLA $abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		absAddr := (hi << 8) | lo
		value := cpu.Read(absAddr)
		// First: Rotate Left (ROL)
		carry := (value & 0x80) != 0
		result := (value << 1) | (cpu.Status&0x01)
		cpu.Write(absAddr, result)
		cpu.SetFlag(C, carry)
		// Second: AND with accumulator
		cpu.A = cpu.A & result
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)

	case 0x37: // RLA $zp,X
		cpu.PC++
		baseAddr := uint16(cpu.Read(cpu.PC))
		zpAddr := (baseAddr + uint16(cpu.X)) & 0xFF
		value := cpu.Read(zpAddr)
		// First: Rotate Left (ROL)
		carry := (value & 0x80) != 0
		result := (value << 1) | (cpu.Status&0x01)
		cpu.Write(zpAddr, result)
		cpu.SetFlag(C, carry)
		// Second: AND with accumulator
		cpu.A = cpu.A & result
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)
		cpu.PC++

	case 0x3B: // RLA $abs,Y
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		baseAddr := (hi << 8) | lo
		effectiveAddr := baseAddr + uint16(cpu.Y)
		value := cpu.Read(effectiveAddr)
		// First: Rotate Left (ROL)
		carry := (value & 0x80) != 0
		result := (value << 1) | (cpu.Status&0x01)
		cpu.Write(effectiveAddr, result)
		cpu.SetFlag(C, carry)
		// Second: AND with accumulator
		cpu.A = cpu.A & result
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)

	case 0x3F: // RLA $abs,X
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		baseAddr := (hi << 8) | lo
		effectiveAddr := baseAddr + uint16(cpu.X)
		value := cpu.Read(effectiveAddr)
		// First: Rotate Left (ROL)
		carry := (value & 0x80) != 0
		result := (value << 1) | (cpu.Status&0x01)
		cpu.Write(effectiveAddr, result)
		cpu.SetFlag(C, carry)
		// Second: AND with accumulator
		cpu.A = cpu.A & result
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)

	// SRE (Shift Right and EOR) - Unofficial instruction
	case 0x43: // SRE ($zp,X)
		cpu.PC++
		zpAddr := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		cpu.PC++
		effectiveAddr := uint16(cpu.Read(zpAddr)) | (uint16(cpu.Read((zpAddr+1)&0xFF)) << 8)
		value := cpu.Read(effectiveAddr)
		// First: Shift Right (LSR)
		carry := (value & 0x01) != 0
		result := value >> 1
		cpu.Write(effectiveAddr, result)
		cpu.SetFlag(C, carry)
		// Second: EOR with accumulator
		cpu.A = cpu.A ^ result
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)

	case 0x47: // SRE $zp
		cpu.PC++
		zpAddr := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		value := cpu.Read(zpAddr)
		// First: Shift Right (LSR)
		carry := (value & 0x01) != 0
		result := value >> 1
		cpu.Write(zpAddr, result)
		cpu.SetFlag(C, carry)
		// Second: EOR with accumulator
		cpu.A = cpu.A ^ result
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)

	case 0x4F: // SRE $abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		effectiveAddr := (hi << 8) | lo
		value := cpu.Read(effectiveAddr)
		// First: Shift Right (LSR)
		carry := (value & 0x01) != 0
		result := value >> 1
		cpu.Write(effectiveAddr, result)
		cpu.SetFlag(C, carry)
		// Second: EOR with accumulator
		cpu.A = cpu.A ^ result
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)

	case 0x53: // SRE ($zp),Y
		cpu.PC++
		zpAddr := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		lo := uint16(cpu.Read(zpAddr))
		hi := uint16(cpu.Read((zpAddr + 1) & 0xFF))
		baseAddr := (hi << 8) | lo
		effectiveAddr := baseAddr + uint16(cpu.Y)
		value := cpu.Read(effectiveAddr)
		// First: Shift Right (LSR)
		carry := (value & 0x01) != 0
		result := value >> 1
		cpu.Write(effectiveAddr, result)
		cpu.SetFlag(C, carry)
		// Second: EOR with accumulator
		cpu.A = cpu.A ^ result
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)

	case 0x57: // SRE $zp,X
		cpu.PC++
		zpAddr := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		cpu.PC++
		value := cpu.Read(zpAddr)
		// First: Shift Right (LSR)
		carry := (value & 0x01) != 0
		result := value >> 1
		cpu.Write(zpAddr, result)
		cpu.SetFlag(C, carry)
		// Second: EOR with accumulator
		cpu.A = cpu.A ^ result
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)

	case 0x5B: // SRE $abs,Y
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		baseAddr := (hi << 8) | lo
		effectiveAddr := baseAddr + uint16(cpu.Y)
		value := cpu.Read(effectiveAddr)
		// First: Shift Right (LSR)
		carry := (value & 0x01) != 0
		result := value >> 1
		cpu.Write(effectiveAddr, result)
		cpu.SetFlag(C, carry)
		// Second: EOR with accumulator
		cpu.A = cpu.A ^ result
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)

	case 0x5F: // SRE $abs,X
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		baseAddr := (hi << 8) | lo
		effectiveAddr := baseAddr + uint16(cpu.X)
		value := cpu.Read(effectiveAddr)
		// First: Shift Right (LSR)
		carry := (value & 0x01) != 0
		result := value >> 1
		cpu.Write(effectiveAddr, result)
		cpu.SetFlag(C, carry)
		// Second: EOR with accumulator
		cpu.A = cpu.A ^ result
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)

	// RRA (Rotate Right and ADC) - Unofficial instruction
	case 0x63: // RRA ($zp,X)
		cpu.PC++
		zpAddr := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		cpu.PC++
		effectiveAddr := uint16(cpu.Read(zpAddr)) | (uint16(cpu.Read((zpAddr+1)&0xFF)) << 8)
		value := cpu.Read(effectiveAddr)
		// First: Rotate Right (ROR)
		oldCarry := uint8(0)
		if cpu.GetFlag(C) == 1 {
			oldCarry = 0x80
		}
		newCarry := (value & 0x01) != 0
		result := (value >> 1) | oldCarry
		cpu.Write(effectiveAddr, result)
		cpu.SetFlag(C, newCarry)
		// Second: ADC with accumulator
		oldA := cpu.A
		temp := uint16(cpu.A) + uint16(result)
		if cpu.GetFlag(C) == 1 {  // Use the carry from ROR operation
			temp++
		}
		cpu.A = uint8(temp)
		cpu.SetFlag(C, temp > 255)
		cpu.SetFlag(V, (oldA^cpu.A)&(result^cpu.A)&0x80 != 0)
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)

	case 0x67: // RRA $zp
		cpu.PC++
		zpAddr := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		value := cpu.Read(zpAddr)
		// First: Rotate Right (ROR)
		oldCarry := uint8(0)
		if cpu.GetFlag(C) == 1 {
			oldCarry = 0x80
		}
		newCarry := (value & 0x01) != 0
		result := (value >> 1) | oldCarry
		cpu.Write(zpAddr, result)
		cpu.SetFlag(C, newCarry)
		// Second: ADC with accumulator
		oldA := cpu.A
		temp := uint16(cpu.A) + uint16(result)
		if cpu.GetFlag(C) == 1 {  // Use the carry from ROR operation
			temp++
		}
		cpu.A = uint8(temp)
		cpu.SetFlag(C, temp > 255)
		cpu.SetFlag(V, (oldA^cpu.A)&(result^cpu.A)&0x80 != 0)
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)

	case 0x6F: // RRA $abs
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		absAddr := (hi << 8) | lo
		cpu.PC++
		value := cpu.Read(absAddr)
		// First: Rotate Right (ROR)
		oldCarry := uint8(0)
		if cpu.GetFlag(C) == 1 {
			oldCarry = 0x80
		}
		newCarry := (value & 0x01) != 0
		result := (value >> 1) | oldCarry
		cpu.Write(absAddr, result)
		cpu.SetFlag(C, newCarry)
		// Second: ADC with accumulator
		oldA := cpu.A
		temp := uint16(cpu.A) + uint16(result)
		if cpu.GetFlag(C) == 1 {  // Use the carry from ROR operation
			temp++
		}
		cpu.A = uint8(temp)
		cpu.SetFlag(C, temp > 255)
		cpu.SetFlag(V, (oldA^cpu.A)&(result^cpu.A)&0x80 != 0)
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)

	case 0x77: // RRA $zp,X
		cpu.PC++
		zpAddr := (uint16(cpu.Read(cpu.PC)) + uint16(cpu.X)) & 0xFF
		cpu.PC++
		value := cpu.Read(zpAddr)
		// First: Rotate Right (ROR)
		oldCarry := uint8(0)
		if cpu.GetFlag(C) == 1 {
			oldCarry = 0x80
		}
		newCarry := (value & 0x01) != 0
		result := (value >> 1) | oldCarry
		cpu.Write(zpAddr, result)
		cpu.SetFlag(C, newCarry)
		// Second: ADC with accumulator
		oldA := cpu.A
		temp := uint16(cpu.A) + uint16(result)
		if cpu.GetFlag(C) == 1 {  // Use the carry from ROR operation
			temp++
		}
		cpu.A = uint8(temp)
		cpu.SetFlag(C, temp > 255)
		cpu.SetFlag(V, (oldA^cpu.A)&(result^cpu.A)&0x80 != 0)
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)

	case 0x73: // RRA ($zp),Y
		cpu.PC++
		zpAddr := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		baseAddr := uint16(cpu.Read(zpAddr)) | (uint16(cpu.Read((zpAddr+1)&0xFF)) << 8)
		effectiveAddr := baseAddr + uint16(cpu.Y)
		value := cpu.Read(effectiveAddr)
		// First: Rotate Right (ROR)
		oldCarry := uint8(0)
		if cpu.GetFlag(C) == 1 {
			oldCarry = 0x80
		}
		newCarry := (value & 0x01) != 0
		result := (value >> 1) | oldCarry
		cpu.Write(effectiveAddr, result)
		cpu.SetFlag(C, newCarry)
		// Second: ADC with accumulator
		oldA := cpu.A
		temp := uint16(cpu.A) + uint16(result)
		if cpu.GetFlag(C) == 1 {  // Use the carry from ROR operation
			temp++
		}
		cpu.A = uint8(temp)
		cpu.SetFlag(C, temp > 255)
		cpu.SetFlag(V, (oldA^cpu.A)&(result^cpu.A)&0x80 != 0)
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)

	case 0x7B: // RRA $abs,Y
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		baseAddr := (hi << 8) | lo
		effectiveAddr := baseAddr + uint16(cpu.Y)
		cpu.PC++
		value := cpu.Read(effectiveAddr)
		// First: Rotate Right (ROR)
		oldCarry := uint8(0)
		if cpu.GetFlag(C) == 1 {
			oldCarry = 0x80
		}
		newCarry := (value & 0x01) != 0
		result := (value >> 1) | oldCarry
		cpu.Write(effectiveAddr, result)
		cpu.SetFlag(C, newCarry)
		// Second: ADC with accumulator
		oldA := cpu.A
		temp := uint16(cpu.A) + uint16(result)
		if cpu.GetFlag(C) == 1 {  // Use the carry from ROR operation
			temp++
		}
		cpu.A = uint8(temp)
		cpu.SetFlag(C, temp > 255)
		cpu.SetFlag(V, (oldA^cpu.A)&(result^cpu.A)&0x80 != 0)
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)

	case 0x7F: // RRA $abs,X
		cpu.PC++
		lo := uint16(cpu.Read(cpu.PC))
		cpu.PC++
		hi := uint16(cpu.Read(cpu.PC))
		baseAddr := (hi << 8) | lo
		effectiveAddr := baseAddr + uint16(cpu.X)
		cpu.PC++
		value := cpu.Read(effectiveAddr)
		// First: Rotate Right (ROR)
		oldCarry := uint8(0)
		if cpu.GetFlag(C) == 1 {
			oldCarry = 0x80
		}
		newCarry := (value & 0x01) != 0
		result := (value >> 1) | oldCarry
		cpu.Write(effectiveAddr, result)
		cpu.SetFlag(C, newCarry)
		// Second: ADC with accumulator
		oldA := cpu.A
		temp := uint16(cpu.A) + uint16(result)
		if cpu.GetFlag(C) == 1 {  // Use the carry from ROR operation
			temp++
		}
		cpu.A = uint8(temp)
		cpu.SetFlag(C, temp > 255)
		cpu.SetFlag(V, (oldA^cpu.A)&(result^cpu.A)&0x80 != 0)
		cpu.SetFlag(Z, cpu.A == 0x00)
		cpu.SetFlag(N, cpu.A&0x80 != 0)

	default:
		// Unknown instruction - log and terminate
		panic(fmt.Sprintf("Unknown instruction $%02X at PC=$%04X", opcode, cpu.PC))
	}
}