package cpu

// Unofficial opcodes that were still missing from executeInstructionEnhanced.
// Stable flag behavior and the SHX/SHY/SHA store formula follow the programming
// guide. XAA/LXA use one digital approximation of an analog result.
// https://www.nesdev.org/wiki/Programming_with_unofficial_opcodes
// https://www.nesdev.org/wiki/CPU_unofficial_opcodes
// https://www.nesdev.org/wiki/Visual6502wiki/6502_Opcode_8B_(XAA,_ANE)

// unstableMagic is ORed into A for XAA ($8B) and LXA ($AB).
// $EE leaves bits 0 and 4 clear, so those bits of A still shine through.
// The XAA page says bits 0 and 4 feed A back directly, and lists $EE on
// several MOS 6502s (the family the 2A03 comes from). The 2A03 itself is not
// in that table. LXA is the immediate slot the programming guide says is
// "affected by line noise on the data bus", so it uses the same constant.
const unstableMagic uint8 = 0xEE

func (cpu *CPU) immediate() uint8 {
	cpu.PC++
	value := cpu.Read(cpu.PC)
	cpu.PC++
	return value
}

func (cpu *CPU) absOperand() uint16 {
	cpu.PC++
	lo := uint16(cpu.Read(cpu.PC))
	cpu.PC++
	hi := uint16(cpu.Read(cpu.PC))
	cpu.PC++
	return (hi << 8) | lo
}

func (cpu *CPU) setZN(value uint8) {
	cpu.SetFlag(Z, value == 0)
	cpu.SetFlag(N, value&0x80 != 0)
}

// nopImmediate is NOP #imm ($82 $89 $C2 $E2): read nothing into a register.
// 2 cycles, same as $80. https://www.nesdev.org/wiki/Programming_with_unofficial_opcodes
func (cpu *CPU) nopImmediate() {
	cpu.PC += 2
}

// anc is ANC #imm ($0B $2B): AND, then copy N into C.
func (cpu *CPU) anc() {
	cpu.A &= cpu.immediate()
	cpu.setZN(cpu.A)
	cpu.SetFlag(C, cpu.A&0x80 != 0)
}

// alr is ALR #imm ($4B): AND, then LSR. N is cleared by the shift.
func (cpu *CPU) alr() {
	cpu.A &= cpu.immediate()
	carry := cpu.A&0x01 != 0
	cpu.A >>= 1
	cpu.SetFlag(C, carry)
	cpu.SetFlag(Z, cpu.A == 0)
	cpu.SetFlag(N, false)
}

// arr is ARR #imm ($6B): AND, then ROR. N and Z come from the rotated value.
// C is bit 6 of the result and V is bit 6 xor bit 5. The 2A03 has no decimal
// mode, so only this binary rule is implemented.
func (cpu *CPU) arr() {
	and := cpu.A & cpu.immediate()
	rotated := and >> 1
	if cpu.GetFlag(C) == 1 {
		rotated |= 0x80
	}
	cpu.A = rotated
	cpu.setZN(cpu.A)
	cpu.SetFlag(C, cpu.A&0x40 != 0)
	bit6 := cpu.A&0x40 != 0
	bit5 := cpu.A&0x20 != 0
	cpu.SetFlag(V, bit6 != bit5)
}

// axs is AXS/SBX #imm ($CB): X = (A & X) - imm, with no borrow-in.
// C is set when (A & X) >= imm, the same way CMP sets C.
func (cpu *CPU) axs() {
	imm := cpu.immediate()
	left := cpu.A & cpu.X
	cpu.X = left - imm
	cpu.SetFlag(C, left >= imm)
	cpu.setZN(cpu.X)
}

// lxa is LXA #imm ($AB): A and X = (A | magic) & imm. Unstable; see unstableMagic.
func (cpu *CPU) lxa() {
	value := (cpu.A | unstableMagic) & cpu.immediate()
	cpu.A = value
	cpu.X = value
	cpu.setZN(value)
}

// xaa is XAA/ANE #imm ($8B): A = (A | magic) & X & imm. Unstable; see unstableMagic.
func (cpu *CPU) xaa() {
	value := (cpu.A | unstableMagic) & cpu.X & cpu.immediate()
	cpu.A = value
	cpu.setZN(value)
}

// las is LAS abs,Y ($BB): A, X, and S become the memory value ANDed with S.
// 4 cycles, plus the indexed-read page-cross cycle from readIndexedAddr.
func (cpu *CPU) las() {
	base := cpu.absOperand()
	addr := cpu.readIndexedAddr(base, cpu.Y)
	value := cpu.Read(addr) & cpu.SP
	cpu.A = value
	cpu.X = value
	cpu.SP = value
	cpu.setZN(value)
}

// unstableStore writes mask&(H+1) to base+index. H is the high byte of base,
// the address before the index is added. A page cross does not add a cycle.
// The high byte of the computed address is then ANDed with andHi.
//
// The programming guide states that for SHX, and says SHY has the same
// caveats and SHA combines that store with A. TAS is not given its own
// paragraph; it is the stack sibling in the same opcode row, so it uses the
// same AND-with-X rule. For an 8-bit index, SHX/SHY's "AND with X/Y" matches
// "replace the high byte with the stored value". SHA and TAS do not: the
// address is ANDed with X, not with A&X.
// DMC DMA can also disturb these stores. DMC is not implemented.
// https://www.nesdev.org/wiki/Programming_with_unofficial_opcodes
func (cpu *CPU) unstableStore(base uint16, index, mask, andHi uint8) {
	addr := base + uint16(index)
	value := mask & (uint8(base>>8) + 1)
	if base&0xFF00 != addr&0xFF00 {
		addr = (addr & 0x00FF) | (uint16(uint8(addr>>8)&andHi) << 8)
	}
	cpu.Write(addr, value)
}

// shaIndirectY is SHA/AHX (zp),Y ($93). 6 cycles, like STA (zp),Y.
func (cpu *CPU) shaIndirectY() {
	cpu.PC++
	zp := uint16(cpu.Read(cpu.PC))
	cpu.PC++
	lo := uint16(cpu.Read(zp))
	hi := uint16(cpu.Read((zp + 1) & 0xFF))
	base := (hi << 8) | lo
	cpu.unstableStore(base, cpu.Y, cpu.A&cpu.X, cpu.X)
}

// shaAbsoluteY is SHA/AHX abs,Y ($9F). 5 cycles, like STA abs,Y.
func (cpu *CPU) shaAbsoluteY() {
	cpu.unstableStore(cpu.absOperand(), cpu.Y, cpu.A&cpu.X, cpu.X)
}

// tas is TAS/SHS abs,Y ($9B): S = A & X, then store S & (H+1). 5 cycles.
func (cpu *CPU) tas() {
	cpu.SP = cpu.A & cpu.X
	cpu.unstableStore(cpu.absOperand(), cpu.Y, cpu.SP, cpu.X)
}

// shy is SHY abs,X ($9C). 5 cycles.
func (cpu *CPU) shy() {
	cpu.unstableStore(cpu.absOperand(), cpu.X, cpu.Y, cpu.Y)
}

// shx is SHX abs,Y ($9E). 5 cycles.
func (cpu *CPU) shx() {
	cpu.unstableStore(cpu.absOperand(), cpu.Y, cpu.X, cpu.X)
}

// stp is STP/JAM. The opcode byte stays at PC. Later Clock calls do not fetch,
// and NMI/IRQ are ignored until Reset. The base cycle count still elapses so
// Complete() becomes true; a real KIL wedges the T-state counter and would
// never retire, which would hang Console.StepInstruction.
// https://www.nesdev.org/wiki/Visual6502wiki/6502_Unsupported_Opcodes
func (cpu *CPU) stp() {
	cpu.jammed = true
}
