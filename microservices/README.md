# microservices — 通しで作るマイクロサービス

教科書の[通しで作る（応用）](https://learning-it-skills.makoto-developer.net/chapters/hands-on-microservices/)で使う一式です。

```
ブラウザ / curl
   │ HTTP
   ▼
[ BFF ]         TypeScript。HTTP で受けて gRPC に変換する
   │ gRPC
   ▼
[ link ]        Go。短縮リンクの業務ロジック
   │ Spanner API
   ▼
[ Spanner ]     学習用はエミュレータ。本番は Cloud Spanner
```

すべて **Kubernetes（kind）の上**で動き、**スキーマは Terraform** で作ります。

## 必要な環境

### 必須

| ツール | 確認済みバージョン | 用途 |
|---|---|---|
| Docker | 29.x（Desktop / Colima / OrbStack いずれでも） | イメージのビルド、kind の土台 |
| kind | 0.32 | 手元に Kubernetes クラスタを作る |
| kubectl | 1.36 | クラスタの操作 |
| Terraform | 1.15 | Spanner のインスタンスとテーブルを作る |
| Go | 1.25 | link サービス |
| Node.js | 24 | BFF |
| pnpm | 11 | BFF の依存管理 |
| buf | 1.72 | `.proto` からコードを生成する |
| GNU Make | 3.81+（macOS 標準で可） | 入口 |
| Python 3 | 3.9+ | `make smoke` の JSON 整形 |

### あると便利

| ツール | 用途 |
|---|---|
| grpcurl | gRPC を直接叩いて確かめる |

### マシンの条件

```
□ メモリ: Docker に 4GB 以上を割り当てられること（クラスタ + エミュレータで実測 1.5GB 程度）
□ ディスク: イメージとクラスタで 3GB 程度
□ CPU アーキテクチャ: arm64 / amd64 のどちらでも動く（動作確認は arm64 / macOS）
□ ネットワーク: 初回のみイメージと依存の取得に外部通信が要る
```

### 使うポート

| ポート | 用途 |
|---|---|
| **18080** | kind から公開する BFF の入口。**8080 は他ツールと衝突しやすいので避けている** |
| 9010 / 9020 | Spanner エミュレータ（`make emulator` で手元に立てた時のみ） |
| 8080 / 3000 | Kubernetes を使わず直接動かす時の link / BFF |

既に使っているポートがある場合は、`kind/cluster.yaml` の `hostPort` と
`Makefile` の `HOST_PORT` を合わせて変えてください。

### 必要ないもの

```
✕ GCP のアカウント
✕ gcloud コマンド
✕ クレジットカード
```

**課金は一切発生しません。** Spanner はエミュレータ（`gcr.io/cloud-spanner-emulator/emulator`）を
Docker で動かします。

## 動かす

### A. Kubernetes に載せる（本番に近い形）

```bash
make cluster    # kind でクラスタを作る
make images     # イメージをビルドして kind に読み込ませる
make deploy     # エミュレータ → Terraform → link → bff の順に載せる
make smoke      # 一通り叩いて確かめる
```

`make deploy` が終わると <http://localhost:18080> で使えます。

```bash
curl -X POST localhost:18080/links \
  -H 'content-type: application/json' \
  -d '{"url":"https://example.com/very/long/url"}'
# {"key":"9jjbqVZ","url":"https://example.com/very/long/url"}

curl -i localhost:18080/9jjbqVZ     # 302 で転送される
curl 'localhost:18080/links?page_size=5'
```

後片付けは `make down` です。

### B. Kubernetes を使わず、手元で直接動かす

デバッグはこちらのほうが速いです。ターミナルを3つ使います。

```bash
make emulator   # 1つ目: Spanner エミュレータ
make schema     # スキーマを作る（Terraform）

make run-link   # 2つ目: Go の gRPC サーバー（:8080）
make run-bff    # 3つ目: TypeScript の BFF（:3000）
```

```bash
curl -X POST localhost:3000/links -H 'content-type: application/json' -d '{"url":"https://example.com"}'
grpcurl -plaintext localhost:8080 list          # gRPC を直接見る
```

## テスト

```bash
make test              # 保存先をメモリに差し替えた単体テスト。数秒で終わる
make test-integration  # Spanner エミュレータに対する結合テストも走らせる
```

`make test` は**エミュレータが無くても通ります**。
Spanner に対するテストは `SPANNER_EMULATOR_HOST` が無ければスキップされるためです。

| 何をテストしているか | どこ |
|---|---|
| URL の検証、キーの衝突と再採番、ページング、エラーコードの対応 | `services/link/internal/server/server_test.go` |
| ページトークンの往復と、中身が透けないこと | `services/link/internal/server/token_internal_test.go` |
| Spanner の commit timestamp、重複、並び順 | `services/link/internal/store/spanner_integration_test.go` |

**メモリ実装で通っても Spanner で通るとは限りません**（型の制約、commit timestamp、
クエリの書き方）。そこは結合テストで押さえています。

## ディレクトリ

```
proto/link/v1/link.proto   API の定義。ここが2つのサービスの契約
buf.yaml / buf.gen.yaml    生成の設定（Go と TypeScript の両方を出す）
gen/                       Go の生成コード（コミットしている。理由は下記）
services/link/             Go: gRPC サーバー + Spanner（テストもここ）
services/bff/              TypeScript: HTTP → gRPC
terraform/                 Spanner のインスタンスとテーブル
k8s/                       Kubernetes のマニフェスト
kind/cluster.yaml          クラスタの定義
```

`.proto` を変えたら `make generate` を実行してください。

<!-- 生成コードは通常コミットしないが、この教材では「clone してすぐ動く」を優先している。
     実務では CI で生成し、buf.build などのレジストリに publish する形が多い -->

## エミュレータと本番の違い

**エミュレータで動いても、本番で同じとは限りません。** 主な差は次のとおりです。

| | エミュレータ | 本番（Cloud Spanner） |
|---|---|---|
| 認証 | 要らない（`SPANNER_EMULATOR_HOST` を設定するだけ） | Workload Identity や ADC が要る |
| 性能 | 単一プロセス。負荷試験の参考にならない | ノード数に応じてスケールする |
| 分散の影響 | ホットスポットが再現しない | 主キーの偏りが性能に直結する |
| バックアップ / レプリカ | 無い | ある |
| 課金 | 無料 | ノード時間とストレージで課金 |

本番へ向ける時に変えるのは、次の2箇所だけです。

```
1. k8s/link.yaml の SPANNER_EMULATOR_HOST を消す
2. terraform/main.tf の spanner_custom_endpoint を消し、config を実際のリージョンにする
```

**アプリのコードは1行も変わりません。** それが Spanner クライアントを使う利点です。

## GKE に載せる場合

この教材は kind までで完結しますが、GKE でも動く構成にしてあります。
違いは3点です。

```
1. イメージを Artifact Registry に push する（kind load の代わり）
2. Workload Identity を設定し、SPANNER_EMULATOR_HOST を外す
3. Service を NodePort から LoadBalancer か Ingress に変える
```

**GKE と Spanner は課金されます。** 試す場合は、必ず費用を確認し、
終わったら `terraform destroy` とクラスタの削除まで行ってください。
