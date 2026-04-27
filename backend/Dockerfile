# ビルド用
FROM golang:1.26-alpine AS builder

# 作業場所
WORKDIR /backend

# 依存関係をコピーしてダウンロード
COPY go.mod go.sum ./
RUN go mod download

# ソースをコピーしてビルド
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/server


# 実行用
FROM alpine:3.19

# non-root ユーザーの追加
RUN adduser -D -u 10001 appuser

# 作業場所
WORKDIR /backend

# ビルド環境からバイナリーをコピー
COPY --from=builder /backend/server ./

# ユーザー指定
USER appuser

# ポートを公開
EXPOSE 8080

# 実行ファイルを実行
CMD [ "./server" ]
