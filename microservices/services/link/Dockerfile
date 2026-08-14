# ビルド用と実行用を分ける（マルチステージ）。実行用に Go のツールチェーンは要らない
FROM golang:1.25-alpine AS build
WORKDIR /src

# 依存だけ先に入れる。ソースを変えてもここのレイヤーは再利用される
COPY go.mod go.sum ./
RUN go mod download

COPY gen ./gen
COPY services/link ./services/link

# CGO を切って静的リンクにする。distroless / scratch でも動くようになる
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /out/server ./services/link/cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
# 非 root で動かす。破られた時にできることを減らす
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/server"]
