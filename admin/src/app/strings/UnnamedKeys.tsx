"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useTranslator } from "@/i18n/client";
import { distinctNames, keyHref } from "@/strings/strings";
import { unapprovedNames } from "./actions";

export function UnnamedKeys({ keys = [] }: { keys?: readonly string[] }) {
  const t = useTranslator();
  return <KeyLinks message={t("strings.unnamed")} keys={keys} />;
}

const separator = "\n";

export function UnnamedKeysAhead({ keys }: { keys: readonly string[] }) {
  const t = useTranslator();
  const names = distinctNames(keys).join(separator);
  const [checked, setChecked] = useState({ names: "", unnamed: [] as string[] });
  useEffect(() => {
    if (names === "") {
      return;
    }
    let current = true;
    unapprovedNames(names.split(separator)).then((unnamed) => {
      if (current) {
        setChecked({ names, unnamed });
      }
    });
    return () => {
      current = false;
    };
  }, [names]);
  return <KeyLinks message={t("strings.unnamed_ahead")} keys={checked.names === names ? checked.unnamed : []} />;
}

function KeyLinks({ message, keys }: { message: string; keys: readonly string[] }) {
  if (keys.length === 0) {
    return null;
  }
  return (
    <div className="flex flex-col gap-1 text-sm text-text-secondary">
      <p>{message}</p>
      <ul className="flex flex-col gap-1">
        {keys.map((key) => (
          <li key={key}>
            <Link href={keyHref(key)} prefetch={false} className="font-mono text-primary underline">
              {key}
            </Link>
          </li>
        ))}
      </ul>
    </div>
  );
}
