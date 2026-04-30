"use client";

import { useState } from "react";
import { Check, Copy, Sparkles, Ticket, Users } from "lucide-react";
import { toast } from "sonner";
import { createClient } from "@/lib/supabase/client";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

export type PlanStatus = {
  plan_code: string;
  display_name: string;
  max_line_users: number;
  hourly_limit: number;
  daily_limit: number;
  used_line_users: number;
};

export function RegisterClient({ initialStatus }: { initialStatus: PlanStatus }) {
  const [status, setStatus] = useState<PlanStatus>(initialStatus);
  const [token, setToken] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [copied, setCopied] = useState(false);

  const isFull = status.used_line_users >= status.max_line_users;

  const refreshStatus = async () => {
    const supabase = createClient();
    const { data } = await supabase.rpc("get_my_plan_status").single<PlanStatus>();
    if (data) setStatus(data);
  };

  const handleGenerate = async () => {
    const supabase = createClient();

    setLoading(true);
    setToken(null);
    setCopied(false);

    const { data, error } = await supabase.rpc("generate_register_token");

    setLoading(false);

    if (error) {
      if (error.message.includes("plan max line users reached")) {
        toast.error("登録枠がいっぱいです", {
          description: `${status.display_name}プランは ${status.max_line_users} 人まで登録できます。既存ユーザーを解除するか、プランを変更してください。`,
        });
        await refreshStatus();
      } else if (error.message.includes("too many unused tokens")) {
        toast.error("発行できません", {
          description:
            "未使用のコードが 3 本あります。使われるか、1 時間で期限切れになるまで新規発行はできません。",
        });
      } else {
        toast.error("発行に失敗しました", { description: error.message });
      }
      return;
    }

    setToken(data as string);
    toast.success("コードを発行しました");
  };

  const handleCopy = async () => {
    if (!token) return;
    try {
      await navigator.clipboard.writeText(token);
      setCopied(true);
      toast.success("コピーしました");
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast.error("コピーに失敗しました");
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-start gap-3">
        <div className="grid size-10 shrink-0 place-items-center rounded-lg bg-emerald-100 text-emerald-700">
          <Ticket className="size-5" />
        </div>
        <div>
          <h2 className="text-2xl font-bold tracking-tight">登録コード発行</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            おばあちゃんに伝える 6 桁のコードを発行します。1 時間有効・1 回限り。
          </p>
        </div>
      </div>

      {/* プラン状況 */}
      <Card>
        <CardHeader>
          <div className="flex items-center gap-3">
            <div className="grid size-9 shrink-0 place-items-center rounded-full bg-slate-100 text-slate-700">
              <Users className="size-4" />
            </div>
            <div className="flex-1">
              <CardTitle className="text-base">
                登録可能人数：
                <span className="tabular-nums">
                  {status.used_line_users} / {status.max_line_users}
                </span>
                <span className="ml-2 rounded-full bg-emerald-100 px-2 py-0.5 text-xs font-medium text-emerald-700">
                  {status.display_name}プラン
                </span>
              </CardTitle>
              <CardDescription>
                1 時間あたり {status.hourly_limit} 回 / 1 日あたり {status.daily_limit} 回まで Bot
                に質問できます。
              </CardDescription>
            </div>
          </div>
        </CardHeader>
      </Card>

      <Card>
        <CardHeader>
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <CardTitle>新しいコードを作る</CardTitle>
              <CardDescription>
                {isFull
                  ? "登録枠がいっぱいのため、新しいコードを発行できません。"
                  : "未使用のコードは最大 3 本までです。"}
              </CardDescription>
            </div>
            <Button
              onClick={() => void handleGenerate()}
              disabled={loading || isFull}
              size="lg"
              className="w-full sm:w-auto"
            >
              <Sparkles />
              {loading ? "発行中..." : "コードを発行する"}
            </Button>
          </div>
        </CardHeader>

        {token && (
          <CardContent>
            <div className="rounded-xl border border-emerald-200 bg-linear-to-br from-emerald-50 to-white p-5 text-center sm:p-6">
              <p className="text-xs font-medium tracking-wider text-emerald-700 uppercase">
                発行されたコード
              </p>
              <p className="mt-3 font-mono text-4xl font-bold tracking-[0.15em] break-all text-emerald-900 tabular-nums sm:text-5xl sm:tracking-[0.2em]">
                {token}
              </p>
              <div className="mt-4 flex justify-center">
                <Button onClick={() => void handleCopy()} variant="outline" size="sm">
                  {copied ? <Check /> : <Copy />}
                  {copied ? "コピーしました" : "コピー"}
                </Button>
              </div>
              <p className="mt-4 text-sm text-muted-foreground">
                このコードを LINE で Bot に送信してもらってください。
              </p>
            </div>
          </CardContent>
        )}
      </Card>
    </div>
  );
}
