"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { itemAssetsHref, type StaticArt } from "@/assets/assets";
import { useTranslator } from "@/i18n/client";
import { staticArt } from "./actions";

export function StaticPreview({ itemId }: { itemId: string }) {
  const t = useTranslator();
  const [loaded, setLoaded] = useState<{ itemId: string; art: StaticArt } | null>(null);
  useEffect(() => {
    let current = true;
    staticArt(itemId).then((art) => {
      if (current) {
        setLoaded({ itemId, art });
      }
    });
    return () => {
      current = false;
    };
  }, [itemId]);
  const art = loaded?.itemId === itemId ? loaded.art : null;
  if (art?.state === "unavailable") {
    return null;
  }
  return (
    <section className="flex flex-col gap-2 rounded-md border border-surface p-3 text-sm">
      <h3 className="text-text-secondary">{t("cosmetics.preview")}</h3>
      {art === null && <p className="text-text-muted">{t("cosmetics.preview_loading")}</p>}
      {art?.state === "passed" && (
        <div className="flex flex-wrap items-end gap-4">
          <img src={art.url} alt={itemId} className="h-32 w-32 rounded-md bg-surface object-contain" />
          <div className="flex flex-col gap-1 text-text-muted">
            <p>{t("cosmetics.preview_static")}</p>
            {art.width !== null && art.height !== null && (
              <p>
                {art.width}×{art.height}
              </p>
            )}
            <p>{t.when(art.uploadedAt)}</p>
          </div>
        </div>
      )}
      {art?.state === "missing" && <p className="text-danger-text">{t("cosmetics.preview_missing")}</p>}
      {art !== null && <p className="text-text-muted">{t("cosmetics.preview_not_built")}</p>}
      <Link href={itemAssetsHref(itemId)} prefetch={false} className="self-start text-primary underline">
        {t("cosmetics.preview_assets")}
      </Link>
    </section>
  );
}
