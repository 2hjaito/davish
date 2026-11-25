---
title: "Không phải root?"
description: "Đăng nhập bằng user có quyền sudo thì các app chạy bằng root những gì server từ chối: Services, Docker , Cron, Logs, Files và Editor, Disk, Nginx, An ninh và Database."
sidebar:
  order: 10
---

Đăng nhập bằng user có quyền `sudo` thì các app chạy bằng root những gì server từ chối: Services, Docker (khi không thuộc nhóm `docker`), Cron, Logs, Files và Editor, Disk, Nginx, An ninh và Database. Davish hiện đúng lệnh và hỏi mật khẩu sudo một lần cho mỗi server trong mỗi phiên: các app của server đó dùng chung cho tới khi 15 phút không dùng tới (giống chính sudo; chọn 5 phút, 1 giờ hoặc cả phiên ở **Cài đặt → Bảo mật**), khi bạn ngắt kết nối, mất kết nối hoặc khóa app. Sau đó sẽ hỏi lại. Mật khẩu chỉ giữ trong bộ nhớ (không ghi ra đĩa, keychain hay log) và đưa cho sudo qua stdin, không bao giờ trên dòng lệnh. **Quên mật khẩu sudo** trong menu server xóa nó ngay. Sudo không cần mật khẩu thì không hỏi gì. User không có quyền sudo thì không được hỏi. Những gì có thể khóa bạn khỏi server (bật ufw) và đường dẫn mà user khác có thể tráo vẫn chỉ qua Terminal.
