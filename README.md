# go-nes-by-pstack

Go NES emulator. Mapper 0, nestest-checked CPU, PPU frame to a PPM from `cmd/nes`.

## Prove it

```
go test ./internal/nes/ -count=1
go run ./cmd/nes testdata/nestest.nes 2
```

## Layout

- `internal/nes` Console
- `internal/cpu` 6502
- `internal/ppu` PPU
- `internal/bus` CPU map
- `internal/cartridge` iNES
- `design/HANDOFF.md` 次セッション向けの現状と未着手
