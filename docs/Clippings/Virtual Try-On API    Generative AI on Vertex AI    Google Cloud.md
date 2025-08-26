---
title: "Virtual Try-On API  |  Generative AI on Vertex AI  |  Google Cloud"
source: "https://cloud.google.com/vertex-ai/generative-ai/docs/model-reference/virtual-try-on-api?hl=en"
author:
published:
created: 2025-08-25
description: "Use Virtual Try-On to generate virtual try-on images of people wearing clothing products"
tags:
  - "clippings"
---
This guide shows you how to use the Virtual Try-On API to generate virtual try-on images of people modeling clothing products.

This page covers the following topics:

- **[Image input options](https://cloud.google.com/vertex-ai/generative-ai/docs/model-reference/?hl=en#image-input-options):** Learn about the different ways to provide images in your API request.
- **[Supported model versions](https://cloud.google.com/vertex-ai/generative-ai/docs/model-reference/?hl=en#supported-model-versions):** Find information about the model version that Virtual Try-On supports.
- **[HTTP request](https://cloud.google.com/vertex-ai/generative-ai/docs/model-reference/?hl=en#http-request):** Review the structure and parameters of an API request.
- **[Request body description](https://cloud.google.com/vertex-ai/generative-ai/docs/model-reference/?hl=en#request-body-description):** See a detailed description of the request body fields.
- **[Sample request](https://cloud.google.com/vertex-ai/generative-ai/docs/model-reference/?hl=en#sample-request):** See a complete example of an API request.

## Image input options

When you provide an image of a person or a clothing product, you can specify it as either a Base64-encoded byte string or a Cloud Storage URI. The following table provides a comparison of these two options to help you decide which one to use.

| Option               | Description                                                       | Pros                                                            | Cons                                                                                   | Use Case                                                                                                              |
| -------------------- | ----------------------------------------------------------------- | --------------------------------------------------------------- | -------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| `bytesBase64Encoded` | The image data is sent directly within the JSON request body.     | Simple for smaller images; no need for a separate storage step. | Increases request size; not suitable for very large images due to JSON payload limits. | Quick tests or applications where images are generated or processed on the fly and not stored long-term.              |
| `gcsUri`             | A URI pointing to an image file stored in a Cloud Storage bucket. | Efficient for large images; keeps the request payload small.    | Requires uploading the image to Cloud Storage first, which adds an extra step.         | Batch processing, workflows where images are already stored in Cloud Storage, or when dealing with large image files. |

## Supported model versions

Virtual Try-On supports the following model:

- `virtual-try-on-preview-08-04`

For more information about the features that the model supports, see [Imagen models](https://cloud.google.com/vertex-ai/generative-ai/docs/models#imagen-models).

## HTTP request

To generate an image, send a `POST` request to the model's `predict` endpoint.

```
curl -X POST \
  -H "Authorization: Bearer $(gcloud auth print-access-token)" \
  -H "Content-Type: application/json" \
https://LOCATION-aiplatform.googleapis.com/v1/projects/PROJECT_ID/locations/LOCATION/publishers/google/models/MODEL_ID:predict \

-d '{
  "instances": [
    {
      "personImage": {
        "image": {
          // Union field can be only one of the following:
          "bytesBase64Encoded": string,
          "gcsUri": string,
        }
      },
      "productImages": [
        {
          "image": {
            // Union field can be only one of the following:
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

<table><thead><tr><th colspan="2">Instances</th></tr></thead><tbody><tr><td><p><code>personImage</code></p></td><td><p><code>string</code></p><p>Required. An image of a person to try-on the clothing product, which can be either of the following:</p><ul><li>A <code>bytesBase64Encoded</code> string that encodes an image.</li><li>A <code>gcsUri</code> string URI to a Cloud Storage bucket location.</li></ul></td></tr><tr><td><p><code>productImages</code></p></td><td><p><code>string</code></p><p>Required. An image of a product to try-on a person, which can be either of the following:</p><ul><li>A <code>bytesBase64Encoded</code> string that encodes an image.</li><li>A <code>gcsUri</code> string URI to a Cloud Storage bucket location.</li></ul></td></tr></tbody></table>

<table><thead><tr><th colspan="2">Parameters</th></tr></thead><tbody><tr><td><code>add<wbr>Watermark</code></td><td><p><code>bool</code></p><p>Optional. Add an invisible watermark to the generated images.</p><p>The default value is <code>true</code>.</p></td></tr><tr><td><p><code>baseSteps</code></p></td><td><p><code>int</code></p><p>Required. An integer that controls image generation, with higher steps trading higher quality for increased latency.</p><p>Integer values greater than <code>0</code>. The default is <code>32</code>.</p></td></tr><tr><td><code>person<wbr>Generation</code></td><td><p><code>string</code></p><p>Optional. Allow generation of people by the model. The following values are supported:</p><ul><li><code>"dont_allow"</code>: Disallow the inclusion of people or faces in images.</li><li><code>"allow_adult"</code>: Allow generation of adults only.</li><li><code>"allow_all"</code>: Allow generation of people of all ages.</li></ul><p>The default value is <code>"allow_adult"</code>.</p></td></tr><tr><td><code>safety<wbr>Setting</code></td><td><p><code>string</code></p><p>Optional. Adds a filter level to safety filtering. The following values are supported:</p><ul><li><code>"block_low_and_above"</code>: Strongest filtering level, most strict blocking. Deprecated value: <code>"block_most"</code>.</li><li><code>"block_medium_and_above"</code>: Block some problematic prompts and responses. Deprecated value: <code>"block_some"</code>.</li><li><code>"block_only_high"</code>: Reduces the number of requests blocked due to safety filters. May increase objectionable content generated by Imagen. Deprecated value: <code>"block_few"</code>.</li><li><code>"block_none"</code>: Block very few problematic prompts and responses. Access to this feature is restricted. Previous field value: <code>"block_fewest"</code>.</li></ul><p>The default value is <code>"block_medium_and_above"</code>.</p></td></tr><tr><td><p><code>sampleCount</code></p></td><td><p><code>int</code></p><p>Required. The number of images to generate.</p><p>An integer value between <code>1</code> and <code>4</code>, inclusive. The default value is <code>1</code>.</p></td></tr><tr><td><code>seed</code></td><td><p><code>Uint32</code></p><p>Optional. The random seed for image generation. This isn't available when <code>addWatermark</code> is set to <code>true</code>.</p></td></tr><tr><td><code>storage<wbr>Uri</code></td><td><p><code>string</code></p><p>Optional. A string URI to a Cloud Storage bucket location to store the generated images.</p></td></tr><tr><td><code>output<wbr>Options</code></td><td><p><code>outputOptions</code></p><p>Optional. Describes the output image format in an <a href="https://cloud.google.com/vertex-ai/generative-ai/docs/model-reference/?hl=en#output-options"><code>outputOptions</code> object</a>.</p></td></tr></tbody></table>

### Output options object

The `outputOptions` object describes the image output.

<table><tbody><tr><th colspan="2">Parameters</th></tr><tr><td><code>output<wbr>Options.<wbr>mime<wbr>Type</code></td><td><p>Optional: <code>string</code></p><p>The image output format.. The following values are supported:</p><ul><li><code>"image/png"</code>: Save as a PNG image</li><li><code>"image/jpeg"</code>: Save as a JPEG image</li></ul><p>The default value is <code>"image/png"</code>.</p></td></tr><tr><td><code>output<wbr>Options.<wbr>compression<wbr>Quality</code></td><td><p>Optional: <code>int</code></p><p>The level of compression if the output type is <code>"image/jpeg"</code>. Accepted values are <code>0</code> through <code>100</code>. The default value is <code>75</code>.</p></td></tr></tbody></table>

## Sample request

Before using any of the request data, make the following replacements:

- REGION: The region that your project is located in. For more information about supported regions, see [Generative AI on Vertex AI locations](https://cloud.google.com/vertex-ai/generative-ai/docs/learn/locations).
- PROJECT\_ID: Your Google Cloud [project ID](https://cloud.google.com/resource-manager/docs/creating-managing-projects#identifiers).
- BASE64\_PERSON\_IMAGE: The Base64-encoded image of the person image.
- BASE64\_PRODUCT\_IMAGE: The Base64-encoded image of the product image.
- IMAGE\_COUNT: The number of images to generate. The accepted range of values is `1` to `4`.
- GCS\_OUTPUT\_PATH: The Cloud Storage path to store the virtual try-on output to.

HTTP method and URL:

```
POST https://REGION-aiplatform.googleapis.com/v1/projects/PROJECT_ID/locations/REGION/publishers/google/models/virtual-try-on-preview-08-04:predict
```

Request JSON body:

```
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

To send your request, choose one of these options:

Save the request body in a file named `request.json`, and execute the following command:

```
curl -X POST \
     -H "Authorization: Bearer $(gcloud auth print-access-token)" \
     -H "Content-Type: application/json; charset=utf-8" \
     -d @request.json \
     "https://REGION-aiplatform.googleapis.com/v1/projects/PROJECT_ID/locations/REGION/publishers/google/models/virtual-try-on-preview-08-04:predict"
```

Save the request body in a file named `request.json`, and execute the following command:

The request returns image objects. In this example, two image objects are returned, with two prediction objects as base64-encoded images.
```
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

Except as otherwise noted, the content of this page is licensed under the [Creative Commons Attribution 4.0 License](https://creativecommons.org/licenses/by/4.0/), and code samples are licensed under the [Apache 2.0 License](https://www.apache.org/licenses/LICENSE-2.0). For details, see the [Google Developers Site Policies](https://developers.google.com/site-policies). Java is a registered trademark of Oracle and/or its affiliates.

Last updated 2025-08-21 UTC.

新しいページが読み込まれました。