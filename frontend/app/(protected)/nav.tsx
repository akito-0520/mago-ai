"use client";

import { useState } from "react";
import Image from "next/image";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import {
  Bell,
  Check,
  Copy,
  ExternalLink,
  LogOut,
  MessageSquare,
  QrCode,
  Ticket,
  Users,
} from "lucide-react";
import { toast } from "sonner";
import { createClient } from "@/lib/supabase/client";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { cn } from "@/lib/utils";

const items = [
  { href: "/line-users", label: "登録ユーザー", icon: Users },
  { href: "/conversations", label: "会話ログ", icon: MessageSquare },
  { href: "/register", label: "登録コード発行", icon: Ticket },
  { href: "/notify-settings", label: "通知設定", icon: Bell },
];

const lineFriendUrl = process.env.NEXT_PUBLIC_LINE_FRIEND_URL ?? "";

export function HeaderNav({ email }: { email: string | null | undefined }) {
  const pathname = usePathname();
  const router = useRouter();
  const [copied, setCopied] = useState(false);

  const handleSignOut = async () => {
    const supabase = createClient();
    await supabase.auth.signOut();
    router.replace("/");
    router.refresh();
  };

  const handleCopyUrl = async () => {
    if (!lineFriendUrl) return;
    try {
      await navigator.clipboard.writeText(lineFriendUrl);
      setCopied(true);
      toast.success("URL をコピーしました");
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast.error("コピーに失敗しました");
    }
  };

  return (
    <header className="sticky top-0 z-30 border-b bg-white/80 backdrop-blur supports-backdrop-filter:bg-white/60">
      <div className="mx-auto flex max-w-5xl items-center justify-between px-4 py-3 sm:px-6">
        <Link href="/line-users" className="flex items-center gap-2">
          <Image
            src="/img/app_icon.png"
            alt="mago.ai"
            width={32}
            height={32}
            priority
            className="size-8 rounded-lg shadow-sm"
          />
          <span className="text-lg font-bold tracking-tight">mago.ai</span>
          <span className="hidden text-xs text-muted-foreground sm:inline">管理画面</span>
        </Link>
        <div className="flex items-center gap-2">
          {email && <span className="hidden text-xs text-muted-foreground md:inline">{email}</span>}
          {lineFriendUrl && (
            <Dialog>
              <DialogTrigger asChild>
                <Button variant="outline" size="sm">
                  <QrCode />
                  <span className="hidden sm:inline">友だち追加</span>
                </Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>LINE 公式アカウント</DialogTitle>
                  <DialogDescription>
                    おばあちゃんに QR を読み取ってもらうか、URL を送ってください。
                  </DialogDescription>
                </DialogHeader>
                <div className="flex flex-col items-center gap-4">
                  <div className="rounded-xl border bg-white p-3">
                    <Image
                      src="/img/mago-ai_prod.png"
                      alt="LINE 公式アカウントの友だち追加 QR"
                      width={240}
                      height={240}
                      className="size-60"
                    />
                  </div>
                  <div className="flex w-full items-center gap-2 rounded-lg border bg-muted/50 px-3 py-2">
                    <code className="flex-1 truncate font-mono text-xs">{lineFriendUrl}</code>
                    <Button
                      onClick={() => void handleCopyUrl()}
                      variant="ghost"
                      size="sm"
                      className="shrink-0"
                    >
                      {copied ? <Check /> : <Copy />}
                    </Button>
                  </div>
                  <Button asChild variant="outline" size="sm" className="w-full">
                    <a href={lineFriendUrl} target="_blank" rel="noopener noreferrer">
                      <ExternalLink />
                      LINE で開く
                    </a>
                  </Button>
                </div>
              </DialogContent>
            </Dialog>
          )}
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
