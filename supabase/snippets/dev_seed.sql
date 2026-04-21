-- =============================================================================
-- ローカル開発用の seed
--
-- 使い方:
--   1. supabase db reset でスキーマを初期化
--   2. Google でログインして auth.users に自分のユーザー行を作る
--   3. Supabase Studio (http://127.0.0.1:54323) の SQL Editor にこのファイルを貼り付けて実行
--   4. 自分の UUID に紐付いた line_users / conversations / register_tokens が作成される
--
-- 本番環境では絶対に実行しないこと。
-- =============================================================================

do $$
declare
  v_admin_id     uuid;
  v_line_user_id uuid := '10000000-0000-0000-0000-000000000001'::uuid;
begin
  -- 最新のユーザーを admin として扱う (ローカルは 1 人しかいない前提)
  select id into v_admin_id
  from auth.users
  order by created_at desc
  limit 1;

  if v_admin_id is null then
    raise exception 'auth.users が空です。先に Google ログインしてください。';
  end if;

  -- idempotent 再実行のため既存 seed を一度削除
  delete from conversations    where line_user_id = v_line_user_id;
  delete from line_users       where id = v_line_user_id;
  delete from register_tokens  where token = '123456';

  -- line_users: テスト用おばあちゃん
  insert into line_users (id, admin_id, line_user_id, display_name)
  values (
    v_line_user_id,
    v_admin_id,
    'U00000000000000000000000000000001',
    'おばあちゃん (テスト)'
  );

  -- conversations: 2 往復分
  insert into conversations (line_user_id, role, content, model, input_tokens, output_tokens) values
    (v_line_user_id, 'user',      'iPhone の写真を孫に送りたい',                  null,                null, null),
    (v_line_user_id, 'assistant', 'まず「写真」アプリを開いてください。',             'claude-sonnet-4-6', 120, 45),
    (v_line_user_id, 'user',      'アプリを開いたあとは？',                         null,                null, null),
    (v_line_user_id, 'assistant', '送りたい写真をタップして選んでください。',          'claude-sonnet-4-6', 180, 60);

  -- register_tokens: 未使用のトークン 1 本
  insert into register_tokens (token, admin_id, expires_at)
  values (
    '123456',
    v_admin_id,
    now() + interval '24 hours'
  );
end $$;
