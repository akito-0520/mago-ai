-- =============================================================================
-- プラン制度の追加
--
-- 1. plans                       … プランのマスター（料金体系の定義）
-- 2. admin_plans                 … admin → plan の紐付け（行が無ければ free 扱い）
-- 3. get_my_plan_status()        … 自分のプラン状況を取得する RPC
-- 4. generate_register_token()   … 改修：プランの max_line_users チェックを追加
--
-- 設計メモ:
--   - プラン値は migration で管理（GUI からの編集は想定しない）
--   - 「行が無ければ free」ルールで admin_plans への INSERT は将来の Stripe webhook
--     経由のみとする（今は誰も INSERT しない）
--   - レート制限のカウント自体は backend インメモリで持つ（DB には保存しない）
-- =============================================================================

-- -----------------------------------------------------------------------------
-- plans: プランのマスター
-- -----------------------------------------------------------------------------
create table public.plans (
  code           text        primary key,
  display_name   text        not null,
  max_line_users int         not null check (max_line_users > 0),
  hourly_limit   int         not null check (hourly_limit > 0),
  daily_limit    int         not null check (daily_limit > 0),
  created_at     timestamptz not null default now(),
  updated_at     timestamptz not null default now()
);

-- 料金体系は公開情報。誰でも参照可（INSERT/UPDATE/DELETE は migration からのみ）
alter table public.plans enable row level security;
create policy plans_select_all on public.plans for select using (true);

-- 初期データ（無料プランのみ。basic/premium は将来の migration で追加予定）
insert into public.plans (code, display_name, max_line_users, hourly_limit, daily_limit) values
  ('free', '無料', 1, 10, 60);


-- -----------------------------------------------------------------------------
-- admin_plans: admin → plan の紐付け
-- -----------------------------------------------------------------------------
create table public.admin_plans (
  admin_id   uuid        primary key references auth.users(id) on delete cascade,
  plan_code  text        not null references public.plans(code) on update cascade,
  -- 将来の billing 統合で追加する列の予約地
  --   stripe_customer_id      text,
  --   stripe_subscription_id  text,
  --   current_period_end      timestamptz,
  --   canceled_at             timestamptz,
  updated_at timestamptz not null default now(),
  created_at timestamptz not null default now()
);

alter table public.admin_plans enable row level security;

-- 自分の行のみ参照可。INSERT/UPDATE/DELETE は明示的に許可しない（将来の webhook 経由のみ）
create policy admin_plans_select_own on public.admin_plans for select
  using (admin_id = (select auth.uid()));


-- -----------------------------------------------------------------------------
-- get_my_plan_status: 自分のプラン状況（プラン情報 + 現在の使用人数）を返す
-- -----------------------------------------------------------------------------
create or replace function public.get_my_plan_status()
returns table (
  plan_code        text,
  display_name     text,
  max_line_users   int,
  hourly_limit     int,
  daily_limit      int,
  used_line_users  int
)
language plpgsql
security definer
set search_path = public
as $$
declare
  v_admin_id  uuid := auth.uid();
  v_plan_code text;
begin
  if v_admin_id is null then
    raise exception 'unauthorized: must be logged in';
  end if;

  -- 行が無ければ free
  v_plan_code := coalesce(
    (select ap.plan_code from public.admin_plans ap where ap.admin_id = v_admin_id),
    'free'
  );

  return query
    select
      p.code,
      p.display_name,
      p.max_line_users,
      p.hourly_limit,
      p.daily_limit,
      (select count(*)::int
         from public.line_users lu
        where lu.admin_id   = v_admin_id
          and lu.revoked_at is null) as used_line_users
    from public.plans p
    where p.code = v_plan_code;
end;
$$;

revoke all on function public.get_my_plan_status() from public;
grant execute on function public.get_my_plan_status() to authenticated;


-- -----------------------------------------------------------------------------
-- generate_register_token: 改修版
--
-- 旧版との差分:
--   - プラン解決ステップを追加（admin_plans → plans、行が無ければ free）
--   - 現役 line_users 数 < plan.max_line_users チェックを追加
--     （未使用本数チェックよりも前に配置：枠超過なら即座に弾く）
-- -----------------------------------------------------------------------------
create or replace function public.generate_register_token()
returns text
language plpgsql
security definer
set search_path = public
as $$
declare
  v_admin_id        uuid := auth.uid();
  v_plan_code       text;
  v_max_line_users  int;
  v_active_count    int;
  v_unused_count    int;
  v_token           text;
  v_attempt         int := 0;
begin
  -- 認証チェック
  if v_admin_id is null then
    raise exception 'unauthorized: must be logged in';
  end if;

  -- プラン解決
  v_plan_code := coalesce(
    (select ap.plan_code from public.admin_plans ap where ap.admin_id = v_admin_id),
    'free'
  );

  select p.max_line_users
    into v_max_line_users
    from public.plans p
   where p.code = v_plan_code;

  -- 現役 line_users 数チェック（プラン上限）
  select count(*)
    into v_active_count
    from public.line_users
   where admin_id   = v_admin_id
     and revoked_at is null;

  if v_active_count >= v_max_line_users then
    raise exception 'plan max line users reached: plan=% max=% used=%',
      v_plan_code, v_max_line_users, v_active_count;
  end if;

  -- 未使用上限チェック（3 本）
  select count(*)
    into v_unused_count
    from public.register_tokens
   where admin_id   = v_admin_id
     and used_at    is null
     and expires_at > now();

  if v_unused_count >= 3 then
    raise exception 'too many unused tokens (max 3)';
  end if;

  -- 衝突しないトークンを生成
  loop
    v_token := lpad(floor(random() * 1000000)::int::text, 6, '0');
    exit when not exists (
      select 1 from public.register_tokens where token = v_token
    );
    v_attempt := v_attempt + 1;
    if v_attempt > 10 then
      raise exception 'token generation collision: retry exhausted';
    end if;
  end loop;

  insert into public.register_tokens (token, admin_id, expires_at)
  values (v_token, v_admin_id, now() + interval '1 hour');

  return v_token;
end;
$$;

revoke all on function public.generate_register_token() from public;
grant execute on function public.generate_register_token() to authenticated;
