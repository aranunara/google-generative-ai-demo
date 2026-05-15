# 新規専用 SA。既存 try-on-demo@ / デフォルト compute SA は削除・流用しない。

resource "google_service_account" "run" {
  project      = var.project_id
  account_id   = "${var.name}-run"
  display_name = "${var.name} Cloud Run runtime"

  depends_on = [google_project_service.this]
}

resource "google_service_account" "cb" {
  project      = var.project_id
  account_id   = "${var.name}-cb"
  display_name = "${var.name} Cloud Build"

  depends_on = [google_project_service.this]
}
