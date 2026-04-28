import { createClient } from "@/lib/supabase/server";
import { redirect } from "next/navigation";
import { HeaderNav } from "./nav";

export default async function ProtectedLayout({ children }: { children: React.ReactNode }) {
  const supabase = await createClient();
  const {
    data: { user },
  } = await supabase.auth.getUser();

  if (!user) {
    redirect("/login");
  }

  return (
    <div className="min-h-screen bg-linear-to-b from-slate-50 to-white">
      <HeaderNav email={user.email} />
      <main className="mx-auto max-w-5xl px-4 py-6 sm:px-6 sm:py-8">{children}</main>
    </div>
  );
}
