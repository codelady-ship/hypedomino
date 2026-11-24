import type { Translate } from "@/i18n/translate";
import { decidedReports, type EarlierDecision } from "@/reviews/reviews";

export function EarlierDecisions({ decisions, t }: { decisions: EarlierDecision[]; t: Translate }) {
  if (decisions.length === 0) {
    return <p className="text-text-secondary">{t("reports.no_decisions")}</p>;
  }
  const totals = decidedReports(decisions);
  return (
    <div className="flex flex-col gap-1">
      <div className="flex flex-wrap gap-x-6 gap-y-1">
        <span className="font-medium">{t("reports.decisions")}</span>
        <span>
          {t("reports.upheld_total")}: {totals.upheld}
        </span>
        <span>
          {t("reports.rejected_total")}: {totals.rejected}
        </span>
      </div>
      <table>
        <thead className="text-left text-text-muted">
          <tr>
            <th className="py-1 pr-4">{t("reports.decision_queue")}</th>
            <th className="py-1 pr-4">{t("reports.decision_outcome")}</th>
            <th className="py-1 pr-4">{t("reports.decision_reasons")}</th>
            <th className="py-1 pr-4">{t("reports.decision_reports")}</th>
            <th className="py-1 pr-4">{t("reports.decided_at")}</th>
          </tr>
        </thead>
        <tbody>
          {decisions.map((decision) => (
            <tr key={`${decision.queue}-${decision.outcome}-${decision.decided_at}`} className="border-t border-surface align-top">
              <td className="py-1 pr-4 font-mono">{decision.queue}</td>
              <td className="py-1 pr-4 font-mono">{decision.outcome}</td>
              <td className="py-1 pr-4 font-mono">{decision.reasons.join(", ")}</td>
              <td className="py-1 pr-4">{decision.queue === "reports" ? decision.reports : "-"}</td>
              <td className="py-1 pr-4 whitespace-nowrap">{t.when(decision.decided_at)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
