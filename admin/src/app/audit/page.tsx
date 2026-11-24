import Link from "next/link";
import { adminApi } from "@/admin/api";
import { googleClientId } from "@/admin/google";
import { can, canView } from "@/admin/navigation";
import { currentAdmin } from "@/admin/session";
import { auditApiPath, beforeIdOf, olderPageHref, readFilter, type AuditEntry, type AuditFilter, type SearchParams } from "@/audit/query";
import { serverTranslator } from "@/i18n/server";
import type { MessageKey, Translate } from "@/i18n/translate";
import { ExportAudit } from "./ExportAudit";

export const dynamic = "force-dynamic";

const textFields: readonly { name: keyof AuditFilter; label: MessageKey }[] = [
  { name: "actor", label: "audit.actor" },
  { name: "object_type", label: "audit.object_type" },
  { name: "object_id", label: "audit.object_id" },
  { name: "action", label: "audit.action" },
  { name: "reason", label: "audit.reason" },
];

const timeFields: readonly { name: keyof AuditFilter; label: MessageKey }[] = [
  { name: "from", label: "audit.from" },
  { name: "to", label: "audit.to" },
];

export default async function AuditPage({ searchParams }: { searchParams: Promise<SearchParams> }) {
  const t = await serverTranslator();
  const admin = await currentAdmin();
  if (!admin) {
    return <Notice text={t("auth.required")} />;
  }
  if (!canView(admin.permissions, "audit_log")) {
    return <Notice text={t("auth.forbidden")} />;
  }
  const params = await searchParams;
  const filter = readFilter(params);
  const reply = await adminApi<{ entries: AuditEntry[] }>("GET", auditApiPath(filter, beforeIdOf(params)));
  const older = reply.status === 200 ? olderPageHref(filter, reply.body.entries) : null;
  return (
    <main className="mx-auto flex max-w-6xl flex-col gap-8 px-6 py-10">
      <h1 className="text-2xl font-semibold">{t("audit.title")}</h1>
      <FilterForm filter={filter} t={t} />
      {can(admin.permissions, "audit_log", "export_audit") && <ExportAudit filter={filter} clientId={googleClientId()} />}
      {reply.status === 400 && <p role="alert" className="text-danger-text">{t("audit.invalid_filter")}</p>}
      {reply.status !== 200 && reply.status !== 400 && <p role="alert" className="text-danger-text">{t("audit.load_failed")}</p>}
      {reply.status === 200 && <Entries entries={reply.body.entries} t={t} />}
      {older && (
        <Link href={older} className="self-start rounded-md bg-surface px-4 py-2 text-text-secondary">
          {t("audit.older")}
        </Link>
      )}
    </main>
  );
}

function FilterForm({ filter, t }: { filter: AuditFilter; t: Translate }) {
  return (
    <form method="get" action="/audit" className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      {textFields.map((field) => (
        <label key={field.name} className="flex flex-col gap-1 text-sm text-text-secondary">
          {t(field.label)}
          <input name={field.name} defaultValue={filter[field.name] ?? ""} className="rounded-md bg-surface px-3 py-2 text-text-primary" />
        </label>
      ))}
      {timeFields.map((field) => (
        <label key={field.name} className="flex flex-col gap-1 text-sm text-text-secondary">
          {t(field.label)}
          <input type="datetime-local" name={field.name} defaultValue={filter[field.name] ?? ""} className="rounded-md bg-surface px-3 py-2 text-text-primary" />
        </label>
      ))}
      <div className="flex items-end gap-4">
        <button type="submit" className="rounded-md bg-primary px-4 py-2 font-semibold text-on-primary">
          {t("audit.search")}
        </button>
        <Link href="/audit" className="py-2 text-text-secondary">
          {t("audit.clear")}
        </Link>
      </div>
    </form>
  );
}

function Entries({ entries, t }: { entries: readonly AuditEntry[]; t: Translate }) {
  if (entries.length === 0) {
    return <p className="text-text-secondary">{t("audit.empty")}</p>;
  }
  return (
    <ol className="flex flex-col gap-3">
      {entries.map((entry) => (
        <li key={entry.id} className="rounded-md bg-background-raised p-4">
          <div className="flex flex-wrap items-baseline gap-x-4 gap-y-1">
            <span className="font-mono text-sm text-text-muted">#{entry.id}</span>
            <time dateTime={entry.at} className="text-sm text-text-secondary">
              {t.when(entry.at)}
            </time>
            <span className="font-semibold">{entry.action}</span>
            <span className="break-all">
              {entry.object_type}
              {entry.object_id && `:${entry.object_id}`}
            </span>
            <span className="text-sm text-text-muted">{entry.environment}</span>
          </div>
          <dl className="mt-2 grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
            <dt className="text-text-muted">{t("audit.actor")}</dt>
            <dd className="break-all">
              {entry.actor_id} ({entry.actor_roles.join(", ")})
            </dd>
            {entry.reason && (
              <>
                <dt className="text-text-muted">{t("audit.reason")}</dt>
                <dd className="break-words">{entry.reason}</dd>
              </>
            )}
            {entry.ref && (
              <>
                <dt className="text-text-muted">{t("audit.ref")}</dt>
                <dd className="break-all">{entry.ref}</dd>
              </>
            )}
            {entry.config_version !== undefined && (
              <>
                <dt className="text-text-muted">{t("audit.config_version")}</dt>
                <dd>{entry.config_version}</dd>
              </>
            )}
            <dt className="text-text-muted">{t("audit.source")}</dt>
            <dd className="break-all">{[entry.source_ip, entry.user_agent, entry.session_id].filter(Boolean).join(" · ")}</dd>
          </dl>
          {(entry.before !== undefined || entry.after !== undefined) && (
            <details className="mt-2">
              <summary className="cursor-pointer text-sm text-text-secondary">{t("audit.changes")}</summary>
              <div className="mt-2 grid gap-2 lg:grid-cols-2">
                <Snapshot label={t("audit.before")} value={entry.before} />
                <Snapshot label={t("audit.after")} value={entry.after} />
              </div>
            </details>
          )}
        </li>
      ))}
    </ol>
  );
}

function Snapshot({ label, value }: { label: string; value: unknown }) {
  return (
    <figure className="min-w-0">
      <figcaption className="text-sm text-text-muted">{label}</figcaption>
      <pre className="overflow-x-auto rounded-md bg-surface p-3 text-xs">{value === undefined ? "-" : JSON.stringify(value, null, 2)}</pre>
    </figure>
  );
}

function Notice({ text }: { text: string }) {
  return (
    <main className="mx-auto max-w-3xl px-6 py-24">
      <p className="text-text-secondary">{text}</p>
    </main>
  );
}
