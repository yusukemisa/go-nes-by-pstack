package apu

// Channel waveforms, envelopes, the sweep, and the triangle linear counter.
// https://www.nesdev.org/wiki/APU_Pulse
// https://www.nesdev.org/wiki/APU_Sweep
// https://www.nesdev.org/wiki/APU_Envelope
// https://www.nesdev.org/wiki/APU_Triangle
// https://www.nesdev.org/wiki/APU_Noise
// https://www.nesdev.org/wiki/APU_Mixer

// dutySequence is the pulse output waveform in time order. The hardware
// counter walks a lookup downward; this table is already that result.
// Duty 0 is 12.5%, 1 is 25%, 2 is 50%, 3 is 25% negated.
// https://www.nesdev.org/wiki/APU_Pulse
var dutySequence = [4][8]uint8{
	{0, 1, 0, 0, 0, 0, 0, 0},
	{0, 1, 1, 0, 0, 0, 0, 0},
	{0, 1, 1, 1, 1, 0, 0, 0},
	{1, 0, 0, 1, 1, 1, 1, 1},
}

// triangleSequence is the 32-step waveform sent to the mixer.
// https://www.nesdev.org/wiki/APU_Triangle
var triangleSequence = [32]uint8{
	15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0,
	0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
}

// noisePeriodNTSC is how many CPU cycles fall between LFSR clocks.
// The values are even because an APU cycle is two CPU cycles.
// https://www.nesdev.org/wiki/APU_Noise
var noisePeriodNTSC = [16]uint16{
	4, 8, 16, 32, 64, 96, 128, 160, 202, 254, 380, 508, 762, 1016, 2034, 4068,
}

// Lookup mixer. Index 0 stays 0 so a silent group is not a division by zero.
// DMC is not implemented and contributes 0 to the TND index.
// https://www.nesdev.org/wiki/APU_Mixer
var (
	pulseTable [31]float32
	tndTable   [203]float32
)

func init() {
	for n := 1; n < len(pulseTable); n++ {
		pulseTable[n] = float32(95.52 / (8128.0/float64(n) + 100))
	}
	for n := 1; n < len(tndTable); n++ {
		tndTable[n] = float32(163.67 / (24329.0/float64(n) + 100))
	}
}

type envelope struct {
	start    bool
	loop     bool
	constant bool
	volume   uint8
	divider  uint8
	decay    uint8
}

func (e *envelope) write(data uint8) {
	e.loop = data&0x20 != 0
	e.constant = data&0x10 != 0
	e.volume = data & 0x0F
}

// clock is a quarter-frame clock.
// https://www.nesdev.org/wiki/APU_Envelope
func (e *envelope) clock() {
	if e.start {
		e.start = false
		e.decay = 15
		e.divider = e.volume
		return
	}
	if e.divider > 0 {
		e.divider--
		return
	}
	e.divider = e.volume
	if e.decay > 0 {
		e.decay--
	} else if e.loop {
		e.decay = 15
	}
}

func (e *envelope) output() uint8 {
	if e.constant {
		return e.volume
	}
	return e.decay
}

type pulse struct {
	duty    uint8
	dutyPos uint8
	period  uint16
	timer   uint16
	env     envelope

	sweepEnabled bool
	sweepPeriod  uint8
	sweepNegate  bool
	sweepShift   uint8
	sweepDivider uint8
	sweepReload  bool
	// complement selects pulse 1's ones' complement negate. Pulse 2 uses
	// two's complement. https://www.nesdev.org/wiki/APU_Sweep
	complement bool
}

func (p *pulse) writeDuty(data uint8) {
	p.duty = data >> 6
	p.env.write(data)
}

func (p *pulse) writeSweep(data uint8) {
	p.sweepEnabled = data&0x80 != 0
	p.sweepPeriod = (data >> 4) & 7
	p.sweepNegate = data&0x08 != 0
	p.sweepShift = data & 7
	p.sweepReload = true
}

func (p *pulse) writeTimerLow(data uint8) {
	p.period = (p.period & 0x0700) | uint16(data)
}

func (p *pulse) writeTimerHigh(data uint8) {
	p.period = (p.period & 0x00FF) | uint16(data&0x07)<<8
	p.dutyPos = 0
	p.env.start = true
}

// target is the sweep unit's continuously calculated period, clamped at 0.
// https://www.nesdev.org/wiki/APU_Sweep
func (p *pulse) target() int {
	change := int(p.period >> p.sweepShift)
	if p.sweepNegate {
		if p.complement {
			change = -change - 1
		} else {
			change = -change
		}
	}
	sum := int(p.period) + change
	if sum < 0 {
		return 0
	}
	return sum
}

func (p *pulse) muted() bool {
	if p.period < 8 {
		return true
	}
	return p.target() > 0x7FF
}

// clockSweep is a half-frame clock. The divider period is P+1 half-frames
// because a counter of P counts P, P-1, ..., 0.
func (p *pulse) clockSweep() {
	if p.sweepDivider == 0 && p.sweepEnabled && p.sweepShift > 0 && !p.muted() {
		p.period = uint16(p.target())
	}
	if p.sweepDivider == 0 || p.sweepReload {
		p.sweepDivider = p.sweepPeriod
		p.sweepReload = false
		return
	}
	p.sweepDivider--
}

// clockTimer runs on APU cycles. The timer counts t, t-1, ..., 0 and clocks
// the sequencer when it reloads from 0 to t. https://www.nesdev.org/wiki/APU_Pulse
func (p *pulse) clockTimer() {
	if p.timer == 0 {
		p.timer = p.period
		p.dutyPos = (p.dutyPos + 1) & 7
		return
	}
	p.timer--
}

func (p *pulse) output(length uint8) uint8 {
	if length == 0 || p.muted() || dutySequence[p.duty][p.dutyPos] == 0 {
		return 0
	}
	return p.env.output()
}

type triangle struct {
	period uint16
	timer  uint16
	step   uint8
	linear uint8
	reload uint8
	flag   bool
}

func (t *triangle) writeLinear(data uint8) {
	t.reload = data & 0x7F
}

func (t *triangle) writeTimerLow(data uint8) {
	t.period = (t.period & 0x0700) | uint16(data)
}

func (t *triangle) writeTimerHigh(data uint8) {
	t.period = (t.period & 0x00FF) | uint16(data&0x07)<<8
	t.flag = true
}

// clockLinear is a quarter-frame clock.
// https://www.nesdev.org/wiki/APU_Triangle
func (t *triangle) clockLinear(control bool) {
	if t.flag {
		t.linear = t.reload
	} else if t.linear > 0 {
		t.linear--
	}
	if !control {
		t.flag = false
	}
}

func (t *triangle) clockTimer(length uint8) {
	if t.timer == 0 {
		t.timer = t.period
		if length > 0 && t.linear > 0 {
			t.step = (t.step + 1) & 31
		}
		return
	}
	t.timer--
}

func (t *triangle) output(length uint8) uint8 {
	// A halted sequencer holds its last step. The NES high-pass filters turn
	// that DC into silence. Those filters are not modeled, so a zero length
	// or linear counter contributes 0 instead of the held step.
	if length == 0 || t.linear == 0 {
		return 0
	}
	return triangleSequence[t.step]
}

type noise struct {
	env    envelope
	mode   bool
	period uint16
	timer  uint16
	shift  uint16
}

func (n *noise) writePeriod(data uint8) {
	n.mode = data&0x80 != 0
	n.period = noisePeriodNTSC[data&0x0F]
}

func (n *noise) clockTimer() {
	if n.timer > 0 {
		n.timer--
		return
	}
	// period is in CPU cycles; this timer ticks once per APU cycle.
	if n.period >= 2 {
		n.timer = n.period/2 - 1
	}
	bit := uint16(1)
	if n.mode {
		bit = 6
	}
	feedback := (n.shift & 1) ^ ((n.shift >> bit) & 1)
	n.shift = (n.shift >> 1) | (feedback << 14)
}

func (n *noise) output(length uint8) uint8 {
	if length == 0 || n.shift&1 == 1 {
		return 0
	}
	return n.env.output()
}
