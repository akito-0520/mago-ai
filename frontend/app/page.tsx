import Link from "next/link";
import { redirect } from "next/navigation";
import { ArrowRight, Heart, MessageCircle, ShieldCheck, Sparkles } from "lucide-react";
import { createClient } from "@/lib/supabase/server";
import { Button } from "@/components/ui/button";

export default async function Home() {
  const supabase = await createClient();
  const {
    data: { user },
  } = await supabase.auth.getUser();

  if (user) {
    redirect("/line-users");
  }

  return (
    <div className="min-h-screen bg-gradient-to-b from-emerald-50/60 via-white to-sky-50/60">
      <header className="mx-auto flex max-w-6xl items-center justify-between px-4 py-4 sm:px-6 sm:py-5">
        <Link href="/" className="flex items-center gap-2">
          <span className="grid size-9 place-items-center rounded-xl bg-emerald-500 text-white shadow-sm">
            <span className="text-base font-bold">孫</span>
          </span>
          <span className="text-lg font-bold tracking-tight">mago.ai</span>
        </Link>
        <Link href="/login">
          <Button variant="ghost" size="sm">
            ログイン
          </Button>
        </Link>
      </header>

      <section className="mx-auto max-w-6xl px-4 pt-8 pb-16 sm:px-6 sm:pt-16 sm:pb-28">
        <div className="grid items-center gap-10 md:grid-cols-2 md:gap-12">
          <div>
            <span className="inline-flex items-center gap-1.5 rounded-full border border-emerald-200 bg-white px-3 py-1 text-xs font-medium text-emerald-700">
              <Sparkles className="size-3" />孫 × AI で見守る LINE Bot
            </span>
            <h1 className="mt-5 text-3xl font-bold tracking-tight text-slate-900 sm:text-4xl md:text-5xl">
              おばあちゃんの
              <br />
              「ちょっと聞きたい」を
              <br />
              <span className="text-emerald-600">いつでも、やさしく。</span>
            </h1>
            <p className="mt-6 max-w-md text-base leading-relaxed text-slate-600">
              スマホ操作の疑問を LINE で気軽に質問。AI が
              やさしい言葉で答え、孫はあとから様子を見守れます。
            </p>
            <div className="mt-8 flex flex-col gap-3 sm:flex-row">
              <Link href="/login" className="w-full sm:w-auto">
                <Button size="lg" className="w-full sm:w-auto">
                  管理画面へログイン
                  <ArrowRight />
                </Button>
              </Link>
              <a href="#features" className="w-full sm:w-auto">
                <Button variant="outline" size="lg" className="w-full sm:w-auto">
                  できること を見る
                </Button>
              </a>
            </div>
          </div>

          <div className="relative order-first md:order-last">
            <div className="absolute -inset-6 rounded-[2.5rem] bg-gradient-to-br from-emerald-200/40 via-sky-200/40 to-transparent blur-2xl" />
            <div className="relative mx-auto w-full max-w-xs rounded-[2rem] border bg-white p-4 shadow-xl sm:max-w-sm sm:p-5">
              <div className="flex items-center gap-2 border-b pb-3">
                <span className="grid size-8 place-items-center rounded-full bg-emerald-100 text-emerald-700">
                  <Heart className="size-4" />
                </span>
                <span className="text-sm font-semibold">mago.ai Bot</span>
              </div>
              <div className="mt-4 space-y-3">
                <div className="ml-auto max-w-[80%] rounded-2xl rounded-tr-sm bg-emerald-500 px-3.5 py-2 text-sm text-white">
                  写真を孫に送りたいんだけど、どうするの？
                </div>
                <div className="max-w-[85%] rounded-2xl rounded-tl-sm bg-slate-100 px-3.5 py-2 text-sm text-slate-800">
                  だいじょうぶです。
                  <br />
                  ① LINE のトーク画面をひらきます
                  <br />
                  ② 左下の「＋」をおします
                  <br />③ 「写真」をえらびます
                </div>
                <div className="max-w-[85%] rounded-2xl rounded-tl-sm bg-slate-100 px-3.5 py-2 text-sm text-slate-800">
                  ここまでできましたか？
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section id="features" className="mx-auto max-w-6xl px-4 pb-16 sm:px-6 sm:pb-24">
        <div className="text-center">
          <h2 className="text-2xl font-bold tracking-tight sm:text-3xl">できること</h2>
          <p className="mt-2 text-sm text-slate-600">
            シニア向けに配慮した UX と、孫のための見守り機能。
          </p>
        </div>
        <div className="mt-8 grid gap-4 sm:mt-10 sm:gap-5 md:grid-cols-3">
          <FeatureCard
            icon={<MessageCircle className="size-5" />}
            title="LINE で気軽に質問"
            body="使い慣れた LINE からそのまま聞ける。アプリのインストールは不要です。"
          />
          <FeatureCard
            icon={<Sparkles className="size-5" />}
            title="やさしい AI の返答"
            body="難しい言葉を避け、1 ステップずつ手順を案内。最後に確認の一言を添えます。"
          />
          <FeatureCard
            icon={<ShieldCheck className="size-5" />}
            title="孫が見守れる管理画面"
            body="複数の祖父母を 1 アカウントで管理。会話ログを確認し、登録コードを発行できます。"
          />
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-4 pb-16 sm:px-6 sm:pb-24">
        <div className="overflow-hidden rounded-2xl border bg-gradient-to-br from-emerald-500 to-emerald-600 px-6 py-8 text-white shadow-lg shadow-emerald-500/20 sm:px-12 sm:py-14">
          <div className="flex flex-col items-start justify-between gap-6 sm:flex-row sm:items-center">
            <div>
              <h3 className="text-xl font-bold tracking-tight sm:text-3xl">さあ、はじめましょう</h3>
              <p className="mt-2 text-sm text-emerald-50">
                Google アカウントでログインしてすぐに使えます。
              </p>
            </div>
            <Link href="/login" className="w-full sm:w-auto">
              <Button size="lg" variant="secondary" className="w-full sm:w-auto">
                管理画面へログイン
                <ArrowRight />
              </Button>
            </Link>
          </div>
        </div>
      </section>

      <footer className="border-t bg-white/60">
        <div className="mx-auto flex max-w-6xl flex-col items-center justify-between gap-2 px-4 py-6 text-center text-xs text-slate-500 sm:flex-row sm:px-6 sm:text-left">
          <span>© {new Date().getFullYear()} mago.ai</span>
          <span>孫 × AI でつくる、おばあちゃんのスマホ相談相手。</span>
        </div>
      </footer>
    </div>
  );
}

function FeatureCard({
  icon,
  title,
  body,
}: {
  icon: React.ReactNode;
  title: string;
  body: string;
}) {
  return (
    <div className="rounded-2xl border bg-white p-6 shadow-sm transition-shadow hover:shadow-md">
      <div className="grid size-10 place-items-center rounded-lg bg-emerald-100 text-emerald-700">
        {icon}
      </div>
      <h3 className="mt-4 font-semibold">{title}</h3>
      <p className="mt-1.5 text-sm leading-relaxed text-slate-600">{body}</p>
    </div>
  );
}
