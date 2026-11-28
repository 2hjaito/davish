---
title: "Monitor"
description: "CPU từng nhân, bộ nhớ và swap, ổ đĩa, mạng theo interface, GPU (NVIDIA, AMD, Intel), tiến trình."
sidebar:
  order: 7
---

CPU từng nhân, bộ nhớ và swap, ổ đĩa, mạng theo interface, GPU (NVIDIA, AMD, Intel), tiến trình. Cảnh báo RAM và ổ đĩa chỉnh được ngưỡng.

Bên dưới các biểu đồ, trang **Tổng quan** mô tả máy trong bốn thẻ: hệ thống (OS, kernel, ảo hóa, khởi động lúc nào, múi giờ, systemd, ai đang đăng nhập), phần cứng (model CPU, số nhân và luồng, RAM, swap), ổ đĩa (mỗi ổ ghi NVMe, SSD, HDD hay ổ ảo, kèm model, dung lượng và những gì được mount trên đó; bấm mount để mở Ổ đĩa; có Docker thì mount chứa dữ liệu Docker được gắn nhãn, và mỗi mount ghi các project hoặc container có volume hay bind mount trên đó) và mạng (interface xếp theo LAN, VPN như Tailscale hay WireGuard, rồi bridge Docker, kèm địa chỉ, gateway mặc định và DNS). Đọc một lần bằng `lsblk`, `ip` và `/proc`; server thiếu các công cụ này thì chỉ hiện ít hơn.

Bên dưới các thẻ GPU, bảng **Tiến trình** liệt kê các tiến trình đang dùng GPU, giống nvtop: PID, người dùng, GPU, loại (tính toán hay đồ họa), mức sử dụng, bộ nhớ GPU và tỉ lệ trên card, CPU, bộ nhớ máy và dòng lệnh, sắp xếp được theo mọi cột. Davish đọc `nvidia-smi` (`--query-compute-apps` và một mẫu `pmon` dài một giây) cùng `/proc` của riêng các tiến trình đó, mỗi 3 giây và chỉ khi đang mở tab GPU. Driver không có `pmon` (chế độ MIG, một số driver cũ) thì chỉ hiện tiến trình tính toán, không có mức sử dụng; Windows ở chế độ WDDM không báo bộ nhớ GPU theo từng tiến trình; tiến trình `nvidia-smi` thấy nhưng phiên đăng nhập không thấy (`nvidia-smi` chạy trong container) vẫn giữ tên, nhưng không có người dùng, CPU hay bộ nhớ máy. GPU AMD và Intel không có `nvidia-smi`: trên Linux, Davish đọc thẻ từ `/sys/class/drm` (AMD: tải, bộ nhớ GPU, nhiệt độ, công suất; Intel: chỉ có tên vì driver không báo tải) và đọc thời gian GPU cùng bộ nhớ GPU của từng tiến trình từ các descriptor DRM trong `/proc/PID/fdinfo`, nguồn nvtop dùng, nên cột mức sử dụng chỉ có từ lần đọc thứ hai, và tài khoản không phải root chỉ thấy tiến trình của chính nó. Trên Windows, mọi hãng đọc từ bộ đếm hiệu năng GPU (tải theo engine, bộ nhớ theo card và theo tiến trình), không có nhiệt độ hay công suất.

Trên server Windows (OpenSSH, Windows PowerShell 5.1, không cài thêm gì) Monitor đọc cùng các số liệu qua PowerShell và CIM: CPU, bộ nhớ, ổ đĩa, mạng, I/O đĩa và tiến trình. Windows không có load average, I/O wait hay steal nên các mục đó được ẩn. Các app cần Linux (Services, Docker, Logs, Cron, Nginx…) chưa dùng được trên server Windows; Launchpad hiện chúng mờ.
