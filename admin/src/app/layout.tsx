import type { Metadata } from "next";
import type { ReactNode } from "react";
import { LocaleProvider } from "@/i18n/client";
import { currentLocale, serverTranslator } from "@/i18n/server";
import { AdminHeader } from "./AdminHeader";
import "./globals.css";

export async function generateMetadata(): Promise<Metadata> {
  const t = await serverTranslator();
  return {
    title: t("app.title"),
    robots: { index: false, follow: false },
    icons: { icon: "data:," },
  };
}

export default async function RootLayout({ children }: { children: ReactNode }) {
  const locale = await currentLocale();
  return (
    <html lang={locale}>
      <body className="min-h-screen antialiased">
        <LocaleProvider locale={locale}>
          <AdminHeader />
          {children}
        </LocaleProvider>
      </body>
    </html>
  );
}
