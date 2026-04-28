"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { LogOut, Users, Ticket } from "lucide-react";
import { createClient } from "@/lib/supabase/client";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

const items = [
  { href: "/line-users", label: "登録ユーザー", icon: Users },
  { href: "/register", label: "登録コード発行", icon: Ticket },
];

export function HeaderNav({ email }: { email: string | null | undefined }) {
  const pathname = usePathname();
  const router = useRouter();

  const handleSignOut = async () => {
    const supabase = createClient();
    await supabase.auth.signOut();
    router.replace("/");
    router.refresh();
  };

  return (
    <header className="sticky top-0 z-30 border-b bg-white/80 backdrop-blur supports-[backdrop-filter]:bg-white/60">
      <div className="mx-auto flex max-w-5xl items-center justify-between px-4 py-3 sm:px-6">
        <Link href="/line-users" className="flex items-center gap-2">
          <span className="grid size-8 place-items-center rounded-lg bg-emerald-500 text-white shadow-sm">
            <span className="text-base font-bold">孫</span>
          </span>
          <span className="text-lg font-bold tracking-tight">mago.ai</span>
          <span className="hidden text-xs text-muted-foreground sm:inline">管理画面</span>
        </Link>
        <div className="flex items-center gap-2">
          {email && <span className="hidden text-xs text-muted-foreground md:inline">{email}</span>}
          <Button onClick={() => void handleSignOut()} variant="ghost" size="sm">
            <LogOut />
            <span className="hidden sm:inline">ログアウト</span>
          </Button>
        </div>
      </div>
      <nav className="mx-auto flex max-w-5xl gap-1 overflow-x-auto px-2 sm:px-4">
        {items.map((item) => {
          const Icon = item.icon;
          const active = pathname === item.href || pathname.startsWith(item.href + "/");
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "relative inline-flex shrink-0 items-center gap-2 px-3 py-2.5 text-sm font-medium whitespace-nowrap transition-colors",
                active ? "text-emerald-700" : "text-muted-foreground hover:text-foreground",
              )}
            >
              <Icon className="size-4" />
              {item.label}
              {active && (
                <span className="absolute inset-x-2 -bottom-px h-0.5 rounded-full bg-emerald-500" />
              )}
            </Link>
          );
        })}
      </nav>
    </header>
  );
}
