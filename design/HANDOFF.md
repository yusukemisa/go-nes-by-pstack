# 次セッションへの引き継ぎ

PR: https://github.com/yusukemisa/go-nes-by-pstack/pull/1  
ブランチ: `feat/nes-emulator`  
参考実装（読み取り専用）: `/Users/misawa/dev/go-nes-hello`

## いま通っているもの

- `CGO_ENABLED=0 go test ./internal/nes/ -count=1`  
  `testdata/nestest.log` の 8991 行について、実行前の PC / A / X / Y / P / SP / PPU scanline,dot / CYC が一致する。
- `nes <rom.nes>` が SDL のウィンドウを開く。SDL は `cmd/nes` だけ。
- `nes -ppm <rom.nes> [frames]` が `frame.ppm` を書く。

公開 API は `internal/nes.Console`。`Open` / `Reset` / `PowerOnNestest` / `StepInstruction` / `Clock` / `StepFrame` / `Frame` / `TraceRegs`。

## 届いた形

- 実行は `executeInstructionEnhanced` の一本。簡易スイッチは削除済み。
- 基本サイクルは `baseCycles` の `[256]uint8`。ページクロスはオペランドの実効アドレスから計算する。
- PPU から sample1、alphabet、`fmt.Printf` は削除済み。`advanceTiming` と NMI は残る。
- 実行本体はまだ一つの switch です。`Instruction.Operate` は空のままです。

## まだやらないもの

- マッパー 0 以外。
- APU の音。
- 未実装 opcode は panic する。nestest はこの経路を通らない。

## 触らないもの

- 参考実装への書き込み。
- nestest の開始状態。PC `$C000`、P `$24`、CYC 7、PPU `(0, 21)`。
- 照合は逆アセンブルの空白ではなくレジスタと CYC（`internal/nes/nestest_test.go`）。
