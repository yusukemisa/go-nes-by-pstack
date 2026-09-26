# Design synthesis

## Problem

Build a new Go NES emulator at `/Users/misawa/dev/go-nes-by-pstack`, using `/Users/misawa/dev/go-nes-hello` as a source of hardware facts, not as a file dump. The reference CPU works on nestest but splits execute across two switches, hardcodes page-cross PCs, and leaves an unused OLC table. The PPU path includes sample1 intercepts. A new project must be smaller and testable without SDL.

## Usage

```
nes, err := nes.Open("game.nes")
nes.Reset()
for running {
    nes.StepFrame()
    frame := nes.Frame() // [240][256]uint8 NES palette indices
}
```

Headless nestest:

```
c := nes.Open("testdata/nestest.nes")
c.PowerOnNestest() // PC=$C000, P=$24, CYC=7, PPU 21 dots
line := c.TraceLine()
c.StepInstruction()
```

## Shape (base)

Pick instruction-step CPU plus catch-up PPU. Nestest compares state at instruction boundaries. A cycle-accurate master loop is still available as `Console.Clock` for the frame path (3 PPU dots per CPU cycle).

Domain types:

- `Cartridge` parsed from iNES bytes at the file boundary. Mapper 0 only. No filename hacks.
- `CPU` registers plus `[256]Opcode` table (name, bytes, base cycles, execute). One execute path.
- `Bus` owns 2KB RAM and address decode. Devices do not peek RAM.
- `PPU` owns VRAM, OAM, screen buffer, scanline/dot. NMI flag polled by Console.
- `Console` is the only public type. SDL is `cmd/nes` only.

Invariants in types: `Opcode` table is the ISA. Illegal combinations of mapper/PRG size fail at parse. Page-cross cycles from address math, not nestest PCs.

## Synthesis decision

Base is instruction-step + opcode table (candidate A). Graft from cycle-accurate candidate B the 3:1 `Clock` used by the visible frame loop and PPU state machine (`Visible` / `PostRender` / `VBlank` / `PreRender`). Reject dual enhanced/simple CPU, sample1 intercepts, and SDL inside packages.

## Tradeoffs

- We accept mapper 0 only in exchange for a nestest-green CPU.
- We accept no APU audio in exchange for a thin bus stub at $4000-$4017.
- We accept PPM or SDL at the CLI in exchange for tests that never link CGO.

## Alternatives rejected

- Pure cycle-accurate CPU (`Clock` every cycle with countdown only). Hides less from nestest; same as the reference.
- Copy go-nes-hello wholesale. Carries dead files and ROM-specific hacks.

## Next implementation step

Cartridge parse + CPU opcode table + nestest log compare against `testdata/nestest.log`.
