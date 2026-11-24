import { KeyedConfigPage } from "../KeyedConfigPage";
import { publishRules, rollbackRules } from "./actions";

export const dynamic = "force-dynamic";

export default function RulesPage() {
  return (
    <KeyedConfigPage
      path="/admin/config/rules"
      text={{ title: "rules.title", loadFailed: "rules.load_failed", intro: "rules.applies_to_new_matches", historyEmpty: "rules.history_empty" }}
      publish={publishRules}
      rollback={rollbackRules}
    />
  );
}
