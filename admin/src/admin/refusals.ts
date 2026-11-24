import type { MessageKey } from "@/i18n/translate";

const signInRefusals: Readonly<Record<string, MessageKey>> = {
  account_refused: "signin.account_refused",
  not_company_account: "signin.not_company_account",
  token_expired: "signin.retry",
  nonce_rejected: "signin.retry",
  provider_unavailable: "signin.unavailable",
  rate_limited: "signin.rate_limited",
};

export function signInRefusalKey(error: string): MessageKey {
  return signInRefusals[error] ?? "signin.failed";
}

export function stepUpRefusalKey(error: string): MessageKey | null {
  if (!error) {
    return null;
  }
  return error === "different_account" ? "stepup.different_account" : signInRefusalKey(error);
}

export function needsStepUp(error: string | undefined): boolean {
  return error === "step_up_required";
}
