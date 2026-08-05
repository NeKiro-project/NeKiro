#!/usr/bin/env python3
"""Refresh read-only satellite documentation snapshots for the central RepoWiki.

The satellite repositories remain the canonical owners. This script copies only
their published Markdown documentation at explicit immutable revisions into the
central reading surface. It never copies satellite source code and is not part
of the Pages build; run it deliberately when a snapshot should be refreshed.
"""

from __future__ import annotations

import argparse
import posixpath
import re
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[1]
DEFAULT_OUTPUT = REPO_ROOT / "repowiki" / "satellites"
MARKDOWN_LINK = re.compile(r"(?<!!)(\[[^\]]+\])\(([^)]+)\)")

# Go module zip paths reject Unicode punctuation such as full-width parentheses
# and colons. Keep the canonical satellite filenames unchanged, but publish
# these two mirrored documents under stable ASCII-only target paths.
CONSOLE_KNOWLEDGE_SAFE_NAMES = {
    "前端依赖管理（npm + Vite）": "frontend-dependency-management-npm-vite",
    "前端错误处理：NekiroApiError 与 PlatformErrorView 统一模型": (
        "frontend-error-handling-nekiro-api-error-platform-error-view"
    ),
}


@dataclass(frozen=True)
class Satellite:
    slug: str
    title: str
    repo: str
    revision: str
    files: tuple[str, ...]


SATELLITES = (
    Satellite(
        slug="console",
        title="NeKiro Console",
        repo="NeKiro-project/NeKiro-Console",
        revision="5e577d86825e2ff80434e752342990b9b313d947",
        files=(
            "README.md",
            ".qoder/repowiki/zh/content/API 参考.md",
            ".qoder/repowiki/zh/content/开发指南.md",
            ".qoder/repowiki/zh/content/快速开始.md",
            ".qoder/repowiki/zh/content/技术架构/API 集成层.md",
            ".qoder/repowiki/zh/content/技术架构/前端架构设计.md",
            ".qoder/repowiki/zh/content/技术架构/技术架构.md",
            ".qoder/repowiki/zh/content/技术架构/数据模型设计.md",
            ".qoder/repowiki/zh/content/技术架构/样式架构.md",
            ".qoder/repowiki/zh/content/故障排除.md",
            ".qoder/repowiki/zh/content/核心功能/数据账本系统.md",
            ".qoder/repowiki/zh/content/核心功能/服务器安装管理.md",
            ".qoder/repowiki/zh/content/核心功能/服务调用监控.md",
            ".qoder/repowiki/zh/content/核心功能/核心功能.md",
            ".qoder/repowiki/zh/content/核心功能/注册表管理.md",
            ".qoder/repowiki/zh/content/部署指南.md",
            ".qoder/repowiki/zh/content/项目概述.md",
            ".qoder/repowiki/knowledge/zh/NeKiro Console 前端工程根 (React + Vite)/技术栈.md",
            ".qoder/repowiki/knowledge/zh/NeKiro Console 前端工程根 (React + Vite)/架构设计.md",
            ".qoder/repowiki/knowledge/zh/NeKiro Console 前端工程根 (React + Vite)/概述.md",
            ".qoder/repowiki/knowledge/zh/NeKiro Console 前端工程根 (React + Vite)/特殊配置与命令.md",
            ".qoder/repowiki/knowledge/zh/NeKiro Console 前端工程根 (React + Vite)/编码规范.md",
            ".qoder/repowiki/knowledge/zh/Tailwind v4 + 暗色玻璃拟态设计系统/Tailwind v4 + 暗色玻璃拟态设计系统.md",
            ".qoder/repowiki/knowledge/zh/Vite + TypeScript 前端构建系统/Vite + TypeScript 前端构建系统.md",
            ".qoder/repowiki/knowledge/zh/Vite 环境变量驱动的浏览器运行时配置系统/Vite 环境变量驱动的浏览器运行时配置系统.md",
            ".qoder/repowiki/knowledge/zh/前端依赖管理（npm + Vite）/前端依赖管理（npm + Vite）.md",
            ".qoder/repowiki/knowledge/zh/前端错误处理：NekiroApiError 与 PlatformErrorView 统一模型/前端错误处理：NekiroApiError 与 PlatformErrorView 统一模型.md",
        ),
    ),
    Satellite(
        slug="sdk-go",
        title="nekiro-sdk-go",
        repo="NeKiro-project/nekiro-sdk-go",
        revision="0bc1bd0495ef877f8583301d6ba8ff128d6cae5f",
        files=("README.md", "agent/README.md", "client/README.md"),
    ),
    Satellite(
        slug="samples",
        title="NeKiro Samples",
        repo="NeKiro-project/NeKiro-Samples",
        revision="89bf743604ddafb77688b22f4fb6e20577a85f3a",
        files=("README.md", "runtime-a/README.md", "runtime-b/README.md"),
    ),
    Satellite(
        slug="stack",
        title="NeKiro Stack",
        repo="NeKiro-project/NeKiro-Stack",
        revision="20a6f36f5b78d8b259d8b6d430ebfaf176ff6ee4",
        files=("README.md",),
    ),
    Satellite(
        slug="a2a-transport-go",
        title="nekiro-a2a-transport-go",
        repo="NeKiro-project/nekiro-a2a-transport-go",
        revision="71fb8ee839be4311b6fd8350274ddf098cad4d5b",
        files=("README.md",),
    ),
)


def raw_url(repo: str, revision: str, source: str) -> str:
    encoded = urllib.parse.quote(source, safe="/")
    return f"https://raw.githubusercontent.com/{repo}/{revision}/{encoded}"


def source_url(repo: str, revision: str, source: str) -> str:
    encoded = urllib.parse.quote(source, safe="/")
    return f"https://github.com/{repo}/blob/{revision}/{encoded}"


def target_path(satellite: Satellite, source: str) -> Path:
    source_path = Path(source)
    if source_path.name == "README.md":
        # MkDocs treats README.md as an index page. Use a distinct filename
        # because Windows also treats README.md and readme.md as the same path.
        return source_path.with_name("readme-page.md")
    if satellite.slug == "console":
        if source.startswith(".qoder/repowiki/zh/content/"):
            return Path("content") / source_path.relative_to(
                ".qoder/repowiki/zh/content"
            )
        if source.startswith(".qoder/repowiki/knowledge/zh/"):
            relative = source_path.relative_to(".qoder/repowiki/knowledge/zh")
            safe_name = CONSOLE_KNOWLEDGE_SAFE_NAMES.get(relative.parts[0])
            if safe_name is not None:
                return Path("knowledge") / safe_name / f"{safe_name}{source_path.suffix}"
            return Path("knowledge") / relative
    return source_path


def fetch(url: str) -> str:
    request = urllib.request.Request(
        url,
        headers={"User-Agent": "NeKiro-RepoWiki-snapshot/1.0"},
    )
    try:
        with urllib.request.urlopen(request, timeout=30) as response:
            return response.read().decode("utf-8")
    except (urllib.error.HTTPError, urllib.error.URLError, UnicodeDecodeError) as error:
        raise RuntimeError(f"failed to fetch satellite document {url}: {error}") from error


def mirror_header(satellite: Satellite, source: str) -> str:
    url = source_url(satellite.repo, satellite.revision, source)
    return (
        '<div class="satellite-source-note">Read-only mirror of '
        f'<a href="{url}"><code>{satellite.repo}/{source}</code></a> at '
        f'<code>{satellite.revision}</code>. The canonical document remains in '
        "the satellite repository; edit it there and refresh this snapshot.</div>\n\n"
    )


def rewrite_links(satellite: Satellite, source: str, text: str) -> str:
    mirrored_sources = set(satellite.files)
    source_posix = source.replace("\\", "/")

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
        is_file_uri = target.startswith("file://")
        if is_file_uri:
            target = target.removeprefix("file://")
        if not is_file_uri and not target.endswith(".md"):
            return match.group(0)

        if is_file_uri:
            resolved = posixpath.normpath(target)
        else:
            resolved = posixpath.normpath(
                posixpath.join(posixpath.dirname(source_posix), target)
            )

        if resolved in mirrored_sources and resolved.endswith(".md"):
            current_target = target_path(satellite, source_posix).as_posix()
            mirrored_target = target_path(satellite, resolved).as_posix()
            relative = posixpath.relpath(
                mirrored_target,
                posixpath.dirname(current_target),
            )
            link = urllib.parse.quote(relative, safe="/") + fragment
        else:
            link = source_url(satellite.repo, satellite.revision, resolved) + fragment
        return f"{match.group(1)}({link}{suffix})"

    return MARKDOWN_LINK.sub(replace, text)


def snapshot_index(satellite: Satellite) -> str:
    lines = [
        f"# {satellite.title}",
        "",
        "This is a read-only documentation snapshot in the central NeKiro RepoWiki.",
        f"The source repository is [{satellite.repo}](https://github.com/{satellite.repo}) "
        f"at commit `{satellite.revision}`.",
        "",
        "Edit the canonical repository and run "
        "`python scripts/sync_repowiki_satellites.py` to refresh this mirror.",
        "",
        "## Mirrored documents",
        "",
    ]
    for source in satellite.files:
        target = target_path(satellite, source)
        link = urllib.parse.quote(target.as_posix(), safe="/")
        lines.append(f"- [{target.as_posix()}]({link})")
    lines.append("")
    return "\n".join(lines)


def sync(output: Path) -> int:
    count = 0
    for satellite in SATELLITES:
        root = output / satellite.slug
        root.mkdir(parents=True, exist_ok=True)
        for source in satellite.files:
            destination = root / target_path(satellite, source)
            if Path(source).name == "README.md":
                stale = root / Path(source)
                if stale != destination and stale.is_file():
                    stale.unlink()
            destination.parent.mkdir(parents=True, exist_ok=True)
            content = fetch(raw_url(satellite.repo, satellite.revision, source))
            content = rewrite_links(satellite, source, content)
            destination.write_text(
                mirror_header(satellite, source) + content,
                encoding="utf-8",
            )
            count += 1
        (root / "index.md").write_text(snapshot_index(satellite), encoding="utf-8")
    return count


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--output",
        type=Path,
        default=DEFAULT_OUTPUT,
        help=f"snapshot directory (default: {DEFAULT_OUTPUT})",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    output = args.output if args.output.is_absolute() else REPO_ROOT / args.output
    try:
        count = sync(output)
    except RuntimeError as error:
        print(f"Satellite RepoWiki sync failed: {error}")
        return 1
    print(f"Satellite documentation snapshot refreshed: {count} documents")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
