#!/usr/bin/env python3
"""Build the MkDocs source tree for the central NeKiro RepoWiki.

Tracked English and Chinese navigation pages live under repowiki/. Canonical
Core Markdown files under docs/ are copied into both locale trees at build
time. The Chinese source-document pages remain explicitly linked to the
English canonical text until an approved translation exists.
"""

from __future__ import annotations

import argparse
import posixpath
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
EXPECTED_CURATED = (
    "index.md",
    "architecture/index.md",
    "architecture/lifecycle.md",
    "contracts/index.md",
    "operations/index.md",
    "decisions/index.md",
    "repositories.md",
)


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


def source_site_path(path: Path) -> Path:
    return Path("source-docs") / path.relative_to(DOCS_ROOT)


def relative_source_link(source_path: Path, target_path: Path) -> str:
    source_site = source_site_path(source_path)
    target_site = source_site_path(target_path)
    return posixpath.relpath(
        target_site.as_posix(),
        start=source_site.parent.as_posix(),
    )


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
        link = relative_source_link(source_path, resolved)
        return f"[{match.group(1)}]({link}{fragment}{suffix})"

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


def source_page(document: Path, language: str) -> str:
    relative = document.relative_to(REPO_ROOT).as_posix()
    title = document_title(document)
    if language == "zh":
        banner = (
            '<div class="source-note">英文 canonical source：'
            f'<a href="{SOURCE_URL_PREFIX}{relative}"><code>{relative}</code></a>。'
            "本页保留英文规范正文，中文导航和摘要页已提供双语入口。</div>"
        )
        heading = f"# {title}（英文规范）"
    else:
        banner = (
            '<div class="source-note">Canonical source: '
            f'<a href="{SOURCE_URL_PREFIX}{relative}"><code>{relative}</code></a>. '
            "This page is rendered from the source document during the MkDocs build.</div>"
        )
        heading = f"# {title}"
    body = rewrite_links(without_title(document.read_text(encoding="utf-8")), document)
    return f"{banner}\n\n{heading}\n\n{body}"


def source_index(documents: list[Path], language: str) -> str:
    if language == "zh":
        lines = [
            "# 源文档",
            "",
            "以下页面由 docs/ 中的 Core canonical Markdown 文件生成。",
            "中文页面保留英文规范正文，避免在未审阅的机器翻译中改变契约语义。",
            "",
        ]
    else:
        lines = [
            "# Source documents",
            "",
            "These pages are generated from the canonical Markdown files under docs/.",
            "Edits belong in the source documents, not in the generated MkDocs tree.",
            "",
        ]
    for section in DOC_SECTIONS:
        lines.extend([f"## {section.title()}", ""])
        for document in documents:
            relative = document.relative_to(DOCS_ROOT)
            if relative.parts[0] != section:
                continue
            title = document_title(document)
            link = relative.as_posix()
            lines.append(f"- [{title}]({link})")
        lines.append("")
    return "\n".join(lines)


def copy_tracked_wiki(output: Path) -> None:
    for path in WIKI_ROOT.rglob("*"):
        if path.is_dir():
            continue
        relative = path.relative_to(WIKI_ROOT)
        if relative.parts[0] == "assets":
            destination = output / relative
        elif relative.parts[0] == "zh":
            destination = output / relative
        elif path.suffix == ".md":
            destination = output / "en" / relative
        else:
            fail(f"unsupported tracked RepoWiki file: {relative}")
        destination.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(path, destination)


def validate(documents: list[Path]) -> None:
    for relative in EXPECTED_CURATED:
        if not (WIKI_ROOT / relative).is_file():
            fail(f"missing English RepoWiki page: repowiki/{relative}")
        if not (WIKI_ROOT / "zh" / relative).is_file():
            fail(f"missing Chinese RepoWiki page: repowiki/zh/{relative}")
    if not (WIKI_ROOT / "assets/stylesheets/extra.css").is_file():
        fail("missing shared MkDocs stylesheet: repowiki/assets/stylesheets/extra.css")

    titles: set[str] = set()
    for document in documents:
        title = document_title(document)
        if title in titles:
            fail(f"duplicate source document title: {title}")
        titles.add(title)
        rewrite_links(document.read_text(encoding="utf-8"), document)

    for path in WIKI_ROOT.rglob("*.md"):
        text = path.read_text(encoding="utf-8")
        if "{{" in text or "relative_url" in text:
            fail(f"Jekyll/Liquid link remains in MkDocs source: {path.relative_to(REPO_ROOT)}")


def build(output: Path, documents: list[Path]) -> None:
    if output.exists():
        if output.is_dir():
            shutil.rmtree(output)
        else:
            output.unlink()
    output.mkdir(parents=True, exist_ok=True)
    copy_tracked_wiki(output)

    for language in ("en", "zh"):
        generated_root = output / language / "source-docs"
        generated_root.mkdir(parents=True, exist_ok=True)
        (generated_root / "index.md").write_text(
            source_index(documents, language),
            encoding="utf-8",
        )
        for document in documents:
            destination = generated_root / document.relative_to(DOCS_ROOT)
            destination.parent.mkdir(parents=True, exist_ok=True)
            destination.write_text(source_page(document, language), encoding="utf-8")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--output",
        type=Path,
        default=REPO_ROOT / ".repowiki-site",
        help="generated MkDocs docs directory (default: .repowiki-site)",
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
            print(f"RepoWiki check passed: {len(documents)} Core source documents, 2 locales")
        else:
            output = args.output if args.output.is_absolute() else REPO_ROOT / args.output
            build(output, documents)
            print(f"MkDocs source generated: {output}")
    except ValueError as error:
        print(f"RepoWiki build failed: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
