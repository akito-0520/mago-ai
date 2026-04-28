import type { Metadata } from "next";
import "./globals.css";
import { Analytics } from "@vercel/analytics/next";
import { Toaster } from "@/components/ui/sonner";

export const metadata: Metadata = {
  title: "mago.ai 管理アプリ",
  description: "孫のための LINE Bot 管理アプリ",
  icons: {
    icon: "/app_icon.png",
    apple: "/app_icon.png",
  },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="ja">
      <body>
        {children}
        <Toaster richColors position="top-center" />
        <Analytics />
      </body>
    </html>
  );
}
