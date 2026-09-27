package apu

import "testing"

func clockUntil(a *APU, pred func() bool, limit int) int {
	for i := 1; i <= limit; i++ {
		a.Clock()
		if pred() {
			return i
		}
	}
	return -1
}

func TestLengthTableEnableAndHalt(t *testing.T) {
	a := New()
	a.Write(0x4003, 0x18) // $18 >> 3 = 3 → 2, but the channel is disabled
	if a.length[0] != 0 {
		t.Fatalf("disabled load %d", a.length[0])
	}
	a.Write(0x4015, 0x0F)
	if a.Read(0x4015) != 0 {
		t.Fatal("enabling a channel does not reload its length")
	}
	for _, addr := range []uint16{0x4003, 0x4007, 0x400B, 0x400F} {
		a.Write(addr, 0x18)
	}
	for i, n := range a.length {
		if n != 2 {
			t.Fatalf("ch %d length %d, $18 should load 2", i, n)
		}
	}
	if got := a.Read(0x4015); got != 0x0F {
		t.Fatalf("status %02X", got)
	}

	a.Write(0x4015, 0x01)
	if a.length[0] != 2 || a.length[1] != 0 || a.length[2] != 0 || a.length[3] != 0 {
		t.Fatalf("disable did not zero lengths: %v", a.length)
	}
	a.Write(0x4007, 0x18)
	if a.length[1] != 0 {
		t.Fatal("length load while disabled")
	}

	a = New()
	a.Write(0x4015, 0x01)
	for i := 0; i < 32; i++ {
		a.Write(0x4003, uint8(i<<3))
		if a.length[0] != lengthTable[i] {
			t.Fatalf("index %d = %d, table %d", i, a.length[0], lengthTable[i])
		}
	}

	a = New()
	a.Write(0x4015, 0x0F)
	for _, addr := range []uint16{0x4003, 0x4007, 0x400B, 0x400F} {
		a.Write(addr, 0x18)
	}
	a.Write(0x4000, 0x20) // pulse 1 halt (bit 5)
	a.Write(0x4008, 0x20) // triangle bit 5 is not halt
	a.Write(0x4017, 0x00)
	n := clockUntil(a, func() bool { return a.halves == 1 }, 20000)
	if n < 0 {
		t.Fatal("no half frame")
	}
	if a.length[0] != 2 {
		t.Fatalf("halted pulse 1 length %d", a.length[0])
	}
	if a.length[1] != 1 || a.length[3] != 1 {
		t.Fatalf("pulse2/noise %d %d", a.length[1], a.length[3])
	}
	if a.length[2] != 1 {
		t.Fatalf("triangle halted by bit 5: %d", a.length[2])
	}

	a.Write(0x4008, 0x80)
	a.Write(0x4004, 0x20)
	a.Write(0x400C, 0x20)
	n = clockUntil(a, func() bool { return a.halves == 2 }, 20000)
	if n < 0 {
		t.Fatal("no second half frame")
	}
	if a.length != [4]uint8{2, 1, 1, 1} {
		t.Fatalf("lengths after halted second half %v", a.length)
	}
}

func TestRegisterReadsAreNotTheWrittenValue(t *testing.T) {
	a := New()
	for addr := uint16(0x4000); addr <= 0x4013; addr++ {
		a.Write(addr, 0xA5)
		if got := a.Read(addr); got != 0 {
			t.Fatalf("%04X read %02X", addr, got)
		}
	}
	a.Write(0x4015, 0xFF)
	a.Write(0x4003, 0x18)
	got := a.Read(0x4015)
	if got&0x01 == 0 || got&0x90 != 0 || got == 0xFF {
		t.Fatalf("status %02X", got)
	}
}

func TestFourStepFrameIRQAndFiveStepClocks(t *testing.T) {
	a := New()
	a.Write(0x4017, 0x00)
	n := clockUntil(a, func() bool { return a.frameIRQ }, 40000)
	if n < 29830-4 || n > 29830+4 {
		t.Fatalf("frame IRQ at clock %d after $4017=$00, want 29830±4", n)
	}
	if got := a.Read(0x4015); got&0x40 == 0 {
		t.Fatalf("read missed bit6: %02X", got)
	}
	if a.frameIRQ {
		t.Fatal("read did not clear the flag")
	}
	if a.Read(0x4015)&0x40 != 0 {
		t.Fatal("flag set again without a clock")
	}

	a = New()
	a.Write(0x4015, 0x01)
	a.Write(0x4003, 0x18)
	a.Write(0x4017, 0x00)
	first := clockUntil(a, func() bool { return a.length[0] == 1 }, 20000)
	second := first + clockUntil(a, func() bool { return a.length[0] == 0 }, 20000)
	if first < 14000 || first > 16000 || second < 29000 || second > 31000 {
		t.Fatalf("length clocks at %d and %d", first, second)
	}
	if !a.frameIRQ {
		t.Fatal("IRQ flag should be up once the length has expired")
	}
	a.Write(0x4017, 0x00) // bit 6 clear: flag stays
	if !a.frameIRQ {
		t.Fatal("$4017=$00 cleared the frame flag")
	}
	a.Write(0x4015, 0x00) // status write does not clear it
	if !a.frameIRQ {
		t.Fatal("$4015 write cleared the frame flag")
	}
	a.Write(0x4017, 0x40)
	if a.frameIRQ {
		t.Fatal("inhibit did not clear the flag on the write")
	}
	if clockUntil(a, func() bool { return a.frameIRQ }, 40000) != -1 {
		t.Fatal("inhibit still raised the flag")
	}

	// Odd write cycle → the other delay (4). Still inside 29830±4.
	a = New()
	a.Clock()
	a.Write(0x4017, 0x00)
	n = clockUntil(a, func() bool { return a.frameIRQ }, 40000)
	if n < 29830-4 || n > 29830+4 {
		t.Fatalf("odd-cycle frame IRQ at %d", n)
	}

	a = New()
	a.Write(0x4015, 0x01)
	a.Write(0x4003, 0x18)
	a.Write(0x4017, 0x80)
	if a.quarters != 0 || a.halves != 0 || a.length[0] != 2 {
		t.Fatal("5-step write clocked before the delayed reset")
	}
	n = clockUntil(a, func() bool { return a.halves == 1 && a.quarters == 1 }, 8)
	if n < 1 || n > 5 {
		t.Fatalf("5-step quarter/half at clock %d", n)
	}
	if a.length[0] != 1 {
		t.Fatalf("immediate half left length %d", a.length[0])
	}
	for i := 0; i < 100; i++ {
		a.Clock()
	}
	if a.halves != 1 || a.quarters != 1 {
		t.Fatalf("extra early clocks q=%d h=%d", a.quarters, a.halves)
	}
	if clockUntil(a, func() bool { return a.frameIRQ }, 40000) != -1 {
		t.Fatal("5-step raised the frame IRQ")
	}
	if a.length[0] != 0 {
		t.Fatalf("5-step second half did not finish the counter (%d)", a.length[0])
	}
}
