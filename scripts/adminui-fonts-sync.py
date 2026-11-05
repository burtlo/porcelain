#!/usr/bin/env python3
"""Build local operator UI fonts from a hand-maintained icon list.

Source of truth: embedui/fonts/icons.txt (one Material Symbols name per line).

  make adminui-fonts          # fetch sources if needed, then subset
  make adminui-fonts-fetch    # download upstream TTFs into chimera/.deps only

Committed outputs (no fonttools needed for normal builds):
  embedui/fonts/*.woff2, *.ttf
  embedui/fonts.css
  embedui/fonts/codepoints.js
  embedui/fonts/NOTICE
"""

from __future__ import annotations

import argparse
import os
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[1]
EMBEDUI = REPO_ROOT / "chimera/chimera-gateway/internal/server/adminui/embed/embedui"
FONTS_OUT = EMBEDUI / "fonts"
FONTS_CSS = EMBEDUI / "fonts.css"
ICONS_TXT = FONTS_OUT / "icons.txt"
DEPS = REPO_ROOT / "chimera" / ".deps"

ICON_NAME = re.compile(r"^[a-z][a-z0-9_]*$")


def die(msg: str, code: int = 1) -> None:
    print(f"adminui-fonts: {msg}", file=sys.stderr)
    raise SystemExit(code)


def read_icons() -> list[str]:
    if not ICONS_TXT.is_file():
        die(f"missing {ICONS_TXT} — add Material Symbols names, one per line")
    out: list[str] = []
    seen: set[str] = set()
    for line in ICONS_TXT.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        if not ICON_NAME.match(line):
            die(f"invalid icon name in icons.txt: {line!r}")
        if line in seen:
            continue
        seen.add(line)
        out.append(line)
    if not out:
        die("icons.txt has no icon names")
    return out


def ensure_fonttools() -> Path:
    try:
        import fontTools  # noqa: F401

        return Path(sys.executable)
    except ImportError:
        pass
    venv = DEPS / "adminui-fonts-venv"
    py = venv / ("Scripts/python.exe" if os.name == "nt" else "bin/python")
    if not py.is_file():
        print("adminui-fonts: creating fonttools venv under chimera/.deps/adminui-fonts-venv")
        DEPS.mkdir(parents=True, exist_ok=True)
        subprocess.check_call([sys.executable, "-m", "venv", str(venv)])
        subprocess.check_call(
            [str(py), "-m", "pip", "install", "--quiet", "fonttools[woff]", "brotli"]
        )
    else:
        subprocess.check_call([str(py), "-c", "import fontTools"])
    return py


def find_material_symbols_ttf() -> Path | None:
    candidates: list[Path] = []
    env = os.environ.get("ADMINUI_MATERIAL_ICONS_SRC", "").strip()
    if env:
        candidates.append(Path(env))
    candidates.extend(
        [
            REPO_ROOT.parent / "material-design-icons",
            DEPS / "material-design-icons",
        ]
    )
    fonts_env = os.environ.get("ADMINUI_FONTS_SRC", "").strip()
    if fonts_env:
        candidates.append(Path(fonts_env))
    candidates.extend([REPO_ROOT.parent / "fonts", DEPS / "google-fonts"])
    for root in candidates:
        if not root.is_dir():
            continue
        hits = list(root.rglob("MaterialSymbolsOutlined*.ttf"))
        hits = [h for h in hits if h.is_file()]
        if hits:
            hits.sort(key=lambda p: (0 if "FILL" in p.name else 1, str(p)))
            return hits[0]
    return None


def find_material_symbols_codepoints(ttf: Path) -> Path | None:
    sibling = ttf.with_suffix(".codepoints")
    if sibling.is_file():
        return sibling
    for p in ttf.parent.glob("MaterialSymbolsOutlined*.codepoints"):
        return p
    for root in (ttf.parent, ttf.parent.parent, DEPS / "material-design-icons"):
        if not root.is_dir():
            continue
        hits = list(root.rglob("MaterialSymbolsOutlined*.codepoints"))
        if hits:
            return hits[0]
    return None


def find_hanken_ttf() -> Path | None:
    candidates: list[Path] = []
    env = os.environ.get("ADMINUI_FONTS_SRC", "").strip()
    if env:
        candidates.append(Path(env))
    candidates.extend(
        [
            REPO_ROOT.parent / "fonts",
            DEPS / "google-fonts",
            REPO_ROOT.parent / "hanken-grotesk",
            DEPS / "hanken-grotesk",
        ]
    )
    for root in candidates:
        if not root.is_dir():
            continue
        for rel in (
            Path("ofl/hankengrotesk/HankenGrotesk[wght].ttf"),
            Path("fonts/variable/HankenGrotesk-VF.ttf"),
        ):
            p = root / rel
            if p.is_file():
                return p
        hits = list(root.rglob("HankenGrotesk*.ttf"))
        if hits:
            hits.sort(key=lambda p: (0 if "wght" in p.name or "VF" in p.name else 1, str(p)))
            return hits[0]
    return None


def load_icon_codepoints(codepoints_path: Path, icons: list[str]) -> dict[str, int]:
    mapping: dict[str, str] = {}
    for line in codepoints_path.read_text(encoding="utf-8").splitlines():
        parts = line.split()
        if len(parts) >= 2:
            mapping[parts[0]] = parts[1].lower().removeprefix("u+")
    missing = [i for i in icons if i not in mapping]
    if missing:
        die(
            "icons in icons.txt missing from Material Symbols codepoints map:\n  "
            + "\n  ".join(missing)
            + f"\n(codepoints file: {codepoints_path})"
        )
    return {name: int(mapping[name], 16) for name in icons}


def instantiate_material(py: Path, src: Path, dest: Path) -> None:
    script = (
        "from fontTools.ttLib import TTFont\n"
        "from fontTools.varLib import instancer\n"
        f"font = TTFont({str(src)!r})\n"
        "inst = instancer.instantiateVariableFont(font, "
        "{'opsz': 24, 'wght': 400, 'FILL': 0, 'GRAD': 0})\n"
        f"inst.save({str(dest)!r})\n"
    )
    print(f"adminui-fonts: instantiating Material Symbols from {src}")
    subprocess.check_call([str(py), "-c", script])


def subset_material(py: Path, src: Path, icons: list[str], mapping: dict[str, int], dest_woff2: Path, dest_ttf: Path) -> None:
    dest_woff2.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="adminui-ms-") as tmp:
        tmp_path = Path(tmp)
        inst = tmp_path / "material-inst.ttf"
        instantiate_material(py, src, inst)
        uni_file = tmp_path / "unicodes.txt"
        text_file = tmp_path / "text.txt"
        uni_file.write_text(",".join(f"{cp:04x}" for cp in mapping.values()), encoding="utf-8")
        text_file.write_text("".join(icons), encoding="utf-8")
        # Keep ligature rules so name text still works; unicodes keep PUA glyphs for codepoints.js.
        base = [
            str(py),
            "-m",
            "fontTools.subset",
            str(inst),
            f"--unicodes-file={uni_file}",
            f"--text-file={text_file}",
            "--layout-features=*",
            "--glyph-names",
            "--symbol-cmap",
            "--legacy-cmap",
            "--notdef-glyph",
            "--notdef-outline",
            "--name-IDs=*",
            "--name-legacy",
            "--name-languages=*",
        ]
        print(f"adminui-fonts: subsetting Material Symbols ({len(icons)} icons from icons.txt)")
        subprocess.check_call(
            base + ["--flavor=woff2", f"--output-file={dest_woff2}"]
        )
        subprocess.check_call(base + [f"--output-file={dest_ttf}"])


def subset_hanken(py: Path, src: Path, dest_woff2: Path, dest_ttf: Path) -> None:
    dest_woff2.parent.mkdir(parents=True, exist_ok=True)
    text = (
        "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
        "0123456789"
        " .,;:!?\"'`()[]{}<>/@#$%^&*+-=_|\\~"
        "—–…·•°€£¥"
    )
    base = [
        str(py),
        "-m",
        "fontTools.subset",
        str(src),
        f"--text={text}",
        "--unicodes=U+0000-00FF,U+0131,U+0152-0153,U+02BB-02BC,U+02C6,U+02DA,U+02DC,"
        "U+0304,U+0308,U+0329,U+2000-206F,U+2074,U+20AC,U+2122,U+2191,U+2193,U+2212,U+2215,"
        "U+FEFF,U+FFFD",
        "--layout-features=*",
        "--glyph-names",
        "--symbol-cmap",
        "--legacy-cmap",
        "--notdef-glyph",
        "--notdef-outline",
        "--recommended-glyphs",
        "--name-IDs=*",
        "--name-legacy",
        "--name-languages=*",
    ]
    print(f"adminui-fonts: subsetting Hanken Grotesk from {src}")
    subprocess.check_call(base + ["--flavor=woff2", f"--output-file={dest_woff2}"])
    subprocess.check_call(base + [f"--output-file={dest_ttf}"])


def write_fonts_css(cache_bust: str) -> None:
    FONTS_CSS.write_text(
        f"""/* Generated by scripts/adminui-fonts-sync.py — do not edit by hand.
 * Icon list: embedui/fonts/icons.txt
 * Rebuild: make adminui-fonts
 */
@font-face {{
  font-family: "Hanken Grotesk";
  font-style: normal;
  font-weight: 100 900;
  font-display: swap;
  src:
    url("/ui/assets/fonts/hanken-grotesk-latin.woff2?v={cache_bust}") format("woff2"),
    url("/ui/assets/fonts/hanken-grotesk-latin.ttf?v={cache_bust}") format("truetype");
}}

@font-face {{
  font-family: "Material Symbols Outlined";
  font-style: normal;
  font-weight: 400;
  font-display: block;
  src:
    url("/ui/assets/fonts/material-symbols-outlined.woff2?v={cache_bust}") format("woff2"),
    url("/ui/assets/fonts/material-symbols-outlined.ttf?v={cache_bust}") format("truetype");
}}
""",
        encoding="utf-8",
    )


def write_codepoints_js(mapping: dict[str, int]) -> None:
    lines = [
        "/* Generated by scripts/adminui-fonts-sync.py from fonts/icons.txt — do not edit by hand. */",
        "(function (g) {",
        '  "use strict";',
        "  var CP = {",
    ]
    for name, cp in mapping.items():
        lines.append(f"    {name!r}: 0x{cp:04x},")
    lines += [
        "  };",
        "  function char(name) {",
        '    name = String(name || "");',
        "    var cp = CP[name];",
        "    return cp != null ? String.fromCodePoint(cp) : name;",
        "  }",
        "  g.ChimeraMaterialIcons = { codepoints: CP, char: char };",
        "})(globalThis);",
        "",
    ]
    (FONTS_OUT / "codepoints.js").write_text("\n".join(lines), encoding="utf-8")


def write_notice(material_src: Path, hanken_src: Path) -> None:
    def rel(p: Path) -> str:
        try:
            return str(p.resolve().relative_to(REPO_ROOT))
        except ValueError:
            return str(p)

    notice = f"""Operator UI font assets (generated from fonts/icons.txt)

Material Symbols Outlined
  License: Apache License 2.0
  Upstream: https://github.com/google/material-design-icons
  Instantiated at: opsz=24 wght=400 FILL=0 GRAD=0
  Source: {rel(material_src)}

Hanken Grotesk
  License: SIL Open Font License 1.1 (see OFL-HankenGrotesk.txt)
  Upstream: https://github.com/google/fonts (ofl/hankengrotesk)
  Source: {rel(hanken_src)}

Edit fonts/icons.txt, then: make adminui-fonts
"""
    (FONTS_OUT / "NOTICE").write_text(notice, encoding="utf-8")
    for cand in (
        hanken_src.parent / "OFL.txt",
        DEPS / "google-fonts/ofl/hankengrotesk/OFL.txt",
    ):
        if cand.is_file():
            shutil.copy2(cand, FONTS_OUT / "OFL-HankenGrotesk.txt")
            break


def cmd_check(_: argparse.Namespace) -> None:
    icons = read_icons()
    js = FONTS_OUT / "codepoints.js"
    if not js.is_file():
        die("missing fonts/codepoints.js — run make adminui-fonts")
    have = set(
        re.findall(r'["\']([a-z][a-z0-9_]*)["\']\s*:\s*0x', js.read_text(encoding="utf-8"))
    )
    missing = [i for i in icons if i not in have]
    extra = sorted(have - set(icons))
    if missing or extra:
        msg = []
        if missing:
            msg.append("in icons.txt but not in codepoints.js: " + ", ".join(missing))
        if extra:
            msg.append("in codepoints.js but not icons.txt: " + ", ".join(extra))
        die("\n".join(msg) + "\nRun: make adminui-fonts")
    for name in (
        "material-symbols-outlined.woff2",
        "material-symbols-outlined.ttf",
        "hanken-grotesk-latin.woff2",
        "hanken-grotesk-latin.ttf",
    ):
        p = FONTS_OUT / name
        if not p.is_file() or p.stat().st_size < 100:
            die(f"missing or empty {p} — run make adminui-fonts")
    print(f"adminui-fonts: ok ({len(icons)} icons in icons.txt)")


def cmd_sync(_: argparse.Namespace) -> None:
    icons = read_icons()
    material = find_material_symbols_ttf()
    hanken = find_hanken_ttf()
    if material is None or hanken is None:
        missing = []
        if material is None:
            missing.append("Material Symbols Outlined TTF")
        if hanken is None:
            missing.append("Hanken Grotesk TTF")
        die(
            "missing source font(s): "
            + ", ".join(missing)
            + "\n  make adminui-fonts-fetch\n"
            "  or clone ../material-design-icons and/or ../fonts\n"
            "  or set ADMINUI_MATERIAL_ICONS_SRC / ADMINUI_FONTS_SRC"
        )
    cp_path = find_material_symbols_codepoints(material)
    if cp_path is None:
        die("Material Symbols .codepoints map not found; re-run make adminui-fonts-fetch")
    mapping = load_icon_codepoints(cp_path, icons)
    py = ensure_fonttools()
    FONTS_OUT.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="adminui-fonts-") as tmp:
        tmp_path = Path(tmp)
        ms_w = tmp_path / "material-symbols-outlined.woff2"
        ms_t = tmp_path / "material-symbols-outlined.ttf"
        hk_w = tmp_path / "hanken-grotesk-latin.woff2"
        hk_t = tmp_path / "hanken-grotesk-latin.ttf"
        subset_material(py, material, icons, mapping, ms_w, ms_t)
        subset_hanken(py, hanken, hk_w, hk_t)
        shutil.copy2(ms_w, FONTS_OUT / "material-symbols-outlined.woff2")
        shutil.copy2(ms_t, FONTS_OUT / "material-symbols-outlined.ttf")
        shutil.copy2(hk_w, FONTS_OUT / "hanken-grotesk-latin.woff2")
        shutil.copy2(hk_t, FONTS_OUT / "hanken-grotesk-latin.ttf")
    # Cache-bust query changes whenever the woff2 bytes change.
    import hashlib

    digest = hashlib.sha256((FONTS_OUT / "material-symbols-outlined.woff2").read_bytes()).hexdigest()[:10]
    write_fonts_css(digest)
    write_codepoints_js(mapping)
    write_notice(material, hanken)
    print(f"adminui-fonts: wrote {FONTS_OUT} ({len(icons)} icons from icons.txt, bust={digest})")


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    sub = ap.add_subparsers(dest="cmd", required=True)
    p = sub.add_parser("check", help="verify generated assets match icons.txt")
    p.set_defaults(func=cmd_check)
    p = sub.add_parser("sync", help="subset fonts from icons.txt")
    p.set_defaults(func=cmd_sync)
    args = ap.parse_args()
    args.func(args)


if __name__ == "__main__":
    main()
