import { cookies, headers } from "next/headers";
import { adminApiHeaders, parseReply } from "./forwarding";

export const sessionCookie = "hd_admin_session";

export type ApiReply<T> = { status: number; body: T };

export type ApiRefusal = { error?: string; detail?: string; keys?: string[] };

export async function sessionToken(): Promise<string | undefined> {
  return (await cookies()).get(sessionCookie)?.value || undefined;
}

export async function hasSession(): Promise<boolean> {
  return Boolean(await sessionToken());
}

export async function adminApi<T>(method: string, path: string, body?: unknown): Promise<ApiReply<T & ApiRefusal>> {
  return callAdminApi<T>(method, path, json(body), await sessionToken());
}

export async function adminApiText<T>(method: string, path: string, text: string): Promise<ApiReply<T & ApiRefusal>> {
  return callAdminApi<T>(method, path, text, await sessionToken());
}

export async function anonymousAdminApi<T>(method: string, path: string, body?: unknown): Promise<ApiReply<T & ApiRefusal>> {
  return callAdminApi<T>(method, path, json(body), undefined);
}

function json(body: unknown): string | undefined {
  return body === undefined ? undefined : JSON.stringify(body);
}

async function callAdminApi<T>(method: string, path: string, payload: string | undefined, token: string | undefined): Promise<ApiReply<T & ApiRefusal>> {
  const response = await fetchAdminApi(method, path, payload, token);
  return { status: response.status, body: parseReply<T & ApiRefusal>(await response.text()) };
}

export async function adminApiResponse(method: string, path: string, body?: unknown): Promise<Response> {
  return fetchAdminApi(method, path, json(body), await sessionToken());
}

async function fetchAdminApi(method: string, path: string, payload: string | undefined, token: string | undefined): Promise<Response> {
  const base = process.env.HD_ADMIN_API_URL;
  if (!base) {
    throw new Error("HD_ADMIN_API_URL is not set");
  }
  return fetch(new URL(path, base), {
    method,
    cache: "no-store",
    headers: adminApiHeaders(await headers(), token),
    body: payload,
  });
}
