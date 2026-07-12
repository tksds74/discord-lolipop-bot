# ビルドステージ
FROM golang:1.25-bookworm AS builder

WORKDIR /build

# 依存関係のダウンロード
COPY go.mod go.sum ./
RUN go mod download

# ソースコードのコピーとビルド
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o bot ./cmd/lolipop-bot

# 実行ステージ (最小構成)
FROM debian:bookworm-slim

# 必要な最小限のパッケージをインストール
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# ビルド済みバイナリをコピー
COPY --from=builder /build/bot .

# バイナリを実行
CMD ["./bot"]
