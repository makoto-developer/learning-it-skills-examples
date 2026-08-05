# hands-on — 短縮 URL サービス

第28章「ハンズオン: サービスを1本作って動かす」で作るサービスの完成形です。
章を読みながら自分で書いてもいいですし、動かないときの答え合わせに使っても構いません。

Go の標準ライブラリだけで動きます。`go get` は不要です。

## 5分で動かす

```bash
cd examples/hands-on
make test          # go test ./...
make run           # :8080 で起動
```

別のターミナルから叩きます。

```bash
# 作る
curl -s -X POST localhost:8080/v1/links \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://go.dev/doc/effective_go"}'
# => {"key":"aZ3k9Qw","url":"https://go.dev/doc/effective_go","created_at":"..."}

# 引く
curl -s localhost:8080/v1/links/aZ3k9Qw

# 一覧(ページネーション)
curl -s 'localhost:8080/v1/links?page_size=2'

# 転送
curl -i localhost:8080/r/aZ3k9Qw   # => 302 Location: https://go.dev/...
```

## API

| メソッド | パス | 用途 |
|---|---|---|
| POST | `/v1/links` | 短縮リンクを作る。ボディは `{"url": "..."}` |
| GET | `/v1/links/{key}` | 1件引く |
| GET | `/v1/links?page_size=&page_token=` | 一覧。`next_page_token` が空なら最終ページ |
| GET | `/r/{key}` | 転送先へ 302 |
| GET | `/healthz` | liveness 用 |
| GET | `/readyz` | readiness 用 |

## ディレクトリ

```
proto/link/v1/link.proto   設計の成果物。この形を先に決めてから実装する
cmd/server/main.go         起動・シグナル処理
cmd/server/router.go       HTTP のルーティングと JSON の入出力
internal/service/          業務ロジック。HTTP も gRPC も知らない
internal/store/            保存先。Store インターフェース + インメモリ実装
k8s/                       Deployment と Service
```

依存の向きは `cmd → service → store` の一方通行です。逆向きの import はありません。

## .proto から生成する

生成コードは**コミットしていません**。`.proto` が正本で、生成物は誰でも作り直せるからです。
コミットすると、`.proto` を直したのに生成物を忘れた差分がレビューをすり抜けます。

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
make gen      # gen/ に出力される(.gitignore 済み)
```

このサンプルは生成物が無くてもビルドできるように、`internal/` を素の Go の型で書いてあります。
gRPC に載せ替えるときは、生成された `LinkServiceServer` を実装する薄い層を足し、
その中から `internal/service` を呼びます。ロジックは書き直しません。

## コンテナと Kubernetes

```bash
make docker            # マルチステージビルド。最終段は distroless + 非 root
make compose-up        # http://localhost:8080
make compose-down

make k8s-apply         # probe と resource limits 入り
kubectl get pods -l app=linkshort
kubectl logs -l app=linkshort -f
make k8s-delete
```

ローカルの kind / minikube に載せる場合は、先にイメージをクラスタへ読み込ませてください。

```bash
kind load docker-image linkshort:dev
# minikube なら: minikube image load linkshort:dev
```

## 手順（章と対応）

| 手順 | やること | 対応する章 |
|---|---|---|
| 1 | `.proto` で API を決める | 第13章 protobuf と gRPC |
| 2 | `store.Store` を切る | 第12章 Go / 第20章 テストを書く |
| 3 | `service` に業務ロジックを書く | 第12章 Go |
| 4 | table-driven test を書く | 第20章 テストを書く |
| 5 | HTTP で公開する | 第9章 Web と HTTP |
| 6 | Dockerfile を書く | 第15章 Kubernetes |
| 7 | マニフェストを書く | 第15章 Kubernetes |
| 8 | CI を通す | 第22章 CI/CD |
| 9 | 構造化ログを見る | 第23章 監視とオンコール |

## 既知の制限（わざと残しています）

- **プロセス内メモリなので再起動で消える。** 永続化は第14章の練習問題です
- **認証がない。** 誰でも作れて誰でも一覧できます
- **転送回数を数えていない。** メトリクスを足す練習に向いています
