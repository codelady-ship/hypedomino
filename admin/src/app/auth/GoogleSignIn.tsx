"use client";

import { signInRefusalKey } from "@/admin/refusals";
import { completeSignIn } from "./actions";
import { GoogleButton } from "./GoogleButton";

export function GoogleSignIn({ clientId }: { clientId: string }) {
  return <GoogleButton clientId={clientId} onCredential={async (credential, nonce) => signInRefusalKey((await completeSignIn(credential, nonce)).error)} />;
}
