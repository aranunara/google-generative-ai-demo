---
title: "Gemini を使用した画像生成  |  Gemini API"
source: "https://ai.google.dev/gemini-api/docs/image-generation?hl=ja"
author:
  - "[[Google AI for Developers]]"
published:
created: 2025-08-28
description: "Gemini API を使用して画像を生成する"
tags:
  - "clippings"
---
Gemini は、会話形式で画像を生成して処理できます。Gemini にテキスト、画像、またはその両方を組み合わせて指示することで、これまでにない制御でビジュアルを作成、編集、反復処理できます。

- **Text-to-Image:** シンプルなテキストの説明から複雑なテキストの説明まで、高品質の画像を生成します。
- **画像 + テキストから画像（編集）:** 画像を指定し、テキスト プロンプトを使用して要素の追加、削除、変更、スタイルの変更、カラー グレーディングの調整を行います。
- **Multi-Image to Image（構図とスタイル転送）:** 複数の入力画像を使用して新しいシーンを構成したり、ある画像のスタイルを別の画像に転送したりします。
- **反復的な調整:** 会話を通じて、画像を複数回にわたって徐々に調整し、完璧になるまで小さな調整を繰り返します。
- **高忠実度のテキスト レンダリング:** ロゴ、図、ポスターなどに最適な、読みやすく配置されたテキストを含む画像を正確に生成します。

すべての生成画像には [SynthID の透かし](https://ai.google.dev/responsible/docs/safeguards/synthid?hl=ja) が埋め込まれています。

## 画像生成（テキスト画像変換）

次のコードは、説明的なプロンプトに基づいて画像を生成する方法を示しています。

```
from google import genai
from google.genai import types
from PIL import Image
from io import BytesIO

client = genai.Client()

prompt = (
    "Create a picture of a nano banana dish in a fancy restaurant with a Gemini theme"
)

response = client.models.generate_content(
    model="gemini-2.5-flash-image-preview",
    contents=[prompt],
)

for part in response.candidates[0].content.parts:
    if part.text is not None:
        print(part.text)
    elif part.inline_data is not None:
        image = Image.open(BytesIO(part.inline_data.data))
        image.save("generated_image.png")
```
```
import { GoogleGenAI, Modality } from "@google/genai";
import * as fs from "node:fs";

async function main() {

  const ai = new GoogleGenAI({});

  const prompt =
    "Create a picture of a nano banana dish in a fancy restaurant with a Gemini theme";

  const response = await ai.models.generateContent({
    model: "gemini-2.5-flash-image-preview",
    contents: prompt,
  });
  for (const part of response.candidates[0].content.parts) {
    if (part.text) {
      console.log(part.text);
    } else if (part.inlineData) {
      const imageData = part.inlineData.data;
      const buffer = Buffer.from(imageData, "base64");
      fs.writeFileSync("gemini-native-image.png", buffer);
      console.log("Image saved as gemini-native-image.png");
    }
  }
}

main();
```
```
package main

import (
  "context"
  "fmt"
  "os"
  "google.golang.org/genai"
)

func main() {

  ctx := context.Background()
  client, err := genai.NewClient(ctx, nil)
  if err != nil {
      log.Fatal(err)
  }

  result, _ := client.Models.GenerateContent(
      ctx,
      "gemini-2.5-flash-image-preview",
      genai.Text("Create a picture of a nano banana dish in a " +
                 " fancy restaurant with a Gemini theme"),
  )

  for _, part := range result.Candidates[0].Content.Parts {
      if part.Text != "" {
          fmt.Println(part.Text)
      } else if part.InlineData != nil {
          imageBytes := part.InlineData.Data
          outputFilename := "gemini_generated_image.png"
          _ = os.WriteFile(outputFilename, imageBytes, 0644)
      }
  }
}
```
```
curl -s -X POST
  "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash-image-preview:generateContent" \
  -H "x-goog-api-key: $GEMINI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [{
      "parts": [
        {"text": "Create a picture of a nano banana dish in a fancy restaurant with a Gemini theme"}
      ]
    }]
  }' \
  | grep -o '"data": "[^"]*"' \
  | cut -d'"' -f4 \
  | base64 --decode > gemini-native-image.png
```
![ナノバナナ料理の AI 生成画像](https://ai.google.dev/static/gemini-api/docs/images/nano-banana.png?hl=ja)

Gemini をテーマにしたレストランでナノバナナ料理を出す AI 生成の画像

## 画像編集（テキストと画像による画像変換）

**リマインダー**: アップロードする画像に対して必要な権利をすべて所有していることをご確認ください。他者の権利を侵害するコンテンツ（他人を欺く、嫌がらせをする、または危害を加える動画や画像など）の生成は禁止されています。この生成 AI 機能の使用は、Google の [使用禁止に関するポリシー](https://policies.google.com/terms/generative-ai/use-policy?hl=ja) の対象となります。

画像編集を行うには、入力として画像を追加します。次の例は、base64 でエンコードされた画像をアップロードする方法を示しています。複数の画像、大きなペイロード、サポートされている MIME タイプについては、 [画像認識](https://ai.google.dev/gemini-api/docs/image-understanding?hl=ja) のページをご覧ください。

```
from google import genai
from google.genai import types
from PIL import Image
from io import BytesIO

client = genai.Client()

prompt = (
    "Create a picture of my cat eating a nano-banana in a "
    "fancy restaurant under the Gemini constellation",
)

image = Image.open("/path/to/cat_image.png")

response = client.models.generate_content(
    model="gemini-2.5-flash-image-preview",
    contents=[prompt, image],
)

for part in response.candidates[0].content.parts:
    if part.text is not None:
        print(part.text)
    elif part.inline_data is not None:
        image = Image.open(BytesIO(part.inline_data.data))
        image.save("generated_image.png")
```
```
import { GoogleGenAI, Modality } from "@google/genai";
import * as fs from "node:fs";

async function main() {

  const ai = new GoogleGenAI({});

  const imagePath = "path/to/cat_image.png";
  const imageData = fs.readFileSync(imagePath);
  const base64Image = imageData.toString("base64");

  const prompt = [
    { text: "Create a picture of my cat eating a nano-banana in a" +
            "fancy restaurant under the Gemini constellation" },
    {
      inlineData: {
        mimeType: "image/png",
        data: base64Image,
      },
    },
  ];

  const response = await ai.models.generateContent({
    model: "gemini-2.5-flash-image-preview",
    contents: prompt,
  });
  for (const part of response.candidates[0].content.parts) {
    if (part.text) {
      console.log(part.text);
    } else if (part.inlineData) {
      const imageData = part.inlineData.data;
      const buffer = Buffer.from(imageData, "base64");
      fs.writeFileSync("gemini-native-image.png", buffer);
      console.log("Image saved as gemini-native-image.png");
    }
  }
}

main();
```
```
package main

import (
 "context"
 "fmt"
 "os"
 "google.golang.org/genai"
)

func main() {

 ctx := context.Background()
 client, err := genai.NewClient(ctx, nil)
 if err != nil {
     log.Fatal(err)
 }

 imagePath := "/path/to/cat_image.png"
 imgData, _ := os.ReadFile(imagePath)

 parts := []*genai.Part{
   genai.NewPartFromText("Create a picture of my cat eating a nano-banana in a fancy restaurant under the Gemini constellation"),
   &genai.Part{
     InlineData: &genai.Blob{
       MIMEType: "image/png",
       Data:     imgData,
     },
   },
 }

 contents := []*genai.Content{
   genai.NewContentFromParts(parts, genai.RoleUser),
 }

 result, _ := client.Models.GenerateContent(
     ctx,
     "gemini-2.5-flash-image-preview",
     contents,
 )

 for _, part := range result.Candidates[0].Content.Parts {
     if part.Text != "" {
         fmt.Println(part.Text)
     } else if part.InlineData != nil {
         imageBytes := part.InlineData.Data
         outputFilename := "gemini_generated_image.png"
         _ = os.WriteFile(outputFilename, imageBytes, 0644)
     }
 }
}
```
```
IMG_PATH=/path/to/cat_image.jpeg

if [[ "$(base64 --version 2>&1)" = *"FreeBSD"* ]]; then
  B64FLAGS="--input"
else
  B64FLAGS="-w0"
fi

IMG_BASE64=$(base64 "$B64FLAGS" "$IMG_PATH" 2>&1)

curl -X POST \
  "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash-image-preview:generateContent" \
    -H "x-goog-api-key: $GEMINI_API_KEY" \
    -H 'Content-Type: application/json' \
    -d "{
      \"contents\": [{
        \"parts\":[
            {\"text\": \"'Create a picture of my cat eating a nano-banana in a fancy restaurant under the Gemini constellation\"},
            {
              \"inline_data\": {
                \"mime_type\":\"image/jpeg\",
                \"data\": \"$IMG_BASE64\"
              }
            }
        ]
      }]
    }"  \
  | grep -o '"data": "[^"]*"' \
  | cut -d'"' -f4 \
  | base64 --decode > gemini-edited-image.png
```
![バナナを食べている猫の AI 生成画像](https://ai.google.dev/static/gemini-api/docs/images/cat-banana.png?hl=ja)

ナノバナナを食べる猫の AI 生成画像

## その他の画像生成モード

Gemini は、プロンプトの構造とコンテキストに基づいて、次のような他の画像操作モードもサポートしています。

- **テキストから画像とテキスト（インターリーブ）:** 関連するテキストを含む画像を出力します。
	- プロンプトの例: 「パエリアのレシピをイラスト付きで生成してください。」
- **画像とテキスト画像変換とテキスト（インターリーブ）**: 入力画像とテキストを使用して、関連する新しい画像とテキストを作成します。
	- プロンプトの例:（家具付きの部屋の画像を提示して）「この部屋に合いそうなソファの色には他にどんなものがありますか？画像を更新してください」。
- **マルチターン画像編集（チャット）:** 対話形式で画像を生成、編集し続けます。
	- プロンプトの例: \[青い車の画像をアップロードして\], 「この車をコンバーチブルにしてください。」, 「次に、色を黄色に変えてください」。

## プロンプトのガイドと戦略

Gemini 2.5 Flash 画像生成を使いこなすには、次の基本原則を理解する必要があります。

> **キーワードを列挙するだけでなく、シーンを説明します。** このモデルの強みは、言語を深く理解していることです。物語や説明文の段落は、ほとんどの場合、関連性のない単語のリストよりも、より優れた一貫性のある画像を生成します。

### 画像を生成するためのプロンプト

次の戦略は、効果的なプロンプトを作成して、思い通りの画像を生成するのに役立ちます。

#### 1\. フォトリアリスティックなシーン

リアルな画像の場合は、写真用語を使用します。カメラアングル、レンズの種類、照明、細部について言及し、モデルを写真のようにリアルな結果に導きます。

```
A photorealistic [shot type] of [subject], [action or expression], set in
[environment]. The scene is illuminated by [lighting description], creating
a [mood] atmosphere. Captured with a [camera/lens details], emphasizing
[key textures and details]. The image should be in a [aspect ratio] format.
```
```
A photorealistic close-up portrait of an elderly Japanese ceramicist with
deep, sun-etched wrinkles and a warm, knowing smile. He is carefully
inspecting a freshly glazed tea bowl. The setting is his rustic,
sun-drenched workshop. The scene is illuminated by soft, golden hour light
streaming through a window, highlighting the fine texture of the clay.
Captured with an 85mm portrait lens, resulting in a soft, blurred background
(bokeh). The overall mood is serene and masterful. Vertical portrait
orientation.
```
```
from google import genai
from google.genai import types
from PIL import Image
from io import BytesIO

client = genai.Client()

# Generate an image from a text prompt
response = client.models.generate_content(
    model="gemini-2.5-flash-image-preview",
    contents="A photorealistic close-up portrait of an elderly Japanese ceramicist with deep, sun-etched wrinkles and a warm, knowing smile. He is carefully inspecting a freshly glazed tea bowl. The setting is his rustic, sun-drenched workshop with pottery wheels and shelves of clay pots in the background. The scene is illuminated by soft, golden hour light streaming through a window, highlighting the fine texture of the clay and the fabric of his apron. Captured with an 85mm portrait lens, resulting in a soft, blurred background (bokeh). The overall mood is serene and masterful.",
)

image_parts = [
    part.inline_data.data
    for part in response.candidates[0].content.parts
    if part.inline_data
]

if image_parts:
    image = Image.open(BytesIO(image_parts[0]))
    image.save('photorealistic_example.png')
    image.show()
```
![日本の高齢の陶芸家のリアルなクローズアップ ポートレート...](https://ai.google.dev/static/gemini-api/docs/images/photorealistic_example.png?hl=ja)

日本の高齢の陶芸家のリアルなクローズアップ ポートレート...

#### 2\. スタイルを適用したイラストとステッカー

ステッカー、アイコン、アセットを作成する場合は、スタイルを明確に指定し、背景を透明にするようリクエストします。

```
A [style] sticker of a [subject], featuring [key characteristics] and a
[color palette]. The design should have [line style] and [shading style].
The background must be transparent.
```
```
A kawaii-style sticker of a happy red panda wearing a tiny bamboo hat. It's
munching on a green bamboo leaf. The design features bold, clean outlines,
simple cel-shading, and a vibrant color palette. The background must be white.
```
```
from google import genai
from google.genai import types
from PIL import Image
from io import BytesIO

client = genai.Client()

# Generate an image from a text prompt
response = client.models.generate_content(
    model="gemini-2.5-flash-image-preview",
    contents="A kawaii-style sticker of a happy red panda wearing a tiny bamboo hat. It's munching on a green bamboo leaf. The design features bold, clean outlines, simple cel-shading, and a vibrant color palette. The background must be white.",
)

image_parts = [
    part.inline_data.data
    for part in response.candidates[0].content.parts
    if part.inline_data
]

if image_parts:
    image = Image.open(BytesIO(image_parts[0]))
    image.save('red_panda_sticker.png')
    image.show()
```
![幸せそうな赤い...](https://ai.google.dev/static/gemini-api/docs/images/red_panda_sticker.png?hl=ja)

幸せそうなレッサーパンダの kawaii スタイルのステッカー...

#### 3\. 画像内の正確なテキスト

Gemini はテキストのレンダリングに優れています。テキスト、フォント スタイル（説明）、全体的なデザインを明確にしてください。

```
Create a [image type] for [brand/concept] with the text "[text to render]"
in a [font style]. The design should be [style description], with a
[color scheme].
```
```
Create a modern, minimalist logo for a coffee shop called 'The Daily Grind'.
The text should be in a clean, bold, sans-serif font. The design should
feature a simple, stylized icon of a a coffee bean seamlessly integrated
with the text. The color scheme is black and white.
```
```
from google import genai
from google.genai import types
from PIL import Image
from io import BytesIO

client = genai.Client()

# Generate an image from a text prompt
response = client.models.generate_content(
    model="gemini-2.5-flash-image-preview",
    contents="Create a modern, minimalist logo for a coffee shop called 'The Daily Grind'. The text should be in a clean, bold, sans-serif font. The design should feature a simple, stylized icon of a a coffee bean seamlessly integrated with the text. The color scheme is black and white.",
)

image_parts = [
    part.inline_data.data
    for part in response.candidates[0].content.parts
    if part.inline_data
]

if image_parts:
    image = Image.open(BytesIO(image_parts[0]))
    image.save('logo_example.png')
    image.show()
```
![「The Daily Grind」というコーヒー ショップのモダンでミニマルなロゴを作成してください。](https://ai.google.dev/static/gemini-api/docs/images/logo_example.png?hl=ja)

#### 4\. 商品のモックアップと広告写真

e コマース、広告、ブランディング用のクリーンでプロフェッショナルな商品写真を撮影するのに最適です。

```
A high-resolution, studio-lit product photograph of a [product description]
on a [background surface/description]. The lighting is a [lighting setup,
e.g., three-point softbox setup] to [lighting purpose]. The camera angle is
a [angle type] to showcase [specific feature]. Ultra-realistic, with sharp
focus on [key detail]. [Aspect ratio].
```
```
A high-resolution, studio-lit product photograph of a minimalist ceramic
coffee mug in matte black, presented on a polished concrete surface. The
lighting is a three-point softbox setup designed to create soft, diffused
highlights and eliminate harsh shadows. The camera angle is a slightly
elevated 45-degree shot to showcase its clean lines. Ultra-realistic, with
sharp focus on the steam rising from the coffee. Square image.
```
```
from google import genai
from google.genai import types
from PIL import Image
from io import BytesIO

client = genai.Client()

# Generate an image from a text prompt
response = client.models.generate_content(
    model="gemini-2.5-flash-image-preview",
    contents="A high-resolution, studio-lit product photograph of a minimalist ceramic coffee mug in matte black, presented on a polished concrete surface. The lighting is a three-point softbox setup designed to create soft, diffused highlights and eliminate harsh shadows. The camera angle is a slightly elevated 45-degree shot to showcase its clean lines. Ultra-realistic, with sharp focus on the steam rising from the coffee. Square image.",
)

image_parts = [
    part.inline_data.data
    for part in response.candidates[0].content.parts
    if part.inline_data
]

if image_parts:
    image = Image.open(BytesIO(image_parts[0]))
    image.save('product_mockup.png')
    image.show()
```
![ミニマルなセラミック製コーヒー マグのスタジオ撮影による高解像度の商品写真。](https://ai.google.dev/static/gemini-api/docs/images/product_mockup.png?hl=ja)

ミニマリストのセラミック製コーヒー マグのスタジオ照明付き高解像度の商品写真...

#### 5\. ミニマルでネガティブ スペースを活かしたデザイン

テキストを重ねて表示するウェブサイト、プレゼンテーション、マーケティング資料の背景の作成に最適です。

```
A minimalist composition featuring a single [subject] positioned in the
[bottom-right/top-left/etc.] of the frame. The background is a vast, empty
[color] canvas, creating significant negative space. Soft, subtle lighting.
[Aspect ratio].
```
```
A minimalist composition featuring a single, delicate red maple leaf
positioned in the bottom-right of the frame. The background is a vast, empty
off-white canvas, creating significant negative space for text. Soft,
diffused lighting from the top left. Square image.
```
```
from google import genai
from google.genai import types
from PIL import Image
from io import BytesIO

client = genai.Client()

# Generate an image from a text prompt
response = client.models.generate_content(
    model="gemini-2.5-flash-image-preview",
    contents="A minimalist composition featuring a single, delicate red maple leaf positioned in the bottom-right of the frame. The background is a vast, empty off-white canvas, creating significant negative space for text. Soft, diffused lighting from the top left. Square image.",
)

image_parts = [
    part.inline_data.data
    for part in response.candidates[0].content.parts
    if part.inline_data
]

if image_parts:
    image = Image.open(BytesIO(image_parts[0]))
    image.save('minimalist_design.png')
    image.show()
```
![1 枚の繊細な紅葉の葉をフィーチャーしたミニマルな構図...](https://ai.google.dev/static/gemini-api/docs/images/minimalist_design.png?hl=ja)

1 枚の繊細な紅葉の葉をフィーチャーしたミニマリストの構図...

#### 6\. 連続したアート（コミック パネル / ストーリーボード）

キャラクターの一貫性とシーンの説明に基づいて、ビジュアル ストーリーテリング用のパネルを作成します。

```
A single comic book panel in a [art style] style. In the foreground,
[character description and action]. In the background, [setting details].
The panel has a [dialogue/caption box] with the text "[Text]". The lighting
creates a [mood] mood. [Aspect ratio].
```
```
A single comic book panel in a gritty, noir art style with high-contrast
black and white inks. In the foreground, a detective in a trench coat stands
under a flickering streetlamp, rain soaking his shoulders. In the
background, the neon sign of a desolate bar reflects in a puddle. A caption
box at the top reads "The city was a tough place to keep secrets." The
lighting is harsh, creating a dramatic, somber mood. Landscape.
```
```
from google import genai
from google.genai import types
from PIL import Image
from io import BytesIO

client = genai.Client()

# Generate an image from a text prompt
response = client.models.generate_content(
    model="gemini-2.5-flash-image-preview",
    contents="A single comic book panel in a gritty, noir art style with high-contrast black and white inks. In the foreground, a detective in a trench coat stands under a flickering streetlamp, rain soaking his shoulders. In the background, the neon sign of a desolate bar reflects in a puddle. A caption box at the top reads \"The city was a tough place to keep secrets.\" The lighting is harsh, creating a dramatic, somber mood. Landscape.",
)

image_parts = [
    part.inline_data.data
    for part in response.candidates[0].content.parts
    if part.inline_data
]

if image_parts:
    image = Image.open(BytesIO(image_parts[0]))
    image.save('comic_panel.png')
    image.show()
```
![ざらざらしたノワール アート スタイルの漫画本の一コマ...](https://ai.google.dev/static/gemini-api/docs/images/comic_panel.png?hl=ja)

ざらざらしたノワール アート スタイルのコミックブックの 1 つのパネル...

### 画像を編集するためのプロンプト

これらの例は、編集、構図、スタイル転送のテキスト プロンプトとともに画像を提供する方法を示しています。

#### 1\. 要素の追加と削除

画像を提供し、変更内容を説明します。モデルは、元の画像のスタイル、照明、遠近法と一致します。

```
Using the provided image of [subject], please [add/remove/modify] [element]
to/from the scene. Ensure the change is [description of how the change should
integrate].
```
```
"Using the provided image of my cat, please add a small, knitted wizard hat
on its head. Make it look like it's sitting comfortably and matches the soft
lighting of the photo."
```
```
from google import genai
from google.genai import types
from PIL import Image
from io import BytesIO

client = genai.Client()

# Base image prompt: "A photorealistic picture of a fluffy ginger cat sitting on a wooden floor, looking directly at the camera. Soft, natural light from a window."
image_input = Image.open('/path/to/your/cat_photo.png')
text_input = """Using the provided image of my cat, please add a small, knitted wizard hat on its head. Make it look like it's sitting comfortably and not falling off."""

# Generate an image from a text prompt
response = client.models.generate_content(
    model="gemini-2.5-flash-image-preview",
    contents=[text_input, image_input],
)

image_parts = [
    part.inline_data.data
    for part in response.candidates[0].content.parts
    if part.inline_data
]

if image_parts:
    image = Image.open(BytesIO(image_parts[0]))
    image.save('cat_with_hat.png')
    image.show()
```

| 入力 | 出力 |
| --- | --- |
| ![ふわふわの茶色の猫のリアルな写真。](https://ai.google.dev/static/gemini-api/docs/images/cat.png?hl=ja)  ふわふわした茶色の猫のリアルな写真... | ![提供した猫の画像を使用して、小さな編み物の魔法使いの帽子を追加してください。](https://ai.google.dev/static/gemini-api/docs/images/cat_with_hat.png?hl=ja)  提供された猫の画像を使用して、小さな編み物の魔法使いの帽子を追加してください。 |

#### 2\. インペイント（セマンティック マスク）

会話形式で「マスク」を定義して、画像の特定の部分を編集し、残りの部分はそのままにします。

```
Using the provided image, change only the [specific element] to [new
element/description]. Keep everything else in the image exactly the same,
preserving the original style, lighting, and composition.
```
```
"Using the provided image of a living room, change only the blue sofa to be
a vintage, brown leather chesterfield sofa. Keep the rest of the room,
including the pillows on the sofa and the lighting, unchanged."
```
```
from google import genai
from google.genai import types
from PIL import Image
from io import BytesIO

client = genai.Client()

# Base image prompt: "A wide shot of a modern, well-lit living room with a prominent blue sofa in the center. A coffee table is in front of it and a large window is in the background."
living_room_image = Image.open('/path/to/your/living_room.png')
text_input = """Using the provided image of a living room, change only the blue sofa to be a vintage, brown leather chesterfield sofa. Keep the rest of the room, including the pillows on the sofa and the lighting, unchanged."""

# Generate an image from a text prompt
response = client.models.generate_content(
    model="gemini-2.5-flash-image-preview",
    contents=[living_room_image, text_input],
)

image_parts = [
    part.inline_data.data
    for part in response.candidates[0].content.parts
    if part.inline_data
]

if image_parts:
    image = Image.open(BytesIO(image_parts[0]))
    image.save('living_room_edited.png')
    image.show()
```

| 入力 | 出力 |
| --- | --- |
| ![明るく照らされたモダンなリビングルームのワイドショット。](https://ai.google.dev/static/gemini-api/docs/images/living_room.png?hl=ja)  明るく照らされたモダンなリビングルームのワイドショット... | ![提供されたリビングルームの画像を使用して、青いソファだけをヴィンテージの茶色の革製チェスターフィールド ソファに変更してください。](https://ai.google.dev/static/gemini-api/docs/images/living_room_edited.png?hl=ja)  提供されたリビングルームの画像を使用して、青いソファだけをヴィンテージの茶色の革製チェスターフィールド ソファに変更してください。 |

#### 3\. 画風変換

画像を提供し、別の芸術的なスタイルでコンテンツを再作成するようモデルに指示します。

```
Transform the provided photograph of [subject] into the artistic style of [artist/art style]. Preserve the original composition but render it with [description of stylistic elements].
```
```
"Transform the provided photograph of a modern city street at night into the artistic style of Vincent van Gogh's 'Starry Night'. Preserve the original composition of buildings and cars, but render all elements with swirling, impasto brushstrokes and a dramatic palette of deep blues and bright yellows."
```
```
from google import genai
from google.genai import types
from PIL import Image
from io import BytesIO

client = genai.Client()

# Base image prompt: "A photorealistic, high-resolution photograph of a busy city street in New York at night, with bright neon signs, yellow taxis, and tall skyscrapers."
city_image = Image.open('/path/to/your/city.png')
text_input = """Transform the provided photograph of a modern city street at night into the artistic style of Vincent van Gogh's 'Starry Night'. Preserve the original composition of buildings and cars, but render all elements with swirling, impasto brushstrokes and a dramatic palette of deep blues and bright yellows."""

# Generate an image from a text prompt
response = client.models.generate_content(
    model="gemini-2.5-flash-image-preview",
    contents=[city_image, text_input],
)

image_parts = [
    part.inline_data.data
    for part in response.candidates[0].content.parts
    if part.inline_data
]

if image_parts:
    image = Image.open(BytesIO(image_parts[0]))
    image.save('city_style_transfer.png')
    image.show()
```

| 入力 | 出力 |
| --- | --- |
| ![にぎやかな街の通りのリアルな高解像度写真...](https://ai.google.dev/static/gemini-api/docs/images/city.png?hl=ja)  賑やかな街並みの高解像度でリアルな写真... | ![夜の現代的な街並みの写真を提供します。](https://ai.google.dev/static/gemini-api/docs/images/city_style_transfer.png?hl=ja)  夜の現代的な街並みの写真を提供します。この写真を... |

#### 4\. 高度な合成: 複数の画像を組み合わせる

複数の画像をコンテキストとして提供し、新しい複合シーンを作成します。これは、商品のモックアップやクリエイティブなコラージュに最適です。

```
Create a new image by combining the elements from the provided images. Take
the [element from image 1] and place it with/on the [element from image 2].
The final image should be a [description of the final scene].
```
```
"Create a professional e-commerce fashion photo. Take the blue floral dress
from the first image and let the woman from the second image wear it.
Generate a realistic, full-body shot of the woman wearing the dress, with
the lighting and shadows adjusted to match the outdoor environment."
```
```
from google import genai
from google.genai import types
from PIL import Image
from io import BytesIO

client = genai.Client()

# Base image prompts:
# 1. Dress: "A professionally shot photo of a blue floral summer dress on a plain white background, ghost mannequin style."
# 2. Model: "Full-body shot of a woman with her hair in a bun, smiling, standing against a neutral grey studio background."
dress_image = Image.open('/path/to/your/dress.png')
model_image = Image.open('/path/to/your/model.png')

text_input = """Create a professional e-commerce fashion photo. Take the blue floral dress from the first image and let the woman from the second image wear it. Generate a realistic, full-body shot of the woman wearing the dress, with the lighting and shadows adjusted to match the outdoor environment."""

# Generate an image from a text prompt
response = client.models.generate_content(
    model="gemini-2.5-flash-image-preview",
    contents=[dress_image, model_image, text_input],
)

image_parts = [
    part.inline_data.data
    for part in response.candidates[0].content.parts
    if part.inline_data
]

if image_parts:
    image = Image.open(BytesIO(image_parts[0]))
    image.save('fashion_ecommerce_shot.png')
    image.show()
```

| 入力 1 | 入力 2 | 出力 |
| --- | --- | --- |
| ![青い花柄の夏のワンピースのプロが撮影した写真...](https://ai.google.dev/static/gemini-api/docs/images/dress.png?hl=ja)  青い花柄の夏のドレスのプロが撮影した写真... | ![髪を束ねた女性の全身写真...](https://ai.google.dev/static/gemini-api/docs/images/model.png?hl=ja)  髪を束ねた女性の全身ショット... | ![プロフェッショナルな e コマースのファッション写真を作成して...](https://ai.google.dev/static/gemini-api/docs/images/fashion_ecommerce_shot.png?hl=ja)  プロフェッショナルな e コマース ファッション写真を作成して... |

#### 5\. 高忠実度のディテールの保持

編集時に重要な詳細（顔やロゴなど）が保持されるように、編集リクエストとともに詳細な説明を記載してください。

```
Using the provided images, place [element from image 2] onto [element from
image 1]. Ensure that the features of [element from image 1] remain
completely unchanged. The added element should [description of how the
element should integrate].
```
```
"Take the first image of the woman with brown hair, blue eyes, and a neutral
expression. Add the logo from the second image onto her black t-shirt.
Ensure the woman's face and features remain completely unchanged. The logo
should look like it's naturally printed on the fabric, following the folds
of the shirt."
```
```
from google import genai
from google.genai import types
from PIL import Image
from io import BytesIO

client = genai.Client()

# Base image prompts:
# 1. Woman: "A professional headshot of a woman with brown hair and blue eyes, wearing a plain black t-shirt, against a neutral studio background."
# 2. Logo: "A simple, modern logo with the letters 'G' and 'A' in a white circle."
woman_image = Image.open('/path/to/your/woman.png')
logo_image = Image.open('/path/to/your/logo.png')
text_input = """Take the first image of the woman with brown hair, blue eyes, and a neutral expression. Add the logo from the second image onto her black t-shirt. Ensure the woman's face and features remain completely unchanged. The logo should look like it's naturally printed on the fabric, following the folds of the shirt."""

# Generate an image from a text prompt
response = client.models.generate_content(
    model="gemini-2.5-flash-image-preview",
    contents=[woman_image, logo_image, text_input],
)

image_parts = [
    part.inline_data.data
    for part in response.candidates[0].content.parts
    if part.inline_data
]

if image_parts:
    image = Image.open(BytesIO(image_parts[0]))
    image.save('woman_with_logo.png')
    image.show()
```

| 入力 1 | 入力 2 | 出力 |
| --- | --- | --- |
| ![茶色の髪と青い目の女性のプロフェッショナルな顔写真...](https://ai.google.dev/static/gemini-api/docs/images/woman.png?hl=ja)  茶色の髪と青い目の女性のプロフェッショナルなヘッドショット... | ![「G」と「A」の文字が入ったシンプルでモダンなロゴ...](https://ai.google.dev/static/gemini-api/docs/images/logo.png?hl=ja)  「G」と「A」の文字を使ったシンプルでモダンなロゴ... | ![茶色の髪、青い目、無表情の女性の最初の画像を取得します。](https://ai.google.dev/static/gemini-api/docs/images/woman_with_logo.png?hl=ja)  茶色の髪、青い目、無表情の女性の最初の画像を撮影して... |

### ベスト プラクティス

結果を優れたものにするには、次のプロフェッショナルな戦略をワークフローに組み込みます。

- **非常に具体的な内容にする:** 詳細な内容にするほど、より細かく制御できます。「ファンタジー アーマー」ではなく、「銀の葉の模様がエッチングされた、装飾的なエルフのプレート アーマー。高い襟とハヤブサの翼の形をした肩当てが付いている」のように説明します。
- **背景と意図を説明する:** 画像の *目的* を説明します。モデルのコンテキストの理解が最終出力に影響します。たとえば、「高級でミニマリストなスキンケア ブランドのロゴを作成して」と入力すると、「ロゴを作成して」と入力するよりも良い結果が得られます。
- **イテレーションと改良:** 最初の試行で完璧な画像が生成されるとは限りません。モデルの会話的な性質を利用して、小さな変更を行います。「素晴らしいですが、照明をもう少し暖かくしてもらえますか？」や「すべてそのままにして、キャラクターの表情をもう少し真剣なものに変えてください」などのプロンプトでフォローアップします。
- **ステップバイステップの手順を使用する:** 多くの要素を含む複雑なシーンでは、プロンプトをステップに分割します。「まず、夜明けの静かで霧がかった森の背景を作成します。次に、前景に苔むした古代の石の祭壇を追加します。最後に、祭壇の上に光る剣を 1 本置きます。」
- **「セマンティック ネガティブ プロンプト」を使用する:** 「車がない」と言う代わりに、「交通の兆候がない空っぽの寂れた通り」のように、望ましいシーンを肯定的に説明します。
- **カメラを制御する:** 写真や映画の用語を使用して、構図を制御します。 `wide-angle shot` 、 `macro shot` 、 `low-angle perspective` などの用語。

## 制限事項

- 最高のパフォーマンスを実現するには、EN、es-MX、ja-JP、zh-CN、hi-IN のいずれかの言語を使用してください。
- 画像生成では、音声や動画の入力はサポートされていません。
- モデルは、ユーザーが明示的にリクエストした画像出力の数を正確に守るとは限りません。
- モデルは、入力として最大 3 枚の画像を使用する場合に最適に動作します。
- 画像のテキストを生成する場合、Gemini は、まずテキストを生成してから、そのテキストを含む画像をリクエストすると、最適な結果が得られます。
- 現在、EEA、スイス、英国では、子供の画像をアップロードすることはできません。
- すべての生成画像には [SynthID の透かし](https://ai.google.dev/responsible/docs/safeguards/synthid?hl=ja) が埋め込まれています。

## Imagen を使用する場面

Gemini の組み込み画像生成機能に加えて、Gemini API を介して、Google の専用画像生成モデルである [Imagen](https://ai.google.dev/gemini-api/docs/imagen?hl=ja) にアクセスすることもできます。

| 属性 | Imagen | Gemini ネイティブ画像 |
| --- | --- | --- |
| 強み | これまでで最も高性能な画像生成モデル。写実的な画像、鮮明な画像、スペルとタイポグラフィの改善におすすめします。 | **デフォルトの推奨事項。**   比類のない柔軟性、コンテキストに沿った理解、シンプルでマスクフリーの編集。マルチターンの会話型編集を独自に実現。 |
| 対象 | 一般提供 | プレビュー（本番環境での使用が許可されています） |
| レイテンシ | **低** 。ほぼリアルタイムのパフォーマンス向けに最適化されています。 | 比較的長い。高度な機能にはより多くの計算が必要です。 |
| 費用 | 特定のタスクに費用対効果が高い。$0.02/画像～ $0.12/画像 | トークンベースの料金。画像出力 100 万トークンあたり $30（画像出力は画像あたり 1,290 トークンでトークン化、最大 1,024×1,024 ピクセル） |
| 推奨されるタスク | - 画質、写真のようなリアルさ、芸術的なディテール、特定のスタイル（印象派、アニメなど）を最優先にしたい場合。 - ブランディング、スタイル、ロゴや商品デザインの生成。 - 高度なスペルやタイポグラフィを生成する。 | - テキストと画像をシームレスにブレンドする、テキストと画像が混在するコンテンツの生成。 - 複数の画像のクリエイティブ要素を 1 つのプロンプトで組み合わせます。 - 画像に非常に具体的な編集を加えたり、簡単な言語コマンドで個々の要素を変更したり、画像を繰り返し操作したりできます。 - 元の被写体の形や詳細を保持しながら、ある画像の特定のデザインやテクスチャを別の画像に適用します。 |

Imagen 4 は、Imagen での画像生成を開始する際に使用するモデルです。高度なユースケースや、最高の画質が必要な場合は、Imagen 4 Ultra を選択します（一度に 1 枚の画像しか生成できません）。

## 次のステップ

- その他の例とコードサンプルについては、 [クックブック ガイド](https://colab.sandbox.google.com/github/google-gemini/cookbook/blob/main/quickstarts/Image_out.ipynb?hl=ja) をご覧ください。
- Gemini API を使用して動画を生成する方法については、 [Veo ガイド](https://ai.google.dev/gemini-api/docs/video?hl=ja) をご覧ください。
- Gemini モデルの詳細については、 [Gemini モデル](https://ai.google.dev/gemini-api/docs/models/gemini?hl=ja) をご覧ください。

特に記載のない限り、このページのコンテンツは [クリエイティブ・コモンズの表示 4.0 ライセンス](https://creativecommons.org/licenses/by/4.0/) により使用許諾されます。コードサンプルは [Apache 2.0 ライセンス](https://www.apache.org/licenses/LICENSE-2.0) により使用許諾されます。詳しくは、 [Google Developers サイトのポリシー](https://developers.google.com/site-policies?hl=ja) をご覧ください。Java は Oracle および関連会社の登録商標です。

最終更新日 2025-08-27 UTC。