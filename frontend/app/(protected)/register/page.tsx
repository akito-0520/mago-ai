import { createClient } from "@/lib/supabase/server";
import { RegisterClient, type PlanStatus } from "./client";

export default async function RegisterPage() {
  const supabase = await createClient();

  const { data, error } = await supabase.rpc("get_my_plan_status").single<PlanStatus>();

  if (error) {
    return (
      <div className="rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700">
        プラン情報の取得に失敗しました：{error.message}
      </div>
    );
  }

  return <RegisterClient initialStatus={data} />;
}
