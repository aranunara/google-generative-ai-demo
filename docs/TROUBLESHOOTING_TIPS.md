# Vertex AI Virtual Try-On トラブルシューティング Tips

## StorageURI 指定時のエラー対処法

### 問題: StorageURIを指定すると Internal Error (Status 500) が発生する

#### 症状

```json
Virtual Try-On failed: try-on generation failed: API request failed with status 500: {
  "error": {
    "code": 500,
    "message": "Internal error encountered.",
    "status": "INTERNAL"
  }
}
```

#### 主な原因と解決策

##### 1. **リージョン不一致（最も多い原因）**

**原因**: Cloud StorageバケットとVertex AIのリージョンが異なる場合にエラーが発生します。

**確認方法**:

```bash
# バケットのリージョンを確認
gcloud storage buckets describe gs://your-bucket-name --format="value(location)"

# 現在のVertex AIリージョン設定を確認（環境変数 LOCATION）
echo $LOCATION
```

**解決策A**: Vertex AIのリージョンをバケットに合わせる

```bash
# 例：バケットが us-east1 の場合
export LOCATION=us-east1
```

**解決策B**: Vertex AIのリージョンに合わせて新しいバケットを作成（推奨）

```bash
# 例：Vertex AIが us-central1 の場合
gcloud storage buckets create gs://your-new-bucket --location=us-central1
```

##### 2. **StorageURIの形式**

**正しい形式**:

- `gs://bucket-name/` （末尾のスラッシュが必要）
- `gs://bucket-name/subdirectory/` （サブディレクトリ指定時）

**間違った形式**:

- `gs://bucket-name` （スラッシュなし）
- `gs://bucket-name/filename.jpg` （ファイル名指定）

##### 3. **権限設定**

Vertex AIサービスアカウントにStorage Admin権限が必要です。

**Google Cloud Consoleでの設定**:

1. IAM → サービスアカウント
2. Vertex AI Service Account を選択
3. 「Storage Admin」ロールを追加

##### 4. **対応リージョン**

Vertex AI Virtual Try-Onが利用可能なリージョン:

- `us-central1` （**必須・推奨**）
- `us-east1`

**重要**: 2024年8月時点では、Virtual Try-On APIは `us-central1` での実行が必要です。

- `asia-northeast1` などの他のリージョンでは以下のエラーが発生します：

  ```json
  {
    "error": {
      "code": 404,
      "message": "Publisher Model `projects/{PROJECT_ID}/locations/asia-northeast1/publishers/google/models/virtual-try-on-preview-08-04` not found.",
      "status": "NOT_FOUND"
    }
  }
  ```

#### デバッグ方法

1. **リージョン確認**:

   ```bash
   # バケットリージョン
   gcloud storage buckets describe gs://your-bucket --format="value(location)"
   
   # アプリケーションのVertex AIリージョン設定
   echo $LOCATION
   ```

2. **権限確認**:

   ```bash
   # バケットへの書き込みテスト
   echo "test" | gcloud storage cp - gs://your-bucket/test.txt
   ```

3. **Vertex AI リージョン設定確認**:

   ```bash
   # アプリケーション設定のリージョン確認
   grep -r "location" config.yaml
   
   # 環境変数の確認
   env | grep -i location
   ```

4. **StorageURI無しでのテスト**:
   StorageURIを空にして通常の画像生成が動作するか確認

#### ベストプラクティス

1. **リージョン統一**: 事前にVertex AIとCloud Storageのリージョンを統一する
2. **バケット命名**: 用途とリージョンを含めた命名（例：`try-on-images-us-central1`）
3. **サブディレクトリ活用**: 日付やユーザー別でディレクトリを分ける
   - `gs://bucket/2025-08-26/`
   - `gs://bucket/users/user123/`

---

---

## その他のTips

### パフォーマンス最適化

- Sample Count は必要以上に増やさない（1-2枚推奨）
- Base Steps は 32 が推奨値

### 画像品質向上

- 人物画像は正面向き、全身が写っているものを使用
- 衣服画像は背景が単色で、商品がはっきり写っているものを使用
- 画像サイズは適度に大きく（推奨：512x512以上）

### エラー対処

- Quota Exceeded エラー → しばらく待ってから再試行
- Safety Filter エラー → 異なる画像を使用
- Timeout エラー → Sample Count を減らす

---

最終更新: 2025-01-27
