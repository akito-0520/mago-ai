import Link from "next/link";
import { ChevronRight, MessageSquare, MessageSquareOff, UserRound } from "lucide-react";
import { createClient } from "@/lib/supabase/server";

export default async function ConversationsPage() {
  const supabase = await createClient();

  // line_users を取得（取り消し済みも含めて全部）
  const { data: users, error: usersError } = await supabase
    .from("line_users")
    .select("id, line_user_id, display_name, revoked_at, created_at")
    .order("created_at", { ascending: false });

  if (usersError) {
    return (
      <div className="rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700">
        エラー: {usersError.message}
      </div>
    );
  }

  // 各ユーザーのメッセージ件数 + 最終発言時刻を集約
  const userIds = (users ?? []).map((u) => u.id);
  let countMap = new Map<string, { count: number; lastAt: string | null }>();

  if (userIds.length > 0) {
    const { data: msgs } = await supabase
      .from("conversations")
      .select("line_user_id, created_at")
      .in("line_user_id", userIds);

    countMap = (msgs ?? []).reduce((acc, m) => {
      const cur = acc.get(m.line_user_id) ?? { count: 0, lastAt: null };
      cur.count += 1;
      if (cur.lastAt === null || cur.lastAt < m.created_at) {
        cur.lastAt = m.created_at;
      }
      acc.set(m.line_user_id, cur);
      return acc;
    }, new Map<string, { count: number; lastAt: string | null }>());
  }

  // 最終発言時刻でソート（発言なしは末尾）
  const sortedUsers = [...(users ?? [])].sort((a, b) => {
    const aLast = countMap.get(a.id)?.lastAt;
    const bLast = countMap.get(b.id)?.lastAt;
    if (!aLast && !bLast) return 0;
    if (!aLast) return 1;
    if (!bLast) return -1;
    return bLast.localeCompare(aLast);
  });

  return (
    <div className="space-y-6">
      <div className="flex items-start gap-3">
        <div className="grid size-10 shrink-0 place-items-center rounded-lg bg-violet-100 text-violet-700">
          <MessageSquare className="size-5" />
        </div>
        <div className="flex-1">
          <h2 className="text-2xl font-bold tracking-tight">会話ログ</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            登録ユーザーごとの会話履歴を確認できます。
          </p>
        </div>
      </div>

      {sortedUsers.length === 0 ? (
        <div className="rounded-xl border border-dashed bg-white p-8 text-center sm:p-10">
          <div className="mx-auto mb-3 grid size-12 place-items-center rounded-full bg-slate-100 text-slate-500">
            <UserRound className="size-6" />
          </div>
          <p className="font-medium">登録されたユーザーがいません</p>
          <p className="mt-1 text-sm text-muted-foreground">「登録コード発行」から始めましょう。</p>
        </div>
      ) : (
        <ul className="space-y-3">
          {sortedUsers.map((user) => {
            const stats = countMap.get(user.id);
            const count = stats?.count ?? 0;
            const lastAt = stats?.lastAt;
            const isRevoked = user.revoked_at !== null;

            return (
              <li key={user.id}>
                <Link
                  href={`/conversations/${user.id}`}
                  className="flex items-center gap-4 rounded-xl border bg-white p-4 shadow-sm transition hover:shadow-md"
                >
                  <div
                    className={`grid size-10 shrink-0 place-items-center rounded-full ${
                      isRevoked ? "bg-slate-100 text-slate-400" : "bg-emerald-100 text-emerald-700"
                    }`}
                  >
                    {isRevoked ? (
                      <MessageSquareOff className="size-5" />
                    ) : (
                      <UserRound className="size-5" />
                    )}
                  </div>

                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="truncate font-medium">
                        {user.display_name ?? (
                          <span className="text-muted-foreground italic">(未設定)</span>
                        )}
                      </span>
                      {isRevoked && (
                        <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600">
                          取り消し済み
                        </span>
                      )}
                    </div>
                    <div className="mt-0.5 flex items-center gap-3 text-xs text-muted-foreground">
                      <span className="tabular-nums">{count} 件</span>
                      {lastAt && (
                        <span>
                          最終:{" "}
                          {new Date(lastAt).toLocaleString("ja-JP", {
                            month: "2-digit",
                            day: "2-digit",
                            hour: "2-digit",
                            minute: "2-digit",
                          })}
                        </span>
                      )}
                    </div>
                  </div>

                  <ChevronRight className="size-5 shrink-0 text-muted-foreground" />
                </Link>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
