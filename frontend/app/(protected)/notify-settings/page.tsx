import { createClient } from "@/lib/supabase/server";
import { NotifySettingsClient, type AdminLineLink } from "./client";

export default async function NotifySettingsPage() {
  const supabase = await createClient();

  const { data, error } = await supabase
    .from("admin_line_links")
    .select("id, line_user_id, display_name, created_at")
    .order("created_at", { ascending: false });

  if (error) {
    return (
      <div className="rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700">
        エラー: {error.message}
      </div>
    );
  }

  return <NotifySettingsClient initialLinks={(data ?? []) as AdminLineLink[]} />;
}
