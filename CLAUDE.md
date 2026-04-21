# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## プロジェクト概要

**mago.ai**（孫 + AI）は、シニア世代（おばあちゃん）がスマホ操作の疑問を LINE で気軽に聞ける Bot。孫（管理者）が自分の祖父母ごとに Bot を設定し、**複数の LINE ユーザー（複数の祖父母）を 1 つの管理者アカウントで見守る**構成。

- **エンドユーザー UI = LINE Bot**（おばあちゃん側、シニア向け UX 配慮が必要な唯一の面）
- **Web = 孫専用の管理画面**（読み取り + 登録コード発行、シニア向け制約は不要）

## 学習スコープと設計水準

- **学習対象は Echo フレームワークの書き方のみ**
- それ以外のレイヤ（Supabase、Clean Architecture、Claude API、Next.js、Fly.io / Vercel 等）は**プロダクション水準で実装**し、学習のためのレベル低下は行わない
- 不要な抽象化・将来拡張の先食いは避け、要件に対して過不足ない実装に留める

## 現在のリポジトリ状態

**コードは未実装。** アーキテクチャ決定のみ固まった状態。次のコミットで以下のスケルトンを作成していく：

```
mago-ai/
├── frontend/          # Next.js（管理画面）
├── backend/           # Go + Echo（LINE webhook）
├── supabase/          # マイグレーション + ローカル設定
├── mise.toml          # ツール固定 + 共通 env
└── .gitignore
```

## アーキテクチャ概要

```
おばあちゃん ── LINE ──▶ LINE Platform ──▶ Go/Echo (Fly.io 東京)
                                              │
                                              ├─▶ Anthropic Claude API
                                              └─▶ Supabase Postgres
                                                     ▲
孫 ──── Web ────▶ Vercel (Next.js) ─── Supabase Auth + RLS
```

- **LINE webhook** は Go/Echo が受信・署名検証 → 即 200 OK 返却 → **goroutine で Claude 呼び出し + reply API**（graceful shutdown で進行中 goroutine を待機）
- **会話コンテキスト**：直近 20 ターンを Supabase から取得し Claude に渡す。システムプロンプトは `cache_control: ephemeral` でプロンプトキャッシュ
- **Claude モデル**：`CLAUDE_MODEL` 環境変数、デフォルト `claude-sonnet-4-6`
- **ナレッジベース・パターンマッチ・画像対応は実装しない**（Claude 直通のテキスト Bot）

### バックエンドのディレクトリ設計（Clean Architecture）

```
backend/
├── cmd/server/main.go              # DI 組み立て / Echo 起動 / graceful shutdown
├── internal/
│   ├── domain/                     # 純粋：標準ライブラリのみ依存
│   ├── usecase/                    # RespondToIncomingMessage / RegisterLineUserByToken 等
│   │   └── port.go                 # Ports（ConversationRepository / ClaudeGateway / LineGateway）
│   ├── infrastructure/             # Ports の具象実装
│   │   ├── postgres/               # sqlx + pgx/v5 stdlib
│   │   ├── claude/                 # anthropic-sdk-go ラッパ
│   │   └── linebot/                # line-bot-sdk-go ラッパ
│   ├── interface/http/             # Echo handler / middleware / router
│   └── config/                     # env → Config struct
├── go.mod
├── Dockerfile
└── fly.toml
```

依存方向：`interface/http → usecase ← infrastructure`、`usecase → domain`。
`domain` には external import を入れない（pgx / echo / anthropic を漏らさない）。

### フロントエンドのディレクトリ設計

```
frontend/
├── app/
│   ├── (auth)/login/               # Google OAuth ボタン
│   ├── (protected)/
│   │   ├── conversations/          # ログ一覧 + 詳細
│   │   ├── line-users/             # 紐付け済み LINE ユーザー管理（display_name 編集）
│   │   └── register/               # 登録コード発行ページ
│   ├── auth/callback/              # Supabase OAuth コールバック
│   └── layout.tsx
├── components/ui/                   # shadcn/ui が生成した component
├── lib/supabase/                    # server / middleware / type 定義
└── middleware.ts                    # @supabase/ssr のセッション更新
```

スタック：**Tailwind + shadcn/ui + TanStack Table + @supabase/ssr**。RSC で `auth.uid()` を使った RLS 越しフェッチ。

## データモデル（MVP）

Multi-tenant：1 管理者 → 複数 LINE ユーザー（1 対多）。

```sql
-- 管理者は auth.users をそのまま使用（独自 admins テーブルは作らない）

create table line_users (
  id              uuid primary key default gen_random_uuid(),
  admin_id        uuid not null references auth.users(id) on delete cascade,
  line_user_id    text not null unique,          -- LINE の生 User ID（U... で始まる）
  display_name    text,                          -- 管理者がつけるニックネーム
  created_at      timestamptz not null default now()
);
create index on line_users (admin_id);

create table conversations (
  id              bigserial primary key,
  line_user_id    uuid not null references line_users(id) on delete cascade,
  role            text not null check (role in ('user','assistant')),
  content         text not null,
  latency_ms      integer,
  model           text,
  created_at      timestamptz not null default now()
);
create index on conversations (line_user_id, created_at desc);

create table register_tokens (
  token       text primary key,                  -- 短い英数字（フォーマット未確定）
  admin_id    uuid not null references auth.users(id) on delete cascade,
  expires_at  timestamptz not null,              -- 24 時間後想定
  used_at     timestamptz,
  created_at  timestamptz not null default now()
);
```

RLS（方針、最終 SQL は未確定）：

- `line_users` / `conversations` / `register_tokens` すべて RLS 有効
- 読み取り・編集は `admin_id = auth.uid()` 相当のポリシーで絞る
- 書き込みは backend が `service_role` キーで bypass

## 登録フロー

1. 孫が管理画面の「登録コード発行」ボタンを押す → 短い英数字トークンが表示（24 時間有効）
2. 孫がおばあちゃんに「これを LINE で送って」と伝える
3. おばあちゃんが Bot にトークンを送信
4. Bot は送信元 `line_user_id` が `line_users` に存在するか確認
   - 未登録 かつ メッセージが `register_tokens` に一致 → `line_users` に upsert、「登録しました」と返信
   - 未登録 かつ コード形式でない → 「まだ登録が済んでいません。お孫さんからもらったコードを送ってください」
   - 登録済み → 通常の会話フロー（Claude 応答 + ログ保存）

Bot 受信処理は **`RegisterLineUserByToken`** と **`RespondToIncomingMessage`** の 2 ユースケースに分離。

## 開発環境

ツールバージョンと env は `mise` で管理。

```toml
# mise.toml（ルート）
[tools]
go = "1.26"
node = "20"
"supabase" = "latest"
"cloudflared" = "latest"
```

機密は `backend/.mise.local.toml` / `frontend/.mise.local.toml` の `[env]`（`.gitignore` 対象）。`.mise.local.toml.example` を commit してキー名を共有。

### ローカル起動フロー

```bash
# 初回セットアップ
mise trust
mise install
supabase init       # すでに初期化済みなら不要
supabase start      # Docker で Postgres/Auth/Studio を起動

# 開発中（別ターミナル）
cloudflared tunnel run mago-dev       # LINE dev チャネル用の公開 URL
cd backend && go run ./cmd/server     # Go/Echo 起動
cd frontend && npm run dev            # Next.js 起動
```

LINE チャネルは **dev / prod の 2 つ**を用意。dev は cloudflared 経由のローカル URL、prod は Fly.io のパブリック URL を webhook に設定。

### マイグレーション運用

```bash
supabase migration new <name>         # supabase/migrations/<ts>_<name>.sql が生成
# SQL を書く
supabase db reset                     # ローカル DB を migration で作り直す
supabase db push                      # リモート（本番）に未適用分を流す
```

Next.js の DB 型は `supabase gen types typescript --local > frontend/lib/database.types.ts` で生成。

## 本番環境

| レイヤ | 場所 |
|---|---|
| Next.js 管理画面 | Vercel |
| Go + Echo バックエンド | Fly.io（東京リージョン `nrt`、`min_machines_running=1` で常時稼働） |
| Supabase | マネージド（`db push` で dev ローカルから同期） |
| LINE Prod チャネル | Fly.io の公開 URL を webhook に設定 |

シークレット注入：

- Backend：`fly secrets set KEY=value`
- Frontend：Vercel ダッシュボードの Environment Variables
- Supabase：ダッシュボードで Google OAuth 設定・RLS ポリシー確認

## CI/CD

GitHub Actions、3 ワークフロー：

- `ci.yml`：PR / push で Go の lint+test、Next.js の lint+typecheck+build、`gitleaks` を実行
- `deploy-backend.yml`：`main` の `backend/**` 変更で `flyctl deploy`
- `deploy-supabase.yml`：`main` の `supabase/migrations/**` 変更で `supabase db push`

Vercel は GitHub 連携の自動デプロイに任せる（PR プレビュー URL を自動生成）。

## テスト方針

単体テストのみ（integration / E2E は導入しない）。

- `domain/`：純粋関数テスト
- `usecase/`：手書き fake（`port.go` の interface を満たす struct）を注入し、Claude / LINE / DB を叩かずに振る舞いを検証
- `interface/http/handler/`：`net/http/httptest` + mock usecase
- `infrastructure/*`：単体テスト対象外（薄いラッパのため手動確認で許容）

標準ツール：Go 標準 `testing` + `testify/require`。table-driven style を基本。`mockery` 等のモック生成は使わない。

TDD 原則：**テストを先に書いて失敗させ、実装を通す** ── グローバル CLAUDE.md の方針に従う。

## 観測性

- **Backend**：Go 1.21+ 標準 `log/slog` の JSON ハンドラ、機密ヘッダはミドルウェアでマスキング
- **Healthcheck**：Echo に `/healthz` を実装、`fly.toml` の `[[http_service.checks]]` で監視
- **Frontend**：Vercel Analytics を有効化

エラー追跡（Sentry 等）・メトリクス・分散トレーシングは**導入しない**。Fly.io / Vercel / Supabase の組み込みログで十分。

## 実装上の制約（LINE Bot 側）

おばあちゃん側の UI 制約：

- **文字の読みやすさ**：LINE メッセージは 600 文字以内を目安、長くなるなら対話型で区切る
- **返答トーン**：難しい言葉を避ける／手順は 1 ステップずつ／最後に確認の一言
- **iPhone（iOS）前提**：Android の案内は書かない
- **iOS バージョン差**：画面が異なる可能性を返答に添える

システムプロンプト本文は**未確定**（次の会話で詰める）。

## 保留中の決定事項

以下は実装着手前 or 初期実装中に詰める：

- システムプロンプト本文（ペルソナの距離感、雑談許容範囲、長さガイド等）
- 登録トークンのフォーマット（長さ・文字種・有効期限）
- RLS ポリシーの最終 SQL 文言
- LINE webhook 署名検証ミドルウェアの実装詳細
- `main.go` の DI ワイヤリング
- sqlx の connection pool 設定（`SetMaxOpenConns` 等）

## 開発方針

- **TDD 原則**：期待入出力に基づきテストを先に書き、失敗を確認してから実装。テストは実装中に書き換えない
- **Echo 以外は過不足ない production 水準**：抽象化・将来拡張のための先行実装はしない
- **捨て実装を書かない**：LLM 導入までの中間実装（パターンマッチエンジン等）を作らず、最初から Claude 直通で構築
