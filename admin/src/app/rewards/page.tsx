import Link from "next/link";
import { adminApi } from "@/admin/api";
import { can, canView } from "@/admin/navigation";
import { currentAdmin } from "@/admin/session";
import type { SearchParams } from "@/audit/query";
import { serverTranslator } from "@/i18n/server";
import type { MissionCatalog } from "@/missions/catalog";
import { nextRewardsHref, readBundleQuery, rewardBundlesApiPath, type NameKeysByDomain, type RewardBundle } from "@/rewards/bundles";
import type { ShopCatalog } from "@/shop/catalog";
import { BundleList } from "./BundleList";
import { NewBundle } from "./NewBundle";

export const dynamic = "force-dynamic";

export default async function RewardsPage({ searchParams }: { searchParams: Promise<SearchParams> }) {
  const t = await serverTranslator();
  const admin = await currentAdmin();
  if (!admin) {
    return <Notice text={t("auth.required")} />;
  }
  if (!canView(admin.permissions, "reward_bundles")) {
    return <Notice text={t("auth.forbidden")} />;
  }
  const query = readBundleQuery(await searchParams);
  const canEdit = can(admin.permissions, "reward_bundles", "publish");
  const [reply, nameKeys] = await Promise.all([adminApi<{ bundles: RewardBundle[] }>("GET", rewardBundlesApiPath(query)), canEdit ? loadRepublishedNameKeys() : {}]);
  if (reply.status !== 200) {
    return <Notice text={t("rewards.load_failed")} />;
  }
  const next = nextRewardsHref(query, reply.body.bundles.map((bundle) => bundle.id));
  return (
    <main className="mx-auto flex max-w-6xl flex-col gap-6 px-6 py-10">
      <h1 className="text-2xl font-semibold">{t("rewards.title")}</h1>
      <p className="text-text-secondary">{t("rewards.intro")}</p>
      {can(admin.permissions, "reward_bundles", "create") && <NewBundle />}
      <BundleList bundles={reply.body.bundles} query={query} canEdit={canEdit} nameKeys={nameKeys} />
      {next && (
        <Link href={next} prefetch={false} className="self-start rounded-md bg-surface px-4 py-2 text-text-secondary">
          {t("queues.next")}
        </Link>
      )}
    </main>
  );
}

async function loadRepublishedNameKeys(): Promise<NameKeysByDomain> {
  const [missions, shop] = await Promise.all([
    adminApi<{ catalog: MissionCatalog | null }>("GET", "/admin/config/missions"),
    adminApi<{ catalog: ShopCatalog | null }>("GET", "/admin/config/shop"),
  ]);
  return {
    missions: missions.status === 200 ? (missions.body.catalog?.missions ?? []).map((mission) => mission.nameKey) : [],
    shop: shop.status === 200 ? (shop.body.catalog?.offers ?? []).map((offer) => offer.nameKey) : [],
  };
}

function Notice({ text }: { text: string }) {
  return (
    <main className="mx-auto max-w-3xl px-6 py-24">
      <p className="text-text-secondary">{text}</p>
    </main>
  );
}
