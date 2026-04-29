# mago.ai

シニア（おばあちゃん）向けに、iPhone やスマートフォン操作の質問に LINE で答える Bot。
お孫さんが管理画面でユーザーを登録 → おばあちゃんが LINE で質問 → Claude が優しく回答する設計。

## 主な機能

- **LINE Bot**：おばあちゃんが質問すると、Claude が iPhone 操作の手順をやさしく回答
- **管理画面**：お孫さんが Google ログイン → 6 桁の登録コードを発行 → 紐付けユーザー管理
- **会話ログ閲覧**：管理画面からユーザーごとの会話履歴を確認
- **セッション管理**：Rich Menu の「新しい質問をはじめる」で会話履歴をリセット
- **取り消し / 復活**：管理者がユーザー登録を取り消し、必要なら新コードで復活

## アーキテクチャ

```
おばあちゃん ── LINE ──▶ LINE Platform ──▶ Go/Echo (Fly.io 東京)
                                              │
                                              ├─▶ Anthropic Claude API（プロンプトキャッシュ有効）
                                              └─▶ Supabase Postgres（pgbouncer pooler 経由）
                                                     ▲
お孫さん ──── Web ──▶ Vercel (Next.js) ─── Supabase Auth + RLS + RPC
```

## 技術スタック

| レイヤ          | 技術                                                                              |
| --------------- | --------------------------------------------------------------------------------- |
| Backend         | Go 1.26 + Echo + sqlx + pgx                                                       |
| Frontend        | Next.js 16 + React 19 + TypeScript + Tailwind + shadcn/ui + sonner + lucide-react |
| Auth            | Supabase Auth（Google OAuth）                                                     |
| Database        | Supabase Postgres + RLS + RPC                                                     |
| LLM             | Anthropic Claude（プロンプトキャッシュ）                                          |
| Hosting         | Backend: Fly.io（東京）/ Frontend: Vercel / DB: Supabase                          |
| Tunneling (dev) | cloudflared                                                                       |
| Tool versions   | mise                                                                              |
| CI/CD           | GitHub Actions（lint + test + gitleaks + Fly.io 自動デプロイ）                    |

## リポジトリ構造

```
mago-ai/
├── backend/                # Go + Echo（Clean Architecture）
│   ├── cmd/server/         # エントリーポイント
│   ├── internal/
│   │   ├── domain/         # 純粋型
│   │   ├── usecase/        # 業務ロジック + Port（interface）
│   │   ├── infrastructure/ # Postgres / LINE SDK / Claude SDK
│   │   ├── interface/http/ # Echo ハンドラー
│   │   └── config/         # env 読み込み
│   ├── Dockerfile
│   └── fly.toml
├── frontend/               # Next.js 16（App Router）
│   ├── app/
│   │   ├── (auth)/login/                    # Google ログイン
│   │   ├── (protected)/
│   │   │   ├── conversations/               # 会話ログ閲覧
│   │   │   ├── line-users/                  # 登録ユーザー管理
│   │   │   ├── register/                    # 登録コード発行
│   │   │   └── layout.tsx
│   │   └── auth/callback/                   # OAuth コールバック
│   ├── lib/supabase/                        # browser / server / middleware
│   └── proxy.ts                             # Next.js 16 middleware
├── supabase/
│   ├── migrations/         # マイグレーション SQL
│   └── snippets/           # 開発用 seed 等
├── .github/workflows/      # ci.yml + cd.yml
├── mise.toml               # ツール固定（go, node, supabase, cloudflared, golangci-lint）
└── CLAUDE.md               # 設計詳細・実装方針
```

## ローカル開発セットアップ

### 必須ツール

[`mise`](https://mise.jdx.dev/) でツールを統一管理。

```bash
brew install mise
echo 'eval "$(mise activate zsh)"' >> ~/.zshrc
source ~/.zshrc
```

### 初回セットアップ

```bash
# リポジトリ取得
git clone https://github.com/akito-0520/mago-ai.git
cd mago-ai

# ツールのインストール（go / node / supabase / cloudflared / golangci-lint）
mise trust
mise install

# 環境変数の準備
cp .mise.local.toml.example .mise.local.toml
cp backend/.mise.local.toml.example backend/.mise.local.toml
cp frontend/.mise.local.toml.example frontend/.mise.local.toml
# それぞれ実際の値を埋める（後述）

# Supabase 起動（Docker で Postgres + Auth + Studio）
supabase start

# フロントの依存をインストール
cd frontend && npm install && cd ..
```

### 環境変数

機密値は `.mise.local.toml`（git 管理外）に記入。サンプルは `.mise.local.toml.example` に commit 済み。

**ルート（`.mise.local.toml`）**：Supabase の Google OAuth 用

```toml
[env]
SUPABASE_AUTH_EXTERNAL_GOOGLE_CLIENT_ID = "<Google Cloud Console で取得>"
SUPABASE_AUTH_EXTERNAL_GOOGLE_SECRET    = "<同上>"
```

**`backend/.mise.local.toml`**：

```toml
[env]
PORT                       = "8080"
DATABASE_URL               = "postgres://postgres:postgres@127.0.0.1:54322/postgres"
SUPABASE_URL               = "http://127.0.0.1:54321"
SUPABASE_SERVICE_ROLE_KEY  = "<supabase status で確認>"
ANTHROPIC_API_KEY          = "<Anthropic コンソール>"
CLAUDE_MODEL               = "claude-sonnet-4-6"
LINE_CHANNEL_SECRET        = "<dev チャネル>"
LINE_CHANNEL_ACCESS_TOKEN  = "<dev チャネル>"
CONVERSATION_WINDOW_HOURS  = "24"
```

**`frontend/.mise.local.toml`**：

```toml
[env]
NEXT_PUBLIC_SUPABASE_URL      = "http://127.0.0.1:54321"
NEXT_PUBLIC_SUPABASE_ANON_KEY = "<supabase status で確認>"
NEXT_PUBLIC_LINE_FRIEND_URL   = "<dev 用 LINE 公式アカウントの URL（任意）>"
```

設定後、`mise trust` を実行して env を有効化。

### 開発時の起動

4 つのターミナルで並行起動：

```bash
# ターミナル 1：Supabase（Docker、起動済みなら省略可）
supabase start

# ターミナル 2：cloudflared（LINE dev チャネルからの Webhook 受信用）
cloudflared tunnel run mago-dev

# ターミナル 3：Go バックエンド
cd backend && go run ./cmd/server

# ターミナル 4：Next.js フロント
cd frontend && npm run dev
```

ブラウザで `http://localhost:3000` にアクセス → Google ログイン → 管理画面利用。
LINE Bot は `https://mago-dev.akiton.net/webhook` に到達（cloudflared が `localhost:8080` に転送）。

### 開発用 seed

ローカル DB にテストデータを入れたい場合：

1. `supabase db reset` でスキーマ再構築
2. フロント（`npm run dev`）起動 → Google ログインで `auth.users` に行を作る
3. Supabase Studio（`http://127.0.0.1:54323`）の SQL Editor で `supabase/snippets/dev_seed.sql` を貼り付けて実行

冪等なので何度でも再実行できる。

## マイグレーション運用

```bash
# 新規マイグレーション作成
supabase migration new <name>
# → supabase/migrations/<ts>_<name>.sql が生成される

# ローカル DB に適用（全マイグレーション再実行）
supabase db reset

# 本番（リモート Supabase）に未適用分を反映
supabase db push
```

Next.js の DB 型生成：

```bash
supabase gen types typescript --local > frontend/lib/database.types.ts
```

## テスト

### バックエンド

```bash
cd backend
go test ./...                     # 全テスト実行
go test -race ./...               # データレース検出付き
golangci-lint run ./...           # lint
```

ユースケース層は手書き fake を注入する単体テストを採用。Claude / LINE / DB を実際には叩かない。

### フロントエンド

```bash
cd frontend
npm run lint
npm run typecheck
npm run format:check
npm run build
```

E2E / インテグレーションテストは導入していない（MVP 方針）。

## 本番環境

| レイヤ   | 場所                                                               | デプロイ                                   |
| -------- | ------------------------------------------------------------------ | ------------------------------------------ |
| Backend  | Fly.io（`api.mago-ai.akiton.net`、東京、`min_machines_running=1`） | `main` ブランチへの push で自動（cd.yml）  |
| Frontend | Vercel                                                             | GitHub 連携で自動                          |
| Database | Supabase（マネージド）                                             | ローカルから `supabase db push` を手動実行 |

シークレット管理：

- Backend：`fly secrets set KEY=value`
- Frontend：Vercel ダッシュボードの Environment Variables
- Supabase：ダッシュボードで Google OAuth + RLS 設定

## CI/CD

| ワークフロー | トリガー                                | 内容                                                                                        |
| ------------ | --------------------------------------- | ------------------------------------------------------------------------------------------- |
| `ci.yml`     | PR                                      | backend lint+test、frontend lint+typecheck+build、gitleaks（PR commits の secret スキャン） |
| `cd.yml`     | `main` への push（`backend/**` 変更時） | `flyctl deploy --remote-only` で Fly.io にバックエンドを自動デプロイ                        |

Vercel は GitHub 連携で自動的に PR プレビュー / 本番デプロイ。

## 設計方針

詳細は [CLAUDE.md](./CLAUDE.md) 参照。要点：

- **学習対象**：Echo フレームワーク + LINE 関連ロジック（webhook 署名検証 / Reply API / Rich Menu / goroutine + graceful shutdown）
- **その他レイヤ**（Supabase / Clean Architecture / Claude / Next.js / Fly.io / Vercel）はプロダクション水準で実装
- **過不足ない実装**：抽象化・将来拡張の先食いはしない
- **TDD**：バックエンドはテストファーストで実装

## おばあちゃん側の UX 制約

- 1 回の LINE 返信は **150 文字目安、上限 250 文字**（スクロールせず読める量）
- 返答トーンは「ですます調」、難しい言葉を避ける、手順は 1〜2 ステップずつ
- iPhone（iOS）前提
- iOS バージョン差を必要に応じて添える
- LINE トーク表示前提：Markdown 装飾やコードブロックは使わない

## ライセンス

このプロジェクトは個人開発のため、ライセンスは未設定です。
