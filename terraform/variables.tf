# 環境固有の識別子は default を持たせず terraform.tfvars（gitignore）で指定する。
# repo には実値を残さない。project_number は data.google_project で自動取得。
variable "project_id" {
  description = "GCP プロジェクト ID"
  type        = string
}

variable "region" {
  description = "デプロイリージョン。Virtual Try-On API の都合で us-central1 固定推奨"
  type        = string
  default     = "us-central1"
}

variable "name" {
  description = "リソース命名ベース（Cloud Run / AR / SA のプレフィックス）"
  type        = string
  default     = "genai-demo"
}

variable "github_owner" {
  description = "連携先 GitHub オーナー"
  type        = string
  default     = "aranunara"
}

variable "github_repo" {
  description = "連携先 GitHub リポジトリ名"
  type        = string
  default     = "google-generative-ai-demo"
}

variable "branch_regex" {
  description = "ビルド対象ブランチ（正規表現）"
  type        = string
  default     = "^main$"
}

variable "vto_model" {
  description = "Virtual Try-On モデル ID（Cloud Run の VTO_MODEL env）"
  type        = string
  default     = "virtual-try-on-preview-08-04"
}

variable "gcs_uri" {
  description = "生成物の GCS 保存先 URI（例: gs://try-on-generated）。空なら GCS_URI env を付与しない"
  type        = string
  default     = ""
}

variable "gcs_bucket" {
  description = "ランタイム SA に objectAdmin を付与する GCS バケット名（gs:// なし）。空なら付与しない"
  type        = string
  default     = ""
}
