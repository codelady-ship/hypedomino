"use client";

import Link from "next/link";
import { itemAssetsHref, itemsWithoutArt } from "@/assets/assets";
import { useTranslator } from "@/i18n/client";

export function ItemsWithoutArt({ error, detail }: { error: string; detail: string }) {
  const t = useTranslator();
  const ids = itemsWithoutArt(error, detail);
  if (ids.length === 0) {
    return null;
  }
  return (
    <div className="flex flex-col gap-1 text-sm text-text-secondary">
      <p>{t("assets.missing_art")}</p>
      <ul className="flex flex-col gap-1">
        {ids.map((id) => (
          <li key={id}>
            <Link href={itemAssetsHref(id)} prefetch={false} className="font-mono text-primary underline">
              {id}
            </Link>
          </li>
        ))}
      </ul>
    </div>
  );
}
