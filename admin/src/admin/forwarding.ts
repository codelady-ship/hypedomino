export type IncomingHeaders = { get(name: string): string | null };

export function adminApiHeaders(incoming: IncomingHeaders, token: string | undefined): Record<string, string> {
  const outgoing: Record<string, string> = { "Content-Type": "application/json" };
  if (token) {
    outgoing.Authorization = `Bearer ${token}`;
  }
  const forwardedFor = incoming.get("x-forwarded-for");
  if (forwardedFor) {
    outgoing["X-Forwarded-For"] = forwardedFor;
  }
  const userAgent = incoming.get("user-agent");
  if (userAgent) {
    outgoing["User-Agent"] = userAgent;
  }
  return outgoing;
}

export function parseReply<T>(text: string): T {
  return (text ? JSON.parse(text) : {}) as T;
}
