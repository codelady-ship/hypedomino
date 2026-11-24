import type { Translate } from "@/i18n/translate";
import type { QueueDeadline } from "@/reviews/reviews";

export function QueueDue({ deadline, t }: { deadline: QueueDeadline; t: Translate }) {
  return (
    <span className={deadline.overdue ? "font-semibold text-danger-text" : undefined}>
      {t("queues.due_at")}: {t.when(deadline.due_at)}
      {deadline.overdue && ` (${t("queues.overdue")})`}
    </span>
  );
}
