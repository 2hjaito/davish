# Changelog

Mọi thay đổi đáng chú ý của Davish. Định dạng theo [Keep a Changelog](https://keepachangelog.com/vi/1.1.0/), phiên bản theo [SemVer](https://semver.org/lang/vi/): `v0.<milestone>.<patch>` trong giai đoạn phát triển, `v1.0.0` khi xong M10 (`docs/GIT.md`).

## [Unreleased]

### Thêm
- App An ninh: lần thử đăng nhập SSH sai trong 7 ngày gom theo IP (nhiều nhất lên đầu, user đã thử), đăng nhập thành công gần đây, `last`/`lastb`, trạng thái ufw và fail2ban; chặn một IP hoặc tắt đăng nhập mật khẩu bằng lệnh sudo gõ sẵn vào Terminal (có `sshd -t` trước khi reload, tự gỡ nếu lỗi).
- App Ổ đĩa: dung lượng từng mount; đo một thư mục mỗi lần bấm (`du -x -d 1`, tối đa 30 giây, hủy được), đi sâu từng tầng, mở trong Files. Gợi ý dọn dẹp kèm dung lượng lấy lại: Docker, journal, log xoay vòng cũ, cache apt/dnf; lệnh luôn hiện trước khi chạy, cần root thì gõ sẵn vào Terminal. Thông báo ổ sắp đầy mở thẳng app ở đúng mount.
- App Cổng: cổng TCP server đang nghe (`ss -tlnp`) với process và container; tunnel một cổng về `127.0.0.1` của máy này qua kết nối SSH đang có (giữ số cổng nếu trống), mở trong trình duyệt hoặc copy địa chỉ; mở như một tên miền (ghi đè Host) cho site nginx; "Thử từ server" bằng curl. Tunnel sống qua reconnect và tắt khi đóng app.
- Menu bar thật: Tệp, Xem, Đi tới, Cửa sổ, Trợ giúp và menu của app đang mở (Git: Fetch, Pull, Thêm thư mục; Services, Docker: khởi động lại…), điều khiển bằng bàn phím, phím tắt hiện bên phải. `⌘R` làm mới cửa sổ đang mở, `⌘⇧T/F/M/S/D/L/E/G` mở từng app.
- Trợ giúp: nút `?` trên mỗi app (app làm gì, thao tác chính, phím tắt, lệnh chạy trên server), bảng phím tắt (`⌘/`), Về Davish; Spotlight tìm được trợ giúp. Hướng dẫn nhanh lần đầu, xem lại từ Trợ giúp.
- Chế độ thử trên màn hình chào: xem Davish với dữ liệu mẫu, không kết nối server nào.
- Mọi nút chỉ có icon đều có tooltip.
- Nhập server từ `~/.ssh/config` (menu chọn server hoặc màn hình chào): chọn nhiều host, host đã có được bỏ qua; Davish chỉ đọc file, không ghi. Hỗ trợ `Include`, `IdentityFile`, `ProxyJump`.
- ProxyJump: server có thể đi qua một server khác (nhiều bậc), mỗi bậc kiểm tra host key riêng; mất kết nối tới máy trung gian thì tự dựng lại cả chuỗi.
- App Git: tìm repo trên server (home, `/var/www`, `/srv`, `/opt` và thư mục thêm tay); nhánh, nhánh remote, số commit chưa đẩy/chưa kéo, file thay đổi và diff; biểu đồ commit của mọi nhánh với merge và tag. Fetch, `pull --ff-only` (chặn khi repo bẩn), đổi nhánh qua hộp xác nhận hiện đúng lệnh; cài dependency theo lockfile trong Terminal; lỗi xác thực remote dẫn tới tạo deploy key. Repo của user khác bị git từ chối thì báo rõ, không ghi đè `safe.directory`.
- App Gói: gói apt/dnf/yum/apk đã cài và có bản cập nhật, bản vá bảo mật lên đầu (thông báo khi kết nối); runtime và chương trình trên server (nginx, MariaDB, Redis, Docker, Node.js qua nvm, PHP với php-fpm, Python…) với phiên bản, file cấu hình, dịch vụ; gói npm global. Cài, nâng cấp, gỡ thì gõ sẵn lệnh `sudo` vào Terminal. Spotlight tìm được chương trình trên server.
- App Khóa SSH: quản lý `authorized_keys` của server (thêm, gỡ có sao lưu, đánh dấu key Davish đang đăng nhập), tạo deploy key trên server và thử với GitHub/GitLab (kiểm tra host key theo fingerprint họ công bố), tạo key trên máy này rồi đưa lên server và dùng để đăng nhập.
- Khóa app (ADR 0007): mật khẩu khóa riêng, mở bằng Touch ID trên macOS; tự khóa khi máy ngủ dậy, sau N phút không dùng, hoặc `⌘L` / `Ctrl+L`. Phiên SSH, log, transfer vẫn chạy khi khóa; thao tác lên server bị chặn. Quên mật khẩu thì đặt lại, đổi lại Davish xóa mật khẩu server đã lưu.
- Launchpad (ô đầu Dock hoặc `⌘⇧A`): mọi app, tìm theo tên, ghim/bỏ ghim khỏi Dock. Dock hiện thêm app đang mở chưa ghim.
- Xóa dịch vụ (Services → `Xóa…`): unit do admin tạo thì xóa file; unit của gói (như `certbot`) thì mask cùng timer kích hoạt nó; unit đã mất file thì chỉ bỏ trạng thái lỗi. Luôn hiện đúng lệnh trước khi chạy.
- Xóa site nginx (`Xóa site…`), có thể xóa luôn chứng chỉ Let's Encrypt chỉ site đó dùng (`certbot delete`).
- Kính lúp trên menu bar và mục `Tìm kiếm…` trong menu hệ thống mở Spotlight.

### Thay đổi
- Nhẹ hơn: mỗi app chỉ tải code khi mở lần đầu (JS ban đầu khoảng 96 kB gzip); CI chặn bundle vượt ngân sách. Khi Davish bị ẩn, khóa hoặc ngủ màn hình, chỉ số đọc 30 giây một lần thay vì 2 giây, cảnh báo vẫn chạy.
- Mở, đóng, thu nhỏ và khôi phục cửa sổ bằng hiệu ứng Genie như macOS: cửa sổ chảy ra từ icon của app trên Dock và chảy về đó.

### Sửa
- Logs: nhãn trên toolbar không còn xuống dòng; cửa sổ hẹp thì ô nguồn và ô lọc co lại, hẹp hơn nữa thì toolbar xuống hàng thay vì che mất nút.
- Logs: journal của một dịch vụ im lặng 5 giây thì giải thích lý do thường gặp (dịch vụ không ghi journal, hoặc tài khoản SSH không được đọc journal hệ thống) và đưa lệnh `sudo journalctl …` để chạy trong Terminal, thay vì chờ mãi.
- Thông báo: đường dẫn dài (ví dụ đường dẫn chứng chỉ TLS) xuống dòng trong banner và bảng Thông báo thay vì tràn ra ngoài và đẩy nút đóng đi.

## [1.0.0] - 2026-10-28

M10 · Hoàn thiện v1.0.

### Thêm
- Spotlight (`⌘K` / `Ctrl+K`): mở app, nhảy tới dịch vụ, container, thư mục; gõ `/đường/dẫn` để mở trong Files hoặc Editor, `> lệnh` để gõ sẵn vào Terminal (không tự chạy).
- Cài đặt: font mono (JetBrains Mono hoặc font hệ thống) cho Terminal, Editor, Logs; ngưỡng cảnh báo RAM và ổ đĩa cùng thời gian kéo dài; chế độ sao lưu của Editor (file trong `/etc`, mọi file, không).
- Trang chào khi chưa có server nào.

### Thay đổi
- Đóng gói thành `Davish.app` / Davish 1.0.0 với icon chính thức, bundle ID `io.github.2hjaito.davish`; macOS tối thiểu 12.
- Server chưa từng mở (kể cả server đầu tiên vừa thêm) mở sẵn Terminal.

### Sửa
- Cron: lệnh thay crontab chạy đúng lệnh người dùng đã xác nhận; trước đây nếu đồng hồ sang giây mới giữa lúc xác nhận và lúc chạy, file sao lưu mang tên khác với tên đã hiển thị.

## [0.9.0] - 2026-10-28

M9 · Đa server.

### Thêm
- Mỗi server nhớ bố cục cửa sổ riêng và mở lại như lúc rời đi.
- Copy file và thư mục từ server này sang server khác, đi qua máy người dùng: kéo thả lên mục của server khác trong sidebar Files, hoặc `⌘⌥S`. Không bao giờ ghi đè; trùng tên thì giữ cả hai.
- Sidebar Files hiện thư mục nhà và thư mục gần nhất của các server khác.

## [0.8.0] - 2026-10-28

M8 · Cron + Nginx.

### Thêm
- App Cron: tác vụ của user và của hệ thống (chỉ đọc), lịch nói bằng tiếng Việt, lần chạy kế tiếp theo giờ server, thêm/sửa/xóa/bật/tắt có xác nhận (dòng thay đổi và đúng lệnh), sao lưu crontab trước khi ghi, xem các lần chạy gần nhất.
- App Nginx: danh sách site và server block, bật/tắt site bằng link, `nginx -t` trước mỗi lần reload, lỗi cấu hình mở đúng file và dòng trong Editor.
- Thông báo khi chứng chỉ TLS của site đang bật còn 14 ngày hoặc ít hơn.
- Icon Dock cho Cron và Nginx.

## [0.7.0] - 2026-10-28

M7 · Services, Docker, Logs.

### Thêm
- App Services: dịch vụ systemd theo trạng thái, chi tiết, journal; start/stop/restart có xác nhận.
- App Docker: container, CPU và RAM, log, inspect; start/stop/restart có xác nhận.
- App Logs: theo dõi journal, file hoặc container; lọc, tạm dừng, tô dòng lỗi; bấm đường dẫn file trong log để mở Editor đúng dòng.
- Lệnh bị từ chối vì thiếu quyền đưa ra nút "Chạy trong Terminal" với lệnh `sudo` gõ sẵn.
- Cửa sổ Nhật ký lệnh: mọi lệnh Davish chạy trên server.
- Thông báo khi dịch vụ bị lỗi hoặc container khởi động lại liên tục.

## [0.6.0] - 2026-10-27

M6 · Editor.

### Thêm
- Editor CodeMirror 6, tô màu theo loại file.
- Lưu atomic, giữ mode và chủ file; sao lưu `.bak-<UTC>` cho file trong `/etc`.
- Hỏi trước khi ghi nếu file trên server đã đổi từ lúc mở.
- File không có quyền ghi mở chỉ đọc, kèm nút mở `sudoedit` trong Terminal.

## [0.5.0] - 2026-10-27

M5 · Files.

### Thêm
- App Files kiểu Finder: duyệt, sắp xếp, tìm, đổi tên, xóa có xác nhận đúng lệnh, nén.
- Tải lên, tải xuống, kéo thả từ hệ điều hành; cửa sổ Chuyển file với tiến độ và hủy.

## [0.4.0] - 2026-10-27

M4 · Chỉ số + Monitor.

### Thêm
- Chỉ số CPU, RAM, ổ đĩa, mạng trên menu bar.
- App Monitor: biểu đồ, từng nhân CPU, bộ nhớ, GPU NVIDIA, ổ đĩa, mạng, tiến trình, thông tin hệ thống.
- Cảnh báo RAM và ổ đĩa vượt ngưỡng.

## [0.3.0] - 2026-10-27

M0–M3 · Khung dự án, desktop, kết nối, Terminal.

### Thêm
- Desktop kiểu macOS: menu bar, Dock, cửa sổ kéo/resize/snap, workspace theo server, kính mờ.
- Kết nối SSH qua agent, key hoặc mật khẩu; bí mật trong keychain; xác nhận host key lần đầu, từ chối khi host key đổi; tự kết nối lại.
- Terminal xterm.js nhiều tab, prompt Davish không cài gì lên server.

[Unreleased]: https://github.com/2hjaito/davish/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/2hjaito/davish/compare/v0.9.0...v1.0.0
[0.9.0]: https://github.com/2hjaito/davish/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/2hjaito/davish/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/2hjaito/davish/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/2hjaito/davish/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/2hjaito/davish/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/2hjaito/davish/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/2hjaito/davish/releases/tag/v0.3.0
