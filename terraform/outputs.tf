output "service_url" {
  description = "Cloud Run サービス URL"
  value       = google_cloud_run_v2_service.genai_demo.uri
}

output "trigger_id" {
  description = "Cloud Build トリガー ID"
  value       = google_cloudbuild_trigger.genai_demo.trigger_id
}

output "artifact_registry_repo" {
  description = "専用 Artifact Registry リポジトリ"
  value       = "${google_artifact_registry_repository.genai_demo.location}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.genai_demo.repository_id}"
}

output "runtime_service_account" {
  description = "Cloud Run ランタイム SA"
  value       = google_service_account.run.email
}

output "cloudbuild_service_account" {
  description = "Cloud Build トリガー SA"
  value       = google_service_account.cb.email
}

output "gemini_secret_id" {
  description = "GEMINI_API_KEY を格納する Secret Manager シークレット"
  value       = google_secret_manager_secret.gemini_api_key.secret_id
}
