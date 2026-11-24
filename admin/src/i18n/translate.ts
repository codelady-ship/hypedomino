import type { Locale } from "./locales";
import { az } from "./messages/az.ts";
import { en } from "./messages/en.ts";

export type MessageKey = keyof typeof en;

export type Messages = Record<MessageKey, string>;

const messagesByLocale: Record<Locale, Messages> = { az, en };

export type Translate = ((key: MessageKey) => string) & {
  when: (instant: string) => string;
  number: (value: number) => string;
  signed: (value: number) => string;
};

export function translator(locale: Locale): Translate {
  const messages = messagesByLocale[locale];
  const dates = new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "medium", hourCycle: "h23", timeZone: "UTC" });
  const numbers = new Intl.NumberFormat(locale);
  const signedNumbers = new Intl.NumberFormat(locale, { signDisplay: "exceptZero" });
  return Object.assign((key: MessageKey) => messages[key], {
    when: (instant: string) => {
      const date = new Date(instant);
      return Number.isNaN(date.getTime()) ? instant : `${dates.format(date)} UTC`;
    },
    number: (value: number) => numbers.format(value),
    signed: (value: number) => signedNumbers.format(value),
  });
}
