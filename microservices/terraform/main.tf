# Spanner のインスタンスとテーブルを Terraform で作る。
#
# 学習用にエミュレータへ向けているが、本番の GCP に向ける時に変えるのは
# provider の設定だけで、resource の書き方は同じになる。

terraform {
  required_version = ">= 1.9"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
  }
}

variable "spanner_endpoint" {
  description = "Spanner の管理 API。エミュレータの REST ポートを指す"
  type        = string
  default     = "http://localhost:9020/v1/"
}

variable "project" {
  type    = string
  default = "learning-project"
}

provider "google" {
  project = var.project
  # 本番ではこの行を消す。認証は Application Default Credentials に任せる
  spanner_custom_endpoint = var.spanner_endpoint
}

resource "google_spanner_instance" "main" {
  name         = "learning-instance"
  config       = "emulator-config" # 本番では regional-asia-northeast1 など
  display_name = "learning"
  num_nodes    = 1
}

resource "google_spanner_database" "links" {
  instance = google_spanner_instance.main.name
  name     = "links"

  ddl = [
    <<-SQL
      CREATE TABLE links (
        key        STRING(16)   NOT NULL,
        url        STRING(2048) NOT NULL,
        created_at TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp = true),
      ) PRIMARY KEY (key)
    SQL
    ,
    # 一覧は作成日時の降順で引くので、その順で並んだ索引を用意する
    "CREATE INDEX links_by_created_at ON links (created_at DESC)",
  ]

  # 学習用なので消せるようにしておく。本番では true のままにする
  deletion_protection = false
}

output "database" {
  value = "projects/${var.project}/instances/${google_spanner_instance.main.name}/databases/${google_spanner_database.links.name}"
}
