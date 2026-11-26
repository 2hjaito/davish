---
title: "Đăng nhập chỉ chạy được qua OpenSSH"
description: "Với agent cấp chứng chỉ theo yêu cầu (authentik), tự mở một ssh master và để Davish gắn vào qua control socket của nó."
sidebar:
  order: 7
---

Có những kiểu đăng nhập Davish không tự làm được. Một agent cấp chứng chỉ ngắn hạn theo yêu cầu, như agent của authentik, không liệt kê key nào cho tới khi client gửi `session-bind@openssh.com`, mà SSH client riêng của Davish không gửi được. OpenSSH thì gửi được. Với các server này, Davish gắn vào một kết nối OpenSSH đã mở sẵn: OpenSSH lo việc đăng nhập, và Davish không bao giờ thấy thông tin đăng nhập của bạn.

Cách này chỉ có trên Linux và macOS.

## Thiết lập

1. Mở một ssh master cho server, từ terminal:

   ```sh
   ssh -M -S ~/.ssh/davish-myserver.sock -fN user@host.example.com
   ```

   Nếu agent không phải agent mặc định của bạn, thêm `-o IdentityAgent=/duong/dan/agent.sock`. Đồng ý với những gì agent hỏi. `ssh` kiểm tra host key theo cách quen thuộc, qua `~/.ssh/known_hosts`.
2. Trong Davish, thêm hoặc sửa server. Mở **Nâng cao: dùng ssh control socket** và nhập đường dẫn socket ở bước 1: đường dẫn tuyệt đối, hoặc bắt đầu bằng `~/`.
3. Lưu và kết nối. Terminal, Files và các app khác chạy như với mọi server.

Ô đó thay thế phần xác thực của server, và không dùng chung được với server nhảy. Davish chỉ gắn vào: nó không mở hay tắt master. Nếu master mất, Davish báo, và bạn mở lại.

Giữ đường dẫn ngắn. Đường dẫn socket Unix bị giới hạn khoảng 104 byte trên macOS và 108 trên Linux, vượt quá thì `ssh` báo "ControlPath too long".

## Giữ master chạy bền (Linux)

Master mất khi khởi động lại máy, đăng xuất hoặc rớt mạng. Một service `systemd --user` có thể mở nó khi đăng nhập và khởi động lại khi nó chết. Lưu thành `~/.config/systemd/user/davish-master-myserver.service`, đổi host, và thêm `-o IdentityAgent=...` nếu cần:

```ini
[Unit]
Description=ssh master for Davish (myserver)

[Service]
ExecStartPre=-/usr/bin/rm -f %t/davish-myserver.sock
ExecStart=/usr/bin/ssh -M -S %t/davish-myserver.sock -N \
  -o ServerAliveInterval=30 -o ServerAliveCountMax=3 -o ExitOnForwardFailure=yes \
  user@host.example.com
Restart=always
RestartSec=5

[Install]
WantedBy=default.target
```

Rồi:

```sh
systemctl --user daemon-reload
systemctl --user enable --now davish-master-myserver.service
ssh -S "$XDG_RUNTIME_DIR/davish-myserver.sock" -O check user@host.example.com
```

Lệnh cuối nên in `Master running`. Trong Davish, dùng đường dẫn `$XDG_RUNTIME_DIR/davish-myserver.sock` viết đầy đủ (`echo $XDG_RUNTIME_DIR` cho biết, dạng như `/run/user/1000`). Thư mục đó chỉ riêng bạn vào được.

Ba chi tiết quan trọng:

- `ExecStartPre` xóa socket cũ. Nếu master bị kill, socket của nó còn sót lại, và `ssh -M` sẽ in "ControlSocket ... already exists, disabling multiplexing" rồi chạy như một kết nối thường, không có master, nên Davish không gắn vào được.
- Service không có terminal, nên `ssh` không hỏi được có tin host mới không. Chạy `ssh user@host.example.com` bằng tay một lần trước, để host key nằm trong `~/.ssh/known_hosts`, và chắc rằng agent trả lời được mà không cần bạn ngồi ở terminal (với authentik là hộp thoại trên desktop).
- Nếu agent hỏi sự đồng ý của bạn qua một hộp thoại trên desktop (authentik làm vậy, qua polkit), nó chỉ hiện được khi bạn đang đăng nhập phiên đồ họa. Lúc khởi động máy, trước khi đăng nhập, yêu cầu bị từ chối ("user denied"), và unit có `WantedBy=default.target` cùng `Restart=always` sẽ cứ lỗi lặp lại cho tới khi bạn đăng nhập. Hãy gắn unit vào phiên đồ họa: trong `[Unit]` thêm `After=graphical-session.target` và `PartOf=graphical-session.target`, còn trong `[Install]` dùng `WantedBy=graphical-session.target`. Agent vẫn sẽ hỏi lại mỗi lần master khởi động lại sau khi mạng rớt. Điều này đến từ lần thử với authentik của @PlayerFann14.

Trên macOS, chạy cùng lệnh `ssh` đó từ công cụ bạn dùng để khởi động việc khi đăng nhập.

## Bảo mật

Ai mở được socket thì có shell của bạn mà không cần đăng nhập. Davish từ chối socket thuộc người dùng khác hoặc mà người khác truy cập được (`ssh` tạo nó với quyền đúng, hoặc bạn `chmod 600`). Một thư mục chỉ riêng bạn vào được, như `$XDG_RUNTIME_DIR`, là chỗ tốt để đặt nó.
