---
title: Kết nối server đầu tiên
description: Thêm server bằng key hoặc mật khẩu, kiểm tra host key, hoặc nhập các host bạn đã có.
sidebar:
  order: 2
---

Lần đầu mở Davish, app sẽ hỏi server. Từ đó về sau, Davish tự kết nối server bạn dùng gần nhất ngay khi mở.

## Thêm server

Bấm **Thêm server…** (ở màn hình chào, menu server trên menu bar, hoặc trong **Tất cả server**) rồi điền host, cổng (mặc định 22) và user. Sau đó chọn cách đăng nhập:

- **Tự động**, giống lệnh `ssh`: các key trong SSH agent, rồi tới key mặc định trong `~/.ssh` (`id_ed25519`, `id_ecdsa`, `id_rsa`).
- **Private key**: file key bạn chọn, kèm passphrase nếu có.
- **Mật khẩu**.

Mật khẩu và passphrase được giữ trong keychain của hệ điều hành, không bao giờ ghi ra file.

## Kiểm tra host key

Lần đầu kết nối một server, Davish hiện vân tay host key của server và hỏi bạn có chấp nhận không, giống `ssh`. Nếu sau này host key đổi, Davish từ chối kết nối để bảo vệ bạn và nói rõ lý do.

## Nhập các server bạn đã có

Đã có host trong `~/.ssh/config` hoặc Ansible inventory? Nhập một lần, kèm key và `ProxyJump`: xem [ssh config và Ansible](/vi/docs/network-and-security/ssh-config-and-ansible/). Server sau bastion kết nối qua nó, mỗi bậc kiểm tra host key riêng.

## Nhiều server

Mỗi server là một workspace với các cửa sổ riêng. Chuyển bằng `⌘1` … `⌘9` (`Ctrl` trên Windows và Linux) hoặc menu server, và xem tất cả trong [Tất cả server](/vi/docs/on-the-server/all-servers/). Có thể kéo file từ server này sang server khác trong [Files](/vi/docs/on-the-server/files/).

## Khi cần root

Davish kết nối bằng user của bạn. Thao tác cần root hiện đúng lệnh trước, rồi chạy qua `sudo` với mật khẩu hỏi một lần mỗi phiên, hoặc chuyển sang Terminal với `sudo` gõ sẵn. Xem [Không phải root?](/vi/docs/services-and-apps/not-root/).
