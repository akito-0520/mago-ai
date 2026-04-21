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

Phase 1（外部サービス設定）と CI 初期設定、バックエンドの config ローダーまで完了。LINE webhook ハンドラやユースケースは未実装。

```
mago-ai/
├── frontend/          # Next.js + Tailwind + shadcn/ui + Prettier（初期 scaffold 済）
├── backend/           # Go + Echo（go.mod + config パッケージのみ実装済）
├── supabase/          # init_schema migration + dev_seed snippet
├── .github/workflows/ # ci.yml（PR でフル lint + test + gitleaks）
├── mise.toml          # ツール固定（go / node / supabase / cloudflared / golangci-lint）
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

スタック：**Tailwind + shadcn/ui + TanStack Table + @supabase/ssr**、整形は Prettier、lint は ESLint。RSC で `auth.uid()` を使った RLS 越しフェッチ。

## データモデル

Multi-tenant：1 管理者 → 複数 LINE ユーザー（1 対多、`line_user_id` は全体ユニーク）。

```sql
create table line_users (
  id                uuid        primary key default gen_random_uuid(),
  admin_id          uuid        not null references auth.users(id) on delete cascade,
  line_user_id      text        not null unique,   -- LINE の生 User ID（U... で始まる）
  display_name      text,                          -- 管理者がつけるニックネーム
  session_reset_at  timestamptz,                   -- おばあちゃんが Rich Menu で押したリセット時刻
  created_at        timestamptz not null default now()
);
create index on line_users (admin_id);

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
  expires_at  timestamptz not null,                -- created_at + 24 時間
  used_at     timestamptz,                         -- 使用時刻。使い捨て（NULL = 未使用）
  used_by     uuid        references line_users(id) on delete set null,
  created_at  timestamptz not null default now()
);
create index on register_tokens (admin_id);
```

実 SQL は `supabase/migrations/<ts>_init_schema.sql` に格納。

### RLS

全テーブル RLS 有効。backend は Supabase の service_role キーで bypass するため RLS 対象外。admin（`auth.users`）の JWT 経由での権限：

```sql
-- line_users: 自分の配下のみ SELECT + display_name などを UPDATE 可
create policy line_users_select_own on line_users for select
  using (admin_id = auth.uid());
create policy line_users_update_own on line_users for update
  using (admin_id = auth.uid())
  with check (admin_id = auth.uid());

-- conversations: 自分の配下の line_user の会話のみ SELECT
create policy conversations_select_own on conversations for select
  using (
    exists (
      select 1 from line_users lu
      where lu.id = conversations.line_user_id
        and lu.admin_id = auth.uid()
    )
  );

-- register_tokens: 自分が発行したもののみ SELECT + INSERT
create policy register_tokens_select_own on register_tokens for select
  using (admin_id = auth.uid());
create policy register_tokens_insert_own on register_tokens for insert
  with check (admin_id = auth.uid());
```

INSERT/UPDATE/DELETE 無し = 明示的に不許可（例：`line_users` の DELETE、`conversations` の任意書き込み、`register_tokens` の UPDATE）。

## 登録フロー

1. 孫が管理画面の「登録コード発行」ボタンを押す → **数字 6 桁**のトークンが表示（24 時間有効、1 管理者 × 未使用最大 3 本）
2. 孫がおばあちゃんに「これを LINE で送って」と伝える
3. おばあちゃんが Bot にトークンを送信
4. Bot は送信元 `line_user_id` が `line_users` に存在するか確認
   - **未登録** かつ メッセージが 6 桁数字 → `register_tokens` を検索
     - 一致 → `line_users` に INSERT、`register_tokens.used_at` と `used_by` を埋めて「登録しました」と返信
     - 不一致 → 「コードが違うみたいです。もう一度確認して送ってください」
   - **未登録** かつ 6 桁数字でない → 「まだ登録が済んでいません。お孫さんからもらったコードを送ってください」
   - **登録済み** かつ `#新しい質問` → `session_reset_at` を更新し「新しい質問をどうぞ」と返信
   - **登録済み** かつ 通常メッセージ → Claude 応答 + 会話ログ保存

トークン入力はレート制限で保護（**1 LINE user あたり 1 時間 5 回 / 1 日 30 回**、超過時は「少し時間を置いてからもう一度試してください」）。レート制限は backend のインメモリカウンタで実装（MVP は単一インスタンス運用）。

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

GitHub Actions。現時点では `ci.yml`（PR トリガー）のみ：

- **backend**：`golangci-lint run` + `go test ./...`
- **frontend**：`npm run lint` + `npm run typecheck` + `npm run format:check` + `npm run build`
- **gitleaks**：リポジトリ全体のシークレットスキャン

デプロイ用の `deploy-backend.yml` / `deploy-supabase.yml` は Fly.io / Supabase のプロジェクト作成後に追加予定。Vercel は GitHub 連携の自動デプロイに任せる（PR プレビュー URL を自動生成）。

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
- **Rich Menu**：1 ボタン（「新しい質問をはじめる」）を LINE チャンネルに登録。押下時は `#新しい質問` が送信される。`#` プレフィックスで通常メッセージと区別

システムプロンプト本文は**未確定**（次の会話で詰める）。

## 保留中の決定事項

以下は実装着手前 or 初期実装中に詰める：

- システムプロンプト本文（ペルソナの距離感、雑談許容範囲、長さガイド等）
- Rich Menu の画像デザインと LINE チャンネルへの登録手順
- LINE webhook 署名検証ミドルウェアの実装詳細（Echo middleware として差し込む）
- `main.go` の DI ワイヤリング
- sqlx の connection pool 設定（`SetMaxOpenConns` 等）

## 開発方針

- **TDD 原則**：期待入出力に基づきテストを先に書き、失敗を確認してから実装。テストは実装中に書き換えない
- **Echo + LINE 関連以外は過不足ない production 水準**：抽象化・将来拡張のための先行実装はしない
- **捨て実装を書かない**：LLM 導入までの中間実装（パターンマッチエンジン等）を作らず、最初から Claude 直通で構築
