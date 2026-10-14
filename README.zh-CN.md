# AI Vault

[日本語](README.ja.md) | [English](README.md) | [한국어](README.ko.md) | [简体中文](README.zh-CN.md)

**面向 Codex、Claude 等 AI 客户端的轻量本地 AI 记忆与 Markdown 知识库。** 无界面的 MCP 服务器和 JSON CLI，以单个可执行文件运行于 Windows、Linux 和 macOS。

支持 Obsidian 风格的 `[[链接]]`、反向链接、标签、搜索和笔记历史，无需编辑器、数据库或常驻 Web 服务器。笔记保存在普通的 UTF-8 Markdown 文件中。运行发行版无需 API 密钥、账户，也无需安装 Git、Node.js、Python 或 Go。

笔记支持日语、英语、韩语、中文及混合语言内容，AI Vault 会保留原文，不会自动翻译。使用指南提供四种语言；主 README、内部元数据、CLI 帮助、命令参数、JSON 字段和配置键统一使用英语。

## 把这个 URL 发给你的 AI

仓库：**<https://github.com/uyuk216/ai-vault>**

将以下请求发送给能够在本机运行终端命令的编程 AI：

```text
请在这台电脑上安装 https://github.com/uyuk216/ai-vault。
先阅读 README 和安装脚本，下载适合当前操作系统和 CPU 的发行版，
验证 SHA-256，并将 ~/AI-Vault 作为所有本地 AI 共用的笔记目录。
使用安装器的 auto-connect 选项连接已安装的 Codex 和 Claude Code。
对于其他本地 MCP 客户端，使用生成的绝对路径配置，保留无关的现有设置。
验证 CLI 和 MCP 工具能列出笔记、创建一个小型测试笔记、读取它，
并将该测试笔记移至回收站。说明哪些客户端仍需重启或手动配置。
```

仓库 URL 为 AI 提供安装说明和下载入口，但并不是适用于所有客户端的通用 MCP 安装 URL。AI 需要访问本机终端并获得配置客户端的权限。仅提供聊天功能或完全托管在云端的客户端，无法仅凭收到链接就启动本地程序。

## 安装

预编译发行版支持 **Windows / Linux / macOS × x86-64 / ARM64**。Linux 需要常规 Linux 运行环境。这是原生可执行程序，不是浏览器或手机应用。

### macOS / Linux

需要 `curl`、`tar`，以及一种 SHA-256 工具（`sha256sum`、`shasum` 或 `openssl`）。macOS 和常见 Linux 环境通常已包含这些工具。

```sh
curl -fsSL https://raw.githubusercontent.com/uyuk216/ai-vault/v0.1.0/install.sh | sh -s -- --version v0.1.0 --auto-connect
```

程序安装到 `~/.local/bin/aivault`，笔记保存在 `~/AI-Vault`，无需管理员权限。可以直接运行 `~/.local/bin/aivault`，也可以将 `~/.local/bin` 加入 shell 的 `PATH`。

### Windows PowerShell

需要 Windows PowerShell 5.1 或 PowerShell 7。使用普通用户权限运行：

```powershell
& ([scriptblock]::Create((Invoke-WebRequest -UseBasicParsing 'https://raw.githubusercontent.com/uyuk216/ai-vault/v0.1.0/install.ps1').Content)) -Version v0.1.0 -AutoConnect
```

程序安装到 `%LOCALAPPDATA%\AI-Vault\bin\aivault.exe`，笔记保存在 `%USERPROFILE%\AI-Vault`。使用安装器输出的程序路径，或将对应目录加入用户 `PATH`。

上述命令执行指定标签下的安装脚本。若希望先检查脚本，请先下载该标签的 `install.sh` 或 `install.ps1`，检查后再运行。两个安装器都从**同一个发行标签**下载程序和 `SHA256SUMS`，校验不符时停止。校验和可以检测下载内容是否发生变化，但不能单独证明发布者身份。

### 指定版本和目录

```sh
sh install.sh --version v0.1.0 --install-dir "$HOME/.local/bin" --vault "$HOME/AI-Vault" --auto-connect
```

```powershell
.\install.ps1 -Version v0.1.0 -InstallDir "$env:LOCALAPPDATA\AI-Vault\bin" -Vault "$env:USERPROFILE\AI-Vault" -AutoConnect
```

也可以使用环境变量 `AIVAULT_VERSION`、`AIVAULT_INSTALL_DIR` 和 `AIVAULT_HOME`。运行已保存的安装器时，若未指定版本，它会查询一次最新 GitHub Release，再使用该标签下载程序和校验和。省略 `--auto-connect` / `-AutoConnect` 时，只安装程序并输出配置，不会注册客户端。

更新前请停止正在运行 AI Vault 的客户端。重新运行目标版本的安装器会保留已有笔记。卸载时删除程序和对应 MCP 注册项；笔记目录可独立保留或备份。

## 连接 AI 客户端

`--auto-connect` / `-AutoConnect` 会检测 `PATH` 中的 **Codex CLI** 和 **Claude Code CLI**，并通过各自支持的 `mcp add` 命令注册。Claude Code 使用用户级配置，因此可跨项目使用。安装器会保留已有的 `ai-vault` 注册项并提示你检查，不会替换整个客户端配置文件。已有注册项可能指向另一笔记目录，确认路径一致后再共享知识。

注册后请重启客户端或重新连接 MCP。客户端自己的信任和工具审批设置仍然有效。

手动配置时，请使用安装器实际输出的绝对路径：

```sh
codex mcp add ai-vault -- /absolute/path/aivault --vault /absolute/path/AI-Vault serve
claude mcp add --scope user --transport stdio ai-vault -- /absolute/path/aivault --vault /absolute/path/AI-Vault serve
```

Windows 使用 `.exe` 路径，含空格的路径需要加引号。例如在 PowerShell 中：

```powershell
codex mcp add ai-vault -- "$env:LOCALAPPDATA\AI-Vault\bin\aivault.exe" --vault "$env:USERPROFILE\AI-Vault" serve
claude mcp add --scope user --transport stdio ai-vault -- "$env:LOCALAPPDATA\AI-Vault\bin\aivault.exe" --vault "$env:USERPROFILE\AI-Vault" serve
```

Claude Desktop 和其他支持**本地 stdio MCP** 的客户端，可以使用以下命令生成准确的 JSON：

```sh
aivault --vault /absolute/path/AI-Vault config --format json
```

按照该客户端的正式配置方式，将生成的 `ai-vault` 服务器项合并到配置中，保留其他已有项。典型配置如下：

```json
{
  "mcpServers": {
    "ai-vault": {
      "command": "/absolute/path/aivault",
      "args": ["--vault", "/absolute/path/AI-Vault", "serve"]
    }
  }
}
```

Codex 的 TOML 片段可通过 `aivault config --format codex` 生成。所有需要共享知识的客户端都应使用**同一个笔记目录的绝对路径**。没有 localhost 监听端口或云端地址；客户端按需启动和停止 stdio 进程。只需读取知识时，使用 `aivault serve --read-only`。

官方连接说明：[Codex MCP](https://learn.chatgpt.com/docs/extend/mcp?surface=cli)、[Claude Code MCP](https://code.claude.com/docs/en/mcp)、[配置本地 MCP 服务器](https://modelcontextprotocol.io/docs/develop/connect-local-servers)。

### 验证安装

```sh
aivault version
aivault doctor
aivault list
aivault write setup-check.md --content '# Setup check'
aivault read setup-check.md
aivault trash setup-check.md --expected-sha HASH_FROM_READ
```

将 `HASH_FROM_READ` 替换为 `read` 返回的 `sha256`。若该测试文件已存在，请换一个名称。然后让重新连接的 AI 客户端调用 `vault_list`，用 `vault_write` 创建唯一命名的测试笔记，通过 `vault_read` 读取，再使用当前哈希调用 `vault_trash`。CLI 能运行，并不等于客户端已成功连接 MCP。

## 通过 GitHub URL 添加知识

导入知识与安装工具是两个不同的操作：

```sh
aivault import https://github.com/OWNER/REPOSITORY
aivault import https://github.com/OWNER/REPOSITORY --prefix sources/reference
```

导入器下载公开 GitHub 仓库的一个确定提交，将 Markdown 笔记默认保存到 `sources/OWNER/REPOSITORY/`，不会执行仓库代码。JSON 结果包含来源、提交、导入路径和跳过路径。已有笔记内容相同时跳过，内容冲突时返回错误而不覆盖。不支持私有仓库或任意下载 URL。

可以对已连接的 AI 说：“将 `https://github.com/OWNER/REPOSITORY` 的 Markdown 知识导入 AI Vault，并总结导入的笔记。”

请将导入的笔记视为外部数据，其中可能包含错误信息或针对 AI 的指令。笔记内容是阅读材料，不构成执行命令或修改规则的授权。

## 日常使用

以下示例假设 `aivault` 已加入 `PATH`，否则请替换为安装器输出的绝对程序路径。标准输出为 JSON，错误写入标准错误流。全局 `--vault PATH` 必须放在命令**前面**，也可以设置 `AIVAULT_HOME`。

```sh
aivault init
aivault doctor
aivault list
aivault write inbox/idea.md --content '# Idea: a shared AI memory'
aivault read inbox/idea.md
aivault search 'shared AI'
aivault tags
aivault categories
aivault links inbox/idea.md
aivault backlinks inbox/idea.md
```

Markdown 可以包含 `[[another-note]]`、`[[folder/note|label]]`、Markdown 链接和 `#tags`。搜索是不区分大小写的字面字符串搜索，不使用嵌入模型。普通 Markdown 文件仍可用于文本编辑器和 Obsidian；AI Vault 不实现 Obsidian 插件或所有 Markdown 扩展功能。

### 分类随知识增长

分类就是相对目录。AI Vault 不预设学科、编程语言或固定分类体系，分类列表来自笔记库中已有的目录：

```sh
aivault categories
aivault write decision.md --category work/decisions --content '# A decision and its reasons'
aivault read work/decisions/decision.md
```

`work/decisions` 仅是示例。可以使用适合自己知识的任意可移植分类路径，首次保存笔记时会创建对应目录。对已连接的 AI，可要求它先调用 `vault_categories`，选择最具体且相关的已有目录，再通过 `vault_write` 的可选 `category` 字段保存。现有分类不适用时，可以创建新的分类。不使用分类选项时，也可以直接指定笔记相对路径。

分类的语义判断由连接的 AI 完成。AI Vault 将该选择保存为目录，不运行本地分类模型，也不需要额外的推理运行环境。

### 安全编辑与恢复

`read` 返回笔记的 `sha256`。编辑、追加或将已有笔记移至回收站时，需要提供该哈希：

```sh
aivault write inbox/idea.md --content '# Updated idea' --expected-sha HASH_FROM_READ
aivault append inbox/idea.md --content 'An additional thought.' --expected-sha HASH_FROM_LATEST_READ
aivault history inbox/idea.md
aivault trash inbox/idea.md --expected-sha HASH_FROM_LATEST_READ
aivault restore REVISION_ID recovered/idea.md
```

请将大写占位符替换为实际返回值。新建笔记无需哈希，但会拒绝覆盖已有路径。哈希变化说明有其他写入者修改了笔记，需要重新读取并协调内容。历史保存之前的内容，回收站中的笔记可以恢复；恢复时不会无条件覆盖目标。`write` 和 `append` 也支持 `--file PATH` 或标准输入，作为 `--content` 的替代。追加按原样连接文本，需要换行时请将换行包含在内容中。

修改操作使用 AI Vault 进程间锁。并发写入者会收到可读错误，应重新读取后再尝试。只接受笔记库内的相对 `.md` 路径，拒绝目录穿越、隐藏路径、符号链接和 Windows 保留名称。每个笔记最多 4 MiB。需要保留历史时，请备份包含 `.aivault` 的整个笔记库。

如果崩溃后 `doctor` 报告写入锁，先停止所有 AI Vault 进程并确认没有正在写入的进程，然后才能删除残留的 `.aivault/write.lock` 目录，再运行 `doctor`。不要删除仍在使用的锁。

### MCP 工具

| 工具 | 用途 |
| --- | --- |
| `vault_list`, `vault_read`, `vault_search` | 查找和读取笔记 |
| `vault_categories` | 发现已有分类目录 |
| `vault_links`, `vault_backlinks` | 查看知识之间的关联 |
| `vault_write`, `vault_append` | 使用哈希校验保存笔记 |
| `vault_history`, `vault_trash`, `vault_restore` | 查看和恢复历史版本 |
| `vault_import` | 从公开 GitHub 仓库导入 Markdown |

请求示例：“在 AI Vault 中搜索我的部署决策，先阅读相关笔记再回答，仅在我要求时保存新决策。”AI Vault 提供存储工具，客户端指令决定何时使用它们。安装后不会自动记录所有聊天。

## 本机共享与多台电脑

同一台电脑上的 AI 客户端通过打开同一个笔记库目录共享知识。AI Vault **不提供跨设备同步**。你可以自行管理备份或同步，但本地目录锁不是独立同步副本之间的分布式锁。请避免多台电脑同时写入，先解决同步冲突再继续。

笔记是本地明文文件，并未加密。连接托管 AI 服务后，选中的笔记内容可能按该服务的政策发送给服务提供者。准备公开的笔记库中不应存放秘密信息。

## 构建与验证

开发者需要 `go.mod` 指定的 Go 版本：

```sh
go test ./...
go build -trimpath -ldflags='-s -w' -o aivault ./cmd/aivault
```

发行文件名为 `aivault_VERSION_OS_ARCH.tar.gz`（Windows 使用 `.zip`），OS 为 `darwin`、`linux` 或 `windows`，架构为 `amd64` 或 `arm64`。归档根目录包含 `aivault` / `aivault.exe`，`SHA256SUMS` 用于校验归档。请查看 [GitHub Releases](https://github.com/uyuk216/ai-vault/releases)。

离线测试安装器时，可以将 `AIVAULT_TEST_RELEASE_DIR` 设置为包含发行文件和 `SHA256SUMS` 的**本地绝对目录**，同时必须显式指定版本。不接受镜像 URL。请使用临时安装和笔记目录；测试 auto-connect 时，通过 `PATH` 中的 `codex` 和 `claude` 桩程序避免修改真实客户端配置。

## 许可证

[MIT](LICENSE)。依赖代码保留其原始许可证，详见[第三方声明](THIRD_PARTY_NOTICES.md)。AI Vault 是独立项目，与 Obsidian、OpenAI、Anthropic 无隶属关系。
