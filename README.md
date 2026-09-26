# go-nes-by-pstack

Mapper 0 の NES エミュレータです。CPU は nestest のログと照合します。画面は `cmd/nes` のウィンドウです。

## 動かし方

ウィンドウを開きます。Z が A、X が B、矢印が十字キー、Enter が START、Shift が SELECT です。Escape かウィンドウを閉じると終わります。

```
go run ./cmd/nes path/to/game.nes
```

`frame.ppm` だけ書くときは `-ppm` を付けます。フレーム数の既定は 2 です。nestest はほとんど描画しないので、画像はほぼ黒です。

```
go run ./cmd/nes -ppm testdata/nestest.nes 1
```

テストに CGO は要りません。

```
CGO_ENABLED=0 go test ./internal/nes/ -count=1
```

## 構成

- `internal/nes` が `Console`
- `internal/cpu` が 6502
- `internal/ppu` が PPU
- `internal/bus` が CPU のメモリマップ
- `internal/cartridge` が iNES
- `cmd/nes` が SDL ウィンドウ

マッパー 0 以外と APU の音はまだありません。
