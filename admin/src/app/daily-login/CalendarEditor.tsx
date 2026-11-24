"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import type { ConfigResult } from "@/config/result";
import { calendarIssues, cycleCoins, sameCalendar, withDayCoins, type Calendar } from "@/dailylogin/calendar";
import { useTranslator } from "@/i18n/client";
import { publishCalendar } from "./actions";

const fieldClass = "w-24 rounded-md bg-surface px-3 py-1 text-text-primary disabled:opacity-50";
const labelClass = "flex flex-col gap-1 text-sm text-text-secondary";

type Props = { published: Calendar | null; starting: Calendar; version: number; canPublish: boolean };

export function CalendarEditor({ published, starting, version, canPublish }: Props) {
  const t = useTranslator();
  const router = useRouter();
  const [calendar, setCalendar] = useState(starting);
  const [reason, setReason] = useState("");
  const [result, setResult] = useState<ConfigResult | null>(null);
  const [pending, startTransition] = useTransition();
  const issues = calendarIssues(calendar);
  const unchanged = published !== null && sameCalendar(published, calendar);
  const ready = !pending && issues.length === 0 && !unchanged && reason.trim() !== "";
  return (
    <section className="flex flex-col gap-4">
      <p className="text-sm text-text-muted">
        {t("daily_login.version")}: {version} · {t("daily_login.cycle_coins")}: {issues.length === 0 ? cycleCoins(calendar) : "—"}
      </p>
      <div className="flex flex-wrap gap-3">
        {calendar.days.map((day, index) => (
          <label key={index} className={labelClass}>
            {`${t("daily_login.day")} ${index + 1}`}
            <input
              type="number"
              min={1}
              disabled={!canPublish}
              value={Number.isNaN(day.coins) ? "" : day.coins}
              onChange={(event) => setCalendar(withDayCoins(calendar, index, event.target.value === "" ? Number.NaN : Number(event.target.value)))}
              className={fieldClass}
            />
          </label>
        ))}
      </div>
      {issues.length > 0 && (
        <ul role="alert" className="flex flex-col gap-1 text-sm text-danger-text">
          {issues.map((issue) => (
            <li key={issue}>{t(issue)}</li>
          ))}
        </ul>
      )}
      {canPublish && (
        <div className="flex flex-col gap-3">
          <label className={labelClass}>
            {t("daily_login.reason")}
            <input value={reason} onChange={(event) => setReason(event.target.value)} className="rounded-md bg-surface px-3 py-1 text-text-primary" />
          </label>
          <button
            type="button"
            disabled={!ready}
            onClick={() =>
              startTransition(async () => {
                const outcome = await publishCalendar(calendar, reason);
                setResult(outcome);
                if (outcome.ok) {
                  router.refresh();
                }
              })
            }
            className="self-start rounded-md bg-primary px-4 py-2 text-on-primary disabled:opacity-40"
          >
            {t("daily_login.publish")}
          </button>
          {result && !result.ok && (
            <p role="status" className="text-sm text-text-secondary">
              {t("daily_login.refused")}: {result.error} {result.detail}
            </p>
          )}
        </div>
      )}
    </section>
  );
}
