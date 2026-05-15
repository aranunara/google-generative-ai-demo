# project number は project_id から一意に決まるため自動取得（手入力・tfvars 不要）。
data "google_project" "this" {
  project_id = var.project_id
}
