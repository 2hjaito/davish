<div align="center">

<img src="assets/icon/png/davish-128.png" width="96" height="96" alt="Icon Davish">

# Davish

**Một desktop cho các server Linux của bạn.**<br>
Terminal, file, dịch vụ, Docker, log, cron và nginx qua SSH thuần, gom trong một giao diện như một hệ điều hành.<br>
Không cài gì lên server.

[![CI](https://github.com/2hjaito/davish/actions/workflows/ci.yml/badge.svg)](https://github.com/2hjaito/davish/actions/workflows/ci.yml)
![Nền tảng](https://img.shields.io/badge/platform-macOS%20%7C%20Windows%20%7C%20Linux-6b7180)
![Go](https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white)
![Svelte](https://img.shields.io/badge/Svelte-5-FF3E00?logo=svelte&logoColor=white)

[English](README.md) · **Tiếng Việt**

<img src="assets/screenshots/desktop.jpg" alt="Desktop Davish với Terminal, Monitor và Files" width="100%">

</div>

## Vì sao là Davish

- **Không cần agent.** Davish chỉ cần `sshd` và SFTP. Mọi thứ chạy từ máy bạn qua SSH; không chép, không cài, không để lại gì trên server.
- **Thấy trước mọi lệnh.** Thao tác phá hủy (dừng dịch vụ, xóa file, khởi động lại container, thay crontab) hiện đúng lệnh sẽ chạy trước khi chạy. Nhật ký lệnh ghi lại mọi thứ Davish đã chạy thay bạn.
- **Bí mật nằm trong keychain.** Mật khẩu và passphrase chỉ lưu trong keychain của hệ điều hành. Davish không bao giờ lưu hay tự gõ mật khẩu `sudo`: khi cần root, nó mở Terminal với lệnh gõ sẵn để bạn tự nhấn Enter.
- **Nhiều server, một desktop.** Mỗi server là một workspace (`⌘1`…`⌘9`) nhớ cửa sổ riêng của nó. Kéo file để copy giữa các server.

## Tính năng

| | |
|---|---|
| **Terminal**<br>xterm.js nhiều tab, copy/paste, vẽ bằng GPU, và prompt powerline tùy chọn được nạp vào bash lúc đăng nhập, không ghi gì lên server. | **Files**<br>Trình duyệt kiểu Finder qua SFTP: tải lên, tải xuống, kéo thả từ hệ điều hành, đổi tên, xóa, nén, và copy giữa các server qua máy bạn. |
| **Monitor**<br>CPU từng nhân, bộ nhớ và swap, ổ đĩa, mạng theo interface, GPU NVIDIA, tiến trình. Cảnh báo RAM và ổ đĩa chỉnh được ngưỡng. | **Editor**<br>CodeMirror 6, tô màu cho nginx, YAML, JSON, shell, env và file cấu hình. Lưu atomic giữ nguyên mode và chủ file, sao lưu file trong `/etc`, cảnh báo khi file đã đổi trên server. |
| **Services**<br>Dịch vụ systemd theo trạng thái, chi tiết và journal; khởi động, dừng, khởi động lại. | **Docker**<br>Container với CPU và RAM, log có đánh dấu stderr, inspect, start/stop/restart, cảnh báo khởi động lại liên tục. |
| **Logs**<br>Theo dõi journal của dịch vụ, một file hoặc một container. Lọc, tạm dừng, tô dòng lỗi, bấm đường dẫn file để mở trong Editor đúng dòng. | **Cron**<br>Crontab của bạn và của hệ thống, lịch nói bằng lời, lần chạy kế tiếp theo giờ server, sửa an toàn có sao lưu. |
| **Nginx**<br>Site, server block và upstream; bật/tắt site, `nginx -t` trước mỗi lần reload, lỗi mở đúng dòng; cảnh báo chứng chỉ TLS sắp hết hạn. | **Spotlight**<br>`⌘K` để mở app, dịch vụ, container, thư mục, một `/đường/dẫn`, hoặc `> một lệnh` gõ sẵn vào Terminal. |

<table>
  <tr>
    <td><img src="assets/screenshots/services.jpg" alt="Services"></td>
    <td><img src="assets/screenshots/docker.jpg" alt="Docker với một container khởi động lại liên tục"></td>
  </tr>
  <tr>
    <td><img src="assets/screenshots/logs.jpg" alt="Logs với đường dẫn file bấm được"></td>
    <td><img src="assets/screenshots/nginx.jpg" alt="Site nginx với chứng chỉ TLS"></td>
  </tr>
  <tr>
    <td><img src="assets/screenshots/cron.jpg" alt="Tác vụ cron với lịch nói bằng lời"></td>
    <td><img src="assets/screenshots/spotlight.jpg" alt="Spotlight"></td>
  </tr>
</table>

> Ảnh chụp dùng dữ liệu mẫu của bản xem trước trên trình duyệt.

## Cài đặt

Bản build sẵn cho macOS (universal), Windows và Linux nằm trong từng [release](https://github.com/2hjaito/davish/releases).

- **macOS 12 trở lên.** App chưa được ký: lần đầu, chuột phải vào `Davish.app` rồi chọn **Mở**.
- **Windows 10 trở lên**, cần WebView2 runtime (Windows 11 có sẵn).
- **Linux** có GTK 3 và WebKitGTK 4.1.

### Server cần gì

| Tính năng | Yêu cầu |
|---|---|
| Tất cả | OpenSSH server có subsystem SFTP |
| Services, Logs (journal) | systemd; muốn đọc log hệ thống thì user thuộc nhóm `adm` hoặc `systemd-journal` |
| Docker | lệnh `docker`; user thuộc nhóm `docker` |
| Cron | `crontab` |
| Nginx | nginx với `sites-available` / `sites-enabled` hoặc `conf.d` |

Thao tác cần root (dừng dịch vụ hệ thống, sửa file của root) được chuyển sang Terminal với `sudo` gõ sẵn, không bao giờ chạy ngầm.

## Phím tắt

| Phím | Tác dụng |
|---|---|
| `⌘K` | Spotlight |
| `⌘1` … `⌘9` | Chuyển workspace server |
| `⌘W` · `⌘M` | Đóng · thu nhỏ cửa sổ |
| `⌘,` | Cài đặt |
| `⌘S` | Lưu trong Editor |
| `⌘F` · `⌘[` · `⌘]` · `⌘↑` | Files: tìm, lùi, tiến, lên thư mục cha |
| `⌘U` · `⌘D` | Files: tải lên, tải xuống |
| `⌘⌥S` | Files: copy mục đang chọn sang server khác |
| `⌘⌥L` | Cửa sổ Chuyển file |

Trên Windows và Linux dùng `Ctrl` thay cho `⌘`. Khi đang ở trong Terminal, phím `Ctrl` thuộc về shell.

## Build từ mã nguồn

Cần: Go 1.27+, Node.js LTS, [Wails](https://wails.io) v2.17 (`go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0`, rồi `wails doctor`), golangci-lint v2, và Docker cho integration test. Trên Linux thêm `libgtk-3-dev` và `libwebkit2gtk-4.1-dev` (build với `-tags webkit2_41`).

```bash
make frontend-install   # lần đầu
wails dev               # chạy, hot reload
wails build             # đóng gói vào build/bin/
```

### Kiểm thử

```bash
make check              # go vet, golangci-lint, go test, svelte-check, eslint, prettier, vitest

make sshd-up            # hai SSH server dùng một lần trong Docker (127.0.0.1:2222 và :2223)
make test-integration   # SSH, SFTP, systemd, Docker, cron và nginx thật trên hai server đó
make sshd-down
```

Integration test chỉ nói chuyện với container trong `test/sshd/`, không bao giờ với server thật.

## Công nghệ

| Lớp | Công nghệ |
|---|---|
| Backend | Go · Wails v2 · `golang.org/x/crypto/ssh` · `pkg/sftp` · `go-keyring` |
| Frontend | Svelte 5 (runes) · TypeScript strict · Vite |
| Thành phần | xterm.js · CodeMirror 6 · icon từ [davi-icons](https://github.com/2hjaito/davi-icons) |

## Nhật ký thay đổi

Xem [CHANGELOG.md](CHANGELOG.md).
