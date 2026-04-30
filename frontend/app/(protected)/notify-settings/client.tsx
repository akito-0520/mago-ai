"use client";

import { useState } from "react";
import Image from "next/image";
import { Bell, BellOff, Check, Copy, ExternalLink, QrCode, Trash2, UserRound } from "lucide-react";
import { toast } from "sonner";
import { createClient } from "@/lib/supabase/client";
import { Button } from "@/components/ui/button";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";

export type AdminLineLink = {
  id: string;
  line_user_id: string;
  display_name: string | null;
  created_at: string;
};

const notifyFriendUrl = process.env.NEXT_PUBLIC_LINE_NOTIFY_FRIEND_URL ?? "";

export function NotifySettingsClient({ initialLinks }: { initialLinks: AdminLineLink[] }) {
  const [links, setLinks] = useState<AdminLineLink[]>(initialLinks);
  const [token, setToken] = useState<string | null>(null);
  const [generating, setGenerating] = useState(false);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [unlinkTarget, setUnlinkTarget] = useState<AdminLineLink | null>(null);
  const [copied, setCopied] = useState(false);
  const [urlCopied, setUrlCopied] = useState(false);

  const handleCopyUrl = async () => {
    if (!notifyFriendUrl) return;
    try {
      await navigator.clipboard.writeText(notifyFriendUrl);
      setUrlCopied(true);
      toast.success("URL をコピーしました");
      setTimeout(() => setUrlCopied(false), 2000);
    } catch {
      toast.error("コピーに失敗しました");
    }
  };

  const handleGenerate = async () => {
    setGenerating(true);
    setToken(null);

    const supabase = createClient();
    const { data, error } = await supabase.rpc("generate_admin_link_token");

    setGenerating(false);

    if (error) {
      if (error.message.includes("too many unused tokens")) {
        toast.error("コードを発行できません", {
          description: "未使用のコードが 3 本あります。使われるまでお待ちください。",
        });
      } else {
        toast.error("コードの発行に失敗しました", { description: error.message });
      }
      return;
    }

    setToken(data as string);
  };

  const handleCopyToken = async () => {
    if (!token) return;
    try {
      await navigator.clipboard.writeText(token);
      setCopied(true);
      toast.success("コードをコピーしました");
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast.error("コピーに失敗しました");
    }
  };

  const confirmUnlink = async () => {
    if (!unlinkTarget) return;
    const target = unlinkTarget;
    const supabase = createClient();

    setBusyId(target.id);
    const { error } = await supabase.rpc("unlink_admin_line", {
      p_link_id: target.id,
    });
    setBusyId(null);
    setUnlinkTarget(null);

    if (error) {
      toast.error("解除に失敗しました", { description: error.message });
      return;
    }

    setLinks((prev) => prev.filter((l) => l.id !== target.id));
    toast.success("連携を解除しました");
  };

  const unlinkName = unlinkTarget?.display_name ?? unlinkTarget?.line_user_id ?? "";

  return (
    <div className="space-y-6">
      <div className="flex items-start gap-3">
        <div className="grid size-10 shrink-0 place-items-center rounded-lg bg-amber-100 text-amber-700">
          <Bell className="size-5" />
        </div>
        <div className="flex-1">
          <h2 className="text-2xl font-bold tracking-tight">LINE 通知設定</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            おばあちゃんからの「解決しなかった」フィードバックを LINE でリアルタイムに受け取れます。
          </p>
        </div>
      </div>

      {/* セットアップ手順 */}
      <div className="space-y-4 rounded-xl border bg-white p-5 shadow-sm">
        <h3 className="font-semibold">セットアップ手順</h3>
        <ol className="space-y-3 text-sm">
          <li className="flex gap-3">
            <span className="grid size-6 shrink-0 place-items-center rounded-full bg-slate-100 text-xs font-semibold tabular-nums">
              1
            </span>
            <div>
              <div className="flex flex-wrap items-center gap-2">
                <p>
                  通知用 LINE 公式アカウントを <b>友だち追加</b> します。
                </p>
                {notifyFriendUrl && (
                  <Dialog>
                    <DialogTrigger asChild>
                      <Button variant="outline" size="sm">
                        <QrCode />
                        友だち追加
                      </Button>
                    </DialogTrigger>
                    <DialogContent>
                      <DialogHeader>
                        <DialogTitle>通知 LINE 公式アカウント</DialogTitle>
                        <DialogDescription>
                          スマホのカメラまたは LINE で読み取って友だち追加してください。
                        </DialogDescription>
                      </DialogHeader>
                      <div className="flex flex-col items-center gap-4">
                        <div className="rounded-xl border bg-white p-3">
                          <Image
                            src="/img/mago-ai-prod-admin.png"
                            alt="通知 LINE 公式アカウントの友だち追加 QR"
                            width={240}
                            height={240}
                            className="size-60"
                          />
                        </div>
                        <div className="flex w-full items-center gap-2 rounded-lg border bg-muted/50 px-3 py-2">
                          <code className="flex-1 truncate font-mono text-xs">
                            {notifyFriendUrl}
                          </code>
                          <Button
                            onClick={() => void handleCopyUrl()}
                            variant="ghost"
                            size="sm"
                            className="shrink-0"
                          >
                            {urlCopied ? <Check /> : <Copy />}
                          </Button>
                        </div>
                        <Button asChild variant="outline" size="sm" className="w-full">
                          <a href={notifyFriendUrl} target="_blank" rel="noopener noreferrer">
                            <ExternalLink />
                            LINE で開く
                          </a>
                        </Button>
                      </div>
                    </DialogContent>
                  </Dialog>
                )}
              </div>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="grid size-6 shrink-0 place-items-center rounded-full bg-slate-100 text-xs font-semibold tabular-nums">
              2
            </span>
            <div>
              下のボタンで <b>連携コード</b> を発行します。
            </div>
          </li>
          <li className="flex gap-3">
            <span className="grid size-6 shrink-0 place-items-center rounded-full bg-slate-100 text-xs font-semibold tabular-nums">
              3
            </span>
            <div>
              友だち追加した通知 Bot に、表示された <b>6 桁のコード</b> を送信します。
            </div>
          </li>
        </ol>
      </div>

      {/* コード発行 */}
      <div className="space-y-4 rounded-xl border bg-white p-5 shadow-sm">
        <h3 className="font-semibold">連携コード発行</h3>
        <Button onClick={() => void handleGenerate()} disabled={generating} size="lg">
          {generating ? "発行中..." : "コードを発行する"}
        </Button>

        {token && (
          <div className="rounded-md bg-emerald-50 p-5 text-center">
            <p className="text-sm text-emerald-900">発行されたコード（1 時間有効）</p>
            <div className="mt-2 flex items-center justify-center gap-3">
              <p className="font-mono text-4xl font-bold tracking-widest tabular-nums">{token}</p>
              <Button
                onClick={() => void handleCopyToken()}
                variant="ghost"
                size="icon-sm"
                aria-label="コピー"
              >
                {copied ? <Check /> : <Copy />}
              </Button>
            </div>
            <p className="mt-3 text-xs text-emerald-900">
              通知 Bot にこの 6 桁を送信してください。
            </p>
          </div>
        )}
      </div>

      {/* 連携済み一覧 */}
      <div className="space-y-3">
        <div className="flex items-center gap-2">
          <h3 className="font-semibold">連携済みアカウント</h3>
          <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-700 tabular-nums">
            {links.length} 件
          </span>
        </div>

        {links.length === 0 ? (
          <div className="rounded-xl border border-dashed bg-white p-8 text-center sm:p-10">
            <div className="mx-auto mb-3 grid size-12 place-items-center rounded-full bg-slate-100 text-slate-500">
              <BellOff className="size-6" />
            </div>
            <p className="font-medium">まだ連携されたアカウントはありません</p>
            <p className="mt-1 text-sm text-muted-foreground">
              上の手順に沿って LINE と連携しましょう。
            </p>
          </div>
        ) : (
          <ul className="space-y-3">
            {links.map((link) => {
              const isBusy = busyId === link.id;
              return (
                <li
                  key={link.id}
                  className="flex items-center gap-4 rounded-xl border bg-white p-4 shadow-sm"
                >
                  <div className="grid size-10 shrink-0 place-items-center rounded-full bg-emerald-100 text-emerald-700">
                    <UserRound className="size-5" />
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="truncate font-medium">
                      {link.display_name ?? (
                        <span className="text-muted-foreground italic">(未設定)</span>
                      )}
                    </div>
                    <div className="mt-0.5 text-xs text-muted-foreground tabular-nums">
                      連携日：
                      {new Date(link.created_at).toLocaleDateString("ja-JP", {
                        year: "numeric",
                        month: "2-digit",
                        day: "2-digit",
                      })}
                    </div>
                  </div>
                  <Button
                    onClick={() => setUnlinkTarget(link)}
                    size="sm"
                    variant="destructive"
                    disabled={isBusy}
                  >
                    <Trash2 />
                    {isBusy ? "処理中..." : "解除"}
                  </Button>
                </li>
              );
            })}
          </ul>
        )}
      </div>

      <AlertDialog
        open={unlinkTarget !== null}
        onOpenChange={(open) => {
          if (!open) setUnlinkTarget(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>連携を解除しますか？</AlertDialogTitle>
            <AlertDialogDescription>
              {unlinkName} との連携を解除します。以降この LINE には通知が届かなくなります。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>キャンセル</AlertDialogCancel>
            <AlertDialogAction onClick={() => void confirmUnlink()}>解除する</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
