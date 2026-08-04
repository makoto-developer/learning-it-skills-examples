# examples — 実行できるサンプル

第4部（第11章 TypeScript / 第12章 Go）から埋め込むサンプルコードの**正本**です。
CodeSandbox 上のコードが古くなった時は、こちらを正として直してください。

| ディレクトリ | 対応する章 | 実行環境 |
|---|---|---|
| `typescript/` | 第11章 TypeScript の型 | ブラウザ内（読者が匿名のまま編集・再実行できる） |
| `go/` | 第12章 Go | VM（読者は実行結果を見るだけ。編集には Fork が必要） |

## 手元で動かす

```bash
# TypeScript
cd examples/typescript
pnpm install --ignore-workspace
pnpm start

# Go
cd examples/go
go run .
go run -race .   # データ競合を検出する
```

## CodeSandbox に載せる手順

サンドボックスの作成はブラウザ上の操作が必要なので、そこだけ手作業です。

1. `examples/` を含むリポジトリを GitHub に push する
2. CodeSandbox で **Import from GitHub** を実行し、`examples/typescript` と `examples/go`
   をそれぞれサンドボックスにする
3. エディタの **Share → Copy embed code** から URL を取り出す
4. `content/sandboxes.ts` の `embedUrl` に貼る（コメントアウトを外す）

`embedUrl` が未設定の間、本文には「準備中」と手元での動かし方が表示されます。
ページが壊れることはないので、埋めるのは後からで構いません。

### 公開範囲の注意

埋め込みは**読者がログインなしで見られる**必要があります。
CodeSandbox 側の公開設定を Public にしてください。
Private のままだと、読者には何も表示されません。

### Go は編集できない

Go は VM Sandbox でしか動かず、その埋め込みでは**読者がコードを書き換えられません**
（書き換えるには CodeSandbox 上で Fork が必要 = ログインが要る）。

読者に書き換えさせたい演習を Go で作る場合は、
[Go Playground](https://go.dev/play/) の共有リンクを併用してください。
ただし Playground では `-race` が使えないため、データ競合の演習は CodeSandbox 側に残します。

## サンプルを追加する時

1. `typescript/src/NN-topic.ts` に `export function run(): void` を持つファイルを足し、
   `src/index.ts` から呼ぶ（Go は `topic_demo.go` に `runTopic()` を足し `main.go` から呼ぶ）
2. 本文の該当箇所に `<Sandbox name="..." module="/src/NN-topic.ts" />` を置く

サンプルは**実行して初めて分かるもの**に絞ってください。
型定義を眺めれば分かることは、本文のコードブロックで十分です。
