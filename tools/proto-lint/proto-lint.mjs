import { readdirSync, readFileSync, statSync } from "node:fs";
import { join, relative } from "node:path";
import { fileURLToPath } from "node:url";

export const FORBIDDEN_TOKENS = new Set([
  "balance",
  "balances",
  "coin",
  "coins",
  "crystal",
  "crystals",
  "price",
  "prices",
  "cost",
  "costs",
  "amount",
  "amounts",
  "reward",
  "rewards",
  "score",
  "scores",
  "points",
  "rating",
  "ratings",
  "mmr",
  "lp",
  "xp",
  "rank",
  "tier",
  "inventory",
  "owned",
  "result",
  "results",
]);

const BLOCK_START = /^\s*(message|enum|oneof)\s+(\w+)\s*\{\s*$/;
const EMPTY_MESSAGE = /^\s*message\s+(\w+)\s*\{\s*\}\s*$/;
const BLOCK_END = /^\s*\}\s*$/;
const FIELD = /^\s*(?:repeated\s+|optional\s+)?(map<[^>]+>|[\w.]+)\s+(\w+)\s*=\s*\d+/;

export function isClientMessage(name) {
  return name.endsWith("Intent") || name === "ClientHello";
}

export function parseProto(source) {
  const messages = new Map();
  const stack = [];
  const lines = source.split("\n");
  for (let index = 0; index < lines.length; index++) {
    const line = lines[index];
    const empty = line.match(EMPTY_MESSAGE);
    if (empty) {
      messages.set(fullName(stack, empty[1]), { fields: [] });
      continue;
    }
    const start = line.match(BLOCK_START);
    if (start) {
      const [, kind, name] = start;
      if (kind === "message") messages.set(fullName(stack, name), { fields: [] });
      stack.push({ kind, name });
      continue;
    }
    if (BLOCK_END.test(line)) {
      stack.pop();
      continue;
    }
    const owner = enclosingMessage(stack);
    if (!owner) continue;
    const field = line.match(FIELD);
    if (!field) continue;
    messages.get(owner).fields.push({ type: field[1], name: field[2], line: index + 1 });
  }
  return messages;
}

function fullName(stack, name) {
  return [...messageScope(stack), name].join(".");
}

function enclosingMessage(stack) {
  if (stack.some((block) => block.kind === "enum")) return null;
  const scope = messageScope(stack);
  return scope.length === 0 ? null : scope.join(".");
}

function messageScope(stack) {
  return stack.filter((block) => block.kind === "message").map((block) => block.name);
}

export function forbiddenTokens(fieldName) {
  return fieldName.split("_").filter((token) => FORBIDDEN_TOKENS.has(token));
}

export function lintSources(sources) {
  const messages = new Map();
  for (const { path, source } of sources) {
    for (const [name, message] of parseProto(source)) messages.set(name, { ...message, path });
  }
  const violations = [];
  const visited = new Set();
  for (const name of messages.keys()) {
    if (isClientMessage(lastSegment(name))) walk(name, name, messages, visited, violations);
  }
  return violations;
}

function walk(name, root, messages, visited, violations) {
  const key = `${root}>${name}`;
  if (visited.has(key)) return;
  visited.add(key);
  const message = messages.get(name);
  for (const field of message.fields) {
    for (const token of forbiddenTokens(field.name)) {
      violations.push({ path: message.path, line: field.line, message: name, field: field.name, token, root });
    }
    const target = resolveType(field.type, name, messages);
    if (target) walk(target, root, messages, visited, violations);
  }
}

function resolveType(type, scope, messages) {
  if (type.startsWith("map<")) return null;
  const simple = lastSegment(type);
  const parts = scope.split(".");
  for (let depth = parts.length; depth >= 0; depth--) {
    const candidate = [...parts.slice(0, depth), simple].join(".");
    if (messages.has(candidate)) return candidate;
  }
  for (const name of messages.keys()) if (lastSegment(name) === simple) return name;
  return null;
}

function lastSegment(name) {
  return name.slice(name.lastIndexOf(".") + 1);
}

export function collectProtoFiles(root) {
  const files = [];
  for (const entry of readdirSync(root)) {
    const path = join(root, entry);
    if (statSync(path).isDirectory()) files.push(...collectProtoFiles(path));
    else if (entry.endsWith(".proto")) files.push(path);
  }
  return files.sort();
}

export function formatViolation(violation, cwd) {
  const path = relative(cwd, violation.path).split("\\").join("/");
  return `${path}:${violation.line}: ${violation.message}.${violation.field} reaches the client message ${violation.root} and carries "${violation.token}"`;
}

function main(argv) {
  const root = argv[0];
  if (!root) {
    console.error("usage: proto-lint <proto-dir>");
    return 2;
  }
  const sources = collectProtoFiles(root).map((path) => ({ path, source: readFileSync(path, "utf8") }));
  const violations = lintSources(sources);
  for (const violation of violations) console.error(formatViolation(violation, process.cwd()));
  if (violations.length > 0) return 1;
  const clientMessages = sources
    .flatMap(({ source }) => [...parseProto(source).keys()])
    .filter((name) => isClientMessage(lastSegment(name)));
  console.log(`proto-lint: ${clientMessages.length} client messages carry no value fields`);
  return 0;
}

if (process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1]) {
  process.exitCode = main(process.argv.slice(2));
}
