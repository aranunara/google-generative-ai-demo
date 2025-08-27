#!/bin/bash

# ローカル環境でアプリケーションを起動するスクリプト

set -e

echo "🔐 ローカル環境でアプリケーションを起動します..."

# OS判定関数
detect_os() {
    case "$(uname -s)" in
        Darwin*)
            echo "mac"
            ;;
        MINGW64*|MSYS_NT*|CYGWIN_NT*)
            echo "windows"
            ;;
        Linux*)
            echo "linux"
            ;;
        *)
            echo "unknown"
            ;;
    esac
}

# ADC認証情報のパス取得
get_adc_path() {
    local os_type=$(detect_os)
    
    case $os_type in
        "mac"|"linux")
            echo "$HOME/.config/gcloud"
            ;;
        "windows")
            # Windowsの場合、APPDATAの実際のパスを取得
            if [ -n "$APPDATA" ]; then
                echo "$APPDATA/gcloud"
            else
                echo "$HOME/AppData/Roaming/gcloud"
            fi
            ;;
        *)
            echo ""
            ;;
    esac
}

# ADC認証状態確認
check_adc_login() {
    local adc_path=$(get_adc_path)
    local credentials_file="$adc_path/application_default_credentials.json"
    
    if [ -f "$credentials_file" ]; then
        # 認証情報ファイルが存在し、有効かどうかをチェック
        if gcloud auth application-default print-access-token >/dev/null 2>&1; then
            return 0  # ログイン済み
        else
            return 1  # 認証情報は存在するが無効
        fi
    else
        return 1  # 認証情報ファイルが存在しない
    fi
}

# Docker Compose用のボリュームマウント設定を生成
generate_volume_mount() {
    local os_type=$(detect_os)
    local adc_path=$(get_adc_path)
    
    case $os_type in
        "mac"|"linux")
            echo "      - $adc_path:/root/.config/gcloud:ro"
            ;;
        "windows")
            # Windowsの場合、パス形式を調整
            local windows_path=$(echo "$adc_path" | sed 's|\\|/|g')
            echo "      - $windows_path:/root/.config/gcloud:ro"
            ;;
        *)
            echo "      # Unsupported OS for ADC mount"
            ;;
    esac
}

# 現在の認証情報とプロジェクト設定を表示する関数
show_current_settings() {
    echo ""
    echo "═══════════════════════════════════════════"
    echo "🔐 現在のGoogle Cloud設定"
    echo "═══════════════════════════════════════════"
    
    # 現在のアカウント表示
    current_account=$(gcloud config get-value account 2>/dev/null || echo "未設定")
    echo "📧 アカウント: $current_account"
    
    # 現在のプロジェクト表示
    current_project=$(gcloud config get-value project 2>/dev/null || echo "未設定")
    echo "📁 プロジェクト: $current_project"
    
    # ADC状態確認
    if check_adc_login; then
        echo "🔑 Application Default Credentials: ✅ 設定済み"
    else
        echo "🔑 Application Default Credentials: ❌ 未設定"
    fi
    
    # config.yamlから読み込む設定も表示
    if [ -f "config.yaml" ]; then
        config_location=$(yq '.location' config.yaml 2>/dev/null || echo "未設定")
        config_vto_model=$(yq '.vto_model' config.yaml 2>/dev/null || echo "未設定")
        config_gcs_uri=$(yq '.gcs_uri' config.yaml 2>/dev/null || echo "未設定")
        config_api_key=$(yq '.api_key' config.yaml 2>/dev/null || echo "未設定")
        echo "🌍 リージョン: $config_location (config.yamlから)"
        echo "🤖 VTOモデル: $config_vto_model (config.yamlから)"
        echo "💾 GCS保存先URI: $config_gcs_uri (config.yamlから)"
        # APIキーは表示しない、未設定の場合は背停を促して終了する
        if [ -z "$config_api_key" ]; then
            echo "🔑 APIキー: ❌ 未設定"
            echo "🔑 APIキーを設定してください。"
            exit 1
        else
            echo "🔑 APIキー: ✅ 設定済み"
        fi
    fi
    echo "═══════════════════════════════════════════"
}

# アカウント切り替え処理
switch_account() {
    echo ""
    echo "📋 利用可能なアカウント一覧:"
    gcloud auth list --format="table(account,status)" 2>/dev/null || {
        echo "⚠️  認証されたアカウントがありません"
    }
    echo ""
    echo "選択してください:"
    echo "1) 既存のアカウントに切り替える"
    echo "2) 新しいアカウントでログインする"
    echo "3) キャンセル"
    echo ""
    read -p "選択 (1-3): " choice
    
    case $choice in
        1)
            echo ""
            read -p "📧 切り替えたいアカウントのメールアドレスを入力してください: " target_account
            if [ -n "$target_account" ]; then
                if gcloud config set account "$target_account" 2>/dev/null; then
                    echo "✅ アカウントを $target_account に切り替えました"
                    # ADCも更新
                    echo "🔄 Application Default Credentialsを更新中..."
                    gcloud auth application-default login --account="$target_account"
                    # アカウント切り替え後はプロジェクトも指定させる
                    echo ""
                    echo "🔄 アカウントを切り替えたため、プロジェクトを指定してください"
                    if change_project; then
                        return 0
                    else
                        echo "❌ プロジェクトの設定が必要です"
                        return 1
                    fi
                else
                    echo "❌ アカウントの切り替えに失敗しました"
                    return 1
                fi
            else
                echo "❌ アカウントが入力されませんでした"
                return 1
            fi
            ;;
        2)
            echo "🔄 新しいアカウントでログイン中..."
            gcloud auth login
            # ADCも設定
            echo "🔄 Application Default Credentialsを設定中..."
            gcloud auth application-default login
            # 新規ログイン後はプロジェクトも指定させる
            echo ""
            echo "🔄 新しいアカウントでログインしたため、プロジェクトを指定してください"
            if change_project; then
                return 0
            else
                echo "❌ プロジェクトの設定が必要です"
                return 1
            fi
            ;;
        3)
            echo "❌ キャンセルしました"
            return 1
            ;;
        *)
            echo "❌ 無効な選択です"
            return 1
            ;;
    esac
}

# プロジェクト変更処理
change_project() {
    echo ""
    echo "📋 利用可能なプロジェクト一覧:"
    gcloud projects list --format="table(projectId,name)" --limit=20 2>/dev/null || {
        echo "⚠️  プロジェクト一覧を取得できませんでした"
    }
    echo ""
    read -p "📁 設定したいプロジェクトIDを入力してください: " target_project
    
    if [ -n "$target_project" ]; then
        if gcloud config set project "$target_project" 2>/dev/null; then
            echo "✅ プロジェクトを $target_project に設定しました"
            return 0
        else
            echo "❌ プロジェクトの設定に失敗しました"
            return 1
        fi
    else
        echo "❌ プロジェクトIDが入力されませんでした"
        return 1
    fi
}

# メイン確認ループ
confirm_settings() {
    while true; do
        show_current_settings
        
        # ADC認証状態チェック
        if ! check_adc_login; then
            echo ""
            echo "❌ Application Default Credentials が設定されていません"
            echo "🔄 認証を実行します..."
            gcloud auth application-default login
            continue
        fi
        
        # プロジェクトが設定されているかチェック
        current_project=$(gcloud config get-value project 2>/dev/null)
        if [ -z "$current_project" ]; then
            echo ""
            echo "⚠️  プロジェクトが設定されていません"
            if change_project; then
                continue
            else
                echo "プロジェクトの設定が必要です。再度お試しください。"
                continue
            fi
        fi
        
        echo ""
        echo "この設定でアプリケーションを起動しますか？"
        echo "1) はい、この設定で起動する"
        echo "2) アカウントを変更する"
        echo "3) プロジェクトを変更する"
        echo "4) 終了する"
        echo ""
        read -p "選択 (1-4): " response
        
        case $response in
            1)
                echo "✅ 設定を確認しました。アプリケーションを起動します..."
                PROJECT_ID="$current_project"
                return 0
                ;;
            2)
                if switch_account; then
                    continue
                else
                    echo "アカウント切り替えに失敗しました。再度確認してください。"
                    continue
                fi
                ;;
            3)
                if change_project; then
                    continue
                else
                    echo "プロジェクト変更に失敗しました。再度確認してください。"
                    continue
                fi
                ;;
            4)
                echo "❌ 終了します"
                exit 0
                ;;
            *)
                echo "❌ 無効な選択です。1-4の番号を入力してください。"
                continue
                ;;
        esac
    done
}

# 環境変数を読み込み
echo "🔐 環境変数を読み込み"

# OS検出
OS_TYPE=$(detect_os)
echo "🖥️  検出されたOS: $OS_TYPE"

# 認証と設定の確認
echo "🔐 Google Cloud認証状態を確認中..."
confirm_settings

# モデルを引数で受け取る（なくてもいい）
if [ -n "$1" ]; then
    echo "🔐 モデルを引数で受け取って環境変数に設定: $1"
    export VTO_MODEL=$1
fi

# config.yamlを読み込み（プロジェクトID以外）
echo "🔐 config.yamlを読み込み"
LOCATION=$(yq '.location' config.yaml 2>/dev/null || echo "us-central1")
VTO_MODEL=${VTO_MODEL:-$(yq '.vto_model' config.yaml 2>/dev/null || echo "default")}
GEMINI_API_KEY=$(yq '.api_key' config.yaml 2>/dev/null || echo "default")
GCS_URI=$(yq '.gcs_uri' config.yaml 2>/dev/null || echo "default")
echo ""
echo "🚀 アプリケーションを起動します..."
echo "🔐 設定された環境変数:"
echo "  - PROJECT_ID: $PROJECT_ID (Google Cloud認証から取得)"
echo "  - LOCATION: $LOCATION"
echo "  - VTO_MODEL: $VTO_MODEL"
echo "  - GCS_URI: $GCS_URI"
# APIキーは表示しない、未設定の場合は背停を促して終了する
if [ -z "$GEMINI_API_KEY" ]; then
    echo "🔑 APIキー: ❌ 未設定"
    echo "🔑 APIキーを設定してください。"
    exit 1
else
    echo "🔑 APIキー: ✅ 設定済み"
fi

# 一時的な.envファイルを作成（確実に環境変数を渡すため）
cat > .env.local <<EOF
PROJECT_ID=$PROJECT_ID
LOCATION=$LOCATION
VTO_MODEL=$VTO_MODEL
GEMINI_API_KEY=$GEMINI_API_KEY
GCS_URI=$GCS_URI
EOF

# OS別のdocker-compose設定を生成
echo "🐳 OS別のDocker Compose設定を生成中..."
ADC_VOLUME_MOUNT=$(generate_volume_mount)

# 一時的なcompose override ファイルを作成
cat > compose.local.override.yml <<EOF
services:
  tryon-app:
    volumes:
$ADC_VOLUME_MOUNT
EOF

# 終了時に一時ファイルを削除するよう設定
trap "rm -f .env.local compose.local.override.yml" EXIT INT TERM

echo "🎯 使用するボリュームマウント:"
echo "$ADC_VOLUME_MOUNT"

# docker composeを利用してアプリケーションを起動
echo "🚀 Docker Composeでアプリケーションを起動中..."
docker compose -f compose.local.yml -f compose.local.override.yml --env-file .env.local up --build