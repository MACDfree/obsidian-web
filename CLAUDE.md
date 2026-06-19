# CLAUDE.md

本文件为 Claude Code (claude.ai/code) 在本代码库工作时提供指导。

## 项目概述

obsidian-web 是一个将 Obsidian 笔记库发布为可浏览网站的 Go Web 应用。它支持公开和私有（密码保护）笔记。主要使用场景是在移动设备上阅读 PC 创建的笔记。代码库中的注释和 UI 文本使用中文。

## 构建与运行

```bash
go build                    # 生成 ./obsidian-web 二进制文件
go run main.go              # 直接运行
./obsidian-web -pre "git pull"  # 运行前执行预启动脚本
```

- 需要**启用 CGO**（使用了 `mattn/go-sqlite3`）
- 配置从工作目录的 `config.yml` 加载
- 本项目没有测试、Makefile 或 linter 配置

## 架构

**启动流程** (`main.go` → `noteloader.Load()` → `server.NewRouter()` → `job.Start()` → `r.Run()`)：
1. 可选的预启动脚本（例如 `git pull` 同步笔记库）
2. 笔记库扫描：遍历 Obsidian 笔记库目录，解析 `.md` 文件的 YAML frontmatter，将笔记/附件/wikilink 索引到 SQLite (`tmp.db`)
3. Gin HTTP 服务器及中间件链
4. Cron 调度器启动（每天 04:12 执行 git 拉取）

**请求流程**：Gin 中间件链为 `logger → session → auth → error handler`。Auth 中间件在 context 中设置 `isLogin`；处理器检查笔记的发布状态以实施访问控制。

**核心包**：
- `config/` — 配置单例，通过 `sync.Once` 从 `config.yml` 加载
- `db/` — GORM 模型（`Note`、`AttachInfo`、`NoteAttachment`、`NoteLink`）及所有数据库查询
- `handler/` — Gin HTTP 处理器，处理笔记、附件、认证、git 拉取和静态文件
- `mdparser/` — Goldmark markdown 处理流程，支持 wikilink 解析、语法高亮（Chroma/Monokai）、MathJax 和自定义图片渲染
- `noteloader/` — 笔记库扫描器，解析 frontmatter，构建 SQLite 索引
- `server/` — 路由设置、中间件注册、模板加载
- `middleware/` — 认证、会话（内存 `memstore`）、日志、错误页面
- `job/` — Cron 定时任务
- `logger/` — Zap 结构化日志，支持文件轮转

**访问控制**：frontmatter 中 `publish: true` 的笔记为公开笔记；其他需要登录。附件继承引用笔记的发布状态。

**Markdown 处理流程**：Goldmark + GFM，`[[wikilink]]` 解析（→ `/note/` 或 `/attachment/`）、Chroma 语法高亮、可选 MathJax、YAML frontmatter 提取、纯文件名图片路径自动添加前缀。

**模板**：Go `html/template`，使用 `multitemplate` 进行布局组合（`base.html` + 视图模板）。提供 Sprig 函数和 `SafeHTML` 辅助函数。

## 配置项（`config.yml`）

- `note_path` — Obsidian 笔记库的绝对路径
- `ignore_paths` — 扫描时跳过的目录名
- `attachment_path` — 附件的子目录名
- `paginate` — 每页笔记数（默认：20）
- `title` — 网站标题
- `password` — 私有笔记的密码（空 = 无需认证）
- `bind_addr` — 服务器绑定地址（默认：`127.0.0.1:8888`）

## 约定

- 提交信息遵循 conventional-commit 风格：`feat(scope):`、`fix(scope):`、`refactor:` 等
- Session 密钥在启动时随机生成（内存存储，重启后会话丢失）
- Dockerfile 使用多阶段构建，启用 UPX 压缩，使用中国镜像源和 Asia/Shanghai 时区
- CI/CD 通过 Gitea Actions（`.gitea/workflows/`）—— 在 tag 推送时构建 Docker 镜像
