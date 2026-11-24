import { roleNames } from "@/accounts/roles";
import { adminApi } from "@/admin/api";
import { googleClientId } from "@/admin/google";
import { can, canView } from "@/admin/navigation";
import { currentAdmin, type AdminAccount } from "@/admin/session";
import { serverTranslator } from "@/i18n/server";
import { Notice } from "../players/Notice";
import { AccountActions, CreateAccount } from "./AccountForms";

export const dynamic = "force-dynamic";

type ListedAccount = AdminAccount & { created_at: string; last_login_at?: string; disabled_at?: string };

export default async function AccountsPage() {
  const t = await serverTranslator();
  const admin = await currentAdmin();
  if (!admin) {
    return <Notice text={t("auth.required")} />;
  }
  if (!canView(admin.permissions, "admin_accounts")) {
    return <Notice text={t("auth.forbidden")} />;
  }
  const reply = await adminApi<{ accounts: ListedAccount[] }>("GET", "/admin/accounts");
  if (reply.status !== 200) {
    return <Notice text={t("accounts.load_failed")} />;
  }
  const permissions = admin.permissions;
  const canCreate = can(permissions, "admin_accounts", "create");
  const canEdit = can(permissions, "admin_accounts", "edit");
  const canDisable = can(permissions, "admin_accounts", "retire");
  const canGrantSuperAdmin = can(permissions, "admin_accounts", "grant_super_admin");
  return (
    <main className="mx-auto flex max-w-6xl flex-col gap-8 px-6 py-10">
      <div className="flex flex-col gap-2">
        <h1 className="text-2xl font-semibold">{t("accounts.title")}</h1>
        <p className="text-sm text-text-secondary">{t("accounts.explain")}</p>
      </div>
      {canCreate && <CreateAccount clientId={googleClientId()} />}
      <table className="text-sm">
        <thead className="text-left text-text-muted">
          <tr>
            <th className="py-2 pr-4">{t("accounts.display_name")}</th>
            <th className="py-2 pr-4">{t("accounts.roles")}</th>
            <th className="py-2 pr-4">{t("accounts.state")}</th>
            <th className="py-2 pr-4">{t("accounts.last_login")}</th>
            <th className="py-2 pr-4" />
          </tr>
        </thead>
        <tbody>
          {reply.body.accounts.map((account) => {
            const own = account.id === admin.account.id;
            const active = account.state === "active";
            return (
              <tr key={account.id} className="border-t border-surface align-top">
                <td className="py-2 pr-4 break-all">
                  <div>{account.display_name}</div>
                  <div className="text-text-muted">{account.email}</div>
                </td>
                <td className="py-2 pr-4">{roleNames(account.roles, t)}</td>
                <td className="py-2 pr-4">{active ? t("accounts.active") : t("accounts.disabled")}</td>
                <td className="py-2 pr-4 whitespace-nowrap">{account.last_login_at ? t.when(account.last_login_at) : t("accounts.never")}</td>
                <td className="py-2 pr-4">
                  {own ? (
                    <span className="text-text-muted">{t("accounts.own")}</span>
                  ) : (
                    active &&
                    (canEdit || canDisable) && (
                      <AccountActions id={account.id} held={account.roles} canEdit={canEdit} canDisable={canDisable} canGrantSuperAdmin={canGrantSuperAdmin} clientId={googleClientId()} />
                    )
                  )}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </main>
  );
}
