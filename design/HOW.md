# How go-nes-hello works

### Overview

go-nes-hello is a Go NES emulator. `cmd/main.go` owns the `NES` type and a 3:1 PPU:CPU master clock. CPU lives in `pkg/cpu`, memory decode in `pkg/bus`, iNES in `pkg/cartridge`, rendering in `pkg/ppu`, SDL in `pkg/display`. nestest is proven by a second `package main` harness, not the windowed loop.

### Key concepts

- `cpu.CPU`: A,X,Y,PC,SP,P. Memory only through `Bus`.
- Live execute: `Clock` in `simple_cpu.go` dispatches enhanced vs simple opcode switches. The `Instruction` lookup table is unused.
- `bus.Bus`: 2KB RAM, PPU MMIO, APU stub, controllers, cart $8000-$FFFF.
- `ppu.PPU`: scanline state machine writing `sprScreen[240][256]`.
- nestest automation starts at `$C000`, not the reset vector.

### How it works

1. Load iNES, attach cart to bus and PPU.
2. Reset reads `$FFFC` (windowed path) or forces `$C000` (nestest harness).
3. Each master tick clocks PPU once and CPU every third tick. NMI is polled from PPU into CPU.
4. After a fixed 89342 PPU ticks, SDL presents `GetScreen()`.

### Where things live

`pkg/cpu`, `pkg/bus`, `pkg/cartridge`, `pkg/ppu`, `pkg/display`, `pkg/controller`, `pkg/apu`, `cmd/main.go`, `test_nestest_compare.go`.

### Gotchas

Dual CPU paths, nestest PC hardcodes for page-cross, sample1 CHR intercept, `$2007` increment disabled, APU never clocked, `FrameComplete` unused, mapper ID unused.
