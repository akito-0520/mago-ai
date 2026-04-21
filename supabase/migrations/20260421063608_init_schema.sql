-- =============================================================================
-- 初期スキーマ: mago.ai
--
-- テーブル:
--   line_users       : 管理者 (auth.users) に紐付く LINE ユーザー (おばあちゃん)
--   conversations    : LINE ユーザーと Bot の発言ログ
--   register_tokens  : 孫が発行する 24 時間有効の 6 桁登録トークン (使い捨て)
--
-- RLS:
--   admin の JWT (auth.uid()) で自分の配下のみ SELECT 可
--   display_name UPDATE / register_tokens INSERT も admin 本人のみ
--   それ以外の書き込みは backend の service_role キーで bypass する前提
-- =============================================================================


-- -----------------------------------------------------------------------------
-- 1. Tables
-- -----------------------------------------------------------------------------

create table line_users (
  id                uuid        primary key default gen_random_uuid(),
  admin_id          uuid        not null references auth.users(id) on delete cascade,
  line_user_id      text        not null unique,
  display_name      text,
  session_reset_at  timestamptz,
  created_at        timestamptz not null default now()
);

create table conversations (
  id                            bigserial   primary key,
  line_user_id                  uuid        not null references line_users(id) on delete cascade,
  role                          text        not null check (role in ('user', 'assistant')),
  content                       text        not null,
  latency_ms                    integer,
  model                         text,
  input_tokens                  integer,
  output_tokens                 integer,
  cache_read_input_tokens       integer,
  cache_creation_input_tokens   integer,
  created_at                    timestamptz not null default now()
);

create table register_tokens (
  token       text        primary key check (token ~ '^[0-9]{6}$'),
  admin_id    uuid        not null references auth.users(id) on delete cascade,
  expires_at  timestamptz not null,
  used_at     timestamptz,
  used_by     uuid        references line_users(id) on delete set null,
  created_at  timestamptz not null default now()
);


-- -----------------------------------------------------------------------------
-- 2. Indexes
-- -----------------------------------------------------------------------------

create index line_users_admin_id_idx
  on line_users (admin_id);

create index conversations_line_user_id_created_at_idx
  on conversations (line_user_id, created_at desc);

create index register_tokens_admin_id_idx
  on register_tokens (admin_id);


-- -----------------------------------------------------------------------------
-- 3. Row Level Security
-- -----------------------------------------------------------------------------

alter table line_users      enable row level security;
alter table conversations   enable row level security;
alter table register_tokens enable row level security;

-- line_users: 自分の配下のみ SELECT + display_name などを UPDATE 可
create policy line_users_select_own
  on line_users for select
  using (admin_id = auth.uid());

create policy line_users_update_own
  on line_users for update
  using (admin_id = auth.uid())
  with check (admin_id = auth.uid());

-- conversations: 自分の配下の line_user の会話のみ SELECT
create policy conversations_select_own
  on conversations for select
  using (
    exists (
      select 1
      from line_users lu
      where lu.id = conversations.line_user_id
        and lu.admin_id = auth.uid()
    )
  );

-- register_tokens: 自分が発行したもののみ SELECT + INSERT
create policy register_tokens_select_own
  on register_tokens for select
  using (admin_id = auth.uid());

create policy register_tokens_insert_own
  on register_tokens for insert
  with check (admin_id = auth.uid());
