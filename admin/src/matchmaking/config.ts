export const numberFields = ["mmrWindowStart", "mmrWindowStep", "mmrWindowStepSeconds", "mmrWindowMax", "rankWindowDivisions", "casualBotFillSeconds"] as const;
export const levelFields = ["rankedBotLevel", "casualBotLevel"] as const;
export const botLevels = ["easy", "medium", "hard"] as const;

type NumberField = (typeof numberFields)[number];
type LevelField = (typeof levelFields)[number];

export type MatchmakingConfig = Partial<Record<NumberField, number> & Record<LevelField, string>>;
export type MatchmakingFields = Record<NumberField | LevelField, string>;

const wholeNumber = /^\d+$/;

export function fieldsFromConfig(config: MatchmakingConfig | null): MatchmakingFields {
  const fields = {} as MatchmakingFields;
  for (const name of [...numberFields, ...levelFields]) {
    const value = config?.[name];
    fields[name] = value === undefined ? "" : String(value);
  }
  return fields;
}

export function badFields(fields: MatchmakingFields): string[] {
  const bad: string[] = numberFields.filter((name) => fields[name] !== "" && !wholeNumber.test(fields[name]));
  return bad.concat(levelFields.filter((name) => fields[name] !== "" && !(botLevels as readonly string[]).includes(fields[name])));
}

export function configFromFields(fields: MatchmakingFields): MatchmakingConfig {
  const config: MatchmakingConfig = {};
  for (const name of numberFields) {
    if (fields[name] !== "") {
      config[name] = Number(fields[name]);
    }
  }
  for (const name of levelFields) {
    if (fields[name] !== "") {
      config[name] = fields[name];
    }
  }
  return config;
}
