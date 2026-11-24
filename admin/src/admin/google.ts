export function googleClientId(): string {
  const clientId = process.env.HD_ADMIN_GOOGLE_CLIENT_ID;
  if (!clientId) {
    throw new Error("HD_ADMIN_GOOGLE_CLIENT_ID is not set");
  }
  return clientId;
}
