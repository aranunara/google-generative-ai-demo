FROM golang:1.25-alpine AS builder

WORKDIR /app

# 依存関係を先にコピー（キャッシュ効率化）
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# ソースコードは最後にコピー
COPY . .
# バイナリサイズを最適化（-w: デバッグ情報削除, -s: シンボルテーブル削除）
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main .

FROM alpine:latest

# Cloud Run向けのメタデータ
LABEL maintainer="tryon-demo-team"
LABEL description="Virtual Try-On Demo optimized for Cloud Run"

# 非rootユーザーの作成
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# 必要なパッケージのインストール
RUN apk --no-cache add ca-certificates curl wget

WORKDIR /app

# アプリケーションファイルをコピーし、所有者を変更
COPY --from=builder /app/main .
COPY docker-entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/docker-entrypoint.sh && \
    chown -R appuser:appgroup /app

# Cloud Run最適化の環境変数
ENV PORT=8080
ENV GOGC=100
ENV GOMEMLIMIT=512MiB

# 非rootユーザーに切り替え
USER appuser

# Cloud Run向けに最適化されたヘルスチェック
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["/app/main"]
