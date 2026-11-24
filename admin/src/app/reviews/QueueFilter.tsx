import Link from "next/link";
import type { MessageKey, Translate } from "@/i18n/translate";

type QueueFilterProps = {
  screen: "/reviews" | "/reports" | "/appeals" | "/matches";
  name: "cause" | "reason" | "kind" | "signal";
  label: MessageKey;
  options: readonly string[];
  selected: string;
  account: string;
  t: Translate;
};

export function QueueFilter({ screen, name, label, options, selected, account, t }: QueueFilterProps) {
  return (
    <form method="get" action={screen} className="flex flex-wrap items-end gap-4 text-sm">
      <label className="flex flex-col gap-1 text-text-secondary">
        {t("queues.account")}
        <input name="account" defaultValue={account} className="rounded-md bg-surface px-3 py-2 font-mono text-text-primary" />
      </label>
      <label className="flex flex-col gap-1 text-text-secondary">
        {t(label)}
        <select name={name} defaultValue={selected} className="rounded-md bg-surface px-3 py-2 font-mono text-text-primary">
          <option value="">{t("queues.any")}</option>
          {options.map((option) => (
            <option key={option} value={option}>
              {option}
            </option>
          ))}
        </select>
      </label>
      <button type="submit" className="rounded-md bg-primary px-4 py-2 font-semibold text-on-primary">
        {t("queues.search")}
      </button>
      <Link href={screen} prefetch={false} className="py-2 text-text-secondary">
        {t("queues.clear")}
      </Link>
    </form>
  );
}
