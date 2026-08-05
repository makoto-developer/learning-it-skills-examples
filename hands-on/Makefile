BINARY := bin/server
IMAGE   := linkshort:dev

.PHONY: help gen build test run lint fmt docker compose-up compose-down k8s-apply k8s-delete clean

help: ## このヘルプを出す
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-14s %s\n", $$1, $$2}'

gen: ## .proto から Go のコードを生成する(生成物はコミットしない)
	@command -v protoc >/dev/null || { echo "protoc がありません: brew install protobuf"; exit 1; }
	@command -v protoc-gen-go >/dev/null || { echo "protoc-gen-go がありません: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"; exit 1; }
	@command -v protoc-gen-go-grpc >/dev/null || { echo "protoc-gen-go-grpc がありません: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"; exit 1; }
	mkdir -p gen
	protoc -I proto \
		--go_out=gen --go_opt=paths=source_relative \
		--go-grpc_out=gen --go-grpc_opt=paths=source_relative \
		proto/link/v1/link.proto

build: ## サーバーをビルドする
	go build -o $(BINARY) ./cmd/server

test: ## テストを走らせる
	go test ./...

run: ## サーバーを起動する(既定 :8080)
	go run ./cmd/server

lint: ## 静的解析
	go vet ./...

fmt: ## 整形
	gofmt -l -w .

docker: ## コンテナイメージを作る
	docker build -t $(IMAGE) .

compose-up: ## Docker Compose で起動する
	docker compose up --build -d

compose-down: ## Docker Compose を止める
	docker compose down

k8s-apply: ## Kubernetes に適用する
	kubectl apply -f k8s/

k8s-delete: ## Kubernetes から消す
	kubectl delete -f k8s/

clean: ## 生成物を消す
	rm -rf bin gen
