import { adminApi } from "@/admin/api";
import { googleClientId } from "@/admin/google";
import { canRollBack, canView } from "@/admin/navigation";
import { currentAdmin } from "@/admin/session";
import { serverTranslator } from "@/i18n/server";
import type { Translate } from "@/i18n/translate";
import { changedOffers, emptyShopCatalog, inEffectWithinDays, type CosmeticItem, type PublishedCatalog, type RotationRules, type ShopCatalog } from "@/shop/catalog";
import { Rollback } from "../config/Rollback";
import { rollbackCatalog } from "./actions";
import { ShopEditor } from "./ShopEditor";

export const dynamic = "force-dynamic";

export default async function ShopPage() {
  const t = await serverTranslator();
  const admin = await currentAdmin();
  if (!admin) {
    return <Notice text={t("auth.required")} />;
  }
  if (!canView(admin.permissions, "shop_offers")) {
    return <Notice text={t("auth.forbidden")} />;
  }
  const [shop, items, history] = await Promise.all([
    adminApi<{ config_version: number; catalog: ShopCatalog | null; rotation: RotationRules; reference_price_days: number }>("GET", "/admin/config/shop"),
    adminApi<{ items: CosmeticItem[] }>("GET", "/admin/cosmetics/items"),
    adminApi<{ versions: PublishedCatalog[] }>("GET", "/admin/config/shop/history"),
  ]);
  if (shop.status !== 200 || items.status !== 200 || history.status !== 200) {
    return <Notice text={t("shop.load_failed")} />;
  }
  return (
    <main className="mx-auto flex max-w-5xl flex-col gap-8 px-6 py-10">
      <h1 className="text-2xl font-semibold">{t("shop.title")}</h1>
      {shop.body.catalog === null && <p className="text-text-secondary">{t("shop.empty")}</p>}
      <ShopEditor
        initialCatalog={shop.body.catalog ?? emptyShopCatalog()}
        version={shop.body.config_version}
        items={items.body.items}
        rules={shop.body.rotation}
        referencePrices={{ days: shop.body.reference_price_days, inEffect: inEffectWithinDays(history.body.versions, shop.body.reference_price_days, new Date()) }}
      />
      <History versions={history.body.versions} rollback={canRollBack(admin.permissions, "shop_offers") ? googleClientId() : null} t={t} />
    </main>
  );
}

function History({ versions, rollback, t }: { versions: PublishedCatalog[]; rollback: string | null; t: Translate }) {
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-lg font-semibold">{t("shop.history")}</h2>
      {versions.length === 0 ? (
        <p className="text-text-secondary">{t("shop.history_empty")}</p>
      ) : (
        <table className="text-sm">
          <thead className="text-left text-text-muted">
            <tr>
              <th className="py-2 pr-4">{t("shop.version")}</th>
              <th className="py-2 pr-4">{t("shop.published_at")}</th>
              <th className="py-2 pr-4">{t("shop.offer_count")}</th>
              <th className="py-2 pr-4">{t("shop.changed_offers")}</th>
              {rollback && <th className="py-2 pr-4" />}
            </tr>
          </thead>
          <tbody>
            {versions.map((version, index) => (
              <tr key={version.config_version} className="border-t border-surface align-top">
                <td className="py-2 pr-4">{version.config_version}</td>
                <td className="py-2 pr-4 whitespace-nowrap">{t.when(version.published_at)}</td>
                <td className="py-2 pr-4">{version.catalog.offers.length}</td>
                <td className="py-2 pr-4 font-mono break-all">{changedOffers(versions[index + 1]?.catalog ?? null, version.catalog).join(", ")}</td>
                {rollback && <td className="py-2 pr-4">{index > 0 && <Rollback version={version.config_version} clientId={rollback} action={rollbackCatalog} />}</td>}
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
}

function Notice({ text }: { text: string }) {
  return (
    <main className="mx-auto max-w-3xl px-6 py-24">
      <p className="text-text-secondary">{text}</p>
    </main>
  );
}
