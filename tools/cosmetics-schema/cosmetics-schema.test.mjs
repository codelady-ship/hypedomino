import { test } from "node:test";
import assert from "node:assert/strict";
import { fileURLToPath } from "node:url";
import { checkManifest, itemTypes, loadExamples, loadSchema } from "./cosmetics-schema.mjs";

const repoRoot = new URL("../../", import.meta.url);
const schema = loadSchema(fileURLToPath(new URL("docs/cosmetics/schema.json", repoRoot)));
const examples = loadExamples(fileURLToPath(new URL("docs/cosmetics/examples/", repoRoot)));
const catalog = examples.find(({ name }) => name === "catalog.json").manifest;

function errorsAfter(change) {
  const copy = structuredClone(catalog);
  change(copy, (id) => copy.items.find((item) => item.id === id));
  return checkManifest(schema, copy);
}

function assertReported(errors, fragment) {
  assert.ok(errors.some((error) => error.includes(fragment)), `expected "${fragment}" in:\n${errors.join("\n")}`);
}

test("every example manifest is valid", () => {
  assert.ok(examples.length >= 2);
  for (const { name, manifest } of examples) assert.deepEqual(checkManifest(schema, manifest), [], name);
});

test("the catalog example has an animated item of every type", () => {
  for (const type of itemTypes) assert.ok(catalog.items.some((item) => item.type === type && item.animated), type);
});

test("an animated item without a static fallback is rejected", () => {
  assertReported(errorsAfter((_, item) => delete item("avatar.comet_fox").staticAddress), "missing staticAddress");
});

test("a static item without a static address is rejected", () => {
  assertReported(errorsAfter((_, item) => delete item("avatar.starter").staticAddress), "missing staticAddress");
});

test("an animated avatar, frame or emoji without an animation address is rejected", () => {
  for (const id of ["avatar.comet_fox", "frame.season_1_top", "emoji.laugh_spin"]) {
    assertReported(errorsAfter((_, item) => delete item(id).animationAddress), "missing animationAddress");
  }
});

test("the removed character and action types are rejected", () => {
  for (const type of ["character", "action"]) {
    assertReported(
      errorsAfter((manifest) => {
        manifest.items.push({ ...structuredClone(manifest.items[0]), id: `${type}.ali`, type, nameKey: `cosmetic.${type}.ali.name`, staticAddress: `cosmetic/${type}/ali/static` });
      }),
      "does not match",
    );
  }
});

test("a static item with animation fields is rejected", () => {
  assertReported(
    errorsAfter((_, item) => {
      item("avatar.comet_fox").animated = false;
    }),
    "unexpected property animationAddress",
  );
  assertReported(
    errorsAfter((_, item) => {
      item("table.night_market").animated = false;
    }),
    "unexpected property layers",
  );
});

test("a model or clips on any item are rejected", () => {
  assertReported(
    errorsAfter((_, item) => {
      item("avatar.tile_wizard").modelAddress = "cosmetic/avatar/tile_wizard/model";
    }),
    "unexpected property modelAddress",
  );
  assertReported(
    errorsAfter((_, item) => {
      item("avatar.tile_wizard").clips = { idle: "cosmetic/avatar/tile_wizard/clip/idle" };
    }),
    "unexpected property clips",
  );
});

test("fields of another type are rejected", () => {
  assertReported(
    errorsAfter((_, item) => {
      item("table.night_market").animationAddress = "cosmetic/table/night_market/anim";
    }),
    "unexpected property animationAddress",
  );
  assertReported(
    errorsAfter((_, item) => {
      item("avatar.comet_fox").group = "basic";
    }),
    "unexpected property group",
  );
});

test("an emoji without a group is rejected", () => {
  assertReported(errorsAfter((_, item) => delete item("emoji.laugh_spin").group), "missing group");
});

test("an animated table needs unique layer names and an animated layer", () => {
  assertReported(
    errorsAfter((_, item) => {
      const table = item("table.night_market");
      table.layers[1].name = "sky";
      table.layers[1].address = "cosmetic/table/night_market/layer/sky";
    }),
    "layer names are not unique",
  );
  assertReported(
    errorsAfter((_, item) => {
      for (const layer of item("table.night_market").layers) layer.animated = false;
    }),
    "animated table has no animated layer",
  );
});

test("a price or any other undeclared field is rejected", () => {
  assertReported(
    errorsAfter((_, item) => {
      item("avatar.comet_fox").price = 500;
    }),
    "unexpected property price",
  );
  assertReported(
    errorsAfter((_, item) => {
      item("avatar.comet_fox").sources[0].price = 500;
    }),
    "unexpected property price",
  );
});

test("a source needs a reference unless it is the default source", () => {
  assertReported(errorsAfter((_, item) => delete item("avatar.comet_fox").sources[0].ref), "missing ref");
  assertReported(
    errorsAfter((_, item) => {
      item("avatar.starter").sources[0].ref = "offer.any";
    }),
    "unexpected property ref",
  );
});

test("an unknown source type and a duplicate source are rejected", () => {
  assertReported(
    errorsAfter((_, item) => {
      item("avatar.comet_fox").sources[0].type = "lootbox";
    }),
    "expected one of",
  );
  assertReported(
    errorsAfter((_, item) => {
      const sources = item("avatar.comet_fox").sources;
      sources.push(structuredClone(sources[0]));
    }),
    "items are not unique",
  );
});

test("an item without a source is rejected", () => {
  assertReported(
    errorsAfter((_, item) => {
      item("avatar.comet_fox").sources = [];
    }),
    "fewer than 1 items",
  );
});

test("a duplicate id is rejected", () => {
  assertReported(
    errorsAfter((manifest, item) => manifest.items.push(structuredClone(item("avatar.comet_fox")))),
    "avatar.comet_fox: duplicate id",
  );
});

test("an id whose prefix differs from the type is rejected", () => {
  assertReported(
    errorsAfter((_, item) => {
      item("avatar.comet_fox").id = "frame.comet_fox";
    }),
    "id prefix does not match type avatar",
  );
});

test("a name key that does not follow the item id is rejected", () => {
  assertReported(
    errorsAfter((_, item) => {
      item("avatar.comet_fox").nameKey = "cosmetic.avatar.other.name";
    }),
    "nameKey is cosmetic.avatar.other.name",
  );
});

test("an address off the naming convention is rejected", () => {
  assertReported(
    errorsAfter((_, item) => {
      item("avatar.comet_fox").animationAddress = "cosmetic/avatar/other/anim";
    }),
    "animationAddress is cosmetic/avatar/other/anim",
  );
  assertReported(
    errorsAfter((_, item) => {
      item("table.night_market").layers[0].address = "cosmetic/table/night_market/layer/other";
    }),
    "is cosmetic/table/night_market/layer/other",
  );
  assertReported(
    errorsAfter((_, item) => {
      item("avatar.comet_fox").staticAddress = "avatars/comet_fox.png";
    }),
    "does not match",
  );
});

test("every type needs a live default item", () => {
  assertReported(
    errorsAfter((manifest) => {
      manifest.items = manifest.items.filter((item) => item.id !== "table.starter");
    }),
    "table: no live item with a default source",
  );
  const errors = errorsAfter((_, item) => {
    item("frame.starter").availability = "disabled";
  });
  assertReported(errors, "frame.starter: default item is disabled, must be live");
  assertReported(errors, "frame: no live item with a default source");
});

test("an animated default item is rejected", () => {
  assertReported(
    errorsAfter((_, item) => {
      const starter = item("avatar.starter");
      starter.animated = true;
      starter.animationAddress = "cosmetic/avatar/starter/anim";
    }),
    "avatar.starter: default item is animated, must be static",
  );
  assertReported(
    errorsAfter((_, item) => {
      item("avatar.comet_fox").sources.push({ type: "default" });
    }),
    "avatar.comet_fox: default item is animated, must be static",
  );
});

test("a published item must have passed cultural review", () => {
  assertReported(
    errorsAfter((_, item) => {
      item("avatar.comet_fox").culturalReview = "pending";
    }),
    "expected \"approved\"",
  );
});

test("an unknown schema version is rejected", () => {
  assertReported(
    errorsAfter((manifest) => {
      manifest.schemaVersion = 2;
    }),
    "$.schemaVersion: expected 1",
  );
});
