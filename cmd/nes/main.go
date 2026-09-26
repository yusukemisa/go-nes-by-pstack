package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/misawa/go-nes-by-pstack/internal/nes"
)

func main() {
	ppm := flag.Bool("ppm", false, "run headless and write frame.ppm")
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		log.Fatal("usage: nes <rom.nes> | nes -ppm <rom.nes> [frames]")
	}
	c, err := nes.Open(args[0])
	if err != nil {
		log.Fatal(err)
	}
	c.Reset()

	if !*ppm {
		if err := runWindow(c); err != nil {
			log.Fatal(err)
		}
		return
	}

	frames := 2
	if len(args) >= 2 {
		fmt.Sscanf(args[1], "%d", &frames)
	}
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
