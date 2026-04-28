"use client";

import { useState } from "react";
import { Pencil, Trash2, UserRound, Users } from "lucide-react";
import { toast } from "sonner";
import { createClient } from "@/lib/supabase/client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
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

export type LineUser = {
  id: string;
  line_user_id: string;
  display_name: string | null;
  created_at: string;
};

export function LineUsersClient({ initialUsers }: { initialUsers: LineUser[] }) {
  const [users, setUsers] = useState<LineUser[]>(initialUsers);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editingValue, setEditingValue] = useState("");
  const [busyId, setBusyId] = useState<string | null>(null);
  const [revokeTarget, setRevokeTarget] = useState<LineUser | null>(null);

  const startEdit = (user: LineUser) => {
    setEditingId(user.id);
    setEditingValue(user.display_name ?? "");
  };

  const cancelEdit = () => {
    setEditingId(null);
    setEditingValue("");
  };

  const saveEdit = async (userId: string) => {
    const supabase = createClient();
    setBusyId(userId);

    const { error } = await supabase.rpc("update_line_user_display_name", {
      p_line_user_id: userId,
      p_display_name: editingValue,
    });

    setBusyId(null);

    if (error) {
      toast.error("保存に失敗しました", { description: error.message });
      return;
    }

    const trimmed = editingValue.trim();
    setUsers((prev) =>
      prev.map((u) =>
        u.id === userId ? { ...u, display_name: trimmed === "" ? null : trimmed } : u,
      ),
    );
    setEditingId(null);
    setEditingValue("");
    toast.success("表示名を保存しました");
  };

  const confirmRevoke = async () => {
    if (!revokeTarget) return;
    const user = revokeTarget;

    const supabase = createClient();
    setBusyId(user.id);

    const { error } = await supabase.rpc("revoke_line_user", {
      p_line_user_id: user.id,
    });

    setBusyId(null);
    setRevokeTarget(null);

    if (error) {
      toast.error("取り消しに失敗しました", { description: error.message });
      return;
    }

    setUsers((prev) => prev.filter((u) => u.id !== user.id));
    toast.success("登録を取り消しました");
  };

  const revokeName = revokeTarget?.display_name ?? revokeTarget?.line_user_id ?? "";

  return (
    <div className="space-y-6">
      <div className="flex items-start gap-3">
        <div className="grid size-10 shrink-0 place-items-center rounded-lg bg-sky-100 text-sky-700">
          <Users className="size-5" />
        </div>
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <h2 className="text-2xl font-bold tracking-tight">登録ユーザー</h2>
            <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-700 tabular-nums">
              {users.length} 人
            </span>
          </div>
          <p className="mt-1 text-sm text-muted-foreground">
            表示名は鉛筆アイコンから編集できます。
          </p>
        </div>
      </div>

      {users.length === 0 ? (
        <div className="rounded-xl border border-dashed bg-white p-8 text-center sm:p-10">
          <div className="mx-auto mb-3 grid size-12 place-items-center rounded-full bg-slate-100 text-slate-500">
            <UserRound className="size-6" />
          </div>
          <p className="font-medium">まだ登録されたユーザーはいません</p>
          <p className="mt-1 text-sm text-muted-foreground">
            「登録コード発行」から新しいコードを作りましょう。
          </p>
        </div>
      ) : (
        <>
          <ul className="space-y-3 md:hidden">
            {users.map((user) => {
              const isEditing = editingId === user.id;
              const isBusy = busyId === user.id;

              return (
                <li key={user.id} className="rounded-xl border bg-white p-4 shadow-sm">
                  {isEditing ? (
                    <div className="space-y-3">
                      <Input
                        type="text"
                        value={editingValue}
                        onChange={(e) => setEditingValue(e.target.value)}
                        placeholder="表示名"
                        disabled={isBusy}
                        autoFocus
                      />
                      <div className="flex gap-2">
                        <Button
                          onClick={() => void saveEdit(user.id)}
                          size="sm"
                          disabled={isBusy}
                          className="flex-1"
                        >
                          保存
                        </Button>
                        <Button
                          onClick={cancelEdit}
                          size="sm"
                          variant="outline"
                          disabled={isBusy}
                          className="flex-1"
                        >
                          キャンセル
                        </Button>
                      </div>
                    </div>
                  ) : (
                    <div className="space-y-3">
                      <div className="flex items-start gap-3">
                        <div className="grid size-9 shrink-0 place-items-center rounded-full bg-emerald-100 text-emerald-700">
                          <UserRound className="size-4" />
                        </div>
                        <div className="min-w-0 flex-1">
                          <div className="truncate font-medium">
                            {user.display_name ?? (
                              <span className="text-muted-foreground italic">(未設定)</span>
                            )}
                          </div>
                          <div className="mt-0.5 text-xs text-muted-foreground tabular-nums">
                            登録：
                            {new Date(user.created_at).toLocaleDateString("ja-JP", {
                              year: "numeric",
                              month: "2-digit",
                              day: "2-digit",
                            })}
                          </div>
                        </div>
                        <Button
                          onClick={() => startEdit(user)}
                          size="icon-sm"
                          variant="ghost"
                          disabled={isBusy}
                          aria-label="表示名を編集"
                        >
                          <Pencil />
                        </Button>
                      </div>
                      <Button
                        onClick={() => setRevokeTarget(user)}
                        size="sm"
                        variant="destructive"
                        disabled={isBusy}
                        className="w-full"
                      >
                        <Trash2 />
                        {isBusy ? "処理中..." : "取り消し"}
                      </Button>
                    </div>
                  )}
                </li>
              );
            })}
          </ul>

          <div className="hidden overflow-hidden rounded-xl border bg-white shadow-sm md:block">
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className="border-b bg-slate-50/80">
                  <tr className="text-left text-xs font-semibold tracking-wider text-slate-600 uppercase">
                    <th className="px-5 py-3">表示名</th>
                    <th className="px-5 py-3">登録日</th>
                    <th className="px-5 py-3 text-right">操作</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {users.map((user) => {
                    const isEditing = editingId === user.id;
                    const isBusy = busyId === user.id;

                    return (
                      <tr key={user.id} className="transition-colors hover:bg-slate-50/60">
                        <td className="px-5 py-3">
                          {isEditing ? (
                            <div className="flex flex-wrap items-center gap-2">
                              <Input
                                type="text"
                                value={editingValue}
                                onChange={(e) => setEditingValue(e.target.value)}
                                placeholder="表示名"
                                className="min-w-40 flex-1"
                                disabled={isBusy}
                                autoFocus
                              />
                              <Button
                                onClick={() => void saveEdit(user.id)}
                                size="sm"
                                disabled={isBusy}
                              >
                                保存
                              </Button>
                              <Button
                                onClick={cancelEdit}
                                size="sm"
                                variant="outline"
                                disabled={isBusy}
                              >
                                キャンセル
                              </Button>
                            </div>
                          ) : (
                            <div className="flex items-center gap-3">
                              <div className="grid size-8 shrink-0 place-items-center rounded-full bg-emerald-100 text-emerald-700">
                                <UserRound className="size-4" />
                              </div>
                              <span className="truncate font-medium">
                                {user.display_name ?? (
                                  <span className="text-muted-foreground italic">(未設定)</span>
                                )}
                              </span>
                              <Button
                                onClick={() => startEdit(user)}
                                size="icon-xs"
                                variant="ghost"
                                disabled={isBusy}
                                aria-label="表示名を編集"
                              >
                                <Pencil />
                              </Button>
                            </div>
                          )}
                        </td>
                        <td className="px-5 py-3 text-sm whitespace-nowrap text-muted-foreground tabular-nums">
                          {new Date(user.created_at).toLocaleDateString("ja-JP", {
                            year: "numeric",
                            month: "2-digit",
                            day: "2-digit",
                          })}
                        </td>
                        <td className="px-5 py-3 text-right whitespace-nowrap">
                          <Button
                            onClick={() => setRevokeTarget(user)}
                            size="sm"
                            variant="destructive"
                            disabled={isBusy || isEditing}
                          >
                            <Trash2 />
                            {isBusy ? "処理中..." : "取り消し"}
                          </Button>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
        </>
      )}

      <AlertDialog
        open={revokeTarget !== null}
        onOpenChange={(open) => {
          if (!open) setRevokeTarget(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>登録を取り消しますか？</AlertDialogTitle>
            <AlertDialogDescription>
              <span className="font-medium text-foreground">{revokeName}</span>{" "}
              の登録を取り消します。会話履歴も Bot から見えなくなります。この操作は元に戻せません。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={busyId !== null}>キャンセル</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => void confirmRevoke()}
              disabled={busyId !== null}
            >
              {busyId !== null ? "処理中..." : "取り消す"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
