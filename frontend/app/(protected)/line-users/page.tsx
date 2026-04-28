import { createClient } from "@/lib/supabase/server";
import { LineUsersClient, type LineUser } from "./client";

export default async function LineUsersPage() {
  const supabase = await createClient();

  const { data, error } = await supabase
    .from("line_users")
    .select("id, line_user_id, display_name, created_at")
    .is("revoked_at", null)
    .order("created_at", { ascending: false });

  if (error) {
    return <div className="rounded-md bg-red-50 p-4 text-red-700">エラー: {error.message}</div>;
  }

  return <LineUsersClient initialUsers={(data ?? []) as LineUser[]} />;
}
