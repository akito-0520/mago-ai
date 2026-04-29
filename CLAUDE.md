# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## プロジェクト概要

**mago.ai**（孫 + AI）は、シニア世代（おばあちゃん）がスマホ操作の疑問を LINE で気軽に聞ける Bot。孫（管理者）が自分の祖父母ごとに Bot を設定し、**複数の LINE ユーザー（複数の祖父母）を 1 つの管理者アカウントで見守る**構成。

- **エンドユーザー UI = LINE Bot**（おばあちゃん側、シニア向け UX 配慮が必要な唯一の面）
- **Web = 孫専用の管理画面**（読み取り + 登録コード発行、シニア向け制約は不要）

## 学習スコープと設計水準

- **学習対象**：
  - Echo フレームワークの書き方
  - LINE 関連ロジック全般（Webhook 署名検証 / Webhook イベント処理 / Reply API 呼び出し / Rich Menu 管理 / goroutine + graceful shutdown）
    - 公式 SDK (`line-bot-sdk-go/v8`) を使うが、SDK 内部の挙動（署名検証アルゴリズム / Messaging API のリクエスト形状等）は都度把握する
- それ以外のレイヤ（Supabase、Clean Architecture、Claude API、Next.js、Fly.io / Vercel 等）は**プロダクション水準で実装**し、学習のためのレベル低下は行わない
- 不要な抽象化・将来拡張の先食いは避け、要件に対して過不足ない実装に留める

## 現在のリポジトリ状態

Phase 1（外部サービス設定）、Phase 2（登録フロー）、Phase 3（Claude API 統合）、フロントの管理画面（ログイン・登録コード発行・登録ユーザー管理）まで完了。Fly.io + Vercel + Supabase に本番デプロイ済み。Rich Menu、会話ログ閲覧画面は未実装。

```
mago-ai/
├── frontend/          # Next.js + Tailwind + shadcn/ui + sonner + lucide-react
├── backend/           # Go + Echo + sqlx + pgx（Clean Architecture、登録フロー実装済み）
├── supabase/          # init_schema + add_generate_register_token + add_revoke_and_update_display_name
├── .github/workflows/ # ci.yml（PR で lint + test + gitleaks）/ cd.yml（main で Fly.io 自動デプロイ）
├── mise.toml          # ツール固定（go / node / supabase / cloudflared / golangci-lint）
└── .gitignore
```

## アーキテクチャ概要

```
おばあちゃん ── LINE ──▶ LINE Platform ──▶ Go/Echo (Fly.io 東京、api.mago-ai.akiton.net)
                                              │
                                              ├─▶ Anthropic Claude API（プロンプトキャッシュ有効）
                                              └─▶ Supabase Postgres（pgbouncer pooler 経由）
                                                     ▲
孫 ──── Web ────▶ Vercel (Next.js) ─── Supabase Auth + RLS + RPC
```

- **LINE webhook** は Go/Echo が受信・署名検証 → 即 200 OK 返却 → **goroutine で Claude 呼び出し + Reply API**（graceful shutdown で進行中 goroutine を待機）
- **会話コンテキスト**：`line_users.session_reset_at` と `CONVERSATION_WINDOW_HOURS`（デフォルト 24 時間）の両方を起点にして、**新しい方**以降の直近 20 ターンを Supabase から取得し Claude に渡す。システムプロンプトは `cache_control: ephemeral` でプロンプトキャッシュ
- **Rich Menu**：おばあちゃんが手動で会話履歴をリセットするボタンを LINE 画面下部に常時表示。押すと `#新しい質問` という固定テキストが Bot に送信され、backend は `line_users.session_reset_at = now()` を更新
- **Claude モデル**：`CLAUDE_MODEL` 環境変数、デフォルト `claude-sonnet-4-6`
- **Claude API 失敗時**：`conversations` の user 行は保存済みのまま残し、assistant 行は作らない。おばあちゃんには「うまく答えられませんでした。少し待ってから、もう一度送ってください」と返信
- **ナレッジベース・パターンマッチ・画像対応は実装しない**（Claude 直通のテキスト Bot）
- **Follow/Unfollow イベント**は `slog` にログ出力のみで無視（`line_users` への反映なし）

### バックエンドのディレクトリ設計（Clean Architecture）

```
backend/
├── cmd/server/main.go              # DI 組み立て / Echo 起動 / graceful shutdown
├── internal/
│   ├── domain/                     # 純粋：標準ライブラリのみ依存（LineUser / RegisterToken）
│   ├── usecase/                    # RespondToIncomingMessage / RegisterLineUserByToken
│   │   └── port.go                 # Ports（LineGateway / LineUserRepository / RegisterTokenRepository / 将来 ClaudeGateway / ConversationRepository）
│   ├── infrastructure/             # Ports の具象実装
│   │   ├── postgres/               # sqlx + pgx/v5 stdlib（toSnakeCase MapperFunc で snake_case 自動変換）
│   │   ├── claude/                 # anthropic-sdk-go ラッパ（未実装）
│   │   └── linebot/                # line-bot-sdk-go ラッパ（Reply / GetProfile）
│   ├── interface/http/handler/     # Echo handler（webhook + healthz）
│   │                               # MessageResponder interface は handler パッケージ内で定義（consumer-defined）
│   └── config/                     # env → Config struct
├── go.mod
├── Dockerfile
└── fly.toml
```

依存方向：`interface/http → usecase ← infrastructure`、`usecase → domain`。
`domain` には external import を入れない（pgx / echo / anthropic を漏らさない）。
`domain` の UUID は `string` 型で扱う（DB が `gen_random_uuid()` で生成、Go 側で生成しない方針）。

### フロントエンドのディレクトリ設計

```
frontend/
├── app/
│   ├── (auth)/login/               # Google OAuth ボタン
│   ├── (protected)/
│   │   ├── conversations/          # ログ一覧 + 詳細（未実装）
│   │   ├── line-users/             # 登録ユーザー管理（display_name 編集 + 取り消し）
│   │   │   ├── page.tsx            # Server Component（初期データを RLS 経由で fetch）
│   │   │   └── client.tsx          # Client Component（編集 / 取り消し RPC + 楽観的更新）
│   │   ├── register/               # 登録コード発行ページ（generate_register_token RPC）
│   │   └── layout.tsx              # 共通ヘッダー + 認証チェック（getUser → /login redirect）
│   ├── auth/callback/route.ts      # Supabase OAuth コールバック（exchangeCodeForSession）
│   ├── layout.tsx                   # ルート layout
│   └── page.tsx
├── components/ui/                   # shadcn/ui が生成した component（Button / Input / AlertDialog）
├── lib/supabase/                    # browser / server / middleware の 3 種クライアント
└── proxy.ts                         # Next.js 16 の middleware（旧 middleware.ts）
```

スタック：**Tailwind + shadcn/ui + sonner + lucide-react + @supabase/ssr**、整形は Prettier、lint は ESLint。RSC で `auth.uid()` を使った RLS 越しフェッチ。

**Next.js 16 の注意点**：

- middleware は `proxy.ts` という名前で root に配置、関数名も `proxy()`（旧 `middleware`）
- Server Component で初期データ fetch、Client Component で操作、楽観的更新で refetch を避ける
- `useEffect` 内で同期的 setState を呼ぶとエラー → Server Component 経由で初期データを渡す形にする
- `'use client'` のコンポーネントでも build 時に評価されるので、`createClient()` は **モジュール / コンポーネントトップではなく、ハンドラ内で呼ぶ**（Vercel build で env 不足を回避）

## データモデル

Multi-tenant：1 管理者 → 複数 LINE ユーザー（1 対多、`line_user_id` は全体ユニーク）。

```sql
create table line_users (
  id                uuid        primary key default gen_random_uuid(),
  admin_id          uuid        not null references auth.users(id) on delete cascade,
  line_user_id      text        not null unique,   -- LINE の生 User ID（U... で始まる）
  display_name      text,                          -- 登録時に LINE プロフィールから取得、管理者が編集可
  session_reset_at  timestamptz,                   -- おばあちゃんが Rich Menu で押したリセット時刻
  revoked_at        timestamptz,                   -- 取り消し時刻（NULL = 現役、soft delete）
  created_at        timestamptz not null default now()
);
create index on line_users (admin_id);
create index line_users_active_idx
  on line_users (line_user_id) where revoked_at is null;  -- 現役ユーザー検索の高速化（部分インデックス）

create table conversations (
  id                            bigserial   primary key,
  line_user_id                  uuid        not null references line_users(id) on delete cascade,
  role                          text        not null check (role in ('user', 'assistant')),
  content                       text        not null,
  latency_ms                    integer,    -- assistant 行のみ
  model                         text,       -- assistant 行のみ
  input_tokens                  integer,    -- assistant 行のみ
  output_tokens                 integer,    -- assistant 行のみ
  cache_read_input_tokens       integer,    -- assistant 行のみ
  cache_creation_input_tokens   integer,    -- assistant 行のみ
  created_at                    timestamptz not null default now()
);
create index on conversations (line_user_id, created_at desc);

create table register_tokens (
  token       text        primary key check (token ~ '^[0-9]{6}$'),  -- 数字 6 桁
  admin_id    uuid        not null references auth.users(id) on delete cascade,
  expires_at  timestamptz not null,                -- created_at + 1 時間
  used_at     timestamptz,                         -- 使用時刻。使い捨て（NULL = 未使用）
  used_by     uuid        references line_users(id) on delete set null,
  created_at  timestamptz not null default now()
);
create index on register_tokens (admin_id);
```

実 SQL は `supabase/migrations/` 配下：

- `<ts>_init_schema.sql`：3 テーブル + RLS の初期スキーマ
- `<ts>_add_generate_register_token.sql`：6 桁トークン発行 RPC
- `<ts>_add_revoke_and_update_display_name.sql`：取り消し + 表示名編集 RPC + RLS 最適化

### RLS

全テーブル RLS 有効。backend は Supabase の service_role キーで bypass するため RLS 対象外。admin（`auth.users`）の JWT 経由での権限：

```sql
-- line_users: 自分の配下のみ SELECT 可。書き込みは RPC 経由のみ
create policy line_users_select_own on line_users for select
  using (admin_id = (select auth.uid()));

-- conversations: 自分の配下の line_user の会話のみ SELECT
create policy conversations_select_own on conversations for select
  using (
    exists (
      select 1 from line_users lu
      where lu.id = conversations.line_user_id
        and lu.admin_id = (select auth.uid())
    )
  );

-- register_tokens: 自分が発行したもののみ SELECT。INSERT は RPC 経由のみ
create policy register_tokens_select_own on register_tokens for select
  using (admin_id = (select auth.uid()));
```

`auth.uid()` は `(select auth.uid())` でラップ → 行ごとに再評価されず InitPlan で 1 回評価（パフォーマンス最適化）。

直接の INSERT/UPDATE/DELETE は **明示的に不許可**。書き込みは以下の RPC（`security definer`）経由：

- `generate_register_token() returns text`：6 桁トークン発行（1 admin あたり未使用 3 本上限、1 時間有効）
- `revoke_line_user(p_line_user_id uuid) returns void`：自分の配下のユーザーを取り消し
- `update_line_user_display_name(p_line_user_id uuid, p_display_name text) returns void`：表示名編集（空白文字は NULL に正規化）

各 RPC は `auth.uid()` 検証 + 自分の配下のみ操作という制約で安全性を確保。

## 登録フロー

1. 孫が管理画面の「登録コード発行」ボタンを押す → **数字 6 桁**のトークンが表示（**1 時間有効**、1 管理者 × 未使用最大 3 本）
2. 孫がおばあちゃんに「これを LINE で送って」と伝える
3. おばあちゃんが Bot にトークンを送信
4. Bot は送信元 `line_user_id` の状態に応じて分岐：

```
1. ExistsActiveByLineUserID?
   ├── Yes → Claude 応答（暫定で「準備中」固定文言）
   └── No  → Step 2

2. text が 6 桁数字?
   ├── Yes → tryRegister()
   │         ├── 成功 → 「登録しました」（新規 INSERT or 取り消し済みからの復活）
   │         ├── ErrTokenNotFound → 「コードが違う」
   │         └── ErrTokenExpired  → 「コードの期限切れ」
   └── No  → Step 3

3. ExistsRevokedByLineUserID?
   ├── Yes → 「登録が解除されています、お孫さんに新しいコードをもらってください」
   └── No  → 「まだ登録が済んでいません、コードを送ってください」
```

**実装のポイント**：

- **`RegisterLineUserByToken` usecase**：トークン検証 → LINE プロフィール取得（ベストエフォート） → `line_users` Upsert → `register_tokens.used_at/used_by` を埋める
- **`RespondToIncomingMessage` usecase**：上記の分岐を担当、handler は `MessageResponder` interface 経由でこれを呼ぶ
- **登録時の display_name 自動取得**：`LineGateway.GetProfile(lineUserID)` で LINE プロフィールから取得。失敗しても登録は成功（`display_name` は NULL）
- **取り消し → 復活**：`INSERT ... ON CONFLICT DO UPDATE WHERE revoked_at IS NOT NULL` で「新規 / 復活」を 1 SQL で吸収。現役行と衝突したら `RETURNING` が空 → `ErrLineUserExists`
- **取り消されたユーザーの応答**：6 桁トークンを送れば復活、それ以外は `msgRevoked` で明示的に解除を通知

レート制限は MVP では未実装。必要になったら backend のインメモリカウンタで保護予定（**1 LINE user あたり 1 時間 5 回 / 1 日 30 回**）。

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
"golangci-lint" = "latest"
```

機密は `backend/.mise.local.toml` / `frontend/.mise.local.toml` の `[env]`（`.gitignore` 対象）。`.mise.local.toml.example` を commit してキー名を共有。

主な env（backend）：

- `PORT`（default `8080`）
- `DATABASE_URL` / `SUPABASE_URL` / `SUPABASE_SERVICE_ROLE_KEY`（必須）
- `ANTHROPIC_API_KEY`（必須） / `CLAUDE_MODEL`（default `claude-sonnet-4-6`）
- `LINE_CHANNEL_SECRET` / `LINE_CHANNEL_ACCESS_TOKEN`（必須）
- `CONVERSATION_WINDOW_HOURS`（default `24`、正の整数）

Supabase CLI 側は `SUPABASE_AUTH_EXTERNAL_GOOGLE_CLIENT_ID` / `SUPABASE_AUTH_EXTERNAL_GOOGLE_SECRET` も必要（`supabase/config.toml` から参照）。

### ローカル起動フロー

```bash
# 初回セットアップ
mise trust
mise install
supabase start      # Docker で Postgres/Auth/Studio を起動

# 開発中（別ターミナル）
cloudflared tunnel run mago-dev       # LINE dev チャネル用の公開 URL
cd backend && go run ./cmd/server     # Go/Echo 起動
cd frontend && npm run dev            # Next.js 起動
```

LINE チャネルは **dev / prod の 2 つ**を用意。dev は cloudflared 経由の固定サブドメイン、prod は Fly.io のパブリック URL を webhook に設定。

### ローカル seed

`supabase/seed.sql` は置かない。本番が Google OAuth のみのため、ローカルでも Google ログインを経由する：

1. `supabase db reset` でスキーマを初期化
2. フロント（`npm run dev`）を起動し、Google で 1 回ログイン → `auth.users` に自分のレコードが作られる
3. Supabase Studio（`http://127.0.0.1:54323`）の SQL Editor で `supabase/snippets/dev_seed.sql` を貼り付けて実行 → テスト用 `line_users` / `conversations` / `register_tokens` が自分のアカウントに紐付いて作成される

この snippet は冪等なので何度でも再実行できる。

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

GitHub Actions：

- **`ci.yml`**（PR トリガー）：
  - **backend**：`golangci-lint run` + `go test ./...`
  - **frontend**：`npm run lint` + `npm run typecheck` + `npm run format:check` + `npm run build`
  - **gitleaks**：リポジトリ全体のシークレットスキャン（`permissions: pull-requests: write` 必要、誤検知は `// gitleaks:allow` で除外）
- **`cd.yml`**（main への push トリガー、`backend/**` 変更時）：
  - `flyctl deploy --remote-only` で Fly.io にバックエンドを自動デプロイ
- **Supabase マイグレーション**：CD なし、`supabase db push` をローカルから手動実行
- **Vercel**：GitHub 連携の自動デプロイ（PR プレビュー URL も自動生成）

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

- **文字の読みやすさ**：LINE メッセージは **150 文字くらいを目安、長くても 250 文字以内**。スマホ画面でスクロールせず読める量に抑える
- **返答トーン**：難しい言葉を避ける／手順は 1〜2 ステップずつ／最後に確認の一言
- **対象 OS**：iPhone（iOS）前提。Android の説明はしない
- **iOS バージョン差**：画面が異なる可能性を返答に添える
- **出力形式**：LINE のトーク表示前提なので、Markdown 装飾やコードブロックは使わず素のテキストで返す
- **Rich Menu**：1 ボタン（「新しい質問をはじめる」）を LINE チャンネルに登録。押下時は `#新しい質問` が送信される。`#` プレフィックスで通常メッセージと区別

システムプロンプトの本文は `backend/internal/usecase/system_prompt.go` に定義済み（プロンプトキャッシュ対象）。チューニングはコード上で直接行う。

## 保留中の決定事項

以下は次の実装フェーズで詰める：

- **Rich Menu の画像デザインと LINE チャンネルへの登録手順**
- **会話ログ閲覧画面**（`/conversations`、TanStack Table 等）
- **レート制限**（インメモリカウンタ、必要になったら）
- **エラーログ検証テスト**（slog 出力をキャプチャしてアサート、必要になったら）
- **システムプロンプトのチューニング**：実運用の応答を見ながら `usecase/system_prompt.go` を調整

完了済み：

- ~~`main.go` の DI ワイヤリング~~ → 完了
- ~~sqlx の connection pool 設定~~ → `postgres.New` で `SetMaxOpenConns(20)` / `SetMaxIdleConns(5)` 等を設定済み
- ~~LINE webhook 署名検証~~ → SDK の `webhook.ParseRequest` を使用、handler 内で完結
- ~~システムプロンプト本文~~ → `usecase/system_prompt.go` に初版を定義（150 文字目安、上限 250 文字）
- ~~Phase 3：Claude API 統合~~ → ClaudeGateway / ConversationRepository / プロンプトキャッシュ実装済み

## 開発方針

- **TDD 原則**：期待入出力に基づきテストを先に書き、失敗を確認してから実装。テストは実装中に書き換えない
- **Echo + LINE 関連以外は過不足ない production 水準**：抽象化・将来拡張のための先行実装はしない
- **捨て実装を書かない**：LLM 導入までの中間実装（パターンマッチエンジン等）を作らず、最初から Claude 直通で構築
