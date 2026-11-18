---
title: Cài đặt
description: Cài Davish trên macOS, Windows hoặc Linux.
sidebar:
  order: 1
---

Bản build sẵn cho macOS (universal), Windows và Linux nằm trong từng [release](https://github.com/2hjaito/davish/releases), hoặc dùng các nút ở [trang chủ](/vi/#download). Davish miễn phí và mã nguồn mở (GPL-3.0).

## macOS

macOS 12 trở lên, Apple silicon hoặc Intel. Mở `Davish-…-macos-universal.pkg` để cài vào Applications.

Davish chưa được ký số nên macOS sẽ chặn lần đầu:

1. Bấm **Done**.
2. Mở **Cài đặt hệ thống → Quyền riêng tư & Bảo mật**.
3. Bấm **Vẫn mở** ở dòng nhắc tới Davish (làm lại cho `Davish.app` nếu bị chặn).

Hoặc chạy `xattr -dr com.apple.quarantine` trên file đã tải.

## Windows

Windows 10 trở lên, 64-bit. Chạy `Davish-…-windows-amd64-setup.exe`; bộ cài tự cài WebView2 runtime nếu máy chưa có. SmartScreen có thể cảnh báo nhà phát hành không xác định: chọn **More info → Run anyway**.

File `.zip` là bản portable: giải nén rồi chạy `Davish.exe`, không cần cài.

## Linux

x86-64, có GTK 3 và WebKitGTK 4.1 (trên Debian và Ubuntu: `libgtk-3-0` và `libwebkit2gtk-4.1-0`). Giải nén `Davish-…-linux-amd64.tar.gz` rồi chạy `Davish`.

## Cập nhật

Davish tự kiểm tra bản mới. Có bản mới thì hiện thay đổi kèm **Cập nhật**, **Để sau** và **Bỏ qua bản này**; Davish tự tải, kiểm chữ ký rồi khởi động lại vào bản mới. Xem [Cập nhật](/vi/docs/the-desktop/updates/).
