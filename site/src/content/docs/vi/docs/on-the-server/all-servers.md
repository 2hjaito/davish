---
title: "Tất cả server"
description: "Mọi server trong một cửa sổ, mở từ menu server, Launchpad hoặc Spotlight: đã kết nối chưa, CPU, RAM, ổ /, load, thời gian chạy và mạng, kèm số cảnh báo  và cảnh báo mới nhất."
sidebar:
  order: 8
---

Mọi server trong một cửa sổ, mở từ menu server, Launchpad hoặc Spotlight: đã kết nối chưa, CPU, RAM, ổ `/`, load, thời gian chạy và mạng, kèm số cảnh báo (service lỗi, container khởi động lại liên tục, ổ đầy…) và cảnh báo mới nhất. Kết nối hoặc chuyển sang server ngay trên thẻ của nó. Số liệu chỉ được thu khi cửa sổ đang mở, qua cùng phiên mà Monitor dùng.

Nút **Bản đồ** trên thẻ server đã kết nối cho thấy request nào tới chương trình nào: mỗi site nginx là một chuỗi `tên miền → nginx :443 → :3000 → container (stack)`, site tĩnh kèm thư mục, và các cổng mở ra ngoài mà không site nào proxy tới. Cổng database mở ra Internet, upstream không ai nghe và chứng chỉ hết hạn trong 14 ngày được đưa lên đầu, mỗi cái có nút mở đúng app để sửa; mỗi mắt xích bấm được để mở app của nó (Nginx, Trình duyệt, Docker tại container đó). Bên dưới là những gì đang chạy (stack compose và chương trình đang nghe) và các ổ đĩa. Bản đồ chỉ đọc những gì Nginx, Cổng, Docker và Ổ đĩa vốn đã đọc. Từ 4 server trở lên, cửa sổ có ô lọc theo tên và xếp server có cảnh báo lên trước. Dạng **Danh sách** hiện mỗi server một dòng gọn (số liệu, cảnh báo, bản đồ, kết nối hoặc ngắt kết nối, sửa), và cửa sổ tự thêm, nhập, sửa, ngắt kết nối server. `⌘⇧S` (Ctrl+Shift+S) mở nó từ bất cứ đâu.

![Mọi server trong một màn hình](../../../../../../../assets/screenshots/servers.jpg)

**Chạy lệnh…** chạy một lệnh trên các server bạn chọn, như `ssh server lệnh`: mỗi server một tab terminal, tối đa 4 server cùng lúc, các server còn lại chờ lượt. Server nào hỏi lại (mật khẩu sudo, `[Y/n]`) thì tab đó được đưa lên để bạn trả lời ngay trong tab; Davish không bao giờ tự nhập mật khẩu sudo khi chạy nhiều server. Hộp xác nhận hiện đủ tên server và đúng lệnh; từ 3 server trở lên, hoặc lệnh trông như xóa, dừng, ghi đè thứ gì đó, phải gõ số server mới chạy. Mỗi tab kết thúc với exit code; **Chạy lại** cho các server lỗi. **Cập nhật package** dựng lệnh cập nhật riêng cho từng server với `sudo -n`: server nào sudo đòi mật khẩu sẽ báo, kèm nút mở Terminal. Mỗi server có một dòng trong Nhật ký lệnh.

![Một lệnh trên 4 server, một tab đang chờ mật khẩu sudo](../../../../../../../assets/screenshots/multi-run.jpg)
