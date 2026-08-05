#!/usr/bin/env python3
"""Build the Jekyll source tree for the central NeKiro RepoWiki.

The tracked repowiki/ directory contains curated navigation pages and the
site shell. Canonical Core Markdown files under docs/ are copied into the
generated tree at build time, so the Wiki cannot silently drift from the
source documents. Satellite repositories are represented by links only; their
source remains owned by those repositories.
"""

from __future__ import annotations

import argparse
import json
import re
import shutil
import sys
from pathlib import Path
from typing import NoReturn


REPO_ROOT = Path(__file__).resolve().parents[1]
DOCS_ROOT = REPO_ROOT / "docs"
WIKI_ROOT = REPO_ROOT / "repowiki"
DOC_SECTIONS = ("architecture", "contracts", "decisions", "usage")
SOURCE_URL_PREFIX = "https://github.com/NeKiro-project/NeKiro/blob/main/"
MARKDOWN_LINK = re.compile(r"(?<!!)\[([^\]]+)\]\(([^)]+)\)")
H1 = re.compile(r"^#\s+(.+?)\s*$")


def fail(message: str) -> NoReturn:
    raise ValueError(message)


def source_documents() -> list[Path]:
    documents: list[Path] = []
    for section in DOC_SECTIONS:
        section_root = DOCS_ROOT / section
        if not section_root.is_dir():
            fail(f"missing documentation section: {section_root}")
        documents.extend(sorted(section_root.glob("*.md")))
    if not documents:
        fail("no Core source documents found")
    return documents


def document_title(path: Path) -> str:
    for line in path.read_text(encoding="utf-8").splitlines():
        match = H1.match(line)
        if match:
            return match.group(1).strip()
    fail(f"source document has no level-one heading: {path.relative_to(REPO_ROOT)}")


def site_slug(path: Path) -> str:
    relative = path.relative_to(DOCS_ROOT).with_suffix("")
    return f"/source-docs/{relative.as_posix()}/"


def liquid_link(path: str, fragment: str = "") -> str:
    return "{{ '%s' | relative_url }}%s" % (path, fragment)


def rewrite_links(text: str, source_path: Path) -> str:
    def replace(match: re.Match[str]) -> str:
        destination = match.group(2).strip()
        if destination.startswith(("http://", "https://", "mailto:", "#", "<")):
            return match.group(0)

        parts = destination.split(None, 1)
        target = parts[0]
        suffix = f" {parts[1]}" if len(parts) == 2 else ""
        fragment = ""
        if "#" in target:
            target, raw_fragment = target.split("#", 1)
            fragment = f"#{raw_fragment}"
        if not target.endswith(".md"):
            return match.group(0)

        resolved = (source_path.parent / target).resolve()
        try:
            resolved.relative_to(DOCS_ROOT.resolve())
        except ValueError:
            fail(
                f"internal link escapes docs/: {source_path.relative_to(REPO_ROOT)} -> {target}"
            )
        if not resolved.is_file():
            fail(
                f"internal link target does not exist: {source_path.relative_to(REPO_ROOT)} -> {target}"
            )
        return f"[{match.group(1)}]({liquid_link(site_slug(resolved), fragment)}{suffix})"

    return MARKDOWN_LINK.sub(replace, text)


def without_title(text: str) -> str:
    lines = text.splitlines()
    for index, line in enumerate(lines):
        if not line.strip():
            continue
        if H1.match(line):
            del lines[index]
            if index < len(lines) and not lines[index].strip():
                del lines[index]
            return "\n".join(lines).lstrip() + "\n"
        break
    return text


def page_front_matter(title: str, source_path: Path) -> str:
    relative = source_path.relative_to(REPO_ROOT).as_posix()
    title_value = json.dumps(title, ensure_ascii=False)
    return "\n".join(
        [
            "---",
            "layout: default",
            f"title: {title_value}",
            f"source_path: {relative}",
            f"permalink: {site_slug(source_path)}",
            "---",
            "",
            '<div class="source-note">Canonical source: '
            f'<a href="{SOURCE_URL_PREFIX}{relative}"><code>{relative}</code></a>. '
            "This page is rendered from the source document during the Pages build.</div>",
            "",
            "",
            f"# {title}",
            "",
        ]
    )


def source_index(documents: list[Path]) -> str:
    lines = [
        "---",
        "layout: default",
        "title: Source documents",
        "description: Core source documents rendered into the central RepoWiki.",
        "permalink: /source-docs/",
        "nav_order: 8",
        "---",
        "",
        "# Source documents",
        "",
        "These pages are generated from the canonical Markdown files under docs/.",
        "The generated pages are a reading surface; edits belong in the source files.",
        "",
    ]
    for section in DOC_SECTIONS:
        lines.extend([f"## {section.title()}", ""])
        for document in documents:
            relative = document.relative_to(DOCS_ROOT)
            if relative.parts[0] != section:
                continue
            title = document_title(document)
            lines.append(f"- [{title}]({{{{ '{site_slug(document)}' | relative_url }}}})")
        lines.append("")
    return "\n".join(lines)


def validate(documents: list[Path]) -> None:
    expected = {
        "_config.yml",
        "_layouts/default.html",
        "assets/style.css",
        "index.md",
        "architecture/index.md",
        "architecture/lifecycle.md",
        "contracts/index.md",
        "operations/index.md",
        "decisions/index.md",
        "repositories.md",
    }
    missing = sorted(path for path in expected if not (WIKI_ROOT / path).is_file())
    if missing:
        fail("missing tracked RepoWiki files: " + ", ".join(missing))

    titles: set[str] = set()
    for document in documents:
        title = document_title(document)
        if title in titles:
            fail(f"duplicate source document title: {title}")
        titles.add(title)
        rewrite_links(document.read_text(encoding="utf-8"), document)


def build(output: Path, documents: list[Path]) -> None:
    if output.exists():
        if output.is_dir():
            shutil.rmtree(output)
        else:
            output.unlink()
    shutil.copytree(WIKI_ROOT, output)

    generated_root = output / "source-docs"
    generated_root.mkdir(parents=True, exist_ok=True)
    (generated_root / "index.md").write_text(source_index(documents), encoding="utf-8")

    for document in documents:
        destination = generated_root / document.relative_to(DOCS_ROOT)
        destination = destination.with_suffix(".md")
        destination.parent.mkdir(parents=True, exist_ok=True)
        source = document.read_text(encoding="utf-8")
        body = rewrite_links(without_title(source), document)
        destination.write_text(
            page_front_matter(document_title(document), document) + body,
            encoding="utf-8",
        )


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--output",
        type=Path,
        default=REPO_ROOT / ".repowiki-site",
        help="generated Jekyll source directory (default: .repowiki-site)",
    )
    parser.add_argument(
        "--check",
        action="store_true",
        help="validate source documents and tracked Wiki inputs without writing output",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    try:
        documents = source_documents()
        validate(documents)
        if args.check:
            print(f"RepoWiki check passed: {len(documents)} Core source documents")
        else:
            output = args.output if args.output.is_absolute() else REPO_ROOT / args.output
            build(output, documents)
            print(f"RepoWiki source generated: {output}")
    except ValueError as error:
        print(f"RepoWiki build failed: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
