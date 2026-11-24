"use client";

import { useEffect, useRef, useState } from "react";
import { useTranslator } from "@/i18n/client";
import type { MessageKey } from "@/i18n/translate";
import { issueNonce } from "./actions";

type GoogleIdentity = {
  accounts: {
    id: {
      initialize(options: { client_id: string; nonce: string; ux_mode: "popup"; callback: (response: { credential: string }) => void }): void;
      renderButton(parent: HTMLElement, options: Record<string, string>): void;
    };
  };
};

declare global {
  interface Window {
    google?: GoogleIdentity;
  }
}

const googleIdentityScript = "https://accounts.google.com/gsi/client";

function documentNonce(): string {
  return document.querySelector<HTMLScriptElement>("script[nonce]")?.nonce ?? "";
}

function loadGoogleIdentity(): Promise<void> {
  if (window.google?.accounts) {
    return Promise.resolve();
  }
  const script =
    document.querySelector<HTMLScriptElement>(`script[src="${googleIdentityScript}"]`) ??
    document.head.appendChild(Object.assign(document.createElement("script"), { src: googleIdentityScript, async: true, nonce: documentNonce() }));
  return new Promise((resolve, reject) => {
    script.addEventListener("load", () => resolve(), { once: true });
    script.addEventListener("error", () => reject(new Error("Google Identity Services did not load")), { once: true });
  });
}

export type CredentialHandler = (credential: string, nonce: string) => Promise<MessageKey | null>;

export function GoogleButton({ clientId, onCredential }: { clientId: string; onCredential: CredentialHandler }) {
  const t = useTranslator();
  const button = useRef<HTMLDivElement>(null);
  const [problem, setProblem] = useState<MessageKey | null>(null);

  async function prepare() {
    const nonce = await issueNonce();
    const identity = window.google?.accounts.id;
    if (!nonce || !identity || !button.current) {
      setProblem("signin.unavailable");
      return;
    }
    identity.initialize({
      client_id: clientId,
      nonce,
      ux_mode: "popup",
      callback: async ({ credential }) => {
        const refusal = await onCredential(credential, nonce);
        setProblem(refusal);
        if (refusal) {
          await prepare();
        }
      },
    });
    identity.renderButton(button.current, { type: "standard", theme: "filled_black", size: "large", text: "signin_with" });
  }

  useEffect(() => {
    loadGoogleIdentity().then(prepare, () => setProblem("signin.unavailable"));
  }, []);

  return (
    <div className="flex flex-col items-start gap-4">
      <div ref={button} />
      {problem && (
        <p role="alert" className="text-danger-text">
          {t(problem)}
        </p>
      )}
    </div>
  );
}
