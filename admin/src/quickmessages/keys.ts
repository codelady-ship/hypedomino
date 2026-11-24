export type QuickMessageCatalog = { keys: string[] };

const keyFormat = /^quick\.[a-z0-9_]+$/;

export function keysFromText(text: string): string[] {
  return text
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line !== "");
}

export function badKeys(keys: readonly string[]): string[] {
  return keys.filter((key, index) => !keyFormat.test(key) || keys.indexOf(key) !== index);
}

export function sameKeys(left: readonly string[], right: readonly string[]): boolean {
  return left.join("\n") === right.join("\n");
}
