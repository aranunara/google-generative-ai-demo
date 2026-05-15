terraform {
  required_version = ">= 1.5"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
  }

  # state は GCS リモート backend。bucket/prefix は repo に実値を残さないため
  # 部分設定にし、init 時に backend.hcl（gitignore）で注入する:
  #   terraform init -backend-config=backend.hcl
  # state バケットは Terraform 管理外（chicken-and-egg 回避のため事前に手動作成）:
  #   gcloud storage buckets create gs://<TF_STATE_BUCKET> \
  #     --project=<PROJECT_ID> --location=us-central1 \
  #     --uniform-bucket-level-access
  backend "gcs" {}
}
