"use client";

import { useState } from "react";
import { Check, Copy, Sparkles, Ticket } from "lucide-react";
import { toast } from "sonner";
import { createClient } from "@/lib/supabase/client";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

export default function RegisterPage() {
  const [token, setToken] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [copied, setCopied] = useState(false);

  const handleGenerate = async () => {
    const supabase = createClient();

    setLoading(true);
    setToken(null);
    setCopied(false);

    const { data, error } = await supabase.rpc("generate_register_token");

    setLoading(false);

    if (error) {
      if (error.message.includes("too many unused tokens")) {
        toast.error("発行できません", {
          description: "未使用のコードが 3 本あります。使われるまで新規発行はできません。",
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
            おばあちゃんに伝える 6 桁のコードを発行します。24 時間有効・1 回限り。
          </p>
        </div>
      </div>

      <Card>
        <CardHeader>
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <CardTitle>新しいコードを作る</CardTitle>
              <CardDescription>未使用のコードは最大 3 本までです。</CardDescription>
            </div>
            <Button
              onClick={handleGenerate}
              disabled={loading}
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
            <div className="rounded-xl border border-emerald-200 bg-gradient-to-br from-emerald-50 to-white p-5 text-center sm:p-6">
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
