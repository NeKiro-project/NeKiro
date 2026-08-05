# 卫星仓库文档

这里是 NeKiro 卫星仓库文档的统一阅读入口。以下页面是固定 commit 的只读快照；
卫星仓库仍是 canonical owner，Core RepoWiki 不建立第二个可写事实源。

需要在审阅上游变更后刷新快照时运行：

```text
python scripts/sync_repowiki_satellites.py
```

## 已整合仓库

| 仓库 | Core 入口 | 快照内容 | 快照时 Wiki 状态 |
| --- | --- | --- | --- |
| **NeKiro Console** | [Console 文档](console/index.md) | `5e577d8` 的 26 个 `.qoder/repowiki` Markdown 页面和 `README.md` | GitHub Wiki 已关闭，使用仓库内已提交的 RepoWiki 导出 |
| **nekiro-sdk-go** | [Go SDK 文档](sdk-go/index.md) | `0bc1bd0` 的 `README.md`、`agent/README.md`、`client/README.md` | Wiki 已启用，但没有可读取的 Wiki git 页面 |
| **NeKiro Samples** | [Samples 文档](samples/index.md) | `89bf743` 的根 README 和两个 Runtime README | Wiki 已启用，但没有可读取的 Wiki git 页面 |
| **NeKiro Stack** | [Stack 文档](stack/index.md) | `20a6f36` 的 `README.md` | Wiki 已启用，但没有可读取的 Wiki git 页面 |
| **nekiro-a2a-transport-go** | [A2A transport 文档](a2a-transport-go/index.md) | `71fb8ee` 的 `README.md` | Wiki 已启用，但没有可读取的 Wiki git 页面 |

“没有 Wiki git 页面”表示公开的 `<repository>.wiki.git` 端点没有仓库；Core
不会因此虚构替代 owner。上游 Wiki 初始化并完成审阅后，再把页面和 revision
加入同步清单；在此之前，上述仓库内 Markdown 是可用的 canonical 文档。

## 所有权与变更规则

- Console、SDK、Samples、Stack 和 transport 事实在各自仓库维护。
- 中央 RepoWiki 负责跨仓库导航和只读快照。
- 每个快照页面标注来源仓库和完整 commit revision。
- 不把卫星源码、凭证、构建产物或可变发布状态复制进 Core。
