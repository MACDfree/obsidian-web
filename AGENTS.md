# AGENTS.md

本文件为 Codex (Codex.ai/code) 在本代码库工作时提供指导。

## 项目概述

obsidian-web 是一个将 Obsidian 笔记库发布为可浏览网站的 Go Web 应用。它支持公开和私有（密码保护）笔记，并提供可选的评论系统。主要使用场景是在移动设备上阅读 PC 创建的笔记。代码库中的注释和 UI 文本使用中文。

## 构建与运行

```bash
go build                    # 生成 ./obsidian-web 二进制文件
go run main.go              # 直接运行
./obsidian-web -gitpull     # 启动前通过 go-git 拉取笔记库（需要配置 git_url）
```

- 需要**启用 CGO**（使用了 `mattn/go-sqlite3`）
- 配置从工作目录的 `config.yml` 加载
- 本项目没有测试、Makefile 或 linter 配置

## 架构

**启动流程** (`main.go` → `flag.Parse()` → `gitutil.Pull()` → `noteloader.Load()` → `db.InitCommentDB()` → `server.NewRouter()` → `job.Start()` → `r.Run()`)：
1. 如果设置了 `-gitpull` 标志且配置了 `git_url`，通过 go-git 拉取（或克隆）笔记库
2. 笔记库扫描 (`noteloader.Load()`)：遍历 Obsidian 笔记库目录，解析 `.md` 文件的 YAML frontmatter，将笔记/附件/wikilink 索引到 SQLite (`tmp.db`)，并构建笔记之间的链接关系
3. 初始化评论数据库 (`db.InitCommentDB()`)：创建 `data/comments.db` 并迁移 `Comment` 表结构
4. Gin HTTP 服务器及中间件链 (`server.NewRouter()`)
5. Cron 调度器启动 (`job.Start()`) — 每天 04:12 执行 go-git 拉取

**请求流程**：Gin 中间件链为 `gin.Recovery() → Logger → Session("webauth") → Auth → Error`。Auth 中间件在 context 中设置 `isLogin`；处理器检查笔记的发布状态以实施访问控制。

**核心包**：
- `config/` — 配置单例，通过 `sync.Once` 从 `config.yml` 加载
- `db/` — GORM 模型（`Note`、`AttachInfo`、`NoteAttachment`、`NoteLink`、`Comment`）及所有数据库查询。还包含 `BacklinkInfo`、`TagCount` 响应类型和敏感词黑名单（`IsContentBlocked`）
- `handler/` — Gin HTTP 处理器，处理笔记、标签、搜索、附件、认证、评论（增删改查+管理）、git 拉取和静态文件
- `mdparser/` — Goldmark markdown 处理流程，支持 wikilink 解析、语法高亮（Chroma/Monokai）、MathJax 和自定义图片渲染
- `noteloader/` — 笔记库扫描器，解析 frontmatter，构建 SQLite 索引，创建笔记链接和笔记附件关系
- `server/` — 路由设置、中间件注册、模板加载（使用 Sprig funcmap 的 multitemplate）
- `middleware/` — 认证、会话（内存 `memstore`）、日志（Gin formatter）、错误页面
- `job/` — Cron 定时任务（每日 go-git 拉取）
- `gitutil/` — go-git 封装：pull、clone、status 文本和 `IsConfigured()` 检查
- `logger/` — Zap 结构化日志，支持文件轮转（`logs/app.log`）

**访问控制**：frontmatter 中 `publish: true` 的笔记为公开笔记；其他需要登录。通过 `NoteAttachment` 连接表检查，附件继承引用笔记的发布状态。

**评论系统**：支持嵌套评论（父子树结构）、基于 IP 的限流（10 分钟内 3 条）、敏感词黑名单过滤和管理员管理（隐藏/删除）。评论存储在独立的 SQLite 数据库（`data/comments.db`）中。

**Markdown 处理流程**：Goldmark + GFM，`[[wikilink]]` 解析（→ 根据文件扩展名解析为 `/note/` 或 `/attachment/`）、Chroma 语法高亮（Monokai，行号）、可选 MathJax（通过 frontmatter 中的 `mathjax: true` 启用）、YAML frontmatter 提取、纯文件名图片路径自动添加前缀。

**反向链接**：通过 `NoteLink` 跟踪笔记间的 wikilink。`GetBacklinks()` 返回所有链接到指定笔记的笔记，对于未登录用户按发布状态过滤。

**搜索**：全文搜索通过正则表达式扫描可访问笔记的标题和文件内容，返回高亮匹配结果。

**模板**：Go `html/template`，使用 `multitemplate` 进行布局组合（`base.html` + 视图模板 + 部分模板）。提供 Sprig 函数和 `SafeHTML` 辅助函数。

## 路由

| 方法 | 路径 | 描述 |
|--------|------|-------------|
| GET | `/` | 笔记列表（分页） |
| GET | `/page/:num` | 笔记列表第 N 页 |
| GET | `/tag/*tag` | 标签列表或按标签筛选笔记 |
| GET | `/note/*path` | 笔记详情页 |
| GET | `/search` | 搜索笔记 |
| GET | `/attachment/:attach` | 提供附件文件 |
| GET | `/auth` | 登录页面 |
| POST | `/auth` | 登录操作 |
| GET | `/gitpull` | Git 拉取页面（需要登录） |
| POST | `/gitpull` | 执行 git 拉取（需要登录） |
| GET | `/api/comments` | 获取笔记的评论列表 |
| POST | `/api/comments` | 创建评论 |
| POST | `/api/comments/:id/delete` | 隐藏评论（需要登录） |
| GET | `/admin/comments` | 评论管理页面（需要登录） |
| GET | `/assets/*` | 静态资源目录 |
| GET | `/:static` | 来自 `static/` 的静态文件 |

## 配置项（`config.yml`）

- `note_path` — Obsidian 笔记库的绝对路径
- `ignore_paths` — 扫描时跳过的目录名
- `attachment_path` — 附件的子目录名
- `paginate` — 每页笔记数（默认：20）
- `title` — 网站标题
- `password` — 私有笔记的密码（空 = 无需认证）
- `bind_addr` — 服务器绑定地址（默认：`127.0.0.1:8888`）
- `git_url` — go-git 拉取/克隆用的 Git 仓库 URL（可选，启用 `-gitpull` 标志和 cron 任务）
- `git_token` — 私有 Git 仓库的认证令牌（可选）

## 约定

- 提交信息遵循 conventional-commit 风格：`feat(scope):`、`fix(scope):`、`refactor:` 等
- Session 密钥在启动时随机生成（内存存储，重启后会话丢失）
- 两个独立的 SQLite 数据库：`tmp.db`（笔记/附件/链接）和 `data/comments.db`（评论），均使用 WAL 模式
- Dockerfile 使用多阶段构建，启用 UPX 压缩，使用中国镜像源和 Asia/Shanghai 时区
- CI/CD 通过 Gitea Actions（`.gitea/workflows/`）—— 在 tag 推送时构建 Docker 镜像
- Git 操作使用 go-git 库（`github.com/go-git/go-git/v6`），而非 shell 命令
