# examples — 実行できるサンプル

第4部（TypeScript の型 / Go）から埋め込むサンプルコードの**正本**です。
公開ミラーが古くなった時は、こちらを正として直し、subtree で押し出してください。

| ディレクトリ | 対応する章 | 実行環境 |
|---|---|---|
| `typescript/` | TypeScript の型 | **ブラウザ内で実行**（読者はログイン不要で編集・再実行できる） |
| `go/` | Go | **埋め込みでは実行不可**。手元 or Go Playground |

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

## 公開ミラーと CodeSandbox

**このディレクトリの正本はこのリポジトリ**で、公開ミラーへ subtree で押し出している。

| | 場所 |
|---|---|
| 正本 | このリポジトリの `examples/`（Private） |
| 公開ミラー | https://github.com/makoto-developer/learning-it-skills-examples（Public） |

サンプルを直したら、ミラーへ反映する。

```bash
git subtree push --prefix=examples examples main
```

埋め込みは**公開ミラーを CodeSandbox に直接読ませる**形にしてある。
サンドボックスを作る操作もアカウントも不要で、URL だけで完結する。

```
https://codesandbox.io/embed/github/makoto-developer/learning-it-skills-examples/tree/main/typescript
```

`content/books/intern/sandboxes.ts` の `embedUrl` がこれ。
`module` パラメータで最初に開くファイルを指定できる（`<Sandbox module="/src/01-unknown.ts" />`）。

### Go は埋め込みでは動かない

ブラウザ内サンドボックスは Go を実行できない（**実機で確認済み**。
コードは色付きで表示されるが、プレビューは空のまま）。
実行するには VM Sandbox が必要で、その作成にはログインが要る。

そのため Go の `embedUrl` は**意図的に空のまま**にしてある。
本文には手元での動かし方と公開ミラーへのリンクが出る。
ブラウザで書き換えて試させたい場合は、本文から
[Go Playground](https://go.dev/play/) へ誘導する（`-race` は使えない）。

## サンプルを追加する時

1. `typescript/src/NN-topic.ts` に `export function run(): void` を持つファイルを足し、
   `src/index.ts` から呼ぶ（Go は `topic_demo.go` に `runTopic()` を足し `main.go` から呼ぶ）
2. 本文の該当箇所に `<Sandbox name="..." module="/src/NN-topic.ts" />` を置く

サンプルは**実行して初めて分かるもの**に絞ってください。
型定義を眺めれば分かることは、本文のコードブロックで十分です。
