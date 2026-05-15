# Cloud Run サービス（名称 genai-demo）。
# 初期 image はプレースホルダ。実アプリ image は Cloud Build トリガーの
# `gcloud run services update` が反映するため、image の変更は無視する。
#
# 注意: GEMINI_API_KEY を secret 参照するため、リビジョン作成時点で
# secret version が存在している必要がある。apply は二段階で行う:
#   1) terraform apply -target=google_secret_manager_secret.gemini_api_key \
#        -target=google_secret_manager_secret_iam_member.run_secret_accessor
#   2) gcloud secrets versions add gemini-api-key --data-file=- で鍵投入
#   3) terraform apply（残り全部）
resource "google_cloud_run_v2_service" "genai_demo" {
  project             = var.project_id
  name                = var.name
  location            = var.region
  ingress             = "INGRESS_TRAFFIC_ALL"
  deletion_protection = false

  template {
    service_account                  = google_service_account.run.email
    timeout                          = "300s"
    max_instance_request_concurrency = 80

    scaling {
      max_instance_count = 100
    }

    containers {
      image = "us-docker.pkg.dev/cloudrun/container/hello"

      ports {
        container_port = 8080
      }

      resources {
        limits = {
          cpu    = "1000m"
          memory = "512Mi"
        }
      }

      env {
        name  = "PROJECT_ID"
        value = var.project_id
      }
      env {
        name  = "LOCATION"
        value = var.region
      }
      env {
        name  = "VTO_MODEL"
        value = var.vto_model
      }
      env {
        name  = "USE_SDK"
        value = "false"
      }

      dynamic "env" {
        for_each = var.gcs_uri == "" ? [] : [1]
        content {
          name  = "GCS_URI"
          value = var.gcs_uri
        }
      }

      env {
        name = "GEMINI_API_KEY"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.gemini_api_key.secret_id
            version = "latest"
          }
        }
      }
    }
  }

  lifecycle {
    ignore_changes = [
      template[0].containers[0].image,
    ]
  }

  depends_on = [
    google_project_service.this,
    google_secret_manager_secret_iam_member.run_secret_accessor,
  ]
}

# 未認証アクセス許可（現行と同じ挙動）。
resource "google_cloud_run_v2_service_iam_member" "invoker" {
  project  = var.project_id
  location = google_cloud_run_v2_service.genai_demo.location
  name     = google_cloud_run_v2_service.genai_demo.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}
