# Grounding: go-nes-hello (read-only reference)

Source: `/Users/misawa/dev/go-nes-hello`

## What works

- nestest.nes log match via a **separate** harness (`test_nestest_compare.go`), not `cmd/main.go`.
- CPU: instruction-cycle countdown; most opcodes in a huge switch (`enhanced_cpu.go`); leftover OLC-style table is unused.
- PPU: cycle `Clock` with scanline states; SDL display of 256x240 palette indices.
- Cart: iNES parse, NROM-like 16/32KB PRG, CHR read. Mapper ID unused.
- APU: register stub, never clocked.

## Do not copy

- Dual CPU execute paths (`simple` vs `enhanced`), nestest PC hardcodes for page-cross.
- sample1.nes / alphabet intercept / deferred nametable hacks.
- Dead `instructions.go` + package-level `addr_abs`.
- `$2007` increment commented out.
- Multiple `package main` at repo root.
- Frame loop that ignores `FrameComplete` plus ROM-specific comments.

## Constraints for go-nes-by-pstack

- Greenfield Go NES emulator. Reference is a source of facts, not a file to clone.
- Predicate: CPU nestest log matches `rom/nestest.log` for the official automated run (PC starts at $C000).
- Mapper 0 (NROM) only in v1. No audio required.
- SDL is optional display; tests must run headless.
- Public surface: load ROM, reset, step, read framebuffer.

## Domain facts

- Master clock: PPU 3 dots per CPU cycle.
- CPU 6502: A,X,Y,PC,SP,P; memory only via bus.
- iNES: 16-byte header, PRG 16KB units, CHR 8KB units, mapper nibble, mirroring.
- PPU: 341 dots x 262 scanlines; NMI at scanline 241 dot 1 if enabled.
