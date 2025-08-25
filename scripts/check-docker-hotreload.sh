#!/bin/bash

echo "🔥 Docker ホットリロード診断スクリプト"
echo "=================================="

# macOSでのDocker設定チェック
if [[ "$OSTYPE" == "darwin"* ]]; then
    echo "✅ macOS環境を検出"
    
    # Docker Desktopの設定チェック
    echo ""
    echo "📋 Docker Desktop設定の確認:"
    echo "   1. Docker Desktop を開く"
    echo "   2. Settings > Resources > File Sharing"
    echo "   3. プロジェクトディレクトリが共有されているか確認"
    echo "   4. 設定が変更された場合は「Apply & Restart」"
    echo ""
fi

# 現在のディレクトリチェック
echo "📂 現在のディレクトリ: $(pwd)"

# .air.tomlの設定チェック
if [ -f ".air.toml" ]; then
    echo "✅ .air.toml が見つかりました"
    if grep -q "poll = true" .air.toml; then
        echo "✅ polling mode が有効です"
    else
        echo "❌ polling mode が無効です (Docker環境では必要)"
    fi
else
    echo "❌ .air.toml が見つかりません"
fi

# compose.local.yml の設定チェック
if [ -f "compose.local.yml" ]; then
    echo "✅ compose.local.yml が見つかりました"
    if grep -q "\- \.:/app" compose.local.yml; then
        echo "✅ ボリュームマウント設定が見つかりました"
    else
        echo "❌ ボリュームマウント設定が見つかりません"
    fi
else
    echo "❌ compose.local.yml が見つかりません"
fi

echo ""
echo "🚀 テスト手順:"
echo "   1. docker-compose -f compose.local.yml up --build"
echo "   2. main.go を編集して保存"
echo "   3. コンテナログでリビルドが実行されるか確認"
echo ""
echo "🐛 問題が続く場合:"
echo "   - Docker Desktop を再起動"
echo "   - docker-compose down --volumes && docker-compose -f compose.local.yml up --build"
echo "   - macOSの場合: System Preferences > Security & Privacy でDocker Desktop権限を確認"