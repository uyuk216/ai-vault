# AI Vault

[日本語](README.ja.md) | [English](README.md) | [한국어](README.ko.md) | [简体中文](README.zh-CN.md)

**Codex, Claude와 다른 AI 클라이언트를 위한 경량 로컬 AI 메모리이자 Markdown 지식 저장소입니다.** 화면 없는 MCP 서버와 JSON CLI를 실행 파일 하나로 제공하며 Windows, Linux, macOS에서 동작합니다.

편집기, 데이터베이스, 상시 실행되는 웹 서버 없이 Obsidian 스타일의 `[[links]]`, 백링크, 태그, 검색, 노트 이력을 제공합니다. 노트는 일반 UTF-8 Markdown 파일로 보관됩니다. 릴리스 실행 파일을 사용하는 데 API 키, 계정, Git, Node.js, Python, Go 설치가 필요하지 않습니다.

설치 및 사용 안내 README는 일본어, 영어, 한국어, 중국어 간체로 제공합니다. 기본 README, 내부 메타데이터, CLI 도움말, 명령 옵션, JSON 필드, 설정 키는 영어를 사용합니다. 노트는 UTF-8 일본어, 영어, 한국어, 중국어를 지원하며 여러 언어를 섞어 써도 됩니다. AI Vault는 저장한 노트를 자동 번역하지 않고 원래 언어와 내용을 그대로 보존합니다.

## AI에 이 URL을 전달하세요

저장소: **<https://github.com/uyuk216/ai-vault>**

터미널 명령을 실행할 수 있는 로컬 코딩 AI에 다음 요청을 붙여 넣으세요.

```text
https://github.com/uyuk216/ai-vault 에서 AI Vault를 이 컴퓨터에 설치해 줘.
먼저 README와 설치 스크립트를 읽어 줘. 이 OS와 CPU에 맞는 릴리스를 다운로드하고,
SHA-256을 검증한 뒤 ~/AI-Vault를 공유 저장소로 사용해 줘.
설치된 Codex와 Claude Code 클라이언트는 설치 스크립트의 auto-connect 옵션으로 연결해 줘.
다른 로컬 MCP 클라이언트는 생성된 절대 경로 설정을 사용하되, 관련 없는 설정을 유지해 줘.
CLI와 MCP 도구에서 노트 목록 조회, 작은 테스트 노트 생성, 읽기, 휴지통으로 이동을 확인해 줘.
아직 재시작이나 수동 설정이 필요한 클라이언트가 있으면 알려 줘.
```

저장소 URL은 AI가 설명과 다운로드를 찾는 출발점입니다. **모든 MCP 클라이언트에서 바로 설치되는 공통 URL은 아닙니다.** AI에는 로컬 터미널 접근 권한과 클라이언트 설정 권한이 필요합니다. 채팅만 지원하거나 호스팅되는 클라이언트는 링크를 받는 것만으로 이 로컬 실행 파일을 시작할 수 없습니다.

## 설치

미리 빌드된 릴리스는 **Windows / Linux / macOS × x86-64 / ARM64**를 지원합니다. Linux에서는 일반적인 Linux 실행 환경이 필요합니다. 브라우저나 모바일 앱이 아닌 네이티브 실행 파일입니다.

### macOS / Linux

`curl`, `tar`, SHA-256 도구 하나(`sha256sum`, `shasum`, `openssl` 중 하나)가 필요합니다. 일반적인 macOS와 Linux 설치 환경에는 보통 이미 포함되어 있습니다.

```sh
curl -fsSL https://raw.githubusercontent.com/uyuk216/ai-vault/v0.1.1/install.sh | sh -s -- --version v0.1.1 --auto-connect
```

실행 파일은 `~/.local/bin/aivault`에, 노트는 `~/AI-Vault`에 저장됩니다. 관리자 권한은 필요하지 않습니다. `~/.local/bin/aivault`를 직접 실행하거나 셸의 `PATH`에 `~/.local/bin`을 추가하세요.

### Windows PowerShell

Windows PowerShell 5.1 또는 PowerShell 7이 필요합니다. 일반 사용자 계정으로 실행하세요.

```powershell
& ([scriptblock]::Create((Invoke-WebRequest -UseBasicParsing 'https://raw.githubusercontent.com/uyuk216/ai-vault/v0.1.1/install.ps1').Content)) -Version v0.1.1 -AutoConnect
```

실행 파일은 `%LOCALAPPDATA%\AI-Vault\bin\aivault.exe`에, 노트는 `%USERPROFILE%\AI-Vault`에 저장됩니다. 출력된 실행 파일 경로를 사용하거나 해당 디렉터리를 사용자 `PATH`에 추가하세요.

위 명령은 지정한 태그의 설치 스크립트를 실행합니다. 먼저 내용을 확인하려면 해당 태그의 `install.sh` 또는 `install.ps1`을 다운로드하여 읽은 뒤 저장한 파일을 실행하세요. 두 설치 스크립트 모두 실행 파일과 `SHA256SUMS`를 **같은 릴리스 태그**에서 다운로드하며, 체크섬이 일치하지 않으면 중단합니다. 체크섬은 다운로드 내용의 변경을 감지하지만, 그 자체로 게시자의 신원을 증명하지는 않습니다.

### 설치 옵션

```sh
sh install.sh --version v0.1.1 --install-dir "$HOME/.local/bin" --vault "$HOME/AI-Vault" --auto-connect
```

```powershell
.\install.ps1 -Version v0.1.1 -InstallDir "$env:LOCALAPPDATA\AI-Vault\bin" -Vault "$env:USERPROFILE\AI-Vault" -AutoConnect
```

`AIVAULT_VERSION`, `AIVAULT_INSTALL_DIR`, `AIVAULT_HOME` 환경 변수로도 같은 기본값을 지정할 수 있습니다. 버전을 생략하고 저장한 설치 스크립트를 실행하면 최신 GitHub Release를 한 번 확인한 뒤, 두 다운로드에 모두 그 태그를 사용합니다. `--auto-connect` / `-AutoConnect`를 생략하면 클라이언트를 등록하지 않고 설치와 설정 출력만 수행합니다.

업데이트 전에는 AI Vault를 실행 중인 클라이언트를 중지하세요. 원하는 버전으로 설치 스크립트를 다시 실행하면 기존 노트는 유지됩니다. 제거하려면 실행 파일과 클라이언트의 MCP 항목을 삭제하세요. 저장소 폴더는 별도로 보관하거나 백업하세요.

## AI 클라이언트 연결

`--auto-connect` / `-AutoConnect`는 `PATH`에서 **Codex CLI**와 **Claude Code CLI**를 찾아 각 클라이언트가 지원하는 `mcp add` 명령으로 등록합니다. Claude Code에는 사용자 범위 항목을 등록하므로 여러 프로젝트에서 사용할 수 있습니다. 기존 `ai-vault` 항목이 있으면 유지하고 확인을 안내합니다. 클라이언트 설정 파일 전체를 교체하지 않습니다. 기존 항목은 다른 저장소를 가리킬 수 있으므로, 모든 클라이언트가 같은 노트를 공유한다고 판단하기 전에 경로를 확인하세요.

등록 후에는 클라이언트를 다시 시작하거나 다시 연결하세요. 클라이언트 자체의 신뢰 및 도구 승인 설정은 계속 적용됩니다.

수동으로 설정하려면 설치 스크립트가 출력한 실제 절대 경로를 사용하세요.

```sh
codex mcp add ai-vault -- /absolute/path/aivault --vault /absolute/path/AI-Vault serve
claude mcp add --scope user --transport stdio ai-vault -- /absolute/path/aivault --vault /absolute/path/AI-Vault serve
```

Windows에서는 `.exe` 경로를 사용하고 공백이 포함된 경로를 따옴표로 감싸세요. PowerShell 예시는 다음과 같습니다.

```powershell
codex mcp add ai-vault -- "$env:LOCALAPPDATA\AI-Vault\bin\aivault.exe" --vault "$env:USERPROFILE\AI-Vault" serve
claude mcp add --scope user --transport stdio ai-vault -- "$env:LOCALAPPDATA\AI-Vault\bin\aivault.exe" --vault "$env:USERPROFILE\AI-Vault" serve
```

Claude Desktop과 다른 **로컬 stdio MCP 클라이언트**에는 다음 명령으로 정확한 JSON 설정을 생성하세요.

```sh
aivault --vault /absolute/path/AI-Vault config --format json
```

생성된 `ai-vault` 서버 항목을 해당 클라이언트의 공식 설정 방법에 따라 병합하세요. 다른 항목을 교체하지 마세요. 일반적인 설정은 다음과 같습니다.

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

Codex용 TOML 설정 조각은 `aivault config --format codex`로 생성할 수 있습니다. 모든 클라이언트에서 **같은 저장소의 절대 경로**를 사용해야 합니다. localhost 포트나 클라우드 접속 주소는 없습니다. 클라이언트가 필요할 때 stdio 프로세스를 시작하고 종료합니다. 읽기만 필요한 클라이언트에는 `aivault serve --read-only`를 사용하세요.

연결 참고 문서: [Codex MCP](https://learn.chatgpt.com/docs/extend/mcp?surface=cli), [Claude Code MCP](https://code.claude.com/docs/en/mcp), [로컬 MCP 서버 설정](https://modelcontextprotocol.io/docs/develop/connect-local-servers).

### 설치 확인

```sh
aivault version
aivault doctor
aivault list
aivault write setup-check.md --content '# Setup check'
aivault read setup-check.md
aivault trash setup-check.md --expected-sha HASH_FROM_READ
```

`HASH_FROM_READ`에는 `read`가 반환한 `sha256`을 사용하세요. 같은 이름의 테스트 파일이 이미 있으면 다른 이름을 선택하세요. 그런 다음 재시작한 AI 클라이언트에 `vault_list`를 호출하고, `vault_write`로 고유한 이름의 테스트 노트를 만들고, `vault_read`로 읽은 뒤 현재 해시를 사용해 `vault_trash`로 휴지통에 옮기도록 요청하세요. CLI가 동작하는 것만으로 클라이언트의 MCP 연결까지 확인된 것은 아닙니다.

## GitHub URL에서 지식 추가

도구 설치와는 별개의 기능입니다.

```sh
aivault import https://github.com/OWNER/REPOSITORY
aivault import https://github.com/OWNER/REPOSITORY --prefix sources/reference
```

가져오기 도구는 공개 GitHub 저장소의 특정 커밋을 기록하여 다운로드하고, 기본적으로 Markdown 노트를 `sources/OWNER/REPOSITORY/`에 복사합니다. 저장소의 코드를 실행하지 않습니다. JSON 보고서에는 출처, 커밋, 가져온 경로와 건너뛴 경로가 포함됩니다. 동일한 기존 노트는 건너뛰며, 기존 노트와 내용이 충돌하면 덮어쓰지 않고 오류를 반환합니다. 비공개 저장소와 임의의 다운로드 URL은 지원하지 않습니다.

연결된 AI에 이렇게 요청할 수 있습니다. “`https://github.com/OWNER/REPOSITORY`의 Markdown 지식을 AI Vault로 가져온 뒤, 가져온 노트를 요약해 줘.”

가져온 노트는 외부 콘텐츠로 취급하세요. 부정확한 정보나 AI를 겨냥한 지시가 포함될 수 있습니다. 노트 내용은 읽고 판단할 자료이며, 명령 실행이나 정책 변경에 대한 허가가 아닙니다.

## 일상적인 사용

아래 명령은 `aivault`가 `PATH`에 있다고 가정합니다. 그렇지 않으면 설치 스크립트가 출력한 절대 실행 파일 경로로 바꾸세요. JSON은 stdout으로, 오류는 stderr로 출력됩니다. 전역 옵션 `--vault PATH`는 명령 **앞에** 지정하거나 `AIVAULT_HOME`을 설정하세요.

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

Markdown에는 `[[another-note]]`, `[[folder/note|label]]`, Markdown 링크, `#tags`를 넣을 수 있습니다. 임베딩 모델은 사용하지 않습니다. 검색은 대소문자를 구분하지 않는 일반 텍스트 검색입니다. 일반 Markdown 파일이므로 텍스트 편집기와 Obsidian에서도 사용할 수 있습니다. Obsidian 플러그인과 모든 Markdown 확장 기능까지 지원하는 것은 아닙니다.

### 지식에 맞춰 늘어나는 분류

분류는 일반적인 상대 경로 폴더입니다. AI Vault는 주제, 프로그래밍 언어, 고정된 분류 체계를 미리 등록하지 않습니다. 분류 목록에는 저장소에 실제로 존재하는 폴더가 반영됩니다.

```sh
aivault categories
aivault write decision.md --category work/decisions --content '# A decision and its reasons'
aivault read work/decisions/decision.md
```

`work/decisions`는 예시일 뿐입니다. 지식에 맞는, 운영체제 간 호환이 가능한 분류 경로를 지정하세요. 첫 노트를 해당 위치에 저장하면 폴더가 생성됩니다. 연결된 AI에는 `vault_categories`를 호출하고 관련 있는 기존 폴더 중 가장 구체적인 것을 선택한 뒤, `vault_write`의 선택적 `category` 인자로 전달하도록 요청하세요. 기존 분류가 맞지 않으면 새 분류를 만들 수 있습니다. 분류 옵션 없이 노트 경로를 직접 지정해도 됩니다.

연결된 AI가 노트의 의미를 바탕으로 분류를 선택합니다. AI Vault는 그 선택을 폴더로 저장하며, 로컬 분류 모델이나 추론 실행 환경을 추가하지 않습니다.

### 안전한 수정과 복구

`read`는 노트의 `sha256`을 반환합니다. 기존 노트를 수정하거나 내용을 추가하거나 휴지통으로 옮길 때 이 해시를 전달하세요.

```sh
aivault write inbox/idea.md --content '# Updated idea' --expected-sha HASH_FROM_READ
aivault append inbox/idea.md --content 'An additional thought.' --expected-sha HASH_FROM_LATEST_READ
aivault history inbox/idea.md
aivault trash inbox/idea.md --expected-sha HASH_FROM_LATEST_READ
aivault restore REVISION_ID recovered/idea.md
```

대문자로 된 자리표시자는 반환된 값으로 바꾸세요. 새 노트 생성에는 해시가 필요하지 않으며, 이미 존재하는 경로에는 생성할 수 없습니다. 해시가 달라졌다면 다른 작성자가 노트를 수정한 것이므로 다시 읽고 내용을 조정하세요. 이력에는 이전 내용이 보존되고 휴지통에서도 복구할 수 있습니다. 복구 시 기존 대상 파일을 무조건 덮어쓰지 않습니다. `write`와 `append`는 `--content` 대신 `--file PATH` 또는 파이프로 전달한 stdin도 받습니다. 추가할 텍스트는 그대로 이어 붙이므로 줄바꿈이 필요하면 내용에 포함하세요.

변경 작업은 프로세스 간 저장소 잠금을 사용합니다. 다른 작성자가 동시에 쓰려고 하면 이해할 수 있는 오류를 반환하며, 노트를 다시 읽은 뒤 재시도해야 합니다. 저장소 기준의 상대 `.md` 경로만 허용합니다. 상위 폴더로 벗어나는 경로, 숨김 경로, 심볼릭 링크, Windows 예약 이름은 거부합니다. 노트 하나의 크기 제한은 4 MiB입니다. 이력까지 보관하려면 `.aivault`를 포함한 저장소 전체를 백업하세요.

비정상 종료 후 `doctor`가 쓰기 잠금을 보고하면 먼저 모든 AI Vault 프로세스를 중지하고 쓰는 프로세스가 없는지 확인하세요. 그다음에만 남아 있는 `.aivault/write.lock` 디렉터리를 삭제하고 `doctor`를 다시 실행하세요. 쓰기 작업이 진행 중인 잠금을 삭제하지 마세요.

### MCP 도구

| 도구 | 용도 |
| --- | --- |
| `vault_list`, `vault_read`, `vault_search` | 노트 검색 및 읽기 |
| `vault_categories` | 기존 분류 폴더 확인 |
| `vault_links`, `vault_backlinks` | 노트 간 연결 탐색 |
| `vault_write`, `vault_append` | 해시를 확인하며 노트 저장 |
| `vault_history`, `vault_trash`, `vault_restore` | 이전 버전 확인 및 복구 |
| `vault_import` | 공개 GitHub 저장소에서 Markdown 가져오기 |

요청 예시: “AI Vault에서 내가 배포에 관해 내린 결정을 찾아 줘. 답변 전에 관련 노트를 읽고, 새 결정은 내가 요청할 때만 저장해 줘.” AI Vault는 저장 도구를 제공하며, 사용 시점은 클라이언트의 지시가 결정합니다. 설치만으로 모든 대화가 자동으로 기록되지는 않습니다.

## 같은 폴더 공유와 여러 컴퓨터 사용

한 컴퓨터의 AI 클라이언트들은 같은 저장소 폴더를 열어 지식을 공유합니다. AI Vault에는 **기기 간 동기화 기능이 없습니다.** 백업이나 동기화를 직접 구성할 수는 있지만, 디렉터리 잠금은 따로 동기화되는 복사본 사이의 분산 잠금이 아닙니다. 여러 컴퓨터에서 동시에 쓰지 말고, 동기화 충돌을 해결한 뒤 계속 사용하세요.

노트는 암호화되지 않은 로컬 일반 텍스트입니다. 호스팅되는 AI 서비스에 연결하면 선택한 노트 내용이 해당 서비스의 정책에 따라 제공업체로 전송될 수 있습니다. 공개적으로 공유할 저장소에는 비밀 정보를 넣지 마세요.

## 빌드와 검증

개발자는 `go.mod`에 선언된 Go 버전이 필요합니다.

```sh
go test ./...
go build -trimpath -ldflags='-s -w' -o aivault ./cmd/aivault
```

릴리스 파일 이름은 `aivault_VERSION_OS_ARCH.tar.gz`이며, Windows에서는 `.zip`을 사용합니다. OS는 `darwin`, `linux`, `windows`, 아키텍처는 `amd64`, `arm64`입니다. 압축 파일의 최상위에는 `aivault` / `aivault.exe`가 들어 있습니다. `SHA256SUMS`는 압축 파일들의 체크섬을 담습니다. [GitHub Releases](https://github.com/uyuk216/ai-vault/releases)를 참조하세요.

오프라인 설치 테스트에서는 해당 릴리스 파일들과 `SHA256SUMS`가 있는 **로컬 디렉터리의 절대 경로**를 `AIVAULT_TEST_RELEASE_DIR`에 지정할 수 있습니다. 이때 버전을 명시해야 합니다. 미러 URL은 사용할 수 없습니다. 임시 설치 및 저장소 디렉터리를 사용하고, 자동 연결 테스트에서는 실제 클라이언트 설정이 바뀌지 않도록 `PATH`의 `codex`와 `claude`를 테스트용 대체 명령으로 바꾸세요.

## Windows 애플리케이션 제어 정책

배포 바이너리는 서명되지 않았습니다. 신뢰할 수 있는 게시자의 서명을 요구하는 Windows 애플리케이션 제어 정책은 AI Vault를 차단할 수 있습니다. 이런 장치에는 해당 정책이 허용하는 서명이 필요합니다. 설치 프로그램은 Windows 보안 설정을 변경하지 않습니다. Windows CI에서는 이 바이너리를 허용하는 환경에서 실행과 설치를 검증합니다.

## 라이선스

[MIT](LICENSE). 의존성에는 각각의 원래 라이선스가 적용됩니다. [타사 고지](THIRD_PARTY_NOTICES.md)를 참조하세요. AI Vault는 Obsidian, OpenAI, Anthropic과 독립된 프로젝트입니다.
