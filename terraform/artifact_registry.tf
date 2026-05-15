# genai-demo 専用の Docker リポジトリ。
# 共有 cloud-run-source-deploy は Terraform 管理外（他サービスと共有のため触らない）。
resource "google_artifact_registry_repository" "genai_demo" {
  project       = var.project_id
  location      = var.region
  repository_id = var.name
  description   = "Container images for ${var.name}"
  format        = "DOCKER"

  depends_on = [google_project_service.this]
}
