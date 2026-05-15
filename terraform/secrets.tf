# GEMINI_API_KEY を Secret Manager で管理。
# secret version（鍵の値）は Terraform 管理しない（state/コードに素鍵を残さない）。
# apply 後に手動投入する:
#   printf %s "<GEMINI_API_KEY>" | gcloud secrets versions add gemini-api-key \
#     --project=<PROJECT_ID> --data-file=-
resource "google_secret_manager_secret" "gemini_api_key" {
  project   = var.project_id
  secret_id = "gemini-api-key"

  replication {
    auto {}
  }

  depends_on = [google_project_service.this]
}
