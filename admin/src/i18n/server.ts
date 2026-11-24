import { cookies } from "next/headers";
import { currentAdmin } from "@/admin/session";
import { localeCookie, preferredLocale, type Locale } from "./locales";
import { translator } from "./translate";

export async function currentLocale(): Promise<Locale> {
  const admin = await currentAdmin();
  return preferredLocale(admin?.account.locale, (await cookies()).get(localeCookie)?.value);
}

export async function serverTranslator() {
  return translator(await currentLocale());
}
