"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import { adminRoles, grantsSuperAdmin, roleLabels, sameRoles, toggledRole } from "@/accounts/roles";
import { needsStepUp } from "@/admin/refusals";
import { StepUpPrompt } from "@/app/auth/StepUpPrompt";
import { useTranslator } from "@/i18n/client";
import { createAccount, disableAccount, setAccountRoles, type AccountNote, type AccountResult } from "./actions";

const fieldClass = "rounded-md bg-surface px-3 py-1 text-text-primary";

function useAccountAction() {
  const router = useRouter();
  const [result, setResult] = useState<AccountResult | null>(null);
  const [last, setLast] = useState<(() => Promise<AccountResult>) | null>(null);
  const [pending, startTransition] = useTransition();

  function run(action: () => Promise<AccountResult>, onDone?: () => void) {
    setLast(() => action);
    startTransition(async () => {
      const outcome = await action();
      setResult(outcome);
      if (outcome.ok) {
        onDone?.();
        router.refresh();
      }
    });
  }

  return { result, pending, run, retry: () => last && run(last) };
}

function RolePicker({ roles, onChange, allowSuperAdmin }: { roles: string[]; onChange: (roles: string[]) => void; allowSuperAdmin: boolean }) {
  const t = useTranslator();
  return (
    <fieldset className="flex flex-wrap gap-3 text-sm">
      <legend className="sr-only">{t("accounts.roles")}</legend>
      {adminRoles.map((role) => (
        <label key={role} className="flex items-center gap-1">
          <input
            type="checkbox"
            disabled={role === "super_admin" && !allowSuperAdmin && !roles.includes(role)}
            checked={roles.includes(role)}
            onChange={(event) => onChange(toggledRole(roles, role, event.target.checked))}
          />
          {t(roleLabels[role])}
        </label>
      ))}
    </fieldset>
  );
}

function NoteFields({ note, onChange }: { note: AccountNote; onChange: (note: AccountNote) => void }) {
  const t = useTranslator();
  return (
    <div className="flex flex-wrap gap-2">
      <input aria-label={t("accounts.reason")} placeholder={t("accounts.reason")} value={note.reason} onChange={(event) => onChange({ ...note, reason: event.target.value })} className={`grow ${fieldClass}`} />
      <input aria-label={t("accounts.ref")} placeholder={t("accounts.ref")} value={note.ref} onChange={(event) => onChange({ ...note, ref: event.target.value })} className={`w-40 ${fieldClass}`} />
    </div>
  );
}

function Outcome({ result, clientId, onStepUp }: { result: AccountResult | null; clientId: string; onStepUp: () => void }) {
  const t = useTranslator();
  if (result === null) {
    return null;
  }
  if (!result.ok && needsStepUp(result.error)) {
    return <StepUpPrompt clientId={clientId} onConfirmed={onStepUp} />;
  }
  if (!result.ok) {
    return (
      <p role="status" className="text-sm text-text-secondary">
        {t("accounts.refused")}: {result.error} {result.detail}
      </p>
    );
  }
  return (
    <p role="status" className="text-sm text-text-secondary">
      {result.approvalId === null ? t("accounts.saved") : `${t("accounts.proposed")}: ${result.approvalId}`}
    </p>
  );
}

const emptyNote: AccountNote = { reason: "", ref: "" };

export function CreateAccount({ clientId }: { clientId: string }) {
  const t = useTranslator();
  const [email, setEmail] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [roles, setRoles] = useState<string[]>([]);
  const [note, setNote] = useState(emptyNote);
  const { result, pending, run, retry } = useAccountAction();

  function send() {
    run(
      () => createAccount(email.trim(), displayName.trim(), roles, note),
      () => {
        setEmail("");
        setDisplayName("");
        setRoles([]);
        setNote(emptyNote);
      },
    );
  }

  return (
    <section className="flex flex-col gap-3 rounded-md border border-surface p-4">
      <h2 className="font-semibold">{t("accounts.create")}</h2>
      <div className="flex flex-wrap gap-2">
        <input type="email" aria-label={t("accounts.email")} placeholder={t("accounts.email")} value={email} onChange={(event) => setEmail(event.target.value)} className={`grow ${fieldClass}`} />
        <input aria-label={t("accounts.display_name")} placeholder={t("accounts.display_name")} value={displayName} onChange={(event) => setDisplayName(event.target.value)} className={`grow ${fieldClass}`} />
      </div>
      <RolePicker roles={roles} onChange={setRoles} allowSuperAdmin={false} />
      <p className="text-sm text-text-muted">{t("accounts.super_admin_hint")}</p>
      <NoteFields note={note} onChange={setNote} />
      <button
        type="button"
        disabled={pending || email.trim() === "" || displayName.trim() === "" || roles.length === 0}
        onClick={send}
        className="self-start rounded-md bg-primary px-4 py-2 text-on-primary disabled:opacity-40"
      >
        {t("accounts.create")}
      </button>
      <Outcome result={result} clientId={clientId} onStepUp={retry} />
    </section>
  );
}

export function AccountActions({ id, held, canEdit, canDisable, canGrantSuperAdmin, clientId }: { id: string; held: string[]; canEdit: boolean; canDisable: boolean; canGrantSuperAdmin: boolean; clientId: string }) {
  const t = useTranslator();
  const [roles, setRoles] = useState(held);
  const [note, setNote] = useState(emptyNote);
  const { result, pending, run, retry } = useAccountAction();
  const grant = grantsSuperAdmin(held, roles);
  const noteMissing = note.reason.trim() === "";

  return (
    <div className="flex flex-col gap-2">
      {canEdit && <RolePicker roles={roles} onChange={setRoles} allowSuperAdmin={canGrantSuperAdmin} />}
      {grant && <p className="text-sm text-text-secondary">{t("accounts.grant_needs_approval")}</p>}
      <NoteFields note={note} onChange={setNote} />
      <div className="flex flex-wrap gap-2">
        {canEdit && (
          <button
            type="button"
            disabled={pending || roles.length === 0 || sameRoles(roles, held) || (grant && noteMissing)}
            onClick={() => run(() => setAccountRoles(id, roles, note))}
            className="rounded-md bg-primary px-3 py-1 text-on-primary disabled:opacity-40"
          >
            {grant ? t("accounts.propose_roles") : t("accounts.save_roles")}
          </button>
        )}
        {canDisable && (
          <button type="button" disabled={pending || noteMissing} onClick={() => run(() => disableAccount(id, note))} className="rounded-md bg-surface px-3 py-1 text-danger-text disabled:opacity-40">
            {t("accounts.disable")}
          </button>
        )}
      </div>
      <Outcome result={result} clientId={clientId} onStepUp={retry} />
    </div>
  );
}
