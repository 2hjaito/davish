<div align="center">

<img src="assets/icon/png/davish-128.png" width="96" height="96" alt="Icon Davish">

# Davish

**Một desktop cho các server Linux của bạn.**<br>
Terminal, file, dịch vụ, Docker, log, cron, nginx, khóa SSH, gói phần mềm và git qua SSH thuần, gom trong một giao diện như một hệ điều hành.<br>
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
- **Tự khóa khi bạn rời máy.** Mật khẩu khóa riêng, Touch ID trên macOS; khóa khi máy ngủ dậy, sau thời gian không dùng, hoặc `⌘L`. Phiên SSH vẫn chạy sau màn khóa, nhưng không thao tác nào tới được server cho tới khi mở khóa.
- **Nhiều server, một desktop.** Mỗi server là một workspace (`⌘1`…`⌘9`) nhớ cửa sổ riêng của nó. Kéo file để copy giữa các server.

## Tính năng

| | |
|---|---|
| **Terminal**<br>xterm.js nhiều tab, copy/paste, vẽ bằng GPU, và prompt powerline tùy chọn được nạp vào bash lúc đăng nhập, không ghi gì lên server. | **Files**<br>Trình duyệt kiểu Finder qua SFTP: tải lên, tải xuống, kéo thả từ hệ điều hành, đổi tên, xóa, nén, và copy giữa các server qua máy bạn. |
| **Monitor**<br>CPU từng nhân, bộ nhớ và swap, ổ đĩa, mạng theo interface, GPU NVIDIA, tiến trình. Cảnh báo RAM và ổ đĩa chỉnh được ngưỡng. | **Editor**<br>CodeMirror 6, tô màu cho nginx, YAML, JSON, shell, env và file cấu hình. Lưu atomic giữ nguyên mode và chủ file, sao lưu file trong `/etc`, cảnh báo khi file đã đổi trên server. |
| **Services**<br>Dịch vụ systemd theo trạng thái, chi tiết và journal; khởi động, dừng, khởi động lại. Xóa dịch vụ không còn dùng: unit do admin tạo thì xóa file, unit của gói (như `certbot` cũ) thì chặn hẳn cùng timer chạy nó. | **Docker**<br>Container với CPU và RAM, log có đánh dấu stderr, inspect, start/stop/restart, cảnh báo khởi động lại liên tục. |
| **Logs**<br>Theo dõi journal của dịch vụ, một file hoặc một container. Lọc, tạm dừng, tô dòng lỗi, bấm đường dẫn file để mở trong Editor đúng dòng. | **Cron**<br>Crontab của bạn và của hệ thống, lịch nói bằng lời, lần chạy kế tiếp theo giờ server, sửa an toàn có sao lưu. |
| **Nginx**<br>Site, server block và upstream; bật/tắt site, `nginx -t` trước mỗi lần reload, lỗi mở đúng dòng; cảnh báo chứng chỉ TLS sắp hết hạn. Xóa site, kèm chứng chỉ Let's Encrypt nếu không site nào khác dùng. | **Spotlight & Launchpad**<br>`⌘K` (hoặc kính lúp trên menu bar) để mở app, dịch vụ, container, thư mục, một `/đường/dẫn`, hoặc `> một lệnh` gõ sẵn vào Terminal. Launchpad (`⌘⇧A`) hiện mọi app và cho ghim vào Dock. |
| **Khóa SSH**<br>`authorized_keys` của server, đánh dấu key Davish đang đăng nhập; thêm, gỡ key có sao lưu. Tạo deploy key trên server và thử với GitHub, GitLab (kiểm tra đúng host key họ công bố). Tạo key trên máy, đưa lên server rồi chuyển sang đăng nhập bằng nó. | **Khóa app**<br>Touch ID hoặc mật khẩu riêng che mọi thứ khi bạn rời máy. Quên mật khẩu thì đặt lại, nhưng Davish xóa mật khẩu server đã lưu trước, nên đặt lại không mở được đường vào server. Cửa sổ mở, đóng, thu nhỏ về Dock bằng hiệu ứng Genie. |
| **Gói**<br>Gói apt, dnf, yum hoặc apk đã cài và có bản cập nhật, bản vá bảo mật lên đầu, có thông báo khi kết nối. Các runtime và chương trình trên server (nginx, MariaDB, Redis, Docker, Node.js qua nvm, PHP với php-fpm, Python…) kèm phiên bản, file cấu hình và dịch vụ; tìm được bằng Spotlight. Cài và nâng cấp thì gõ sẵn lệnh `sudo` vào Terminal cho bạn. | **Git**<br>Tìm repo trong home, `/var/www`, `/srv`, `/opt` hoặc thư mục bạn thêm. Nhánh, nhánh remote và số commit chưa đẩy/chưa kéo, file thay đổi kèm diff, biểu đồ commit của mọi nhánh với merge và tag. Fetch, `pull --ff-only` (bị chặn khi repo còn thay đổi chưa commit) và đổi nhánh sau khi xác nhận đúng lệnh; `npm ci` / yarn / pnpm theo lockfile trong Terminal. Remote từ chối key thì dẫn tới tạo deploy key. |

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
  <tr>
    <td><img src="assets/screenshots/keys.jpg" alt="Khóa SSH: authorized_keys của server"></td>
    <td><img src="assets/screenshots/remove.jpg" alt="Xóa site nginx cùng chứng chỉ Let's Encrypt"></td>
  </tr>
  <tr>
    <td><img src="assets/screenshots/packages.jpg" alt="Gói: bản cập nhật, bản vá bảo mật lên đầu"></td>
    <td><img src="assets/screenshots/packages-apps.jpg" alt="Chương trình trên server, Node.js với các phiên bản nvm"></td>
  </tr>
  <tr>
    <td><img src="assets/screenshots/git.jpg" alt="Git: biểu đồ commit của mọi nhánh, kèm diff của một merge"></td>
    <td><img src="assets/screenshots/git-changes.jpg" alt="Git: file thay đổi và diff của một file"></td>
  </tr>
  <tr>
    <td><img src="assets/screenshots/launchpad.jpg" alt="Launchpad"></td>
    <td><img src="assets/screenshots/genie.jpg" alt="Cửa sổ chảy vào Dock với hiệu ứng Genie"></td>
  </tr>
  <tr>
    <td><img src="assets/screenshots/lock.jpg" alt="Màn khóa"></td>
    <td><img src="assets/screenshots/welcome.jpg" alt="Trang chào"></td>
  </tr>
</table>

> Ảnh chụp dùng dữ liệu mẫu của bản xem trước trên trình duyệt.

## Cài đặt

Bản build sẵn cho macOS (universal), Windows và Linux nằm trong từng [release](https://github.com/2hjaito/davish/releases).

- **macOS 12 trở lên.** Mở `Davish-…-macos-universal.pkg` để cài vào Applications. App chưa được ký nên macOS sẽ chặn lần đầu: bấm **Done**, mở **Cài đặt hệ thống → Quyền riêng tư & Bảo mật**, bấm **Vẫn mở** ở dòng nhắc tới Davish (làm lại cho `Davish.app` nếu bị chặn). Hoặc chạy `xattr -dr com.apple.quarantine` trên file đó.
- **Windows 10 trở lên.** Chạy `Davish-…-windows-amd64-setup.exe`; bộ cài tự cài WebView2 runtime nếu máy chưa có. SmartScreen có thể cảnh báo nhà phát hành không xác định: chọn **More info** → **Run anyway**.
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
