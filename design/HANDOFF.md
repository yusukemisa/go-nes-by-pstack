# 次セッションへの引き継ぎ

PR: https://github.com/yusukemisa/go-nes-by-pstack/pull/1  
ブランチ: `feat/nes-emulator`  
参考実装（読み取り専用）: `/Users/misawa/dev/go-nes-hello`

確認用 ROM はローカルの `roms/` にある。`.gitignore` の `/roms/` でコミットしない。`testdata/nestest.nes` だけはテストが読むので追跡したまま。

## いま通っているもの

`CGO_ENABLED=0 go test ./internal/nes/ -count=1` が、`testdata/nestest.log` の 8991 行について、実行前の PC、A、X、Y、P、SP、PPU の scanline と dot、CYC を一致させる。

ウィンドウは `cmd/nes` だけが SDL を使う。この環境ではビルドの前に `export SDKROOT="$(xcrun --show-sdk-path)"` が要る。

```
go run ./cmd/nes roms/sample1.nes
go run ./cmd/nes -ppm testdata/nestest.nes 1
```

Z が A、X が B、矢印が十字キー、Enter が START、Shift が SELECT、Escape で閉じる。`roms/test_ppu_read_buffer.nes` はマッパー 3 なので、今の NROM 読みでは確認に使わない。

公開 API は `internal/nes.Console`。`Open`、`Reset`、`PowerOnNestest`、`StepInstruction`、`Clock`、`StepFrame`、`Frame`、`TraceRegs`。

実行は `executeInstructionEnhanced` の一本。基本サイクルは `baseCycles`。ページクロスの PC 比較と、PPU の sample1、alphabet、`fmt.Printf` は削除済み。

## 残作業

この順でやる。各項目のあとで nestest を再実行し、赤くなったらそこで止める。

1. 完了。`$2007` の読みは `CPURead` の `PPUDATA` で `Increment` する。`TestPPUDataReadAdvancesByOne` と `TestPPUDataReadAdvancesByThirtyTwo` が赤から緑になった。
2. 完了。空の `lookup` と `Disassemble` と `setupInstructionTable` は消した。実行は `executeInstructionEnhanced` のまま。`[256]Opcode` へは移していない。
3. ページクロスの加算を一箇所にする。`willCrossPage` と、`enhanced_cpu.go` 内の `addPageCross` 相当の `totalCycles++` が両方ある。LDA abs,Y などは実行中の加算だけが効く。`willCrossPage` を真にすると二重になる。実行側へ寄せて `willCrossPage` を消すと、nestest の 4767 行で CYC が 13836 ではなく 13837 になり、PPU が 121,247 ではなく 121,250 になった。その差分は戻してある。次は、加算を足した命令を `willCrossPage` が担当していた opcode だけに限る。
4. マッパー 0 以外。`Cartridge.Mapper` は保存するが、`CPURead` は 16KB ならミラー、32KB なら直マップしかしない。未知のマッパーを `LoadCartridge` で拒否するか、実装する。
5. APU の音。`internal/apu` はレジスタ配列だけで、`Console.Clock` は APU を進めない。`bus.go` の `$4017` 読みは常にコントローラ 2 を返し、その分岐の `apu.Read` には到達しない。
6. 未実装 opcode。`executeInstructionEnhanced` の default は panic する。nestest はこの経路を通らない。

キー入力が `$4016` の読みに届くことは、パッドを読む ROM ではまだ確認していない。

## 触らないもの

- `/Users/misawa/dev/go-nes-hello` への書き込み。
- nestest の開始状態。PC `$C000`、P `$24`、CYC 7、PPU `(0, 21)`。
- 照合の対象。逆アセンブルの空白ではなく、レジスタと CYC（`internal/nes/nestest_test.go`）。
