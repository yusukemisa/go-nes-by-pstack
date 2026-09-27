package apu

// APU clocks the frame sequencer, the length counters, and the audible
// channels, then emits a mono sample stream. DMC is not implemented; its
// mixer contribution stays 0, and $4010–$4013 are ignored.
// https://www.nesdev.org/wiki/APU
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
//
// Pulse and noise timers use that same even-cycle approximation for the APU
// clock (every second CPU cycle). Triangle timers run on every CPU cycle.
// Samples are point-sampled at SampleRate against the NTSC CPU clock; the
// analog high-pass and low-pass filters after the mixer are not modeled.
type APU struct {
	length  [4]uint8
	enabled [4]bool
	halt    [4]bool

	pulse [2]pulse
	tri   triangle
	noise noise

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

	sampleAccum int
	samples     []float32
}

// SampleRate is the mono stream rate in hertz. One sample is the mixer
// output at that instant; cmd/nes queues the stream to SDL.
const (
	SampleRate = 44100
	// ntscCPUHz is 1.789773 MHz. https://www.nesdev.org/wiki/APU_Pulse
	ntscCPUHz         = 1789773
	maxPendingSamples = SampleRate
)

// lengthTable is indexed by bits 7–3 of $4003/$4007/$400B/$400F.
// https://www.nesdev.org/wiki/APU_Length_Counter
var lengthTable = [32]uint8{
	10, 254, 20, 2, 40, 4, 80, 6,
	160, 8, 60, 10, 14, 12, 26, 14,
	12, 16, 24, 18, 48, 20, 96, 22,
	192, 24, 72, 26, 16, 28, 32, 30,
}

func New() *APU {
	a := &APU{}
	// Pulse 1's sweep negate is ones' complement. https://www.nesdev.org/wiki/APU_Sweep
	a.pulse[0].complement = true
	// https://www.nesdev.org/wiki/APU_Noise
	a.noise.shift = 1
	return a
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
	case 0x4000, 0x4004:
		ch := 0
		if addr == 0x4004 {
			ch = 1
		}
		a.pulse[ch].writeDuty(data)
		a.halt[ch] = a.pulse[ch].env.loop
	case 0x4001, 0x4005:
		ch := 0
		if addr == 0x4005 {
			ch = 1
		}
		a.pulse[ch].writeSweep(data)
	case 0x4002, 0x4006:
		ch := 0
		if addr == 0x4006 {
			ch = 1
		}
		a.pulse[ch].writeTimerLow(data)
	case 0x4003, 0x4007:
		ch := 0
		if addr == 0x4007 {
			ch = 1
		}
		a.pulse[ch].writeTimerHigh(data)
		a.loadLength(ch, data)
	case 0x4008:
		a.tri.writeLinear(data)
		a.halt[2] = data&0x80 != 0
	case 0x400A:
		a.tri.writeTimerLow(data)
	case 0x400B:
		a.tri.writeTimerHigh(data)
		a.loadLength(2, data)
	case 0x400C:
		a.noise.env.write(data)
		a.halt[3] = a.noise.env.loop
	case 0x400E:
		a.noise.writePeriod(data)
	case 0x400F:
		a.noise.env.start = true
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
	a.clockChannels()
	a.emitSample()
	a.cpuCycle++
}

// clockChannels advances timers. Even cpuCycle is "during an APU cycle", the
// same parity already used for the $4017 delay, so pulse and noise tick then.
// Triangle ticks every CPU cycle. https://www.nesdev.org/wiki/APU_Pulse
// https://www.nesdev.org/wiki/APU_Triangle
// https://www.nesdev.org/wiki/APU_Noise
func (a *APU) clockChannels() {
	a.tri.clockTimer(a.length[2])
	if a.cpuCycle&1 == 0 {
		a.pulse[0].clockTimer()
		a.pulse[1].clockTimer()
		a.noise.clockTimer()
	}
}

// Sample is the current mixer output, in about [0, 1).
func (a *APU) Sample() float32 {
	p := int(a.pulse[0].output(a.length[0])) + int(a.pulse[1].output(a.length[1]))
	tnd := 3*int(a.tri.output(a.length[2])) + 2*int(a.noise.output(a.length[3]))
	return pulseTable[p] + tndTable[tnd]
}

// TakeSamples drains mono samples emitted at SampleRate since the last take.
func (a *APU) TakeSamples() []float32 {
	out := a.samples
	a.samples = nil
	return out
}

func (a *APU) emitSample() {
	a.sampleAccum += SampleRate
	if a.sampleAccum < ntscCPUHz {
		return
	}
	a.sampleAccum -= ntscCPUHz
	if len(a.samples) >= maxPendingSamples {
		return
	}
	a.samples = append(a.samples, a.Sample())
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
	a.pulse[0].env.clock()
	a.pulse[1].env.clock()
	a.noise.env.clock()
	a.tri.clockLinear(a.halt[2])
}

func (a *APU) half() {
	a.halves++
	for i := range a.length {
		if a.halt[i] || a.length[i] == 0 {
			continue
		}
		a.length[i]--
	}
	a.pulse[0].clockSweep()
	a.pulse[1].clockSweep()
}
