-- =============================================================================
-- 取り消し機能 + 表示名編集用の RPC 追加
--
-- 変更点:
--   - line_users.revoked_at カラム追加（soft delete 用）
--   - 部分インデックス追加（現役ユーザー検索の高速化）
--   - 既存の line_users_update_own ポリシーを削除（広い UPDATE を禁止）
--   - RPC: revoke_line_user(uuid) — 取り消し
--   - RPC: update_line_user_display_name(uuid, text) — 表示名編集
-- =============================================================================


-- -----------------------------------------------------------------------------
-- 1. Schema changes
-- -----------------------------------------------------------------------------

alter table line_users
  add column revoked_at timestamptz;

create index line_users_active_idx
  on line_users (line_user_id)
  where revoked_at is null;


-- -----------------------------------------------------------------------------
-- 2. Tighten RLS: 広い UPDATE ポリシーを削除
-- -----------------------------------------------------------------------------

drop policy if exists line_users_update_own on line_users;


-- -----------------------------------------------------------------------------
-- 3. RPC: revoke_line_user
-- -----------------------------------------------------------------------------

create or replace function public.revoke_line_user(p_line_user_id uuid)
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

  update line_users
    set revoked_at = now()
  where id          = p_line_user_id
    and admin_id    = v_admin_id
    and revoked_at  is null;

  if not found then
    raise exception 'line_user not found, not owned, or already revoked';
  end if;
end;
$$;

revoke all on function public.revoke_line_user(uuid) from public;
grant execute on function public.revoke_line_user(uuid) to authenticated;


-- -----------------------------------------------------------------------------
-- 4. RPC: update_line_user_display_name
-- -----------------------------------------------------------------------------

create or replace function public.update_line_user_display_name(
    p_line_user_id uuid,
    p_display_name text
)
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

  update line_users
    set display_name = nullif(trim(p_display_name), '')
  where id       = p_line_user_id
    and admin_id = v_admin_id;

  if not found then
    raise exception 'line_user not found or not owned';
  end if;
end;
$$;

revoke all on function public.update_line_user_display_name(uuid, text) from public;
grant execute on function public.update_line_user_display_name(uuid, text) to authenticated;

-- -----------------------------------------------------------------------------
-- 5. RLS パフォーマンス改善: auth.uid() を (select auth.uid()) でラップ
-- 行ごとに再評価されるのを防ぎ、InitPlan で 1 回だけ評価されるようにする
-- -----------------------------------------------------------------------------

alter policy line_users_select_own on line_users
  using (admin_id = (select auth.uid()));

alter policy conversations_select_own on conversations
  using (
    exists (
      select 1
      from line_users lu
      where lu.id = conversations.line_user_id
        and lu.admin_id = (select auth.uid())
    )
  );

alter policy register_tokens_select_own on register_tokens
  using (admin_id = (select auth.uid()));

alter policy register_tokens_insert_own on register_tokens
  with check (admin_id = (select auth.uid()));