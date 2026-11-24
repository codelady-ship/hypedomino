import Link from "next/link";
import { adminApi } from "@/admin/api";
import { canView } from "@/admin/navigation";
import { currentAdmin } from "@/admin/session";
import { calendarKinds, running, type CalendarView } from "@/calendar/calendar";
import { serverTranslator } from "@/i18n/server";

export const dynamic = "force-dynamic";

export default async function CalendarPage() {
  const t = await serverTranslator();
  const admin = await currentAdmin();
  if (!admin) {
    return <Notice text={t("auth.required")} />;
  }
  if (!canView(admin.permissions, "live_events")) {
    return <Notice text={t("auth.forbidden")} />;
  }
  const calendar = await adminApi<CalendarView>("GET", "/admin/calendar");
  if (calendar.status !== 200) {
    return <Notice text={t("calendar.load_failed")} />;
  }
  const now = new Date(calendar.body.server_time);
  return (
    <main className="mx-auto flex max-w-4xl flex-col gap-8 px-6 py-10">
      <div className="flex flex-col gap-2">
        <h1 className="text-2xl font-semibold">{t("calendar.title")}</h1>
        <p className="text-sm text-text-secondary">{t("calendar.explain")}</p>
      </div>
      {calendar.body.entries.length === 0 ? (
        <p className="text-sm text-text-secondary">{t("calendar.empty")}</p>
      ) : (
        <table className="w-full text-left text-sm">
          <thead className="text-text-muted">
            <tr>
              <th className="py-2">{t("calendar.start")}</th>
              <th className="py-2">{t("calendar.end")}</th>
              <th className="py-2">{t("calendar.kind")}</th>
              <th className="py-2">{t("calendar.id")}</th>
            </tr>
          </thead>
          <tbody>
            {calendar.body.entries.map((entry) => (
              <tr key={`${entry.kind}:${entry.id}:${entry.start}`} className={running(entry, now) ? "font-semibold" : "text-text-secondary"}>
                <td className="py-1">{t.when(entry.start)}</td>
                <td className="py-1">{entry.end ? t.when(entry.end) : t("calendar.open_ended")}</td>
                <td className="py-1">{t(calendarKinds[entry.kind].labelKey)}</td>
                <td className="py-1 font-mono">
                  <Link href={calendarKinds[entry.kind].href} className="underline">
                    {entry.id}
                  </Link>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </main>
  );
}

function Notice({ text }: { text: string }) {
  return (
    <main className="mx-auto max-w-3xl px-6 py-24">
      <p className="text-text-secondary">{text}</p>
    </main>
  );
}
