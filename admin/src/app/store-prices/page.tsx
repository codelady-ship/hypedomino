import { adminApi } from "@/admin/api";
import { googleClientId } from "@/admin/google";
import { can, canRollBack, canView } from "@/admin/navigation";
import { currentAdmin } from "@/admin/session";
import { serverTranslator } from "@/i18n/server";
import type { Translate } from "@/i18n/translate";
import { pricedCountries, type StorePrices, type Target } from "@/storeprices/prices";
import { Rollback } from "../config/Rollback";
import { rollbackStorePrices } from "./actions";
import { StoreExport } from "./StoreExport";
import { StorePricesEditor } from "./StorePricesEditor";
import { StorePush } from "./StorePush";

export const dynamic = "force-dynamic";

type PricesVersion = { config_version: number; published_at: string; catalog: StorePrices };

type PricesConfig = {
  config_version: number;
  catalog: StorePrices | null;
  rules: { default: StorePrices; products: string[]; countries: string[]; targets: Target[] };
};

export default async function StorePricesPage() {
  const t = await serverTranslator();
  const admin = await currentAdmin();
  if (!admin) {
    return <Notice text={t("auth.required")} />;
  }
  if (!canView(admin.permissions, "store_products")) {
    return <Notice text={t("auth.forbidden")} />;
  }
  const [config, history] = await Promise.all([
    adminApi<PricesConfig>("GET", "/admin/config/store-prices"),
    adminApi<{ versions: PricesVersion[] }>("GET", "/admin/config/store-prices/history"),
  ]);
  if (config.status !== 200 || history.status !== 200) {
    return <Notice text={t("store_prices.load_failed")} />;
  }
  const { catalog, rules, config_version: version } = config.body;
  return (
    <main className="mx-auto flex max-w-6xl flex-col gap-8 px-6 py-10">
      <h1 className="text-2xl font-semibold">{t("store_prices.title")}</h1>
      <p className="text-text-secondary">{t("store_prices.intro")}</p>
      {catalog === null && (
        <p role="alert" className="text-danger-text">
          {t("store_prices.empty")}
        </p>
      )}
      {catalog !== null && <StoreExport version={version} targets={rules.targets} />}
      {catalog !== null && <StorePush applyClientId={can(admin.permissions, "store_products", "push_store_prices") ? googleClientId() : null} />}
      <StorePricesEditor
        key={version}
        published={catalog}
        starting={catalog ?? rules.default}
        products={rules.products}
        countries={rules.countries}
        version={version}
        canPublish={can(admin.permissions, "store_products", "publish")}
      />
      <History versions={history.body.versions} rollback={canRollBack(admin.permissions, "store_products") ? googleClientId() : null} t={t} />
    </main>
  );
}

function History({ versions, rollback, t }: { versions: PricesVersion[]; rollback: string | null; t: Translate }) {
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-lg font-semibold">{t("store_prices.history")}</h2>
      {versions.length === 0 ? (
        <p className="text-text-secondary">{t("store_prices.history_empty")}</p>
      ) : (
        <table className="text-sm">
          <thead className="text-left text-text-muted">
            <tr>
              <th className="py-2 pr-4">{t("store_prices.version")}</th>
              <th className="py-2 pr-4">{t("store_prices.published_at")}</th>
              <th className="py-2 pr-4">{t("store_prices.tiers")}</th>
              <th className="py-2 pr-4">{t("store_prices.listed_countries")}</th>
              {rollback && <th className="py-2 pr-4" />}
            </tr>
          </thead>
          <tbody>
            {versions.map((version, index) => (
              <tr key={version.config_version} className="border-t border-surface align-top">
                <td className="py-2 pr-4">{version.config_version}</td>
                <td className="py-2 pr-4 whitespace-nowrap">{t.when(version.published_at)}</td>
                <td className="py-2 pr-4">{Object.entries(version.catalog.tiers).map(([tier, percent]) => `${tier} ${percent}%`).join(", ")}</td>
                <td className="py-2 pr-4">{pricedCountries(version.catalog).length}</td>
                {rollback && <td className="py-2 pr-4">{index > 0 && <Rollback version={version.config_version} clientId={rollback} action={rollbackStorePrices} />}</td>}
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
