# --- 1段目: ビルドだけを行う。Go ツールチェインは成果物に持ち込まない ---
FROM golang:1.24-alpine AS build

WORKDIR /src

# 依存の解決だけを先に済ませると、コードを直してもこの層のキャッシュが効く
COPY go.mod ./
RUN go mod download

COPY . .

# CGO を切って静的リンクにする。これで libc の無いイメージでも動く
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# --- 2段目: バイナリ1つだけの最小イメージ。シェルもパッケージマネージャも無い ---
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/server /server

# root で動かさない。コンテナを破られても、できることを減らしておく
USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/server"]
