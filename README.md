# examples — 手元で動かすサンプル

教科書から参照している**動くコードの正本**です。
読者はこれを clone して、自分のマシンで動かします。

| ディレクトリ | 対応する章 | 動かし方 |
|---|---|---|
| `typescript/` | TypeScript の型 | `pnpm install && pnpm start` |
| `go/` | Go | `go run .` |
| `hands-on/` | ハンズオン: サービスを1本作って動かす | `make` / `docker compose up` |

## 手元で動かす

```bash
# TypeScript（Node 20 以上）
cd examples/typescript
pnpm install
pnpm start          # 5本まとめて実行される
pnpm typecheck      # 型だけ確認する

# Go（1.22 以上）
cd examples/go
go run .
go run -race .      # データ競合を検出する
```

どちらも**ターミナルに出力を出すだけ**です。サーバーも認証も要りません。

## 公開ミラー

**正本はこのリポジトリ**で、読者向けに公開ミラーへ subtree で押し出しています。

| | 場所 |
|---|---|
| 正本 | このリポジトリの `examples/`（Private） |
| 公開ミラー | https://github.com/makoto-developer/learning-it-skills-examples（Public） |

サンプルを直したら、ミラーへ反映します。

```bash
git subtree push --prefix=examples examples main
```

本文からは、この公開ミラーの clone コマンドを案内しています。

## 埋め込み実行環境は置かない

以前は CodeSandbox の埋め込みを本文に置いていましたが、**2026-08-10 に全廃**しました。

- 外部サービスに依存する。仕様変更・障害・料金体系の変更で、教材が一斉に壊れる
- Go はブラウザ内で実行できず、言語によって体験が揃わない
- 読者が**自分の環境で動かす経験**そのものが、実務では価値になる

代わりに、本文には**実際に動かした出力をそのまま載せる**方針にしました。
出力を更新する時は、手元で実行した結果を貼り直してください（想像で書かない）。

<!-- Python など、ノートブック形式が自然な言語を足す場合は、
     .ipynb を置いて「clone して jupyter で開く」導線にする。
     ブラウザ内で完結させようとしないこと。 -->

## サンプルを追加する時

1. `typescript/src/NN-topic.ts` に `export function run(): void` を持つファイルを足し、
   `src/index.ts` から呼ぶ（Go は `topic_demo.go` に `runTopic()` を足し `main.go` から呼ぶ）
2. 実行して、**その出力を本文のコードブロックに貼る**
3. `git subtree push` でミラーへ反映する

サンプルは**実行して初めて分かるもの**に絞ってください。
型定義を眺めれば分かることは、本文のコードブロックで十分です。
