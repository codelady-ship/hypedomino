import json
import sys
from pathlib import Path


def main(package_dir, list_path, out_path):
    package = Path(package_dir)
    meta = json.loads((package / "package.json").read_text(encoding="utf-8"))
    wanted = json.loads(Path(list_path).read_text(encoding="utf-8"))
    if f"{meta['name']}@{meta['version']}" != wanted["package"]:
        raise SystemExit(f"package is {meta['name']}@{meta['version']}, list wants {wanted['package']}")
    manifest = json.loads((package / "dist" / "manifest.json").read_text(encoding="utf-8"))
    categories = {}
    sources = {}
    keys = set()
    for icon in wanted["icons"]:
        if icon["key"] in keys:
            raise SystemExit(f"duplicate key {icon['key']}")
        keys.add(icon["key"])
        entry = manifest[icon["source"]]
        if wanted["style"] not in entry["styles"]:
            raise SystemExit(f"{icon['source']} has no {wanted['style']} style")
        category = entry["category"]
        if category not in categories:
            categories[category] = json.loads((package / "dist" / "data" / f"{category}.json").read_text(encoding="utf-8"))
        sources[icon["key"]] = categories[category][icon["source"]][wanted["style"]]
    Path(out_path).write_text(json.dumps(sources, indent=2, sort_keys=True) + "\n", encoding="utf-8", newline="\n")
    print(f"{len(sources)} {wanted['style']} icons from {wanted['package']} into {out_path}")


if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2], sys.argv[3])
