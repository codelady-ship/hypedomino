import argparse
import hashlib
import io
import json
import sys
import urllib.parse
import urllib.request
from pathlib import Path

import fontTools
from fontTools.ttLib import TTFont
from fontTools.varLib import instancer

REPO_ROOT = Path(__file__).resolve().parents[2]
FONTS_FOLDER = REPO_ROOT / "client" / "Assets" / "HypeDomino" / "UI" / "Fonts"
MANIFEST = FONTS_FOLDER / "fonts.json"
SOURCE_FOLDER = FONTS_FOLDER / "Source"
WINDOWS = (3, 1, 0x409)
DROPPED_NAME_IDS = (16, 17, 21, 22, 25)


def raw_url(repository, commit, path):
    return f"https://raw.githubusercontent.com/{repository}/{commit}/{urllib.parse.quote(path)}"


def fetch(url):
    with urllib.request.urlopen(url) as response:
        return response.read()


def checked(data, expected, path):
    actual = hashlib.sha256(data).hexdigest()
    if actual != expected:
        raise SystemExit(f"{path}: sha256 {actual}, manifest pins {expected}")
    return data


def style_name(italic):
    return "Italic" if italic else "Regular"


def postscript_name(family, weight, italic):
    return f"{family.replace(' ', '')}{weight}-{style_name(italic)}"


def rename(font, family, weight, italic):
    name = font["name"]
    full_family = f"{family} {weight}"
    style = style_name(italic)
    values = {
        1: full_family,
        2: style,
        3: f"{postscript_name(family, weight, italic)};{font['head'].fontRevision:.3f}",
        4: f"{full_family} {style}" if italic else full_family,
        6: postscript_name(family, weight, italic),
    }
    for name_id in DROPPED_NAME_IDS:
        name.removeNames(nameID=name_id)
    for name_id, value in values.items():
        name.removeNames(nameID=name_id)
        name.setName(value, name_id, *WINDOWS)


def static_face(variable_bytes, family, face, output):
    font = TTFont(io.BytesIO(variable_bytes), recalcTimestamp=False)
    instance = instancer.instantiateVariableFont(font, {"wght": face["weight"]}, static=True)
    rename(instance, family["modifiedName"], face["weight"], family["italic"])
    output.parent.mkdir(parents=True, exist_ok=True)
    instance.save(output)


def main():
    parser = argparse.ArgumentParser(prog="build_fonts.py", description="Rebuild the static font faces of client/Assets/HypeDomino/UI/Fonts from the pinned Google Fonts commit")
    parser.parse_args()
    manifest = json.loads(MANIFEST.read_text(encoding="utf-8"))
    source = manifest["source"]
    if fontTools.version != source["fontTools"]:
        raise SystemExit(f"fontTools {fontTools.version} installed, the manifest pins {source['fontTools']}")
    families = {family["token"]: family for family in manifest["families"]}
    variable_fonts = {}
    for family in manifest["families"]:
        url = raw_url(source["repository"], source["commit"], family["file"])
        variable_fonts[family["token"]] = checked(fetch(url), family["sha256"], family["file"])
        license_text = fetch(raw_url(source["repository"], source["commit"], family["license"]))
        folder = SOURCE_FOLDER / family["folder"]
        folder.mkdir(parents=True, exist_ok=True)
        (folder / "OFL.txt").write_bytes(license_text)
    for face in manifest["faces"]:
        family = families[face["family"]]
        output = SOURCE_FOLDER / family["folder"] / f"{face['id']}.ttf"
        static_face(variable_fonts[face["family"]], family, face, output)
        print(f"wrote {output.relative_to(REPO_ROOT).as_posix()}")


if __name__ == "__main__":
    sys.exit(main())
