import { readFileSync, readdirSync } from "node:fs";
import { join, relative, sep } from "node:path";

export const projectPrefix = "HypeDomino.";
export const independentAssembly = "HypeDomino.Core";

export function parseLayout(readme) {
  const section = readme.split(/^## /m).find((part) => part.startsWith("Layout\n") || part.startsWith("Layout\r\n"));
  if (!section) throw new Error("client/README.md has no Layout section");
  const rows = section
    .split(/\r?\n/)
    .filter((line) => line.startsWith("|"))
    .slice(2)
    .map((line) => line.split("|").slice(1, -1).map((cell) => cell.trim()));
  return rows
    .filter((cells) => cells[1])
    .map(([path, assembly, references]) => ({
      path: unquote(path),
      assembly: unquote(assembly),
      references: references === "none" ? [] : references.split(",").map((name) => projectPrefix + name.trim()),
    }));
}

function unquote(cell) {
  return cell.replace(/^`|`$/g, "");
}

export function readAssemblies(clientRoot) {
  return asmdefFiles(join(clientRoot, "Assets")).map((file) => {
    const definition = JSON.parse(readFileSync(file, "utf8"));
    const guid = /^guid: ([0-9a-f]{32})$/m.exec(readFileSync(`${file}.meta`, "utf8"))?.[1];
    const folder = relative(clientRoot, join(file, "..")).split(sep).join("/");
    return {
      path: folder,
      name: definition.name,
      guid,
      references: definition.references ?? [],
      precompiledReferences: definition.precompiledReferences ?? [],
    };
  });
}

function asmdefFiles(directory) {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = join(directory, entry.name);
    if (entry.isDirectory()) return asmdefFiles(path);
    return entry.name.endsWith(".asmdef") ? [path] : [];
  });
}

export function checkLayout(rows, assemblies) {
  const errors = [];
  const rowIndex = new Map();
  rows.forEach((row, index) => {
    if (rowIndex.has(row.assembly)) errors.push(`${row.assembly}: listed twice in the table`);
    else rowIndex.set(row.assembly, index);
  });
  const nameByGuid = new Map(assemblies.map((assembly) => [assembly.guid, assembly.name]));
  const projectNames = new Set(assemblies.map((assembly) => assembly.name));
  for (const row of rows) {
    if (!projectNames.has(row.assembly)) errors.push(`${row.assembly}: in the table but no asmdef defines it`);
  }
  for (const assembly of assemblies) {
    const at = rowIndex.get(assembly.name);
    if (at === undefined) {
      errors.push(`${assembly.name}: not in the table`);
      continue;
    }
    errors.push(...checkAssembly(assembly, rows[at], at, rowIndex, nameByGuid, projectNames));
  }
  return errors;
}

function checkAssembly(assembly, row, at, rowIndex, nameByGuid, projectNames) {
  const errors = [];
  if (row.path !== assembly.path) errors.push(`${assembly.name}: defined in ${assembly.path}, the table says ${row.path}`);
  if (assembly.name === independentAssembly && assembly.references.length + assembly.precompiledReferences.length > 0)
    errors.push(`${assembly.name}: must reference nothing`);
  const references = assembly.references.map((reference) => resolve(reference, nameByGuid)).filter((name) => projectNames.has(name));
  for (const reference of references) {
    if (!(rowIndex.get(reference) < at)) errors.push(`${assembly.name}: references ${reference}, which is not above it in the table`);
    if (!row.references.includes(reference)) errors.push(`${assembly.name}: references ${reference}, which the table does not list`);
  }
  for (const listed of row.references) {
    if (!references.includes(listed)) errors.push(`${assembly.name}: the table lists ${listed}, which the asmdef does not reference`);
  }
  return errors;
}

function resolve(reference, nameByGuid) {
  return reference.startsWith("GUID:") ? nameByGuid.get(reference.slice(5)) ?? reference : reference;
}
