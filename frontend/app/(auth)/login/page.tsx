"use client";

import Image from "next/image";
import { toast } from "sonner";
import { createClient } from "@/lib/supabase/client";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

export default function LoginPage() {
  const handleGoogleLogin = async () => {
    const supabase = createClient();

    const { error } = await supabase.auth.signInWithOAuth({
      provider: "google",
      options: {
        redirectTo: `${process.env.NEXT_PUBLIC_SITE_URL ?? window.location.origin}/auth/callback`,
      },
    });
    if (error) {
      console.error("login error", error);
      toast.error("ログインに失敗しました", { description: error.message });
    }
  };

  return (
    <main className="flex min-h-screen items-center justify-center bg-linear-to-br from-emerald-50 via-white to-sky-50 px-6">
      <div className="w-full max-w-sm">
        <div className="mb-8 text-center">
          <Image
            src="/app_icon.png"
            alt="mago.ai"
            width={80}
            height={80}
            priority
            className="mx-auto mb-4 size-20 rounded-2xl shadow-lg shadow-emerald-500/15"
          />
          <h1 className="text-2xl font-bold tracking-tight">mago.ai</h1>
          <p className="mt-1 text-sm text-muted-foreground">おばあちゃん見守り Bot の管理画面</p>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>ログイン</CardTitle>
            <CardDescription>Google アカウントでログインしてください。</CardDescription>
          </CardHeader>
          <CardContent>
            <Button onClick={handleGoogleLogin} variant="outline" size="lg" className="w-full">
              <GoogleIcon />
              Google でログイン
            </Button>
          </CardContent>
        </Card>

        <p className="mt-6 text-center text-xs text-muted-foreground">
          孫専用の管理画面です。おばあちゃんは LINE からご利用ください。
        </p>
      </div>
    </main>
  );
}

function GoogleIcon() {
  return (
    <svg viewBox="0 0 48 48" aria-hidden="true" className="size-4">
      <path
        fill="#4285F4"
        d="M47.5 24.55c0-1.64-.15-3.22-.42-4.73H24v8.94h13.18c-.57 3.06-2.3 5.65-4.9 7.39v6.13h7.92c4.63-4.27 7.3-10.55 7.3-17.73z"
      />
      <path
        fill="#34A853"
        d="M24 48c6.62 0 12.18-2.2 16.24-5.96l-7.92-6.13c-2.2 1.47-5.01 2.34-8.32 2.34-6.4 0-11.82-4.32-13.76-10.13H1.99v6.36C6.03 42.51 14.4 48 24 48z"
      />
      <path
        fill="#FBBC05"
        d="M10.24 28.12c-.5-1.47-.78-3.04-.78-4.62s.28-3.15.78-4.62v-6.36H1.99A23.96 23.96 0 0 0 0 23.5c0 3.87.93 7.53 2.57 10.78l7.67-6.16z"
      />
      <path
        fill="#EA4335"
        d="M24 9.5c3.6 0 6.83 1.24 9.38 3.66l7.03-7.03C36.16 2.34 30.6 0 24 0 14.4 0 6.03 5.49 1.99 13.49l8.25 6.39C12.18 13.82 17.6 9.5 24 9.5z"
      />
    </svg>
  );
}
