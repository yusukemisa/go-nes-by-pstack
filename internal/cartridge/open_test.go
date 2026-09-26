package cartridge_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/misawa/go-nes-by-pstack/internal/nes"
)

func TestOpenRejectsUnsupportedMapper(t *testing.T) {
	hdr := make([]byte, 16+16384+8192)
	copy(hdr[0:4], "NES\x1a")
	hdr[4] = 1
	hdr[5] = 1
	hdr[6] = 0x40 // mapper 4

	path := filepath.Join(t.TempDir(), "mapper4.nes")
	if err := os.WriteFile(path, hdr, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := nes.Open(path)
	if err == nil || !strings.Contains(err.Error(), "unsupported mapper 4") {
		t.Fatalf("nes.Open err = %v", err)
	}
}
