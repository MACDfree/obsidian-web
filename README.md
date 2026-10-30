# obsidian-web

obsidian-web的功能是将Obsidian笔记发布为网站。特点是可以将公开和隐私笔记统一发布，对于隐私笔记可以进行密码保护。

## 初衷

在本人的笔记工作流中，笔记的创建和维护都是在PC上完成，而笔记阅读使用则有移动端需求。在这样的流程中，移动端需要的功能就只是笔记的查询和展示。因此，obsidian-web就应运而生了。

## Obsidian 语法兼容

支持 GFM、`[[笔记|显示文字]]`、标题和块链接、反向链接，以及常见 Obsidian 内容：

- 文本高亮：`==高亮内容==`。
- Callout：`> [!note] 标题`，`+` 和 `-` 控制初始展开状态。
- 笔记嵌入：`![[笔记]]`、`![[笔记#标题]]`、`![[笔记#^块ID]]`。支持段落、列表项以及以独立块 ID 标记的引用块、代码块和表格；同一标题多次出现时链接到第一个。
- 附件嵌入：图片可使用 `![[图片.png|300]]` 或 `![[图片.png|300x200]]`；支持 PDF（如 `![[文档.pdf#page=2]]`）、MP3、WAV、MP4 和 WebM。附件必须位于配置的附件目录，且要被可访问的笔记引用。
- Mermaid：使用 `mermaid` 代码块；渲染库从 jsDelivr 加载，加载失败时显示原始代码。
- 公式：在笔记 frontmatter 中设置 `mathjax: true`；MathJax 从 jsDelivr 加载。

wikilink 和 Markdown 笔记链接的正文、嵌入及反向链接使用同一套查找规则：带目录的链接优先从笔记库根目录查找；显式 `./` 或 `../` 从当前笔记所在目录查找；不带目录的名称优先查当前目录，然后按完整路径、标题、文件名或别名查找。同一优先级有多个候选时不会随机选择。外部网址、网站绝对路径和已识别的附件链接保持原有处理；未解析的笔记链接会以虚线标识。索引在服务启动或笔记库重新扫描时重建。

嵌入笔记需要浏览器启用 JavaScript。匿名访问仅能读取公开笔记和公开附件，嵌入检查访问权限并限制循环引用。相同文件名的多个附件应使用相对附件目录的完整路径。其他 Obsidian 插件语法及 Canvas、Bases 等专有格式尚未支持。

## Excalidraw 只读预览

确保 `config.yml` 的 `ignore_paths` 没有排除画布所在目录。将 Obsidian Excalidraw 插件保存的画布 Markdown 文件（`.excalidraw.md` 或 frontmatter 含 `excalidraw-plugin` 的 `.md`）放入笔记库并按普通笔记设置 `publish: true`，即可打开画布；在其他笔记中可以使用 `![[画布.excalidraw]]` 嵌入。支持 JSON 和 `compressed-json`，提供放大、缩小和重置。画布继承自身笔记的权限：公开页面嵌入画布时，画布文件也需要设置 `publish: true`，否则仅登录后可见。支持画布 JSON 中以 data URL 内嵌的图片。

预览为只读 SVG，依赖 esm.sh 加载 Excalidraw 和 LZ-String；脱网时无法加载渲染库。画布引用的 Obsidian 外部文件、画布内嵌笔记和旧版独立 `.excalidraw` 文件暂不支持，若画布依赖外部文件，可先在 Obsidian 中导出 SVG/PNG 作为附件发布。
