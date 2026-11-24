import { cache } from "react";
import { adminApi, hasSession } from "./api";
import type { Permissions } from "./navigation";

export type AdminAccount = {
  id: string;
  email: string;
  display_name: string;
  roles: string[];
  state: string;
  locale?: string;
};

export type AdminSession = {
  id: string;
  authenticated_at: string;
  step_up_until: string;
  idle_until: string;
  expires_at: string;
};

export type SignedInAdmin = {
  account: AdminAccount;
  session: AdminSession;
  permissions: Permissions;
};

export const currentAdmin = cache(async (): Promise<SignedInAdmin | null> => {
  if (!(await hasSession())) {
    return null;
  }
  const reply = await adminApi<SignedInAdmin>("GET", "/admin/me");
  return reply.status === 200 ? reply.body : null;
});
