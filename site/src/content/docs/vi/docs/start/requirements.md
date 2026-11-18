---
title: Server cần gì
description: Chỉ cần OpenSSH có SFTP; một vài app dùng công cụ mà server có thể đã có sẵn.
sidebar:
  order: 3
---

Davish không cần agent: không cài, không chép, không để lại gì trên server. Mọi thứ chạy từ máy bạn qua SSH.

| Tính năng | Yêu cầu |
|---|---|
| Tất cả | OpenSSH server có subsystem SFTP |
| Services, Logs (journal) | systemd; muốn đọc log hệ thống thì user thuộc nhóm `adm` hoặc `systemd-journal` |
| Docker | lệnh `docker`; user thuộc nhóm `docker` (chưa có thì Davish gợi ý lệnh cài) |
| Cron | `crontab` |
| Nginx | nginx với `sites-available` / `sites-enabled` hoặc `conf.d` |
| Database | client của từng engine: `mysql`, `psql`, `mongosh` hoặc `sqlite3` |

Thao tác cần root (dừng dịch vụ hệ thống, sửa file của root) hiện đúng lệnh trước, rồi chạy qua `sudo` với mật khẩu hỏi một lần mỗi phiên, hoặc chuyển sang Terminal với `sudo` gõ sẵn. Không có gì chạy ngầm bằng root, và những gì có thể khóa bạn khỏi server vẫn chỉ qua Terminal.

Server Windows cũng dùng được ở mức cơ bản: Monitor đọc được, phần còn lại tùy thứ app cần.
