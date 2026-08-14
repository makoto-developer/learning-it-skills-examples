# 手元で通しで動かすための入口。make help で一覧が出る。
.DEFAULT_GOAL := help
SHELL := /bin/bash

CLUSTER   := learning
DATABASE  := projects/learning-project/instances/learning-instance/databases/links
EMULATOR  := gcr.io/cloud-spanner-emulator/emulator:latest
# kind から公開するホスト側のポート。8080 は他と衝突しやすいので避けている
HOST_PORT := 18080

.PHONY: help
help: ## このヘルプを出す
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

## ---- 手元で直接動かす（Kubernetes を使わない） ----

.PHONY: emulator
emulator: ## Spanner エミュレータを docker で起動する
	docker rm -f spanner-emulator 2>/dev/null || true
	docker run -d --name spanner-emulator -p 9010:9010 -p 9020:9020 $(EMULATOR)
	@echo "エミュレータの起動を待っています..."
	@until curl -sf http://localhost:9020/v1/projects/x/instances >/dev/null 2>&1; do sleep 1; done
	@echo "起動しました"

.PHONY: schema
schema: ## Terraform で Spanner のインスタンスとテーブルを作る
	cd terraform && GOOGLE_OAUTH_ACCESS_TOKEN=dummy terraform init -input=false
	cd terraform && GOOGLE_OAUTH_ACCESS_TOKEN=dummy terraform apply -auto-approve

.PHONY: generate
generate: ## .proto から Go と TypeScript のコードを生成する
	buf lint
	buf generate

.PHONY: run-link
run-link: ## link サービス（Go / gRPC）を起動する
	SPANNER_EMULATOR_HOST=localhost:9010 SPANNER_DATABASE=$(DATABASE) \
		go run ./services/link/cmd/server

.PHONY: run-bff
run-bff: ## BFF（TypeScript / HTTP）を起動する
	cd services/bff && LINK_SERVICE_URL=http://localhost:8080 pnpm start

.PHONY: test
test: ## Go のテストを走らせる
	go test ./...

## ---- Kubernetes（kind）で動かす ----

.PHONY: cluster
cluster: ## kind でクラスタを作る
	kind create cluster --config kind/cluster.yaml || true
	kubectl config use-context kind-$(CLUSTER)

.PHONY: images
images: ## コンテナイメージを作り、kind に読み込ませる
	docker build -f services/link/Dockerfile -t learning/link:dev .
	docker build -f services/bff/Dockerfile  -t learning/bff:dev .
	kind load docker-image learning/link:dev learning/bff:dev --name $(CLUSTER)

.PHONY: deploy
deploy: ## クラスタに載せる
	kubectl apply -f k8s/spanner-emulator.yaml
	kubectl rollout status deploy/spanner-emulator --timeout=180s
	$(MAKE) schema-in-cluster
	kubectl apply -f k8s/link.yaml -f k8s/bff.yaml
	kubectl rollout status deploy/link --timeout=180s
	kubectl rollout status deploy/bff  --timeout=180s
	@echo "http://localhost:$(HOST_PORT) で使えます"

.PHONY: schema-in-cluster
schema-in-cluster: ## クラスタ内のエミュレータに対して Terraform を流す
	kubectl port-forward svc/spanner-emulator 9020:9020 & echo $$! > /tmp/pf.pid; sleep 3
	cd terraform && GOOGLE_OAUTH_ACCESS_TOKEN=dummy terraform init -input=false && \
		GOOGLE_OAUTH_ACCESS_TOKEN=dummy terraform apply -auto-approve || true
	kill $$(cat /tmp/pf.pid) 2>/dev/null || true

.PHONY: smoke
smoke: ## 動いているかを一通り叩いて確かめる
	@set -e; \
	echo "作成:"; \
	KEY=$$(curl -sf -X POST localhost:$(HOST_PORT)/links -H 'content-type: application/json' \
	  -d '{"url":"https://example.com/hello"}' | python3 -c 'import sys,json;print(json.load(sys.stdin)["key"])'); \
	echo "  key=$$KEY"; \
	echo "一覧:"; curl -sf 'localhost:$(HOST_PORT)/links?page_size=5' | head -c 200; echo; \
	echo "転送:"; curl -sf -o /dev/null -w '  %{http_code} -> %{redirect_url}\n' localhost:$(HOST_PORT)/$$KEY; \
	echo "不正な URL:"; curl -s -o /dev/null -w '  %{http_code}\n' -X POST localhost:$(HOST_PORT)/links \
	  -H 'content-type: application/json' -d '{"url":"javascript:alert(1)"}'

.PHONY: logs
logs: ## link サービスのログを追う
	kubectl logs -l app=link -f --tail=50

.PHONY: down
down: ## 後片付け（クラスタとエミュレータを消す）
	kind delete cluster --name $(CLUSTER) || true
	docker rm -f spanner-emulator 2>/dev/null || true
