"use client";

import { useTranslator } from "@/i18n/client";
import { csvFileName, storeCsv, stores, type Store, type Target } from "@/storeprices/prices";

function download(store: Store, version: number, targets: Target[]) {
  const link = document.createElement("a");
  link.href = URL.createObjectURL(new Blob([storeCsv(targets)], { type: "text/csv" }));
  link.download = csvFileName(store, version);
  link.click();
  URL.revokeObjectURL(link.href);
}

export function StoreExport({ version, targets }: { version: number; targets: Target[] }) {
  const t = useTranslator();
  return (
    <section className="flex flex-col gap-2">
      <h2 className="text-lg font-semibold">{t("store_prices.export")}</h2>
      <p className="text-sm text-text-muted">{t("store_prices.export_hint")}</p>
      <div className="flex flex-wrap gap-3">
        {stores.map((store) => (
          <button key={store} type="button" onClick={() => download(store, version, targets)} className="rounded-md bg-surface px-3 py-1 text-text-secondary">
            {t(`store_prices.store.${store}`)}
          </button>
        ))}
      </div>
    </section>
  );
}
