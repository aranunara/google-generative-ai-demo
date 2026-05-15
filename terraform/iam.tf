# ---- Cloud Build SA (genai-demo-cb) ----

# Cloud Run サービスの image 更新（gcloud run services update）に必要。
resource "google_project_iam_member" "cb_run_admin" {
  project = var.project_id
  role    = "roles/run.admin"
  member  = "serviceAccount:${google_service_account.cb.email}"
}

# CLOUD_LOGGING_ONLY のためビルドログ書き込みに必要。
resource "google_project_iam_member" "cb_log_writer" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.cb.email}"
}

# 専用 AR リポジトリへの push 権限。
resource "google_artifact_registry_repository_iam_member" "cb_ar_writer" {
  project    = var.project_id
  location   = google_artifact_registry_repository.genai_demo.location
  repository = google_artifact_registry_repository.genai_demo.repository_id
  role       = "roles/artifactregistry.writer"
  member     = "serviceAccount:${google_service_account.cb.email}"
}

# ランタイム SA で動く Cloud Run をデプロイ更新するため actAs が必要。
resource "google_service_account_iam_member" "cb_actas_run" {
  service_account_id = google_service_account.run.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.cb.email}"
}

# トリガーの service_account 指定（カスタム SA）利用のため、
# Cloud Build サービスエージェントに cb SA への actAs を付与。
resource "google_service_account_iam_member" "cloudbuild_agent_actas_cb" {
  service_account_id = google_service_account.cb.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:service-${data.google_project.this.number}@gcp-sa-cloudbuild.iam.gserviceaccount.com"
}

# ---- Cloud Run runtime SA (genai-demo-run) ----

# GEMINI_API_KEY シークレットの参照。
resource "google_secret_manager_secret_iam_member" "run_secret_accessor" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.gemini_api_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.run.email}"
}

# Vertex AI Virtual Try-On 呼び出し。
resource "google_project_iam_member" "run_aiplatform_user" {
  project = var.project_id
  role    = "roles/aiplatform.user"
  member  = "serviceAccount:${google_service_account.run.email}"
}

# 生成物を GCS に保存する場合のみ（var.gcs_bucket 指定時）。
resource "google_storage_bucket_iam_member" "run_gcs_object_admin" {
  count = var.gcs_bucket == "" ? 0 : 1

  bucket = var.gcs_bucket
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.run.email}"
}
