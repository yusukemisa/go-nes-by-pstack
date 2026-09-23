package nes_test

import (
	"bufio"
	"os"
	"regexp"
	"testing"

	"github.com/misawa/go-nes-by-pstack/internal/nes"
)

var nestestLine = regexp.MustCompile(`^([0-9A-F]{4})\s+.*A:([0-9A-F]{2}) X:([0-9A-F]{2}) Y:([0-9A-F]{2}) P:([0-9A-F]{2}) SP:([0-9A-F]{2}) PPU:\s*(\d+),\s*(\d+) CYC:(\d+)`)

func TestNestestLog(t *testing.T) {
	c, err := nes.Open("../../testdata/nestest.nes")
	if err != nil {
		t.Fatal(err)
	}
	c.PowerOnNestest()

	f, err := os.Open("../../testdata/nestest.log")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	n := 0
	for sc.Scan() {
		n++
		m := nestestLine.FindStringSubmatch(sc.Text())
		if m == nil {
			t.Fatalf("cannot parse nestest.log line %d: %q", n, sc.Text())
		}
		got := c.TraceRegs()
		want := m[1] + " A:" + m[2] + " X:" + m[3] + " Y:" + m[4] + " P:" + m[5] + " SP:" + m[6] +
			" PPU:" + pad3(m[7]) + "," + pad3(m[8]) + " CYC:" + m[9]
		if got != want {
			t.Fatalf("line %d\n got %s\nwant %s", n, got, want)
		}
		c.StepInstruction()
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if n != 8991 {
		t.Fatalf("expected 8991 log lines, compared %d", n)
	}
}

func pad3(s string) string {
	for len(s) < 3 {
		s = " " + s
	}
	return s
}
