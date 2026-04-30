-- =============================================================================
-- 管理者（孫）の LINE 通知連携機能を追加
--
-- 変更点:
--   - admin_line_links: 孫の LINE User ID と admin の紐付け
--   - admin_link_tokens: 連携用の 6 桁ワンタイムトークン
--   - RLS: 自分の配下のみ SELECT 可
--   - RPC: generate_admin_link_token / unlink_admin_line
-- =============================================================================


-- -----------------------------------------------------------------------------
-- 1. Tables
-- -----------------------------------------------------------------------------

create table admin_line_links (
    id           uuid        primary key default gen_random_uuid(),
    admin_id     uuid        not null references auth.users(id) on delete cascade,
    line_user_id text        not null unique,    -- 孫の LINE User ID（"U..." で始まる）
    display_name text,                            -- LINE プロフィールから取得
    created_at   timestamptz not null default now()
);
create index admin_line_links_admin_id_idx on admin_line_links (admin_id);

create table admin_link_tokens (
    token      text        primary key check (token ~ '^[0-9]{6}$'),
    admin_id   uuid        not null references auth.users(id) on delete cascade,
    expires_at timestamptz not null,
    used_at    timestamptz,
    used_by    uuid        references admin_line_links(id) on delete set null,
    created_at timestamptz not null default now()
);
create index admin_link_tokens_admin_id_idx on admin_link_tokens (admin_id);


-- -----------------------------------------------------------------------------
-- 2. Row Level Security
-- -----------------------------------------------------------------------------

alter table admin_line_links  enable row level security;
alter table admin_link_tokens enable row level security;

-- admin_line_links: 自分の配下のみ SELECT 可。書き込みは RPC 経由のみ
create policy admin_line_links_select_own
    on admin_line_links for select
    using (admin_id = (select auth.uid()));

-- admin_link_tokens: 自分が発行したもののみ SELECT 可。INSERT は RPC 経由のみ
create policy admin_link_tokens_select_own
    on admin_link_tokens for select
    using (admin_id = (select auth.uid()));


-- -----------------------------------------------------------------------------
-- 3. RPC: generate_admin_link_token
--   - 認証された admin (auth.uid()) のみ実行可能
--   - 1 admin あたり「未使用 + 有効期限内」が最大 3 本まで
--   - 6 桁の半角数字、1 時間有効
-- -----------------------------------------------------------------------------

create or replace function public.generate_admin_link_token()
returns text
language plpgsql
security definer
set search_path = public
as $$
declare
    v_admin_id     uuid := auth.uid();
    v_unused_count int;
    v_token        text;
    v_attempt      int := 0;
begin
    if v_admin_id is null then
        raise exception 'unauthorized: must be logged in';
    end if;

    select count(*)
        into v_unused_count
        from admin_link_tokens
        where admin_id   = v_admin_id
          and used_at    is null
          and expires_at > now();

    if v_unused_count >= 3 then
        raise exception 'too many unused tokens (max 3)';
    end if;

    loop
        v_token := lpad(floor(random() * 1000000)::int::text, 6, '0');
        exit when not exists (
            select 1 from admin_link_tokens where token = v_token
        );
        v_attempt := v_attempt + 1;
        if v_attempt > 10 then
            raise exception 'token generation collision: retry exhausted';
        end if;
    end loop;

    insert into admin_link_tokens (token, admin_id, expires_at)
    values (v_token, v_admin_id, now() + interval '1 hour');

    return v_token;
end;
$$;

revoke all on function public.generate_admin_link_token() from public;
grant execute on function public.generate_admin_link_token() to authenticated;


-- -----------------------------------------------------------------------------
-- 4. RPC: unlink_admin_line
--   - 自分の配下の連携を削除
-- -----------------------------------------------------------------------------

create or replace function public.unlink_admin_line(p_link_id uuid)
returns void
language plpgsql
security definer
set search_path = public
as $$
declare
    v_admin_id uuid := auth.uid();
begin
    if v_admin_id is null then
        raise exception 'unauthorized: must be logged in';
    end if;

    delete from admin_line_links
    where id       = p_link_id
      and admin_id = v_admin_id;

    if not found then
        raise exception 'admin_line_link not found or not owned';
    end if;
end;
$$;

revoke all on function public.unlink_admin_line(uuid) from public;
grant execute on function public.unlink_admin_line(uuid) to authenticated;
