export type ConfigKey = {
  key: string;
  type: string;
  unit: string;
  min: number;
  max?: number;
  default: number;
  value: number;
  set: boolean;
};

export type KeyedConfig = { config_version: number; keys: ConfigKey[] };

export type KeyValues = Readonly<Record<string, number>>;

export type KeyedVersion = { config_version: number; published_at: string; values: KeyValues };

export type KeyChange = { key: string; before: number; after: number };

function valueOf(key: ConfigKey, values: KeyValues): number {
  return values[key.key] ?? key.value;
}

export function withinBounds(key: ConfigKey, value: number): boolean {
  return Number.isInteger(value) && value >= key.min && (key.max === undefined || value <= key.max);
}

export function invalidKeys(keys: readonly ConfigKey[], values: KeyValues): string[] {
  return keys.filter((key) => !withinBounds(key, valueOf(key, values))).map((key) => key.key);
}

export function changedKeys(keys: readonly ConfigKey[], values: KeyValues): string[] {
  return keys.filter((key) => valueOf(key, values) !== key.value).map((key) => key.key);
}

export function publishedValues(keys: readonly ConfigKey[], values: KeyValues): KeyValues {
  return Object.fromEntries(keys.filter((key) => valueOf(key, values) !== key.default).map((key) => [key.key, valueOf(key, values)]));
}

function publishedValue(key: ConfigKey, values: KeyValues): number {
  return values[key.key] ?? key.default;
}

export function versionChanges(keys: readonly ConfigKey[], older: KeyValues, newer: KeyValues): KeyChange[] {
  return keys
    .filter((key) => publishedValue(key, older) !== publishedValue(key, newer))
    .map((key) => ({ key: key.key, before: publishedValue(key, older), after: publishedValue(key, newer) }));
}
