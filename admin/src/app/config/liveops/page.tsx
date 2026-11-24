import { KeyedConfigPage } from "../KeyedConfigPage";
import { publishLiveops, rollbackLiveops } from "./actions";

export const dynamic = "force-dynamic";

export default function LiveopsPage() {
  return (
    <KeyedConfigPage
      path="/admin/config/liveops"
      text={{ title: "liveops.title", loadFailed: "liveops.load_failed", intro: "liveops.intro", historyEmpty: "liveops.history_empty" }}
      publish={publishLiveops}
      rollback={rollbackLiveops}
    />
  );
}
