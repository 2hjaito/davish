---
title: "An ninh"
description: "Ai đăng nhập qua SSH và ai đang dò."
sidebar:
  order: 4
---

Ai đăng nhập qua SSH và ai đang dò. Phía trên là số liệu 7 ngày (lần sai, user không tồn tại, số địa chỉ, đã ban, đăng nhập thành công), biểu đồ lần sai theo giờ, địa chỉ và user bị thử nhiều nhất, đánh dấu địa chỉ fail2ban đã ban hoặc ufw đã chặn. Bên dưới là ai đang đăng nhập lúc này, mọi địa chỉ kèm tên user đã thử, đăng nhập gần đây, ufw và fail2ban. Đăng nhập sai và Đã chặn là bảng bôi đen, copy được: mỗi địa chỉ kèm số lần, user đã thử, lần thử cuối, đã chặn hay chưa, nút Chặn hoặc Bỏ chặn ngay trên dòng; tick nhiều dòng để chặn một lần. Mọi lần chặn, bỏ chặn, ban, bỏ ban đều hiện đúng lệnh trước, xác nhận là chạy qua `sudo`, hỏi mật khẩu sudo một lần mỗi phiên; những gì cần root để đọc (`lastb`, ufw, fail2ban) cũng được đọc bằng root. **Cho phép đăng nhập bằng mật khẩu** là một công tắc: tắt thì trước hết thử key đăng nhập được qua một kết nối mới, sao lưu cấu hình sshd, thêm một file đứng trước các file khác, kiểm tra bằng `sshd -t` và `sshd -T`, reload sshd rồi thử lại key, và hoàn tác nếu key không còn đăng nhập được. Không cho tắt khi một khối `Match` hoặc `AuthenticationMethods` sẽ khiến người ta bị khóa hoặc thay đổi không có tác dụng, hoặc khi chính Davish đang đăng nhập bằng mật khẩu. Hộp thoại hiện cả script trước; cùng công tắc đó bật lại.

![Lần đăng nhập sai theo giờ, địa chỉ và user bị thử nhiều nhất](../../../../../../../assets/screenshots/security.jpg)
