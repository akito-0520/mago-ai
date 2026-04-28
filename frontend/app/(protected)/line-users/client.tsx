"use client";

import { useState } from "react";
import { createClient } from "@/lib/supabase/client";
import { Button } from "@/components/ui/button";

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
      alert(`保存に失敗しました：${error.message}`);
      return;
    }

    // 楽観的更新：ローカル state を直接書き換える
    const trimmed = editingValue.trim();
    setUsers((prev) =>
      prev.map((u) =>
        u.id === userId ? { ...u, display_name: trimmed === "" ? null : trimmed } : u,
      ),
    );
    setEditingId(null);
    setEditingValue("");
  };

  const handleRevoke = async (user: LineUser) => {
    const name = user.display_name ?? user.line_user_id;
    if (!confirm(`${name} を登録解除します。よろしいですか？`)) {
      return;
    }

    const supabase = createClient();
    setBusyId(user.id);

    const { error } = await supabase.rpc("revoke_line_user", {
      p_line_user_id: user.id,
    });

    setBusyId(null);

    if (error) {
      alert(`取り消しに失敗しました：${error.message}`);
      return;
    }

    // 楽観的更新：リストから即削除
    setUsers((prev) => prev.filter((u) => u.id !== user.id));
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold">登録済みユーザー ({users.length} 人)</h2>
        <p className="mt-2 text-sm text-gray-600">表示名は名前部分をクリックして編集できます。</p>
      </div>

      {users.length === 0 ? (
        <div className="rounded-md bg-gray-50 p-6 text-center text-gray-600">
          まだ登録されたユーザーはいません。
          <br />
          「登録コード発行」から始めてください。
        </div>
      ) : (
        <div className="overflow-x-auto rounded-md border bg-white">
          <table className="w-full">
            <thead className="border-b bg-gray-50">
              <tr className="text-left text-sm font-semibold text-gray-700">
                <th className="px-4 py-3">表示名</th>
                <th className="px-4 py-3">登録日</th>
                <th className="px-4 py-3">操作</th>
              </tr>
            </thead>
            <tbody>
              {users.map((user) => {
                const isEditing = editingId === user.id;
                const isBusy = busyId === user.id;

                return (
                  <tr key={user.id} className="border-b last:border-0">
                    <td className="px-4 py-3">
                      {isEditing ? (
                        <div className="flex items-center gap-2">
                          <input
                            type="text"
                            value={editingValue}
                            onChange={(e) => setEditingValue(e.target.value)}
                            placeholder="表示名"
                            className="flex-1 rounded-md border px-3 py-1.5 text-sm focus:border-blue-500 focus:outline-none"
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
                        <button
                          onClick={() => startEdit(user)}
                          className="rounded px-2 py-1 text-left hover:bg-gray-100"
                          disabled={isBusy}
                        >
                          {user.display_name ?? <span className="text-gray-400">(なし)</span>}
                        </button>
                      )}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-600">
                      {new Date(user.created_at).toLocaleDateString("ja-JP")}
                    </td>
                    <td className="px-4 py-3">
                      <Button
                        onClick={() => void handleRevoke(user)}
                        size="sm"
                        variant="destructive"
                        disabled={isBusy || isEditing}
                      >
                        {isBusy ? "処理中..." : "取り消し"}
                      </Button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
