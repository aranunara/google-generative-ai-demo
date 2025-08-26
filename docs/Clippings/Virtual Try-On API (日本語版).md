---
title: "Virtual Try-On API | Vertex AI での生成AI | Google Cloud"
source: "https://cloud.google.com/vertex-ai/generative-ai/docs/model-reference/virtual-try-on-api?hl=ja"
author:
published:
created: 2025-08-25
description: "Virtual Try-Onを使用して、衣類製品を着用した人物のバーチャル試着画像を生成します"
tags:
  - "clippings"
  - "日本語版"
---

このガイドでは、Virtual Try-On APIを使用して、衣類製品をモデリングする人物のバーチャル試着画像を生成する方法について説明します。

このページでは以下のトピックをカバーしています：

- **[画像入力オプション](#image-input-options):** APIリクエストで画像を提供するさまざまな方法について学習します。
- **[サポートされているモデルバージョン](#supported-model-versions):** Virtual Try-Onがサポートするモデルバージョンに関する情報を見つけます。
- **[HTTPリクエスト](#http-request):** APIリクエストの構造とパラメータを確認します。
- **[リクエストボディの説明](#request-body-description):** リクエストボディフィールドの詳細な説明を参照します。
- **[サンプルリクエスト](#sample-request):** APIリクエストの完全な例を参照します。

## 画像入力オプション

人物または衣類製品の画像を提供する場合、Base64エンコードされたバイト文字列またはCloud Storage URIのいずれかとして指定できます。以下の表は、これら2つのオプションの比較を提供し、どちらを使用するかを決定するのに役立ちます。

| オプション | 説明 | メリット | デメリット | 使用例 |
| --- | --- | --- | --- | --- |
| `bytesBase64Encoded` | 画像データがJSONリクエストボディ内で直接送信されます。 | 小さな画像では簡単；別のストレージステップが不要。 | リクエストサイズが増加；JSONペイロードの制限により、非常に大きな画像には適さない。 | クイックテストまたは画像がその場で生成・処理され、長期保存されないアプリケーション。 |
| `gcsUri` | Cloud Storageバケットに保存された画像ファイルを指すURI。 | 大きな画像で効率的；リクエストペイロードを小さく保つ。 | 最初にCloud Storageに画像をアップロードする必要があり、追加のステップが必要。 | バッチ処理、画像がすでにCloud Storageに保存されているワークフロー、または大きな画像ファイルを扱う場合。 |

## サポートされているモデルバージョン

Virtual Try-Onは以下のモデルをサポートしています：

- `virtual-try-on-preview-08-04`

モデルがサポートする機能の詳細については、[Imagenモデル](https://cloud.google.com/vertex-ai/generative-ai/docs/models#imagen-models)を参照してください。

## HTTPリクエスト

画像を生成するには、モデルの`predict`エンドポイントに`POST`リクエストを送信します。

```bash
curl -X POST \
  -H "Authorization: Bearer $(gcloud auth print-access-token)" \
  -H "Content-Type: application/json" \
https://LOCATION-aiplatform.googleapis.com/v1/projects/PROJECT_ID/locations/LOCATION/publishers/google/models/MODEL_ID:predict \

-d '{
  "instances": [
    {
      "personImage": {
        "image": {
          // Union fieldは以下のいずれか一つのみ:
          "bytesBase64Encoded": string,
          "gcsUri": string,
        }
      },
      "productImages": [
        {
          "image": {
            // Union fieldは以下のいずれか一つのみ:
            "bytesBase64Encoded": string,
            "gcsUri": string,
          }
        }
      ]
    }
  ],
  "parameters": {
    "addWatermark": boolean,
    "baseSteps": integer,
    "personGeneration": string,
    "safetySetting": string,
    "sampleCount": integer,
    "seed": integer,
    "storageUri": string,
    "outputOptions": {
      "mimeType": string,
      "compressionQuality": integer
    }
  }
}'
```

### インスタンス

| フィールド | 型 | 説明 |
| --- | --- | --- |
| `personImage` | `string` | 必須。衣類製品を試着する人物の画像。以下のいずれかになります：<ul><li>画像をエンコードする`bytesBase64Encoded`文字列</li><li>Cloud Storageバケットの場所への`gcsUri`文字列URI</li></ul> |
| `productImages` | `string` | 必須。人物に試着させる製品の画像。以下のいずれかになります：<ul><li>画像をエンコードする`bytesBase64Encoded`文字列</li><li>Cloud Storageバケットの場所への`gcsUri`文字列URI</li></ul> |

### パラメータ

| パラメータ | 型 | 説明 |
| --- | --- | --- |
| `addWatermark` | `bool` | オプション。生成された画像に透明なウォーターマークを追加します。<br>デフォルト値は`true`です。 |
| `baseSteps` | `int` | 必須。画像生成を制御する整数で、ステップ数が多いほど品質が向上しますが、レイテンシが増加します。<br>`0`より大きい整数値。デフォルトは`32`です。 |
| `personGeneration` | `string` | オプション。モデルによる人物の生成を許可します。以下の値がサポートされています：<ul><li>`"dont_allow"`: 画像に人物や顔の含有を禁止</li><li>`"allow_adult"`: 成人のみの生成を許可</li><li>`"allow_all"`: あらゆる年齢の人物の生成を許可</li></ul>デフォルト値は`"allow_adult"`です。 |
| `safetySetting` | `string` | オプション。安全フィルタリングにフィルタレベルを追加します。以下の値がサポートされています：<ul><li>`"block_low_and_above"`: 最も強いフィルタレベル、最も厳格なブロック</li><li>`"block_medium_and_above"`: 一部の問題のあるプロンプトと応答をブロック</li><li>`"block_only_high"`: 安全フィルタによりブロックされるリクエスト数を削減</li><li>`"block_none"`: 問題のあるプロンプトと応答をほとんどブロックしない</li></ul>デフォルト値は`"block_medium_and_above"`です。 |
| `sampleCount` | `int` | 必須。生成する画像の数。<br>`1`から`4`の間の整数値（両端含む）。デフォルト値は`1`です。 |
| `seed` | `Uint32` | オプション。画像生成のランダムシード。`addWatermark`が`true`に設定されている場合は利用できません。 |
| `storageUri` | `string` | オプション。生成された画像を保存するCloud Storageバケットの場所への文字列URI。 |
| `outputOptions` | `outputOptions` | オプション。[`outputOptions`オブジェクト](#output-options-object)で出力画像形式を記述します。 |

### 出力オプションオブジェクト

`outputOptions`オブジェクトは画像出力を記述します。

| パラメータ | 型 | 説明 |
| --- | --- | --- |
| `outputOptions.mimeType` | `string`（オプション） | 画像出力形式。以下の値がサポートされています：<ul><li>`"image/png"`: PNG画像として保存</li><li>`"image/jpeg"`: JPEG画像として保存</li></ul>デフォルト値は`"image/png"`です。 |
| `outputOptions.compressionQuality` | `int`（オプション） | 出力タイプが`"image/jpeg"`の場合の圧縮レベル。受け入れられる値は`0`から`100`です。デフォルト値は`75`です。 |

## サンプルリクエスト

リクエストデータを使用する前に、以下の置換を行ってください：

- REGION: プロジェクトが配置されているリージョン。サポートされているリージョンの詳細については、[Vertex AIでの生成AIの場所](https://cloud.google.com/vertex-ai/generative-ai/docs/learn/locations)を参照してください。
- PROJECT_ID: Google Cloud [プロジェクトID](https://cloud.google.com/resource-manager/docs/creating-managing-projects#identifiers)。
- BASE64_PERSON_IMAGE: 人物画像のBase64エンコード画像。
- BASE64_PRODUCT_IMAGE: 製品画像のBase64エンコード画像。
- IMAGE_COUNT: 生成する画像の数。受け入れられる値の範囲は`1`から`4`です。
- GCS_OUTPUT_PATH: バーチャル試着出力を保存するCloud Storageパス。

HTTPメソッドとURL:

```
POST https://REGION-aiplatform.googleapis.com/v1/projects/PROJECT_ID/locations/REGION/publishers/google/models/virtual-try-on-preview-08-04:predict
```

リクエストJSONボディ:

```json
{
  "instances": [
    {
      "personImage": {
        "image": {
          "bytesBase64Encoded": "BASE64_PERSON_IMAGE"
        }
      },
      "productImages": [
        {
          "image": {
            "bytesBase64Encoded": "BASE64_PRODUCT_IMAGE"
          }
        }
      ]
    }
  ],
  "parameters": {
    "sampleCount": IMAGE_COUNT,
    "storageUri": "GCS_OUTPUT_PATH"
  }
}
```

リクエストを送信するには、以下のオプションのいずれかを選択してください：

リクエストボディを`request.json`という名前のファイルに保存し、以下のコマンドを実行してください：

```bash
curl -X POST \
     -H "Authorization: Bearer $(gcloud auth print-access-token)" \
     -H "Content-Type: application/json; charset=utf-8" \
     -d @request.json \
     "https://REGION-aiplatform.googleapis.com/v1/projects/PROJECT_ID/locations/REGION/publishers/google/models/virtual-try-on-preview-08-04:predict"
```

リクエストは画像オブジェクトを返します。この例では、Base64エンコードされた画像を持つ2つの予測オブジェクトとともに、2つの画像オブジェクトが返されます。

```json
{
  "predictions": [
    {
      "mimeType": "image/png",
      "bytesBase64Encoded": "BASE64_IMG_BYTES"
    },
    {
      "bytesBase64Encoded": "BASE64_IMG_BYTES",
      "mimeType": "image/png"
    }
  ]
}
```

特に明記されていない限り、このページのコンテンツは[Creative Commons Attribution 4.0 License](https://creativecommons.org/licenses/by/4.0/)の下でライセンスされ、コードサンプルは[Apache 2.0 License](https://www.apache.org/licenses/LICENSE-2.0)の下でライセンスされます。詳細については、[Google Developers Site Policies](https://developers.google.com/site-policies)を参照してください。Javaは、OracleおよびOracle関連会社の登録商標です。

最終更新日: 2025-08-21 UTC。