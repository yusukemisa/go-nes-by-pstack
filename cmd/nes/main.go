package main

import (
	"fmt"
	"log"
	"os"

	"github.com/misawa/go-nes-by-pstack/internal/nes"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: nes <rom.nes> [frames]")
	}
	frames := 2
	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[2], "%d", &frames)
	}
	c, err := nes.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	c.Reset()
	fmt.Printf("reset PC=$%04X first=$%02X\n", c.CPU.PC, c.Bus.CPURead(c.CPU.PC))
	for i := 0; i < frames; i++ {
		c.StepFrame()
	}
	if err := writePPM("frame.ppm", c.Frame()); err != nil {
		log.Fatal(err)
	}
	fmt.Println("wrote frame.ppm")
}

func writePPM(path string, screen *[240][256]uint8) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	fmt.Fprintf(f, "P3\n256 240\n255\n")
	for y := 0; y < 240; y++ {
		for x := 0; x < 256; x++ {
			v := int(screen[y][x]) * 4
			if v > 255 {
				v = 255
			}
			fmt.Fprintf(f, "%d %d %d\n", v, v, v)
		}
	}
	return nil
}
