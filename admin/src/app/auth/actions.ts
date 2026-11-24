"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { adminApi, anonymousAdminApi, sessionCookie } from "@/admin/api";
import type { AdminSession } from "@/admin/session";

export type SignInRefusal = { error: string };

export async function issueNonce(): Promise<string | null> {
  const reply = await anonymousAdminApi<{ nonce: string }>("POST", "/admin/nonce");
  return reply.status === 200 ? reply.body.nonce : null;
}

export async function completeSignIn(idToken: string, nonce: string): Promise<SignInRefusal> {
  const reply = await anonymousAdminApi<{ token: string; session: AdminSession }>("POST", "/admin/session", { id_token: idToken, nonce });
  if (reply.status !== 200) {
    return { error: reply.body.error ?? String(reply.status) };
  }
  (await cookies()).set(sessionCookie, reply.body.token, {
    httpOnly: true,
    sameSite: "strict",
    secure: process.env.NODE_ENV === "production",
    path: "/",
    expires: new Date(reply.body.session.expires_at),
  });
  redirect("/");
}

export async function completeStepUp(idToken: string, nonce: string): Promise<SignInRefusal> {
  const reply = await adminApi<{ step_up_until: string }>("POST", "/admin/session/step-up", { id_token: idToken, nonce });
  return { error: reply.status === 200 ? "" : (reply.body.error ?? String(reply.status)) };
}

export async function signOut(): Promise<void> {
  await adminApi("DELETE", "/admin/session");
  (await cookies()).delete(sessionCookie);
  redirect("/");
}
