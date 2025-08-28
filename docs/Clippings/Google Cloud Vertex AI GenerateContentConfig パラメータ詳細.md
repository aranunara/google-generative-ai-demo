# Google Cloud Vertex AI GenerateContentConfig パラメータ詳細

## 概要
Google Cloud Vertex AIの`GenerateContentConfig`構造体の詳細なパラメータ説明。マルチモーダルコンテンツ生成の設定に使用される。

## 基本設定パラメータ

### HTTPOptions
- **型**: `*HTTPOptions`
- **説明**: HTTPリクエストオプションのオーバーライドに使用
- **用途**: カスタムHTTP設定が必要な場合

### SystemInstruction
- **型**: `*Content`
- **説明**: モデルのパフォーマンス向上のための指示
- **例**: 
  - "Answer as concisely as possible"
  - "Don't use technical terms in your response"
- **用途**: モデルの動作を特定の方向に誘導

## 生成制御パラメータ

### Temperature
- **型**: `*float32`
- **説明**: トークン選択のランダム性を制御
- **値の意味**:
  - 低い値: より確定的で創造性の低い回答
  - 高い値: より多様で創造的な結果
- **推奨**: タスクの性質に応じて調整

### TopP
- **型**: `*float32`
- **説明**: 確率の合計がこの値に達するまで、最も確率の高いトークンから選択
- **値の意味**:
  - 低い値: より確定的な回答
  - 高い値: よりランダムな回答
- **用途**: Temperatureと組み合わせて使用

### TopK
- **型**: `*float32`
- **説明**: 各トークン選択ステップで、最も確率の高い`top_k`個のトークンをサンプリング
- **処理フロー**: TopK → TopP → Temperature sampling
- **値の意味**:
  - 低い数値: より確定的な回答
  - 高い数値: よりランダムな回答

## 出力制御パラメータ

### CandidateCount
- **型**: `int32`
- **説明**: 返す応答バリエーションの数
- **デフォルト**: 1（システムが選択）
- **用途**: 複数の候補を生成したい場合

### MaxOutputTokens
- **型**: `int32`
- **説明**: 応答で生成できる最大トークン数
- **デフォルト**: モデルによって異なる
- **用途**: 応答長の制限

### StopSequences
- **型**: `[]string`
- **説明**: 応答でこれらの文字列のいずれかが検出された場合、テキスト生成を停止
- **用途**: 特定の条件で生成を停止

## ログ・分析パラメータ

### ResponseLogprobs
- **型**: `bool`
- **説明**: 各ステップでモデルが選択したトークンのログ確率を返すかどうか
- **用途**: モデルの内部動作分析

### Logprobs
- **型**: `*int32`
- **説明**: 各生成ステップでログ確率を返す上位候補トークンの数
- **用途**: 詳細な確率分析

## ペナルティパラメータ

### PresencePenalty
- **型**: `*float32`
- **説明**: 既に生成されたテキストに出現するトークンに正の値でペナルティ
- **効果**: より多様なコンテンツ生成を促進
- **用途**: 繰り返しを避けたい場合

### FrequencyPenalty
- **型**: `*float32`
- **説明**: 繰り返し出現するトークンに正の値でペナルティ
- **効果**: より多様なコンテンツ生成を促進
- **用途**: 同じ単語の繰り返しを避けたい場合

## 再現性制御

### Seed
- **型**: `*int32`
- **説明**: 特定の数値に固定すると、モデルは繰り返しリクエストに対して同じ応答を提供するよう最善を尽くす
- **デフォルト**: ランダム数値
- **用途**: 結果の再現性が必要な場合

## 出力形式制御

### ResponseMIMEType
- **型**: `string`
- **説明**: 生成された候補テキストの出力応答MIMEタイプ
- **サポート形式**:
  - `text/plain`: デフォルトのテキスト出力
  - `application/json`: JSON応答
- **注意**: モデルに適切な応答タイプを出力するよう促す必要がある
- **ステータス**: プレビュー機能

### ResponseSchema
- **型**: `*Schema`
- **説明**: 入力・出力データ型の定義を可能にするSchemaオブジェクト
- **制約**: 設定する場合は互換性のある`response_mime_type`も設定必須
- **互換MIMEタイプ**: `application/json`
- **ベース**: OpenAPI 3.0 schema objectのサブセット

### ResponseJsonSchema
- **型**: `any`
- **説明**: JSON Schemaを受け入れる`response_schema`の代替
- **制約**: 
  - 設定する場合は`response_schema`は省略必須
  - `response_mime_type`は必須
- **サポート機能**: 
  - `$id`, `$defs`, `$ref`, `$anchor`
  - `type`, `format`, `title`, `description`
  - `enum`, `items`, `prefixItems`
  - `minItems`, `maxItems`, `minimum`, `maximum`
  - `anyOf`, `oneOf`, `properties`, `additionalProperties`, `required`
  - 非標準の`propertyOrdering`プロパティ

## 高度な設定

### RoutingConfig
- **型**: `*GenerationConfigRoutingConfig`
- **説明**: モデルルーターリクエストの設定
- **用途**: 複数モデル間でのルーティング制御

### ModelSelectionConfig
- **型**: `*ModelSelectionConfig`
- **説明**: モデル選択の設定
- **用途**: 特定のモデル選択ロジック

### SafetySettings
- **型**: `[]*SafetySetting`
- **説明**: 応答の安全でないコンテンツをブロックするための安全設定
- **用途**: コンテンツフィルタリング

### Tools
- **型**: `[]*Tool`
- **説明**: モデルの知識と範囲外のアクションを実行するための外部システムとの相互作用を可能にするコード
- **用途**: 関数呼び出しや外部API連携

### ToolConfig
- **型**: `*ToolConfig`
- **説明**: モデル出力を特定の関数呼び出しに関連付ける
- **用途**: ツール使用の制御

## メタデータ・管理

### Labels
- **型**: `map[string]string`
- **説明**: 請求料金を分類するためのユーザー定義メタデータのラベル
- **用途**: コスト管理・追跡

### CachedContent
- **型**: `string`
- **説明**: 後続のリクエストで使用できるコンテキストキャッシュのリソース名
- **用途**: パフォーマンス最適化

## マルチモーダル設定

### ResponseModalities
- **型**: `[]string`
- **説明**: 応答の要求されたモダリティ。モデルが返すことができるモダリティのセットを表す
- **用途**: テキスト、画像、音声などの出力形式指定

### MediaResolution
- **型**: `MediaResolution`
- **説明**: 指定された場合、指定されたメディア解像度が使用される
- **用途**: 画像・動画の解像度制御

### SpeechConfig
- **型**: `*SpeechConfig`
- **説明**: 音声生成の設定
- **用途**: テキスト読み上げ機能

### AudioTimestamp
- **型**: `bool`
- **説明**: 有効にすると、モデルへのリクエストに音声タイムスタンプが含まれる
- **用途**: 音声処理の詳細制御

### ThinkingConfig
- **型**: `*ThinkingConfig`
- **説明**: 思考機能の設定
- **用途**: モデルの内部推論プロセスの制御

## 使用例

```go
config := &GenerateContentConfig{
    Temperature:     &[]float32{0.7}[0],
    MaxOutputTokens: 1000,
    TopP:           &[]float32{0.9}[0],
    SystemInstruction: &Content{
        Parts: []*Part{
            {Text: "You are a helpful assistant that provides concise answers."},
        },
    },
    SafetySettings: []*SafetySetting{
        {
            Category:  HarmCategory_HARM_CATEGORY_HATE_SPEECH,
            Threshold: HarmBlockThreshold_BLOCK_MEDIUM_AND_ABOVE,
        },
    },
}
```

## 参考リンク
- [Content generation parameters](https://cloud.google.com/vertex-ai/generative-ai/docs/multimodal/content-generation-parameters)
- [OpenAPI 3.0 schema object](https://spec.openapis.org/oas/v3.0.3#schema)
- [JSON Schema](https://json-schema.org/)

---
*作成日: 2024年*
*カテゴリ: Google Cloud, Vertex AI, API Documentation*