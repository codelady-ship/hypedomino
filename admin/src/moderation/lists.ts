export const wordLists = ["any", "az", "en", "tr", "ru"] as const;

export type WordList = (typeof wordLists)[number];
export type WordEntry = { entry: string; match: "substring" | "word" };
export type Moderation = { words: Partial<Record<WordList, WordEntry[]>>; reserved: string[] };
export type ModerationText = Record<WordList | "reserved", string>;

const wholeWord = "=";

function lines(text: string): string[] {
  return text
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line !== "" && line !== wholeWord);
}

export function moderationToText(moderation: Moderation | null): ModerationText {
  const text = { reserved: (moderation?.reserved ?? []).join("\n") } as ModerationText;
  for (const list of wordLists) {
    text[list] = (moderation?.words?.[list] ?? []).map((e) => (e.match === "word" ? wholeWord + e.entry : e.entry)).join("\n");
  }
  return text;
}

export function textToModeration(text: ModerationText): Moderation {
  const words: Moderation["words"] = {};
  for (const list of wordLists) {
    const entries = lines(text[list]).map((line): WordEntry => (line.startsWith(wholeWord) ? { entry: line.slice(1).trim(), match: "word" } : { entry: line, match: "substring" }));
    if (entries.length > 0) {
      words[list] = entries;
    }
  }
  return { words, reserved: lines(text.reserved) };
}

export function entryCount(moderation: Moderation): number {
  return wordLists.reduce((sum, list) => sum + (moderation.words?.[list]?.length ?? 0), 0) + (moderation.reserved?.length ?? 0);
}
