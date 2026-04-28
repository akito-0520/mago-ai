"use client";

import { createClient } from "@/lib/supabase/client";
import { Button } from "@/components/ui/button";
import { useState } from "react";

export default function RegisterPage() {
  const supabase = createClient();
  const [token, setToken] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleGenerate = async () => {
    setLoading(true);
    setError(null);
    setToken(null);

    const { data, error } = await supabase.rpc("generate_register_token");

    setLoading(false);

    if (error) {
      if (error.message.includes("too many unused tokens")) {
        setError("未使用のコードが 3 本あります。使われるまで新規発行はできません。");
      } else {
        setError(`発行に失敗しました：${error.message}`);
      }
      return;
    }

    setToken(data as string);
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold">登録コード発行</h2>
        <p className="mt-2 text-sm text-gray-600">
          おばあちゃんに伝える 6 桁のコードを発行します。1 時間有効・1 回限り。
        </p>
      </div>

      <Button onClick={handleGenerate} disabled={loading} size="lg">
        {loading ? "発行中..." : "コードを発行する"}
      </Button>

      {error && <div className="rounded-md bg-red-50 p-4 text-red-700">{error}</div>}

      {token && (
        <div className="rounded-md bg-green-50 p-6 text-center">
          <p className="text-sm text-gray-600">発行されたコード</p>
          <p className="mt-2 font-mono text-5xl font-bold tracking-widest">{token}</p>
          <p className="mt-4 text-sm text-gray-600">
            このコードを LINE で Bot に送信してもらってください。
          </p>
        </div>
      )}
    </div>
  );
}
