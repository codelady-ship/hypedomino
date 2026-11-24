import type { SearchParams } from "../audit/query.ts";

export type Status = "missing" | "draft" | "approved";

export type Placeholder = { name: string; type: string };

export type StringValue = { text: string; status: Status };

export type GameString = {
  key: string;
  note: string;
  max_length: number;
  placeholders: Placeholder[];
  deprecated: boolean;
  values: Partial<Record<string, StringValue>>;
};

export type Coverage = { language: string; keys: number; approved: number; percent: number };

export type Bundle = { language: string; version: number; hash: string; by: string; at: string };

export type Problem = { key: string; error: string };

export const editableStatuses = ["draft", "approved"] as const;

export const transferFormats = ["csv", "json"] as const;

export const stringsPageSize = 50;

export type StringsQuery = { q: string; missing: string; after: string; bundles: string; before: string };

const versionPattern = /^[1-9][0-9]{0,8}$/;

function single(value: string | string[] | undefined): string {
  return (Array.isArray(value) ? value[0] : value)?.trim() ?? "";
}

export function readStringsQuery(params: SearchParams, languages: readonly string[]): StringsQuery {
  const missing = single(params.missing);
  const bundles = single(params.bundles);
  const before = single(params.before);
  return {
    q: single(params.q),
    missing: languages.includes(missing) ? missing : "",
    after: single(params.after),
    bundles: languages.includes(bundles) ? bundles : (languages[0] ?? ""),
    before: versionPattern.test(before) ? before : "",
  };
}

function searchOf(fields: Record<string, string>): URLSearchParams {
  return new URLSearchParams(Object.entries(fields).filter(([, value]) => value));
}

export function stringsApiPath(query: StringsQuery): string {
  return `/admin/strings?${searchOf({ q: query.q, missing: query.missing, after: query.after, limit: String(stringsPageSize) })}`;
}

export function bundlesApiPath(query: StringsQuery): string {
  return `/admin/strings/bundles?${searchOf({ language: query.bundles, before: query.before, limit: String(stringsPageSize) })}`;
}

export function stringsHref(query: StringsQuery, change: Partial<StringsQuery>): string {
  return `/strings?${searchOf({ ...query, ...change })}`;
}

export function distinctNames(keys: readonly string[]): string[] {
  return [...new Set(keys.map((key) => key.trim()).filter((key) => key !== ""))];
}

export function keyHref(key: string): string {
  return `/strings?${searchOf({ q: key })}`;
}

export function nextStringsHref(query: StringsQuery, keys: readonly string[]): string | null {
  return keys.length < stringsPageSize ? null : stringsHref(query, { after: keys[keys.length - 1] });
}

export function olderBundlesHref(query: StringsQuery, versions: readonly number[]): string | null {
  return versions.length < stringsPageSize ? null : stringsHref(query, { before: String(versions[versions.length - 1]) });
}

export function valueOf(entry: GameString, language: string): StringValue {
  return entry.values[language] ?? { text: "", status: "missing" };
}

export function placeholdersFromText(text: string): Placeholder[] {
  return text
    .split(",")
    .map((part) => part.trim())
    .filter((part) => part !== "")
    .map((part) => {
      const [name, type = ""] = part.split(":").map((piece) => piece.trim());
      return { name, type };
    });
}

export function placeholdersText(placeholders: readonly Placeholder[]): string {
  return placeholders.map((placeholder) => `${placeholder.name}:${placeholder.type}`).join(", ");
}

export function problemsText(problems: readonly Problem[]): string {
  return problems.map((problem) => `${problem.key}: ${problem.error}`).join("; ");
}

export type ChainError = "start" | "end" | "unknown" | "repeat";

const finalFallback = "en";

export function chainFromText(text: string): string[] {
  return text
    .split(",")
    .map((part) => part.trim())
    .filter((part) => part !== "");
}

export function chainError(language: string, chain: readonly string[], languages: readonly string[]): ChainError | null {
  if (chain[0] !== language) {
    return "start";
  }
  if (chain[chain.length - 1] !== finalFallback) {
    return "end";
  }
  if (chain.some((step) => !languages.includes(step))) {
    return "unknown";
  }
  return new Set(chain).size === chain.length ? null : "repeat";
}
