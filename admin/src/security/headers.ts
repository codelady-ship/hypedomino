const googleSignIn = "https://accounts.google.com/gsi/";

export function newNonce(): string {
  return Buffer.from(crypto.getRandomValues(new Uint8Array(16))).toString("base64");
}

export function assetStoreOrigin(configured: string | undefined): string {
  try {
    return new URL(configured ?? "").origin;
  } catch {
    return "";
  }
}

export function contentSecurityPolicy(nonce: string, development: boolean, assetOrigin = ""): string {
  const asset = assetOrigin === "" ? "" : ` ${assetOrigin}`;
  return [
    "default-src 'self'",
    `script-src 'self' 'nonce-${nonce}' 'strict-dynamic'${development ? " 'unsafe-eval'" : ""}`,
    `style-src 'self' 'nonce-${nonce}' ${googleSignIn}style`,
    `img-src 'self' data:${asset}`,
    "font-src 'self'",
    `connect-src 'self' ${googleSignIn}${asset}`,
    `frame-src ${googleSignIn}`,
    "object-src 'none'",
    "base-uri 'none'",
    "form-action 'self'",
    "frame-ancestors 'none'",
  ].join("; ");
}

export function staticSecurityHeaders(production: boolean): { key: string; value: string }[] {
  const headers = [
    { key: "X-Content-Type-Options", value: "nosniff" },
    { key: "Referrer-Policy", value: "no-referrer" },
    { key: "X-Frame-Options", value: "DENY" },
    { key: "Cross-Origin-Opener-Policy", value: "same-origin-allow-popups" },
    { key: "Permissions-Policy", value: "camera=(), microphone=(), geolocation=(), payment=(), usb=()" },
  ];
  if (production) {
    headers.push({ key: "Strict-Transport-Security", value: "max-age=63072000; includeSubDomains" });
  }
  return headers;
}
