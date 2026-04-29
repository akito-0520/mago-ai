import Link from "next/link";
import { notFound } from "next/navigation";
import { ArrowLeft, Bot, MessageSquare, UserRound } from "lucide-react";
import { createClient } from "@/lib/supabase/server";

type Conversation = {
  id: number;
  role: "user" | "assistant";
  content: string;
  latency_ms: number | null;
  model: string | null;
  input_tokens: number | null;
  output_tokens: number | null;
  cache_read_input_tokens: number | null;
  cache_creation_input_tokens: number | null;
  created_at: string;
};

export default async function ConversationDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const supabase = await createClient();

  // ユーザー情報
  const { data: user, error: userError } = await supabase
    .from("line_users")
    .select("id, line_user_id, display_name, revoked_at, created_at, session_reset_at")
    .eq("id", id)
    .maybeSingle();

  if (userError) {
    return (
      <div className="rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700">
        エラー: {userError.message}
      </div>
    );
  }

  if (!user) {
    notFound();
  }

  // 会話履歴（古い順）
  const { data: conversations, error: convError } = await supabase
    .from("conversations")
    .select(
      "id, role, content, latency_ms, model, input_tokens, output_tokens, cache_read_input_tokens, cache_creation_input_tokens, created_at",
    )
    .eq("line_user_id", id)
    .order("created_at", { ascending: true });

  if (convError) {
    return (
      <div className="rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700">
        エラー: {convError.message}
      </div>
    );
  }

  const messages = (conversations ?? []) as Conversation[];

  return (
    <div className="space-y-6">
      <div>
        <Link
          href="/conversations"
          className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="size-4" />
          一覧に戻る
        </Link>
      </div>

      <div className="rounded-xl border bg-white p-4 shadow-sm">
        <div className="flex items-start gap-3">
          <div className="grid size-10 shrink-0 place-items-center rounded-full bg-emerald-100 text-emerald-700">
            <UserRound className="size-5" />
          </div>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <h2 className="truncate text-xl font-bold">
                {user.display_name ?? (
                  <span className="text-muted-foreground italic">(未設定)</span>
                )}
              </h2>
              {user.revoked_at !== null && (
                <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600">
                  取り消し済み
                </span>
              )}
            </div>
            <div className="mt-1 text-xs text-muted-foreground">
              登録：
              {new Date(user.created_at).toLocaleDateString("ja-JP")}
              {user.session_reset_at && (
                <>
                  {" / 最終リセット："}
                  {new Date(user.session_reset_at).toLocaleString("ja-JP", {
                    month: "2-digit",
                    day: "2-digit",
                    hour: "2-digit",
                    minute: "2-digit",
                  })}
                </>
              )}
            </div>
          </div>
        </div>
      </div>

      {messages.length === 0 ? (
        <div className="rounded-xl border border-dashed bg-white p-8 text-center sm:p-10">
          <div className="mx-auto mb-3 grid size-12 place-items-center rounded-full bg-slate-100 text-slate-500">
            <MessageSquare className="size-6" />
          </div>
          <p className="font-medium">まだ会話はありません</p>
        </div>
      ) : (
        <div className="space-y-3">
          {messages.map((m) => (
            <MessageBubble key={m.id} message={m} />
          ))}
        </div>
      )}
    </div>
  );
}

function MessageBubble({ message }: { message: Conversation }) {
  const isUser = message.role === "user";
  const isAssistant = message.role === "assistant";

  return (
    <div className={`flex gap-3 ${isUser ? "flex-row-reverse" : ""}`}>
      <div
        className={`grid size-9 shrink-0 place-items-center rounded-full ${
          isUser ? "bg-emerald-100 text-emerald-700" : "bg-violet-100 text-violet-700"
        }`}
      >
        {isUser ? <UserRound className="size-4" /> : <Bot className="size-4" />}
      </div>

      <div className={`max-w-[80%] ${isUser ? "items-end" : ""}`}>
        <div
          className={`rounded-2xl px-4 py-3 text-sm whitespace-pre-wrap ${
            isUser ? "bg-emerald-50 text-emerald-950" : "border bg-white text-slate-900 shadow-sm"
          }`}
        >
          {message.content}
        </div>

        <div
          className={`mt-1 flex items-center gap-2 text-xs text-muted-foreground tabular-nums ${
            isUser ? "justify-end" : ""
          }`}
        >
          <span>
            {new Date(message.created_at).toLocaleString("ja-JP", {
              month: "2-digit",
              day: "2-digit",
              hour: "2-digit",
              minute: "2-digit",
            })}
          </span>
          {isAssistant && message.latency_ms !== null && (
            <span title="応答時間">{message.latency_ms} ms</span>
          )}
          {isAssistant && message.input_tokens !== null && (
            <span title="入力 / 出力トークン">
              in: {message.input_tokens} / out: {message.output_tokens}
              {message.cache_read_input_tokens
                ? ` / cached: ${message.cache_read_input_tokens}`
                : ""}
            </span>
          )}
        </div>
      </div>
    </div>
  );
}
