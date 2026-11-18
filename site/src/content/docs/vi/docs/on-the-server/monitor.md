---
title: "Monitor"
description: "CPU từng nhân, bộ nhớ và swap, ổ đĩa, mạng theo interface, GPU NVIDIA, tiến trình."
sidebar:
  order: 7
---

CPU từng nhân, bộ nhớ và swap, ổ đĩa, mạng theo interface, GPU NVIDIA, tiến trình. Cảnh báo RAM và ổ đĩa chỉnh được ngưỡng.

Bên dưới các biểu đồ, trang **Tổng quan** mô tả máy trong bốn thẻ: hệ thống (OS, kernel, ảo hóa, khởi động lúc nào, múi giờ, systemd, ai đang đăng nhập), phần cứng (model CPU, số nhân và luồng, RAM, swap), ổ đĩa (mỗi ổ ghi NVMe, SSD, HDD hay ổ ảo, kèm model, dung lượng và những gì được mount trên đó; bấm mount để mở Ổ đĩa; có Docker thì mount chứa dữ liệu Docker được gắn nhãn, và mỗi mount ghi các project hoặc container có volume hay bind mount trên đó) và mạng (interface xếp theo LAN, VPN như Tailscale hay WireGuard, rồi bridge Docker, kèm địa chỉ, gateway mặc định và DNS). Đọc một lần bằng `lsblk`, `ip` và `/proc`; server thiếu các công cụ này thì chỉ hiện ít hơn.

Trên server Windows (OpenSSH, Windows PowerShell 5.1, không cài thêm gì) Monitor đọc cùng các số liệu qua PowerShell và CIM: CPU, bộ nhớ, ổ đĩa, mạng, I/O đĩa và tiến trình. Windows không có load average, I/O wait hay steal nên các mục đó được ẩn. Các app cần Linux (Services, Docker, Logs, Cron, Nginx…) chưa dùng được trên server Windows; Launchpad hiện chúng mờ.
