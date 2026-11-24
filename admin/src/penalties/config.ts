export const wholeFields = [
  "afkReturnedWeight",
  "afkWeight",
  "leaveReturnedWeight",
  "leaveWeight",
  "offenseWindowDays",
  "cleanMatchesToForgive",
  "warningLevel",
  "queueDelayLevel",
  "banLevel",
  "reviewLevel",
  "queueDelaySeconds",
  "queueDelayMatches",
  "banHours",
  "offenseRatingPenalty",
  "partnerLossReductionPercent",
  "casualOffenseWindowDays",
  "casualCleanMatchesToForgive",
  "casualWarningLevel",
  "casualQueueDelayMatches",
] as const;
export const decimalFields = ["banEscalationFactor"] as const;
export const listFields = ["casualQueueDelayLevels", "casualQueueDelaySeconds"] as const;

type WholeField = (typeof wholeFields)[number];
type DecimalField = (typeof decimalFields)[number];
type ListField = (typeof listFields)[number];

export type PenaltiesConfig = Partial<Record<WholeField | DecimalField, number> & Record<ListField, number[]>>;
export type PenaltiesFields = Record<WholeField | DecimalField | ListField, string>;

const wholeNumber = /^\d+$/;
const decimalNumber = /^\d+(\.\d+)?$/;

export function fieldsFromConfig(config: PenaltiesConfig | null): PenaltiesFields {
  const fields = {} as PenaltiesFields;
  for (const name of [...wholeFields, ...decimalFields]) {
    const value = config?.[name];
    fields[name] = value === undefined ? "" : String(value);
  }
  for (const name of listFields) {
    fields[name] = config?.[name]?.join(", ") ?? "";
  }
  return fields;
}

function listItems(text: string): string[] {
  return text.split(",").map((item) => item.trim());
}

export function badFields(fields: PenaltiesFields): string[] {
  const bad: string[] = wholeFields.filter((name) => fields[name] !== "" && !wholeNumber.test(fields[name]));
  bad.push(...decimalFields.filter((name) => fields[name] !== "" && !decimalNumber.test(fields[name])));
  bad.push(...listFields.filter((name) => fields[name].trim() !== "" && !listItems(fields[name]).every((item) => wholeNumber.test(item))));
  return bad;
}

export function configFromFields(fields: PenaltiesFields): PenaltiesConfig {
  const config: PenaltiesConfig = {};
  for (const name of [...wholeFields, ...decimalFields]) {
    if (fields[name] !== "") {
      config[name] = Number(fields[name]);
    }
  }
  for (const name of listFields) {
    if (fields[name].trim() !== "") {
      config[name] = listItems(fields[name]).map(Number);
    }
  }
  return config;
}
