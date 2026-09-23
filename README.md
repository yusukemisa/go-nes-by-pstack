# go-nes-by-pstack
<<<<<<< HEAD
Go NES emulator with nestest-backed CPU
=======

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
>>>>>>> 3ddb09c (NES の Console と nestest 照合を追加し、参考実装なしで CPU を検証できるようにする。)
