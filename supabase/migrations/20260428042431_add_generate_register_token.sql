-- =============================================================================
-- generate_register_token: 6 桁の登録トークンを発行する RPC 関数
--
-- 制約:
--   - 認証された admin (auth.uid()) のみ実行可能
--   - 1 admin あたり「未使用 + 有効期限内」が最大 3 本まで
--   - トークンは 6 桁の半角数字、1 時間有効
--   - 衝突時は 10 回までリトライ、それ以降は例外
-- =============================================================================

create or replace function public.generate_register_token()
returns text
language plpgsql
security definer
set search_path = public
as $$
declare
  v_admin_id      uuid := auth.uid();
  v_unused_count  int;
  v_token         text;
  v_attempt       int := 0;
begin
  -- 認証チェック
  if v_admin_id is null then
    raise exception 'unauthorized: must be logged in';
  end if;

  -- 未使用上限チェック (3 本)
  select count(*)
    into v_unused_count
    from register_tokens
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
      select 1 from register_tokens where token = v_token
    );
    v_attempt := v_attempt + 1;
    if v_attempt > 10 then
      raise exception 'token generation collision: retry exhausted';
    end if;
  end loop;

  -- INSERT
  insert into register_tokens (token, admin_id, expires_at)
  values (v_token, v_admin_id, now() + interval '1 hour');

  return v_token;
end;
$$;

-- 公開権限を制限：認証済みユーザーのみ呼べる
revoke all on function public.generate_register_token() from public;
grant execute on function public.generate_register_token() to authenticated;