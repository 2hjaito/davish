# Davish

Desktop cho server Linux của bạn: quản lý VPS qua SSH với giao diện như một hệ điều hành, gồm menu bar, dock và các cửa sổ Terminal, Files, Monitor, Services, Docker, Logs, Editor. Không cài gì lên server.

## Công nghệ

- **Backend:** Go 1.27, Wails v2, `golang.org/x/crypto/ssh`, `pkg/sftp`, `go-keyring`
- **Frontend:** Svelte 5, TypeScript (strict), Vite, xterm.js, CodeMirror 6
- **Icon:** [`davi-icons`](https://github.com/2hjaito/davi-icons), trích thành dữ liệu SVG lúc build ([ADR 0002](docs/adr/0002-davi-icons.md))
- **Kiểm tra:** golangci-lint, `go test -race`, svelte-check, ESLint, Prettier, Vitest

## Yêu cầu

- Go 1.27+
- Node.js LTS
- Wails CLI v2: `go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0`, rồi `wails doctor`
- golangci-lint v2
- Docker (cho test SSH server)
- Linux: `libgtk-3-dev`, `libwebkit2gtk-4.1-dev` (build với `-tags webkit2_41`)

## Chạy

```bash
make frontend-install   # cài dependency frontend lần đầu
wails dev               # phát triển, hot reload
make check              # lint + test (Go và frontend)
wails build             # đóng gói, kết quả ở build/bin/

make sshd-up            # bật SSH server test ở 127.0.0.1:2222 (lần đầu tự sinh key test)
make test-integration
make sshd-down
```

## Cấu trúc

```
main.go, internal/     backend Go (xem docs/ARCHITECTURE.md §2)
frontend/              Svelte app; frontend/wailsjs/ là code tự sinh
build/                 tài nguyên đóng gói Wails (icon, plist, manifest)
assets/icon/           file gốc icon app
test/sshd/             container SSH server cho integration test
docs/                  tài liệu, ADR, mockup HTML
```

## Tài liệu

- [CLAUDE.md](CLAUDE.md): quy tắc cho Claude Code
- [Kiến trúc](docs/ARCHITECTURE.md)
- [Kế hoạch](docs/PLAN.md)
- [Quy ước code](docs/CONVENTIONS.md)
- [Quy tắc Git](docs/GIT.md)
- [Thiết kế giao diện](docs/DESIGN.md)
- [Bảo mật](docs/SECURITY.md)
- [Quyết định kiến trúc](docs/adr/)
