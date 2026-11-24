import Link from "next/link";
import { visibleScreens } from "@/admin/navigation";
import { currentAdmin } from "@/admin/session";
import { LocaleSwitcher } from "@/i18n/LocaleSwitcher";
import { serverTranslator } from "@/i18n/server";
import { signOut } from "./auth/actions";

export async function AdminHeader() {
  const t = await serverTranslator();
  const admin = await currentAdmin();
  return (
    <header className="flex flex-wrap items-center justify-between gap-4 px-6 py-4">
      <nav aria-label={t("nav.label")} className="flex flex-wrap gap-4">
        <Link href="/" className="text-text-secondary hover:text-text-primary">
          {t("nav.home")}
        </Link>
        {admin &&
          visibleScreens(admin.permissions).map((screen) => (
            <Link key={screen.href} href={screen.href} className="text-text-secondary hover:text-text-primary">
              {t(screen.labelKey)}
            </Link>
          ))}
      </nav>
      <div className="flex flex-wrap items-center gap-4">
        {admin && (
          <form action={signOut} className="flex flex-wrap items-center gap-4">
            <span className="text-sm break-all text-text-muted">{admin.account.email}</span>
            <button type="submit" className="rounded-md bg-surface px-3 py-1 text-sm text-text-secondary">
              {t("signin.sign_out")}
            </button>
          </form>
        )}
        <LocaleSwitcher />
      </div>
    </header>
  );
}
