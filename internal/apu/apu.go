package apu

// APU clocks the frame sequencer and the length counters.
// Waveforms, envelopes, the sweep, the linear counter, and the DMC are not
// implemented. https://www.nesdev.org/wiki/APU
//
// The frame sequencer is advanced once per CPU cycle. Its steps follow the
// NTSC table on https://www.nesdev.org/wiki/APU_Frame_Counter : quarter-frame
// clocks at 7457, 14913, 22371, and 29829 CPU cycles after the $4017 reset
// (5-step uses 37281 instead of the last step), half-frame clocks at 14913
// and 29829 (5-step: 14913 and 37281). In 4-step mode the frame IRQ flag is
// raised on cycles 29828, 29829, and 29830, and the sequence period is 29830
// CPU cycles.
//
// A write to $4017 applies 3 or 4 CPU cycles after the write cycle. Even
// cpuCycle (GET, the same parity approximation as OAM DMA) is treated as
// "during an APU cycle" and waits 3; odd (PUT) waits 4. Mode 1 generates one
// quarter-frame and one half-frame clock when that delayed reset happens,
// not on the write itself and not by waiting for step 2.
type APU struct {
	length  [4]uint8
	enabled [4]bool
	halt    [4]bool

	mode5        bool
	pendingMode5 bool
	inhibit      bool
	frameIRQ     bool

	// cpuCycle is the number of Clock calls so far. The write that sets
	// wroteFrame is assigned to the next Clock, whose index is cpuCycle.
	cpuCycle   uint64
	wroteFrame bool
	resetIn    int
	seq        int

	quarters int
	halves   int
}

// lengthTable is indexed by bits 7–3 of $4003/$4007/$400B/$400F.
// https://www.nesdev.org/wiki/APU_Length_Counter
var lengthTable = [32]uint8{
	10, 254, 20, 2, 40, 4, 80, 6,
	160, 8, 60, 10, 14, 12, 26, 14,
	12, 16, 24, 18, 48, 20, 96, 22,
	192, 24, 72, 26, 16, 28, 32, 30,
}

func New() *APU {
	return &APU{}
}

// FrameIRQ reports the frame interrupt flag, which is wired to the CPU IRQ
// line while it is set. Reading $4015 or setting $4017 bit 6 clears it.
func (a *APU) FrameIRQ() bool {
	return a.frameIRQ
}

func (a *APU) Read(addr uint16) uint8 {
	if addr == 0x4015 {
		return a.readStatus()
	}
	// $4000–$4013 and $4017 are write-only. Open bus is a later change;
	// these reads are 0, not the last value written.
	// https://www.nesdev.org/wiki/APU_registers
	return 0
}

func (a *APU) Write(addr uint16, data uint8) {
	switch addr {
	case 0x4000:
		a.halt[0] = data&0x20 != 0
	case 0x4003:
		a.loadLength(0, data)
	case 0x4004:
		a.halt[1] = data&0x20 != 0
	case 0x4007:
		a.loadLength(1, data)
	case 0x4008:
		a.halt[2] = data&0x80 != 0
	case 0x400B:
		a.loadLength(2, data)
	case 0x400C:
		a.halt[3] = data&0x20 != 0
	case 0x400F:
		a.loadLength(3, data)
	case 0x4015:
		a.writeStatus(data)
	case 0x4017:
		a.writeFrame(data)
	}
}

func (a *APU) loadLength(ch int, data uint8) {
	if !a.enabled[ch] {
		return
	}
	a.length[ch] = lengthTable[data>>3]
}

func (a *APU) writeStatus(data uint8) {
	for i := 0; i < 4; i++ {
		on := data&(1<<uint(i)) != 0
		a.enabled[i] = on
		if !on {
			a.length[i] = 0
		}
	}
}

func (a *APU) readStatus() uint8 {
	var s uint8
	for i := 0; i < 4; i++ {
		if a.length[i] > 0 {
			s |= 1 << uint(i)
		}
	}
	if a.frameIRQ {
		s |= 0x40
	}
	// Bit 4 (DMC active) and bit 7 (DMC IRQ) stay 0. Bit 5 is open bus.
	// https://www.nesdev.org/wiki/APU
	a.frameIRQ = false
	return s
}

func (a *APU) writeFrame(data uint8) {
	a.pendingMode5 = data&0x80 != 0
	a.inhibit = data&0x40 != 0
	if a.inhibit {
		a.frameIRQ = false
	}
	// https://www.nesdev.org/wiki/APU_Frame_Counter
	if a.cpuCycle&1 == 0 {
		a.resetIn = 3
	} else {
		a.resetIn = 4
	}
	a.wroteFrame = true
}

// Clock advances the frame sequencer by one CPU cycle.
func (a *APU) Clock() {
	switch {
	case a.wroteFrame:
		// The write cycle itself is not one of the 3 or 4 delay cycles.
		a.wroteFrame = false
		a.stepSequence()
	case a.resetIn > 0:
		a.resetIn--
		if a.resetIn == 0 {
			a.applyReset()
		} else {
			a.stepSequence()
		}
	default:
		a.stepSequence()
	}
	a.cpuCycle++
}

func (a *APU) applyReset() {
	a.seq = 0
	a.mode5 = a.pendingMode5
	if a.mode5 {
		a.quarter()
		a.half()
	}
}

func (a *APU) stepSequence() {
	a.seq++
	if a.mode5 {
		switch a.seq {
		case 7457, 14913, 22371, 37281:
			a.quarter()
		}
		if a.seq == 14913 || a.seq == 37281 {
			a.half()
		}
		if a.seq >= 37282 {
			a.seq = 0
		}
		return
	}
	switch a.seq {
	case 7457, 14913, 22371, 29829:
		a.quarter()
	}
	if a.seq == 14913 || a.seq == 29829 {
		a.half()
	}
	if (a.seq == 29828 || a.seq == 29829 || a.seq == 29830) && !a.inhibit {
		a.frameIRQ = true
	}
	if a.seq >= 29830 {
		a.seq = 0
	}
}

func (a *APU) quarter() {
	a.quarters++
}

func (a *APU) half() {
	a.halves++
	for i := range a.length {
		if a.halt[i] || a.length[i] == 0 {
			continue
		}
		a.length[i]--
	}
}
