locals {
  image = "${var.region}-docker.pkg.dev/${var.project_id}/${var.name}/${var.name}:$COMMIT_SHA"
}

# GitHub 連携 Cloud Build トリガー（1st-gen GitHub）。
# included_files により、Go / ビルド関連ファイルが変更された push のときだけ起動。
# README・docs・HTML・compose 等のみの変更ではビルドが起動しない。
resource "google_cloudbuild_trigger" "genai_demo" {
  project     = var.project_id
  name        = "${var.name}-deploy"
  location    = "global"
  description = "Build & deploy ${var.name} to Cloud Run on push to ${var.branch_regex}"

  service_account = google_service_account.cb.id

  github {
    owner = var.github_owner
    name  = var.github_repo

    push {
      branch = var.branch_regex
    }
  }

  included_files = [
    "**/*.go",
    "*.go",
    "go.mod",
    "go.sum",
    "dockerfile",
    "docker-entrypoint.sh",
    "cloudbuild.yaml",
    ".dockerignore",
  ]

  build {
    images = [local.image]

    options {
      logging = "CLOUD_LOGGING_ONLY"
    }

    step {
      id   = "Build"
      name = "gcr.io/cloud-builders/docker"
      args = ["build", "--no-cache", "-t", local.image, ".", "-f", "dockerfile"]
    }

    step {
      id   = "Push"
      name = "gcr.io/cloud-builders/docker"
      args = ["push", local.image]
    }

    step {
      id         = "Deploy"
      name       = "gcr.io/google.com/cloudsdktool/cloud-sdk:slim"
      entrypoint = "gcloud"
      args = [
        "run", "services", "update", var.name,
        "--platform=managed",
        "--image=${local.image}",
        "--region=${var.region}",
        "--quiet",
      ]
    }
  }

  depends_on = [
    google_project_service.this,
    google_artifact_registry_repository.genai_demo,
    google_cloud_run_v2_service.genai_demo,
    google_project_iam_member.cb_run_admin,
    google_project_iam_member.cb_log_writer,
    google_artifact_registry_repository_iam_member.cb_ar_writer,
    google_service_account_iam_member.cb_actas_run,
    google_service_account_iam_member.cloudbuild_agent_actas_cb,
  ]
}
