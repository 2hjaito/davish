# Changelog

Mọi thay đổi đáng chú ý của Davish. Định dạng theo [Keep a Changelog](https://keepachangelog.com/vi/1.1.0/), phiên bản theo [SemVer](https://semver.org/lang/vi/): `v0.<milestone>.<patch>` trong giai đoạn phát triển, `v1.0.0` khi xong M10 (`docs/GIT.md`).

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

[1.0.0]: https://github.com/2hjaito/davish/compare/v0.9.0...v1.0.0
[0.9.0]: https://github.com/2hjaito/davish/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/2hjaito/davish/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/2hjaito/davish/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/2hjaito/davish/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/2hjaito/davish/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/2hjaito/davish/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/2hjaito/davish/releases/tag/v0.3.0
