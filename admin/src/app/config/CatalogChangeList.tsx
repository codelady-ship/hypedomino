import { changeText, type CatalogChanges, type EntryChange } from "@/config/changes";
import type { Translate } from "@/i18n/translate";

export function CatalogChangeList({ changes, t }: { changes: CatalogChanges; t: Translate }) {
  if (changes.settings.length === 0 && changes.entries.length === 0) {
    return <>{t("config.no_changes")}</>;
  }
  return (
    <ul className="flex flex-col gap-1">
      {changes.settings.map((change) => (
        <li key={change.field}>{changeText(change)}</li>
      ))}
      {changes.entries.map((entry) => (
        <li key={entry.id}>
          {entry.id}: <EntryText entry={entry} t={t} />
        </li>
      ))}
    </ul>
  );
}

function EntryText({ entry, t }: { entry: EntryChange; t: Translate }) {
  if (entry.change === "added") {
    return <>{t("config.added")}</>;
  }
  if (entry.change === "removed") {
    return <>{t("config.removed")}</>;
  }
  return <>{entry.fields.map(changeText).join(", ")}</>;
}
