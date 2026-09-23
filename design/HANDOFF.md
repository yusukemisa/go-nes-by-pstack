# 次セッションへの引き継ぎ

ドラフト PR: https://github.com/yusukemisa/go-nes-by-pstack/pull/1  
ブランチ: `feat/nes-emulator`  
参考実装（読み取り専用）: `/Users/misawa/dev/go-nes-hello`

## いま通っているもの

- `go test ./internal/nes/ -count=1`  
  `testdata/nestest.log` の 8991 行について、実行前の PC / A / X / Y / P / SP / PPU scanline,dot / CYC が一致する。
- `go run ./cmd/nes testdata/nestest.nes 1` が `frame.ppm` を書く。nestest は描画をほぼ出さないので画面はほぼ黒で正しい。

公開 API は `internal/nes.Console`。`Open` / `Reset` / `PowerOnNestest` / `StepInstruction` / `Clock` / `StepFrame` / `Frame` / `TraceRegs`。

## 意図した形と、まだ届いていない点

設計は `design/SKETCH.md`。命令単位の CPU と追いつき PPU、フレームは 3:1 の `Clock`。ISA は `[256]Opcode` 表が本命。

現状の CPU は go-nes-hello から移植した二重パスのまま。

- 実行: `internal/cpu/simple_cpu.go` の `Clock` が `enhanced_cpu.go` と簡易スイッチに分岐する
- ページクロスに nestest 向け PC ハードコードが残っている
- `Instruction` ルックアップは未使用

PPU はほぼ丸コピー。`alphabet_stub.go` でコンパイルを通している。sample1 横取りと `fmt.Printf` デバッグが残る。

## 次にやる順

1. CPU を表駆動の一本に寄せる。nestest が赤くなったら止める。
2. ページクロスをアドレス計算にする。PC ハードコードを消す。
3. PPU から sample1 / alphabet / 余剰 Printf を削除する。nestest は緑のまま。
4. ウィンドウ表示は `cmd/nes` だけ。テストは CGO なし。
5. マッパー 0 以外と APU 音声は後回し。

## 触らないもの

- 参考実装への書き込み
- nestest の開始状態: PC `$C000`, P `$24`, CYC 7, PPU `(0, 21)`
- 照合は逆アセンブルの空白ではなくレジスタと CYC（`internal/nes/nestest_test.go`）
