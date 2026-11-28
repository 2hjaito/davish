# Changelog

Mọi thay đổi đáng chú ý của Davish. Định dạng theo [Keep a Changelog](https://keepachangelog.com/vi/1.1.0/), phiên bản theo [SemVer](https://semver.org/lang/vi/): `v0.<milestone>.<patch>` trong giai đoạn phát triển, `v1.0.0` khi xong M10 (`.claude/docs/GIT.md`).

## [Unreleased]

### Thêm
- **Tiến trình GPU, giống nvtop**: Monitor → GPU giờ liệt kê các tiến trình trên mỗi GPU NVIDIA bên dưới các thẻ: PID, người dùng, GPU, loại, mức sử dụng, bộ nhớ GPU và tỉ lệ trên card, CPU, bộ nhớ máy và lệnh, sắp xếp được theo mọi cột. Davish chạy `nvidia-smi` và đọc `/proc` của các tiến trình đó mỗi 3 giây, chỉ khi đang mở tab GPU. Driver không có `pmon` thì chỉ hiện tiến trình tính toán, không có mức sử dụng; Windows ở chế độ WDDM không có bộ nhớ GPU theo từng tiến trình.
- Giao diện tiếng Đức: chọn Deutsch trong Cài đặt → Chung → Ngôn ngữ; lần đầu Davish theo máy đặt tiếng Đức.
- **Giao diện tiếng Hàn** (#327, của @wbg1991): chọn 한국어 trong Cài đặt → Chung → Ngôn ngữ; lần đầu Davish theo máy đặt tiếng Hàn. Mô tả cron giờ cũng đọc đúng với những ngôn ngữ đặt giờ sau chữ "lúc".
- **Giao diện tiếng Pháp**: chọn Français trong Cài đặt → Chung → Ngôn ngữ; lần đầu Davish theo máy đặt tiếng Pháp (Pháp, Canada, Bỉ, Thụy Sĩ). Bản dịch đầu làm bằng AI, từ nào sai hay gượng xin báo ở [Discussion #323](https://github.com/2hjaito/davish/discussions/323).
- **Sắp xếp lại Dock**: kéo một icon đã ghim dọc theo Dock để đổi chỗ, như macOS. Các icon khác dịch ra để cho thấy chỗ nó sẽ nằm, và thứ tự mới được giữ khi thả chuột. Icon chỉ đi trong khoảng giữa Launchpad và các app đang mở mà không ghim, nhấn Esc khi đang kéo để trả về chỗ cũ, còn bấm một cái vẫn mở app như trước. Các app được ghim nhưng chỉ có trên hệ điều hành của server khác vẫn giữ nguyên chỗ.
- **Giữ một widget để chỉnh sửa widget**: giữ yên một widget trên desktop nửa giây là vào chế độ chỉnh sửa widget, như màn hình chính của điện thoại, không cần qua Xem → Chỉnh sửa widget. Thả chuột lúc đó không mở gì, còn di chuột mà chưa thả thì kéo widget như trước. Bấm một cái vẫn mở app của widget như trước.
- **Server đi qua ProxyCommand** (#320, của @nullmax): server sau relay WebSocket, Cloudflare Access, AWS SSM hay bastion chỉ vào được bằng `ProxyCommand` giờ thêm được, hoặc nhập từ `~/.ssh/config` kèm lệnh của nó. Davish chạy lệnh đó trên máy này qua shell của bạn, như ssh, với `%h`, `%n`, `%p`, `%r` và `%%` được thay. Lệnh bạn gõ trong hộp thoại server chạy ngay; lệnh đến từ nhập thì được hiện trước, đúng dòng sẽ chạy, Davish hỏi một lần và hỏi lại nếu lệnh đổi. Host hoặc user có ký tự shell đọc được bị từ chối trước khi vào lệnh. Không dùng cùng server jump hay control socket.

### Sửa
- Monitor → GPU giờ đọc cả GPU AMD và Intel, không chỉ NVIDIA (#313): server có Radeon hay iGPU Intel trước đây báo "Không thấy GPU NVIDIA". Trên Linux, thẻ đọc từ kernel (`/sys/class/drm`: mức sử dụng, bộ nhớ GPU, nhiệt độ và công suất với AMD; chỉ có tên với Intel vì driver của nó không báo tải ở đó), còn bảng tiến trình đọc từ các descriptor DRM mà mỗi tiến trình đang mở (`/proc/PID/fdinfo`, nguồn nvtop dùng): thời gian GPU kể từ lần đọc trước và bộ nhớ GPU. Không có root thì chỉ thấy tiến trình của chính bạn. Trên server Windows, mọi hãng đọc từ bộ đếm GPU của Windows (tải theo engine, bộ nhớ theo card và theo tiến trình), không có nhiệt độ hay công suất. Không cài gì lên server. Chưa thử trên GPU AMD hay Intel thật; cảm ơn @tian2992 đã thử NVIDIA và gửi ảnh Intel.
- SSH agent không ký (#332, 1Password trên Windows): khi agent từ chối đăng nhập hoặc đóng kết nối trong lúc chờ bạn xác nhận khóa, Davish báo "ssh handshake failed" (không tới được), dễ làm người ta đi tìm lỗi mạng. Giờ nó báo thẳng là agent không ký, và chỉ tới hộp xác nhận của agent cùng log của chính agent.
- Ngày tháng luôn theo một thứ tự cố định dù chọn ngôn ngữ nào (#326, của @wbg1991): app Git luôn ghi ngày commit theo kiểu Việt Nam, còn đồng hồ menu bar, ngày của file và giờ chạy cron luôn ghi ngày trước tháng. Giờ mỗi ngôn ngữ ghi theo cách của nó.
- SSH agent hỏi bạn trước (Bitwarden): Davish đóng kết nối sau 10 giây, ngay lúc hộp "Confirm SSH key usage" chưa ai kịp bấm, và không ghi gì vào log (#168). Khi agent đang ký đăng nhập, Davish giờ chờ bạn tới một phút (hoặc `DAVISH_AGENT_WAIT` nếu dài hơn), rồi cho server thêm một khoảng để trả lời sau khi đã ký. Log ghi khi agent bắt đầu ký, mất bao lâu và kết thúc ra sao; lỗi bắt tay vì lý do khác cũng được ghi. Nếu quá một phút, lỗi giờ nói rõ agent chưa ký kịp và nhắc tìm cửa sổ xác nhận của nó.
- Linux: gói tải về giờ có icon và `install-desktop.sh`, thêm Davish vào menu ứng dụng. Trên Wayland (KDE, GNOME) icon của cửa sổ lấy từ một mục desktop, nên trước đây cửa sổ không có icon (#305). Chưa thử trên desktop Wayland thật.
- Control socket: server đi qua control socket của OpenSSH không còn báo "Mất kết nối" khoảng 15 giây sau khi kết nối (#301). Yêu cầu keepalive của Davish không được proxy mux của OpenSSH chuyển đi, và nó đóng kết nối đã gửi. Davish giờ không gửi keepalive qua control socket: master tự lo cho server (`ServerAliveInterval`), và master thoát thì socket đóng, Davish vẫn phát hiện. Do @PlayerFann14 tìm ra.
- Cài đặt: một mục có lời giải thích dài và hàng nút bên cạnh làm chữ bị ép thành cột hẹp (mục "Kéo cửa sổ" dài tới 13 dòng ở tiếng Việt). Kéo cửa sổ, Khi mở Davish, Số dòng cuộn, Giữ cảnh báo và Quên mật khẩu sudo giờ đặt nút bên dưới chữ, giống mục Mức hiệu năng phía trên (#304).
- Docker: nút **‹ Container** trên trang của một container lại quay về bảng; trước đây bấm không có tác dụng (#310). Menu của app đọc container của trang sau khi việc quay lại đã bỏ nó đi, và lỗi đó chặn trang đóng lại. Các mục một app thêm vào menu bar giờ cũng biến mất khi phần cửa sổ của chúng đóng; trước đây chúng bị bỏ lại.
- Tất cả server → Bản đồ: bản đồ không có lề nên các thẻ dính sát mép sidebar và mép cửa sổ. Giờ có lề giống danh sách server. Tiêu đề từng hiện hai lần và có hai nút làm mới; giờ tiêu đề là "tên map" ở thanh trên của cửa sổ, và nút làm mới ở đó đọc lại bản đồ (trước đây nó chỉ tải lại danh sách server).
- Xóa một server giờ nêu tên các server khác đang đi qua nó (jump host) và sẽ chuyển sang kết nối thẳng, để thấy điều đó trước khi xác nhận. Phần rà soát còn lại sau #296 không thấy gì cần đổi ở lối tắt script, bộ cập nhật (checksum có chữ ký, không hạ phiên bản) hay đặt lại khóa app.

## [1.15.2] - 2026-11-26

### Thêm
- **Ảnh làm hình nền desktop** (M53): Cài đặt → Giao diện có mục **Ảnh** dưới màu nền: chọn một ảnh PNG, JPEG hoặc WebP (dưới 50 MB) từ máy bạn. Ảnh được thu nhỏ về 2560 px và giữ trên máy này, không tải lên đâu, còn **Làm tối ảnh** làm ảnh tối đi (tối đa 70%) để icon và chữ vẫn dễ đọc. **Cài đặt → Hiệu năng → Ảnh nền** tắt riêng được: Beautiful và Balanced hiện ảnh, Light thì không, và khi chưa chọn ảnh thì không đổi gì và không tốn gì. Trên máy Mac dùng để phát triển, một ảnh tốn thêm khoảng 30 MB bộ nhớ và không đo được thêm CPU hay rớt khung (bench render, 3 lần: `scripts/bench.sh --wallpaper`, ai cũng chạy được trên máy mình).
- **Kéo cửa sổ** (#262): Cài đặt → Hiệu năng có thêm một cài đặt cho cách cửa sổ đi theo chuột khi kéo. **Trực tiếp** như trước. **Đơn giản** vẫn di chuyển cửa sổ nhưng tắt kính cho tới khi thả. **Viền** chỉ di chuyển một đường viền và đặt cửa sổ vào đó khi thả, cho máy vẽ cửa sổ bằng CPU, nơi kéo một cửa sổ mờ lớn thì chậm. Viền có nền mờ bên trong, **Tô nền viền** tắt nền đó để kéo nhẹ nhất. Mặc định là Trực tiếp nếu bạn không đổi, và các mức hiệu năng không đổi.
- **Logo hệ điều hành cạnh tên server**: thẻ và danh sách trong Tất cả server, cùng menu chọn server, hiện logo hệ điều hành mà mỗi server chạy (Ubuntu, Debian, Fedora, Arch, Windows, macOS và nhiều hơn). Davish đọc từ `/etc/os-release` khi server kết nối rồi nhớ lại; server chưa biết hệ hoặc không có logo riêng thì hiện hình server đơn giản
- **Chọn cái gì mở khi Davish khởi động** (#273, #281): Cài đặt → Chung có mục **Khi mở Davish**: **Tất cả server** hiện danh sách server, **Server cuối** mở và kết nối server bạn dùng lần trước (như Davish vẫn làm), còn **Không gì** mở workspace của server cuối mà không kết nối, cho server sau VPN hoặc đang chạy thật. Bản cài mới mở Tất cả server; nếu bạn đã có server thì giữ Server cuối.
- **Tắt đăng nhập bằng mật khẩu từ An ninh**: Đăng nhập bằng mật khẩu ở An ninh giờ là một công tắc thay vì một lệnh gõ trong terminal. Tắt thì thử key đăng nhập được qua một kết nối mới, sao lưu cấu hình sshd, thêm một file, kiểm tra bằng `sshd -t` và `sshd -T`, reload sshd rồi thử lại key, và hoàn tác nếu key không còn đăng nhập được. Không cho tắt khi một khối `Match` hoặc `AuthenticationMethods` sẽ khiến người ta bị khóa hoặc thay đổi không có tác dụng, hoặc khi Davish đang đăng nhập bằng mật khẩu. Hộp thoại hiện cả script trước, và cùng công tắc đó bật lại
- **Nhân bản server** (#257): icon sao chép trên thẻ hoặc hàng trong Tất cả server mở hộp thoại thêm server đã điền sẵn từ server đó, tên thành `tên (copy)`, nên chỉ gõ chỗ khác biệt. Chưa lưu thì chưa tạo gì. Mật khẩu, passphrase của key và control socket không được sao chép.
- **Kết nối qua control socket của OpenSSH** (#240): với kiểu đăng nhập chỉ chạy được qua OpenSSH, như agent cấp chứng chỉ theo yêu cầu (authentik), hộp thoại server có ô **Nâng cao: dùng control socket của ssh**. Bạn tự chạy một master (`ssh -M -S <đường dẫn> -fN user@host`) rồi đưa đường dẫn của nó cho Davish: Davish gắn vào kết nối đó, nên OpenSSH lo việc đăng nhập và Davish không bao giờ thấy thông tin đăng nhập của bạn. Nó thay thế phần xác thực của server đó, và chỉ có trên Linux và macOS. Đã thử với authentik bởi @PlayerFann14.

### Đổi
- Disk → Dọn dẹp: trước khi chạy, hộp xác nhận liệt kê các file sẽ bị xóa (200 file lớn nhất, kèm dung lượng) cho log cũ, log container, core dump, file tạm và `~/.cache`. Danh sách lấy từ đúng cách chọn file của lệnh xóa, chỉ đọc. Khi đọc mà chưa có quyền root, hộp báo rõ. Câu cũ nói log đang ghi không bị đụng tới là sai nên đã bỏ (#296).
- Khi dùng ảnh làm hình nền, menu bar giờ theo độ sáng của ảnh chứ không theo giao diện sáng/tối: chữ sáng trên kính khói khi ảnh tối, chữ tối trên kính sáng khi ảnh sáng, để nhãn, icon và đồng hồ luôn dễ đọc (như macOS). Độ sáng được tính một lần lúc chọn ảnh, từ dải trên cùng và mức làm tối, và chỉ khi kính còn trong suốt.
- Các chỉ số trên thanh menu (CPU, RAM, Ổ đĩa, mạng) to hơn: vòng, số trong vòng và tên tăng khoảng một phần mười, dễ đọc hơn (#288).
- Nhật ký giờ ghi lại khi không kết nối được SSH agent và vì sao (ví dụ pipe của Windows đang bận hoặc không có), để nhìn log biết được "không có agent" khác "có agent nhưng không mở được", và agent trả lời sau bao lâu. Agent chờ bạn mở khóa hoặc xác nhận, như của Bitwarden, có thể cần hơn 5 giây Davish chờ: chạy Davish với `DAVISH_AGENT_WAIT=60` (giây, từ 1 đến 300) để chờ lâu hơn (#168).

### Sửa
- An ninh: Chặn và Ban từ chối địa chỉ có thể khóa bạn ra ngoài: `0.0.0.0/0` và các dải quá rộng khác, loopback, và địa chỉ Davish đang dùng để kết nối (server báo qua `SSH_CLIENT`). Chạy trên nhiều server: bước gõ xác nhận giờ cũng áp cho `apt remove/purge`, `find -delete`, `xargs`, `mv`, `cp`, `sed -i`, `tee`, `docker compose down`, `git reset --hard`, `ufw disable` và các lệnh tương tự. Database: từ chối xóa database hệ thống (`mysql`, `postgres`, `admin`…), và khôi phục MongoDB nói rõ là thay thế từng collection cùng tên.
- Disk: rà lại các mục dọn sau #296. File tạm giờ phải không được đọc lẫn sửa suốt 7 ngày và bỏ qua `/tmp` riêng của dịch vụ. Dọn container đã dừng và volume không dùng của Docker có danh sách xem trước. Các chuỗi hứa quá giờ nói rõ cái gì sẽ mất: image tự build không tải lại được, log container mất lịch sử, `~/.cache` có thể chứa thứ tải về rất nặng. Nginx: xóa chứng chỉ Let's Encrypt của site giờ nói rõ Davish không thấy được mail server hay chương trình khác có thể đang dùng nó.
- Tất cả server: danh sách cuộn lại được khi các server cao hơn cửa sổ, ở cả dạng thẻ và dạng bảng. Ở bản 1.15.1 nó bị cắt ở mép cửa sổ và không tới được các server bên dưới (#256).
- Key `.ppk` của PuTTY dùng làm private key của server giờ được báo rõ cách chuyển đổi (PuTTYgen → Conversions → Export OpenSSH key), thay vì "không đọc được private key" chung chung (#264). Davish vẫn chưa tự đọc được `.ppk`.
- Disk: dọn "log đã xoay vòng" không còn xóa nhầm file như volume lưu trữ đang ghi của PCP (`20261125.00.10.0`). Trước đây mọi tên kết thúc bằng `.0`-`.9` đều khớp; giờ chữ số phải đứng sau một chữ cái, như `syslog.1` hay `auth.log.2` (#296).

## [1.15.1] - 2026-11-24

### Thêm
- **Nhãn server** (#243): gắn một nhãn ngắn như `local`, `staging`, `prod` hoặc tên bạn tự đặt (tối đa 24 ký tự) vào server, trong hộp thoại thêm/sửa server; màu của nhãn là màu server có sẵn. Nhãn hiện cạnh tên server trên menu bar, trong menu chọn server, ở Tất cả server và trên thanh tiêu đề của cửa sổ, kèm một vạch màu mảnh dưới menu bar, để biết mình đang ở môi trường nào. Menu chọn server và Tất cả server có thêm bộ lọc theo nhãn khi có từ hai nhãn trở lên. Server không có nhãn giữ nguyên như cũ.
- **Nhóm server** (#243): trong hộp thoại thêm/sửa server có ô **Nhóm** (một cấp, tối đa 40 ký tự, gợi ý các nhóm đang dùng). Cửa sổ **Tất cả server** giờ dùng bố cục như Docker: thanh bên lọc theo trạng thái kết nối (Tất cả, Đã kết nối, Bị lỗi, Chưa kết nối), theo nhóm và theo nhãn, mỗi mục có số đếm. Server cùng nhóm nằm dưới một tiêu đề thư mục gập được (nhớ lần gập gần nhất, kèm số server đang chạy và số cảnh báo), cả ở dạng thẻ lẫn dạng bảng. Tìm kiếm khớp cả nhóm. Không có server nào có nhóm thì danh sách trông như cũ. Menu chọn server trên thanh menu cũng xếp theo nhóm, gập được, và phím `⌘1…⌘9` đánh số theo đúng thứ tự đang hiện (nhóm A là 1, 2, 3, nhóm B là 4, 5, 6); không dùng nhóm thì không đổi gì.

### Đổi
- Trong Tất cả server, bộ lọc nhãn chuyển từ thanh nút sang thanh bên, các nút Thêm, Nhập và Chạy lệnh thành icon trên thanh công cụ, và nút Thẻ/Bảng thành hai icon như Docker.
- Chấm trạng thái trong menu chọn server giờ chỉ có một màu, là trạng thái kết nối, và chữ dưới tên server bỏ đi (rê chuột vào chấm để xem). Màu riêng của server, vốn đã có chip nhãn mang, hiện thành một vạch mảnh ở mép dòng khi server chưa có nhãn. Các phím `⌘1…⌘9` thẳng một cột, và số server của một nhóm nằm sát tên nhóm.

### Sửa
- Server có tên đăng nhập chứa `@` giờ lưu và nhập từ `~/.ssh/config` được: Mac gia nhập thư mục công ty (`whoami` ra `jane.doe@example.com`), đăng nhập `user@realm` của SSSD hoặc FreeIPA. Tên dài tối đa 64 ký tự (#249, cảm ơn @andre0xFF).

## [1.15.0] - 2026-11-23

### Thêm
- App **Widget** (Launchpad, Spotlight, Dock nếu bạn ghim): thư viện các widget có xem trước trực tiếp, số lượng mỗi loại đang có trên desktop của server đang dùng, và nút thêm cỡ nhỏ hoặc vừa. Icon là ba thẻ trắng trên nền xanh lá.
- Widget **Docker**: bao nhiêu container đang chạy, và cái nào lỗi health hoặc khởi động lặp (cỡ nhỏ, hoặc cỡ vừa kèm tên). Nó đọc `docker ps` mỗi phút một lần khi Davish đang ở phía trước và đọc lại khi bạn quay lại, cho từng server có widget này, và không bao giờ hỏi mật khẩu sudo; server không có Docker hoặc cần sudo thì bị bỏ qua cho đến khi bạn quay lại. Đây là widget duy nhất chạy lệnh. Không có trên server Windows.
- Widget **Services**: bao nhiêu service đang chạy dưới dạng thanh màu, và cái nào lỗi hoặc khởi động lặp, cỡ nhỏ hoặc vừa. Nó đọc `systemctl list-units` mỗi phút một lần khi Davish đang ở phía trước và đọc lại khi bạn quay lại, cẩn thận như widget Docker (không bao giờ hỏi mật khẩu sudo, server không có systemd thì bị bỏ qua). Docker và Services dùng chung một đoạn mã đọc.
- Widget **Ổ đĩa**: ổ đĩa đầy nhất đầy bao nhiêu kèm vòng đo (cỡ nhỏ), hoặc mọi ổ đĩa với thanh và dung lượng (cỡ vừa). Nó đổi sang vàng và đỏ khi gần tới và vượt ngưỡng cảnh báo ổ đĩa trong Cài đặt → Cảnh báo, nên đỏ đúng lúc cảnh báo sẽ báo. Nó dùng chung collector với Monitor: không có lệnh riêng.
- Widget trên desktop: **Xem → Thêm widget: Monitor** đặt CPU, hoặc CPU, RAM và mạng kèm biểu đồ 15 phút, lên desktop của server đang dùng; Kéo một widget để di chuyển, như kéo icon; **Xem → Chỉnh sửa widget** còn để đổi cỡ nhỏ hoặc vừa, và xóa. Mỗi loại có màu và biểu tượng riêng nên dễ phân biệt. Mỗi server có widget riêng. Tắt ở mức hiệu năng Nhẹ, hoặc tắt riêng trong Cài đặt → Hiệu năng → Nâng cao.
- Cài đặt theo từng app. Terminal: bảng màu (Davish, Phosphor, Solarized, Dracula), font riêng, kiểu con trỏ và nhấp nháy, nền đặc và số dòng cuộn lại (1.000, 5.000 hoặc 20.000). Editor: cỡ chữ, xuống dòng, số dòng, dấu cho khoảng trắng và tab, và phím Tab chèn gì. Files: hiện hoặc ẩn các tên bắt đầu bằng dấu chấm.

### Đổi
- Monitor không hỏi server danh sách tiến trình khi cửa sổ không hiện trên màn hình (đã thu nhỏ, ở server khác, hoặc Davish bị ẩn).

### Sửa
- Widget nằm ngay dưới menu bar và cách mép phải, cách thanh menu một khoảng bằng nhau, dù cửa sổ rộng bao nhiêu. Trước đây một hàng phía trên bị để trống và khoảng cách bên phải đổi theo cửa sổ.
- Terminal: màu xám của stack trace, mốc thời gian và gợi ý (ANSI bright black) đọc được trên nền kính tối; bảng màu Davish giờ đặt đủ mọi màu ANSI.
- Terminal: tiến trình của deploy hoặc script nằm trên thanh tiêu đề, cạnh các tab, thay vì nổi lên trên output và che đầu bên phải của những dòng đầu.
- Nhãn phím tắt ghi `Ctrl+Shift+U` và `Ctrl+Alt+L` ngoài Mac, Touch ID chỉ được nhắc trên macOS, nút ⋮ trên thanh công cụ Git và Docker giống các nút bên cạnh, và vài chỗ lệch bố cục (nút Bản đồ ở Tất cả server, chú thích của Disk) đã hết.
- Trình duyệt: trang chuyển hướng bằng đường dẫn tương đối (ví dụ `/` sang `/ui/`, hay thêm dấu `/` cuối) không còn báo "this address only opens inside Davish" (403).

## [1.14.11] - 2026-11-21

### Thêm
- Giao diện: 8 màu hình nền (thêm hồng, vàng, xanh trời, xám) và vòng màu đầu hàng để chọn màu bất kỳ bằng bộ chọn màu của hệ điều hành. Màu mới chuyển dần trong nửa giây.
- Hiệu năng: **Kiểm tra lại** đo khung hình với mức Đẹp rồi nói mức nào hợp với máy này (Đẹp, Cân bằng hoặc Nhẹ), kèm số khung hình/giây và nút dùng ngay.

### Đổi
- Tài khoản: tên nằm dưới ảnh, bấm để sửa ngay tại chỗ. Rê chuột vào ảnh hiện máy ảnh, bấm để chọn ảnh mới hoặc xóa ảnh.
- Ảnh đại diện lưu ở 256 px nên nét trên màn hình Retina (chọn lại ảnh một lần để có bản nét).

### Sửa
- Icon và chữ trong cửa sổ không còn mờ sau khi thu nhỏ rồi mở lại, hoặc khi cửa sổ nằm ở vị trí lẻ điểm ảnh.
- Thanh menu, Dock và icon trên desktop đổi theo màu hình nền ngay, không phải chờ hay chuyển cửa sổ.
- Tên dài không còn tràn khỏi sidebar Cài đặt, màn hình khóa và màn hình ngủ.

## [1.14.10] - 2026-11-21

### Sửa
- Danh sách không có khung chi tiết dùng hết chiều ngang cửa sổ: Bảo mật hiện đủ 5 thẻ trên một hàng, bảng Docker không còn bị cắt mất cột Thao tác.

## [1.14.9] - 2026-11-20

### Thêm
- Git: tài khoản ở đầu sidebar, kèm ô tìm repo. Trang tài khoản cho biết người dùng SSH, tên và email git, có lưu token HTTPS không, các key trong `~/.ssh` và tài khoản GitHub/GitLab mà key của server đăng nhập được (chỉ đọc, mỗi host hỏi riêng, host không kết nối tới thì ẩn).
- Git: repo được nhóm theo user hoặc org của remote, nhóm của tài khoản server đang đăng nhập đứng đầu, repo chưa có remote ở cuối.
- Cài đặt: mục **Tài khoản** ở đầu sidebar với họ tên, ảnh đại diện và mật khẩu khóa app (chuyển từ Bảo mật sang). Tên và ảnh chỉ lưu trên máy này, hiện ở màn hình khóa và màn hình ngủ.

### Đổi
- Git: Clone chạy bằng một nút khi server đã có key được chấp nhận hoặc repo public, xong thì mở luôn repo mới. Repo cần token mới gõ sẵn lệnh vào Terminal như trước.
- Cài đặt: thẻ Davish trước đây mở trang Chung nên bị trùng; giờ là **Về Davish** ở cuối sidebar, chứa phiên bản, giấy phép và phần cập nhật.
- Màn hình ngủ hiện ảnh và tên của bạn như macOS; đã bật khóa app thì thức dậy sẽ hỏi mật khẩu.

## [1.14.8] - 2026-11-19

### Đổi
- Cài đặt dựng lại theo kiểu System Settings của macOS: sidebar có ô tìm kiếm và icon màu cho từng mục (Chung, Giao diện, Hiệu năng, Thanh menu, Dock, Terminal, Editor, Cảnh báo, Bảo mật), bên phải là trang của mục đó với các nhóm thẻ, nút lùi/tiến giữa các trang đã xem. Chế độ sáng/tối chọn bằng ảnh xem trước, số liệu thanh menu bật tắt từng dòng. Ngôn ngữ và Cập nhật chuyển sang Chung, khóa app và sudo sang Bảo mật. Davish nhớ mục mở gần nhất.

### Sửa
- Tất cả server giờ mở bằng `⌘O` (Ctrl+O). Trên Windows, khung web giữ Ctrl+0 cho việc phóng to thu nhỏ và không chuyển cho Davish, nên `⌘0` của bản 1.14.7 không làm gì. Cảm ơn @HungNth (#209).
- Docker: bảng container giờ hiện đúng icon của từng project compose như dạng thẻ, kể cả project chạy image tự build (trước đó chỉ hiện chữ cái đầu).

## [1.14.7] - 2026-11-18

### Sửa
- Tất cả server giờ mở bằng `⌘0` (Ctrl+0), cạnh `⌘1`…`⌘9` cho từng server. Bản 1.14.3 đặt nó vào `⌘⇧S`, trùng với Dịch vụ nên chỉ một trong hai chạy; `⌘⇧S` trả lại cho Dịch vụ. Cảm ơn @HungNth (#202).
- Editor: ô tìm, ô thay, các nút và tùy chọn của thanh tìm kiếm dùng cỡ chữ của giao diện thay vì khoảng 9px, đọc được mà không phải phóng to. Cảm ơn @HungNth (#202).

## [1.14.6] - 2026-11-18

### Thêm
- Database: lưới dữ liệu SQLite sắp xếp được (bấm tiêu đề cột: tăng, giảm, bỏ) và lọc theo chữ một cột có chứa. Davish tự dựng câu SQL từ tên cột có thật trong bảng, vẫn không có ô gõ SQL, vẫn chỉ đọc 100 dòng mỗi lần.

### Sửa
- Chạy lệnh trên nhiều server: bấm Dừng ngay lúc lệnh vừa khởi động có lúc không dừng được và giữ cả lượt chạy tới khi lệnh tự xong. Giờ Davish gửi tín hiệu dừng thẳng tới lệnh trên server, và nếu server vẫn giữ phiên thì job được coi là đã dừng sau 5 giây.

### Thay đổi
- Linux: cửa sổ giờ được vẽ bằng GPU (mặc định của WebKitGTK), nên cuộn, kéo cửa sổ và lớp kính mờ mượt hẳn thay vì do CPU vẽ. Trước đây Davish giữ cách né lỗi của Wails là tắt hẳn GPU cho mọi máy. Đã thử trên card Intel và AMD; với driver NVIDIA độc quyền, nơi lỗi cũ xảy ra, GPU vẫn tắt mặc định, đặt `DAVISH_WEBVIEW_GPU=always` để bật (kèm `WEBKIT_DISABLE_DMABUF_RENDERER=1`, trừ khi bạn tự đặt). `DAVISH_WEBVIEW_GPU=never` tắt GPU trên bất kỳ máy nào, và log của app ghi chế độ đã dùng kèm lý do. Cảm ơn @tian2992 đã phát hiện và thử trên Intel, AMD và NVIDIA (#181, #159, #185).

## [1.14.5] - 2026-11-18

### Thêm
- Docker: bảng container gọn và dễ đọc hơn. Mỗi dòng có logo công nghệ, tên kèm image ở dòng dưới, trạng thái có chấm màu, cổng dạng `8080→80` bấm để mở, cột Tạo, và nút khởi động lại, dừng/chạy, xóa, thêm luôn hiện. Container của một compose project nối với project bằng đường dạng cây. Cột ID chuyển vào trang chi tiết.

### Sửa
- Cửa sổ: ba nút đóng, thu nhỏ, phóng to vẽ lại theo đúng kiểu macOS (viền đậm hơn quanh nút, glyph đặc sắc nét ở 12px, màu riêng khi nhấn và khi cửa sổ không focus). Chỉ đổi màu fill nên hover và nhấn nhẹ hơn trước, không còn dùng `filter`.

## [1.14.4] - 2026-11-18

### Thêm
- Database: bấm tên một bảng SQLite để xem dữ liệu dạng lưới, chỉ đọc, 100 dòng mỗi trang. Chữ dài bị cắt ở 200 ký tự và BLOB chỉ hiện kích thước, lệnh chỉ chạy khi bấm, nên app không nặng thêm.

## [1.14.3] - 2026-11-17

### Thêm
- Tất cả server: xem dạng **Danh sách** gọn bên cạnh dạng thẻ (nhớ lựa chọn), thêm, nhập, sửa và **ngắt kết nối** server ngay trong cửa sổ, và phím tắt **⌘⇧S** (Ctrl+Shift+S) để mở nó. Menu chọn server cũng có nút ngắt kết nối cạnh mỗi server đang kết nối. Theo góp ý của @HungNth và @tian2992 (#178).
- Database: khi server chưa có `sqlite3`, phần SQLite hiện luôn lệnh cài như MariaDB và Postgres thay vì bảo vào Gói phần mềm. Mỗi engine và phần SQLite nằm trong một thẻ riêng.

### Sửa
- Monitor: đường và vùng màu của biểu đồ (Bộ nhớ, CPU, GPU, Mạng, Ổ đĩa và các biểu đồ nhỏ ở Tổng quan) không còn tràn qua mép trái của thẻ. Cảm ơn @thanhtrung5763 (#186).
- macOS và Linux: bôi đen và copy được lệnh hiển thị trong hộp xác nhận, inspector và hộp báo lỗi.


## [1.14.2] - 2026-11-17

### Sửa
- Linux: cửa sổ Davish có icon của app trên thanh tác vụ và Alt+Tab, thay vì icon mặc định của X11 (#181).

## [1.14.1] - 2026-11-17

### Thêm
- Cài đặt → Hiệu năng: bốn chế độ **Tự động**, **Đẹp**, **Cân bằng** và **Nhẹ** thay cho hai công tắc Giảm hiệu ứng kính và Giảm chuyển động. Mỗi chế độ đặt sẵn độ mờ kính, bóng cửa sổ, nền đặc, các mảng màu hình nền và chuyển động; **Nâng cao** chỉnh từng thứ riêng, và trên Linux có thêm hai tùy chọn của WebKitGTK, áp dụng sau khi khởi động lại. **Tự động** giữ giao diện đầy đủ, đo khoảng 1,5 giây khung hình lúc mở app mà không chặn gì, và gợi ý chuyển sang Nhẹ (Dùng Nhẹ / Để sau) khi máy vẽ bằng phần mềm hoặc rớt khung hình; **Đo lại** đo thêm lần nữa. Cảm ơn @tian2992 (#176).

### Sửa
- Terminal: khi bật tăng tốc GPU, ký tự vừa gõ hoặc màn hình một chương trình để lại lúc thoát (ví dụ `btop`) không còn chờ tới lần bấm phím hoặc click kế tiếp mới hiện (#175).
- Giảm hiệu ứng kính giờ dùng nền đặc và bóng ngắn nên vẽ nhẹ hơn thật: nền 96% vẫn phải trộn với hình nền ở mỗi khung hình, và bóng sâu bị vẽ lại khi cuộn. Cảm ơn @tian2992 (#174).

## [1.14.0] - 2026-11-17

### Thêm
- Tất cả server: **Chạy lệnh…** chạy một lệnh trên nhiều server, mỗi server một tab terminal, tối đa 4 server cùng lúc. Tab nào hỏi lại (mật khẩu sudo, `[Y/n]`) thì được đưa lên để bạn trả lời ngay trong tab; Davish không bao giờ tự nhập mật khẩu sudo khi chạy nhiều server. Hộp xác nhận liệt kê đủ server và đúng lệnh, phải gõ số server khi chạy từ 3 server hoặc lệnh trông nguy hiểm. Có exit code từng server, Dừng tất cả, Chạy lại trên server lỗi, và **Cập nhật package** theo trình quản lý package của từng server với `sudo -n`. Mỗi server một dòng trong Nhật ký lệnh.
- Nhật ký lệnh: **Chỉ lệnh thay đổi server** (bật sẵn) ẩn lệnh đọc và polling, và **Xuất script…** biến các lệnh được tick thành bash script chạy lại chúng, mỗi lệnh qua một `ssh -t` riêng để sudo hỏi được. Mỗi server một khối, sudo của Davish viết lại thành `sudo` thường, nội dung gửi qua stdin chỉ để lại comment (không bao giờ được lưu), lệnh lỗi chỉ xuất khi bạn chọn kèm. Sao chép hoặc lưu thành file chạy được.
- Files: ô tìm có ba phạm vi. **Thư mục này** lọc như trước; **Cả cây** tìm tên file trong mọi thư mục con bằng `find`; **Nội dung** tìm dòng chứa chữ đó bằng `grep` (bỏ qua file nhị phân, `.git`, `node_modules`). Nhấn Enter để tìm trên server: tối đa 500 kết quả trong 30 giây, hiện đúng lệnh đã chạy, có nút Dừng và số chỗ không có quyền đọc. Kết quả nội dung mở Editor đúng dòng; nút thư mục đưa về thư mục chứa file. Chỉ đọc, không thử lại bằng sudo.
- Logs: **Tìm trong toàn bộ** (hoặc Enter trong ô lọc) tìm trong cả lịch sử của file, journal hay container ngay trên server, không chỉ 200 dòng đang hiện, và trả 500 dòng khớp gần nhất. Với file, bấm một kết quả để đọc các dòng xung quanh, lật trang trước/sau mà không tải cả file về (file log vài GB vẫn nhanh).
- Nhập server từ Ansible inventory (INI): tab mới trong hộp nhập, cạnh `~/.ssh/config`. Đọc `ansible_host`, `ansible_user`, `ansible_port`, file key và `ProxyJump` trong `ansible_ssh_common_args`, có tính biến của group, group con và dải như `web[01:03]`; chọn cả group một lần. File chỉ được đọc; mật khẩu trong file không bao giờ được đọc, host dùng template, giá trị Vault hoặc kết nối không phải SSH được liệt kê là bỏ qua kèm lý do. Inventory YAML báo rõ là chưa hỗ trợ.

### Sửa
- Database: server chưa có sqlite3 thì hộp Dò đưa luôn lệnh cài sqlite3 theo trình quản lý package của server, kèm nút Chạy trong Terminal (hộp Dò tự đóng để Terminal không bị che), thay vì chỉ bảo đi cài trong Gói phần mềm. Các mục trong app Database hiện thành từng thẻ.
- macOS và Linux: bôi đen và sao chép được chữ ở những chỗ đáng lẽ chọn được (đường dẫn file trong hộp Dò, lệnh trong các hộp xác nhận, diff của Git…). WebKit chỉ đọc `-webkit-user-select`, nên trước đây những chỗ này không chọn được.
- Menu chọn server: nhiều server thì danh sách tự cuộn, các mục Tất cả server, Thêm server, Nhập luôn nằm trong tầm bấm (#178).

## [1.13.10] - 2026-11-16

### Sửa
- Cài Docker từ Setup giờ cài kèm plugin compose (thiếu gói compose thì vẫn cài Docker). Hộp thoại HTTPS của Nginx kiểm tra certbot trước và đưa lệnh cài nếu chưa có, thay vì một lệnh certbot chắc chắn lỗi. Setup cho biết công cụ đã được cài hay chưa, và không còn báo lỗi đúng lúc công cụ chưa cài. Docker cũ hơn 23 khi xem volume báo "Docker quá cũ" thay vì lỗi khó hiểu.
- Windows: chế độ đăng nhập **Automatic** giờ đọc được các file key mặc định trong `~\.ssh` (`id_ed25519`, `id_ecdsa`, `id_rsa`). Trước đây đường dẫn có dấu `\` không được mở rộng thành thư mục home nên các file này không bao giờ được thử (#168).
- SSH agent không trả lời (pipe không ai phục vụ, agent đang chờ mở khóa vault) không còn làm kết nối treo ở "Đang kết nối…": sau 5 giây Davish bỏ qua agent, thử các file key và ghi lý do vào log (#168).
- Đường dẫn key trỏ vào một thư mục (ví dụ `~/.ssh`) giờ báo rõ là cần chọn file private key bên trong, thay vì "Incorrect function" trên Windows (#168).

## [1.13.9] - 2026-11-14

### Sửa
- Windows: chế độ đăng nhập **Automatic** dùng được key trong SSH agent của Windows (dịch vụ “OpenSSH Authentication Agent”, và agent của 1Password, Bitwarden cùng chạy trên pipe đó). Trước đây Davish chỉ tìm agent qua `SSH_AUTH_SOCK`, mà Windows không đặt biến này, nên chỉ thử các file key trong `~/.ssh` (#164). Không tìm thấy key thì thông báo chỉ cách bật dịch vụ.

## [1.13.8] - 2026-11-14

### Sửa
- Dùng được private key mã hóa bằng aes256-gcm, aes128-gcm, chacha20-poly1305, aes128/192-ctr hay aes128/192-cbc (kiểu `ssh-keygen -Z`, Termix và vài công cụ khác tạo ra): trước đây Davish chỉ đọc được key aes256-ctr và aes256-cbc, còn lại báo "unknown cipher" (#162). Passphrase đã lưu không mở được key thì giờ báo đúng như vậy, thay vì "Không đọc được file private key".

## [1.13.7] - 2026-11-14

### Thêm
- Git: mỗi commit trong lịch sử có ảnh tác giả, là avatar GitHub nếu email dạng noreply, không thì Gravatar của email, nằm trên chữ cái đầu tô màu theo tên (không tải được ảnh hay máy offline thì giữ chữ cái). Để tìm ảnh, email tác giả (hoặc mã băm SHA-256 với Gravatar) được gửi tới máy chủ ảnh của GitHub hay Gravatar. Cuộn gần cuối thì tự tải thêm commit, tới 1000 commit; nút Tải thêm chỉ hiện khi vượt mức đó.
- App có danh sách và chi tiết (Git, Dịch vụ, Docker…): kéo đường chia giữa hai bên để nới hay thu hẹp danh sách, nhớ riêng từng app; nhấp đúp để đặt lại, phím mũi tên cũng di chuyển được.

### Thay đổi
- Thanh cuộn là một viên thuôn mảnh, mờ khi để yên, rõ hơn khi rê chuột vào vùng đó và rõ nhất khi rê lên chính thanh cuộn.

### Sửa
- Kết nối: server không kết nối được thì hiện hộp thoại nói rõ lý do, kèm một báo cáo (phiên bản Davish, hệ điều hành, cách xác thực, mã lỗi, nguyên nhân kỹ thuật, các dòng log kết nối) bôi đen hoặc sao chép được để gửi khi báo lỗi. Trong đó không có mật khẩu, key, địa chỉ server hay tên user, thư mục home hiện thành `~`. Bấm vào dòng lỗi màu đỏ trên thẻ server để mở lại. Trước đây có lỗi chỉ báo "Đã có lỗi xảy ra." và mất luôn nguyên nhân, kể cả trong log (#162).
- File private key lưu bằng Notepad (có BOM ở đầu) không đọc được; giờ đã đọc được.

## [1.13.6] - 2026-11-13

### Thêm
- Docker: xem container dạng thẻ, mỗi project compose (hoặc container chạy riêng) một thẻ có logo công nghệ khi nhận ra image (n8n, WordPress, Postgres…), số container đang chạy, image, cổng publish bấm để mở trong Trình duyệt, CPU và RAM; viền đỏ khi có container khởi động lặp, bấm thẻ để xem và điều khiển từng container. Thẻ của project compose lấy favicon của chính dự án, hoặc framework trong `package.json`, `requirements.txt`, `composer.json`, `Gemfile`, `go.mod`… (Next, Nest, Express, Django, FastAPI, Laravel…), bằng cách đọc thư mục project qua SFTP; không chạy lệnh nào. Nút Thẻ/Bảng trên thanh công cụ, mặc định là bảng, Davish nhớ lựa chọn.
- Tất cả server: nút **Bản đồ** trên thẻ server đã kết nối cho thấy request đi đường nào: mỗi site nginx là một chuỗi `tên miền → nginx :443 → :3000 → container (stack)`, site tĩnh kèm thư mục, và các cổng mở ra ngoài mà không đi qua proxy. Cảnh báo lên đầu: cổng database mở ra Internet, upstream không ai nghe, chứng chỉ hết hạn trong 14 ngày, mỗi cảnh báo có nút mở đúng app để sửa. Bên dưới là những gì đang chạy và các ổ đĩa. Chỉ đọc. Từ 4 server trở lên có ô lọc theo tên và xếp server có cảnh báo lên trước.
- Monitor → Tổng quan: thông tin máy gộp vào cuối trang (bỏ mục Thông tin riêng và dải Tải · Thời gian hoạt động · Swap · GPU: load nằm dưới số CPU, swap đang dùng và GPU vào thẻ Phần cứng, GPU chỉ hiện khi có), bốn thẻ Hệ thống (khởi động lúc, múi giờ, systemd, ai đang đăng nhập), Phần cứng (số nhân, luồng, socket), Ổ đĩa (ổ vật lý NVMe/SSD/HDD/ảo, model, dung lượng, phân vùng và mount; mount nào chứa dữ liệu Docker và volume/bind mount của project nào, qua `docker info` và `docker inspect` ghi vào Nhật ký lệnh) và Mạng (interface LAN, VPN, Docker kèm IP, gateway, DNS). Chỉ đọc qua `lsblk`, `ip` và `/proc`.
- Launchpad: hàng **Gần đây** ở đầu tab Tất cả, gồm 5 app mở gần nhất (từ Dock, Spotlight hay Launchpad), nhớ qua các lần mở app.
- Cài đặt → Dock: đặt Dock ở cạnh dưới, trái, phải hoặc trên (cửa sổ, thông báo chừa chỗ; tự ẩn hiện lại khi chạm đúng cạnh đó), và công tắc nền Dock trong suốt chỉ còn các icon.

### Sửa
- Logs: bôi đen để copy được rồi. Trên macOS và Linux, WebKit chỉ đọc thuộc tính `-webkit-user-select` nên mọi dòng log bị khóa chọn. Khi đang có vùng chọn, log tạm đứng yên (không cuộn, không thêm dòng) để vùng chọn không trượt, bỏ chọn thì chạy tiếp; `⌘C` copy đúng từng dòng kể cả xuống dòng. Thêm nút Sao chép các dòng đang hiện (theo bộ lọc).

## [1.13.5] - 2026-11-12

### Thay đổi
- Mật khẩu sudo tự được quên sau 15 phút không dùng, tính riêng từng server, giống chính sudo; trước đây Davish giữ tới khi ngắt kết nối hoặc khóa app. Chọn 5 phút, 1 giờ hoặc cả phiên trong Cài đặt → Khóa app → Quên mật khẩu sudo. Hỏi trạng thái sudo không tính là dùng.

## [1.13.4] - 2026-11-11

### Sửa
- Docker: đăng nhập bằng user không thuộc nhóm `docker` thì danh sách container, image, volume và các thao tác start, stop, restart, xóa, tạo network/volume không còn chỉ báo "cần root": Davish chạy lại đúng lệnh đó qua `sudo`. Không cần mật khẩu thì chạy luôn; cần mật khẩu thì app hiện nút "Xem bằng sudo…" và hỏi một lần mỗi phiên, như Dịch vụ và Nginx (#117). Lệnh gõ sẵn vào Terminal (run, pull, exec, compose) giữ nguyên.
- Cron: đăng nhập bằng user có quyền sudo thì Cron hiện thêm crontab của root, thêm, sửa, xóa, bật, tắt job trong đó qua `sudo crontab -u root -`. Crontab cũ được backup vào `/root` trước, hộp xác nhận hiện đúng lệnh và cảnh báo job sẽ chạy với quyền root. File trong `/etc/crontab`, `/etc/cron.d` không đọc được và nhật ký chạy job cũng đọc qua sudo. Cần mật khẩu thì có nút "Xem bằng sudo…", hỏi một lần mỗi phiên (#117). User không có sudo không thấy crontab của root. File trong `/etc/cron.d` vẫn sửa bằng Editor.
- Logs: log không đọc được vì thiếu quyền (journal hệ thống khi không thuộc nhóm `adm`/`systemd-journal`, file của root, docker khi không thuộc nhóm `docker`) được theo dõi lại qua `sudo`: không cần mật khẩu thì chạy luôn, cần thì có nút "Xem bằng sudo…" và hỏi một lần mỗi phiên (#117). Mật khẩu chỉ đi qua stdin của phiên, không nằm trên dòng lệnh hay trong audit log; sudo từ chối thì mật khẩu bị quên và hỏi lại.
- Files và Editor: đăng nhập bằng user thường thì xóa file của root và sửa file chỉ đọc không còn chỉ lùi về Terminal (#117). Xóa: server từ chối thì Davish chạy lại đúng lệnh `rm` đã xác nhận qua `sudo`. Davish không bao giờ xóa bằng root `/`, các thư mục hệ thống (`/usr`, `/boot`, `/proc`…), thư mục home, hay các file cần để đăng nhập (`/etc/passwd`, `/etc/shadow`, `/etc/sudoers`, `/etc/ssh`…). Editor: nút "Sửa bằng sudo…"; khi lưu, Davish hiện đúng lệnh rồi ghi qua `sudo` (file tạm do `mktemp` tạo, giữ owner và quyền, có bản sao lưu), từ chối nếu file đã đổi kể từ lúc mở. Hỏi mật khẩu sudo một lần mỗi phiên; nút Terminal trong hộp hỏi mật khẩu gõ `sudoedit`. Xóa trong Disk cũng chạy qua `sudo`, vẫn theo danh sách cấm của Disk.
- Disk: đăng nhập bằng user thường thì các cleanup cần root (journal, log cũ trong `/var/log`, apt, dnf, crash dump, `/tmp`, snap, log container) chạy qua `sudo` sau khi xác nhận, thay vì chỉ gõ sẵn vào Terminal (#117); vẫn có nút Terminal. Có mật khẩu sudo trong phiên thì Davish đo cả ổ đĩa và dung lượng dọn được bằng root; chưa có thì nút "Đo bằng sudo…" hiện đúng lệnh sẽ chạy. Làm rỗng file của root cũng chạy lại qua `sudo`, vẫn theo danh sách cấm của Disk. Cache trong home của user (`~/.cache`, npm) luôn chạy bằng chính user.
- An ninh: chặn, bỏ chặn bằng ufw và ban, unban bằng fail2ban không còn đòi sudo không mật khẩu: Davish chạy qua `sudo` và hỏi mật khẩu sudo một lần mỗi phiên, như các app khác (#117). Những gì cần root để đọc (`lastb`, trạng thái ufw và fail2ban, đăng nhập trong journal) cũng được đọc bằng root khi có thể; cần mật khẩu thì có nút "Xem bằng sudo…" hiện đúng lệnh. Bật ufw và tắt đăng nhập bằng mật khẩu, vốn có thể khóa bạn khỏi server, vẫn chỉ qua Terminal. User không có quyền sudo không được mời nhập mật khẩu.
- Database: khi chỉ root được kết nối MariaDB hoặc PostgreSQL, Davish không còn đòi sudo không mật khẩu: liệt kê, tạo, xóa, dump và restore chạy qua `sudo`, hỏi mật khẩu sudo một lần mỗi phiên (#117); có nút "Xem bằng sudo…". Mật khẩu không bao giờ nằm trên dòng lệnh: lệnh SQL đi sau mật khẩu trên stdin, file dump vẫn do chính user ghi vào `~/db-backups`, restore chuyển đúng dòng mật khẩu cho sudo rồi mới tới nội dung dump. User không có quyền sudo không được mời nhập mật khẩu.

## [1.13.3] - 2026-11-11

### Thêm
- Monitor trên server Windows: CPU từng nhân, bộ nhớ và swap, ổ đĩa, mạng, I/O đĩa, GPU, tiến trình, thông tin hệ thống và cảnh báo service/container, đọc qua Windows PowerShell 5.1 và CIM trên SSH. Không cài gì lên server. Load average, I/O wait và steal không có trên Windows nên được ẩn. Chạy được dù shell SSH của server là cmd.exe hay PowerShell.

### Sửa
- Monitor trên server Windows hoặc macOS không còn chờ "số liệu đầu tiên" mãi mãi. Davish nhận biết hệ điều hành của server ngay sau khi kết nối, chạy nền nên không làm chậm việc kết nối; Monitor báo chưa hỗ trợ hệ điều hành đó (hoặc server không trả số liệu nào) thay vì thử lại trong im lặng. Server Linux không đổi.
- Terminal trên server Windows mở shell đăng nhập thường thay vì gửi dòng lệnh khởi động chỉ dành cho Linux.

### Thay đổi
- Các app cần shell POSIX (Services, Docker, Git, Logs, Cron, Nginx…) được ẩn khỏi Dock, Spotlight và menu Go trên server Windows, và hiện mờ trong Launchpad kèm lý do. Server macOS không đổi gì ngoài Monitor, nay báo chưa hỗ trợ thay vì chờ mãi.

## [1.13.2] - 2026-11-11

### Thêm
- Giao diện tiếng Tây Ban Nha (Tây Ban Nha): chọn Español trong Cài đặt → Giao diện → Ngôn ngữ; lần đầu Davish theo máy đặt tiếng Tây Ban Nha. Bản dịch là bản nháp đầu, chờ người bản xứ đọc lại (#124).

### Bảo mật
- Build bằng Go 1.27.4, bản sửa lỗi server HTTP/1 bị sập trong net/http (GO-2026-6859).

## [1.13.1] - 2026-11-11

### Sửa
- Hộp xác nhận lệnh: nút chạy trong Terminal nằm ở hàng nút dưới cùng, có chữ, thay vì một icon nhỏ cạnh lệnh; lệnh dài xuống dòng ở khoảng trắng nên địa chỉ IP hay đường dẫn không bị cắt đôi.
- Cài đặt: ngôn ngữ chọn từ danh sách thả xuống thay vì một hàng nút, để thêm ngôn ngữ vẫn vừa.

## [1.13.0] - 2026-11-10

### Thêm
- Database: MongoDB (cài trên server). Liệt kê database kèm số collection và dung lượng, tạo database kèm user có mật khẩu ngẫu nhiên, xóa, dump và khôi phục (`mongodump`/`mongorestore`), mở tunnel cho Compass. MongoDB bật xác thực thì đăng nhập một lần bằng từng trường hoặc connection string; Davish thử trước, được mới lưu vào keychain của máy (như mật khẩu SSH), có nút "Quên thông tin đăng nhập". Connection string chỉ đi qua stdin, không nằm trên dòng lệnh hay nhật ký lệnh.
- Database: SQLite. Thêm file bằng đường dẫn, bằng nút Dò, hoặc chuột phải file `.db`/`.sqlite`/`.sqlite3` trong Files → "Mở bằng Database". Xem bảng và số dòng (mở chỉ đọc), backup an toàn khi app đang ghi, khôi phục bằng lệnh gõ sẵn vào Terminal.
- Database: nút Dò (chỉ đọc) tìm database server đang chạy và file SQLite trên server.
- Database: khối "Davish kết nối thế nào?" giải thích lệnh chạy ở đâu và mật khẩu được giữ thế nào.

### Sửa
- Nginx: đăng nhập bằng user thường thì bật, tắt, xóa, tạo, sửa site, test và reload không còn chỉ lùi về Terminal: Davish chạy lại đúng lệnh đã xác nhận qua `sudo`, hỏi mật khẩu sudo một lần mỗi phiên như Dịch vụ (#117). `certbot` vẫn chạy trong Terminal vì có thể hỏi thêm.

## [1.12.3] - 2026-11-10

### Sửa
- Sudo: nhận đúng "cần mật khẩu" trên server không dùng tiếng Anh (sudo chạy với `LC_ALL=C`); hai thao tác cùng cần mật khẩu giờ chờ chung một hộp hỏi thay vì thao tác trước biến mất; server bật `requiretty` (CentOS/RHEL cũ) thì lùi về Terminal thay vì báo lỗi; user không thuộc nhóm sudo, wheel hay admin được cảnh báo trước khi nhập mật khẩu, nút Terminal được đưa lên trước.

## [1.12.2] - 2026-11-10

### Sửa
- Dịch vụ: đăng nhập bằng user thường (không phải root) thì start, stop, restart, xóa service và xem journal không còn bị từ chối (#76, #113). Davish chạy lại qua `sudo`: không cần mật khẩu thì chạy luôn, cần thì hỏi mật khẩu sudo một lần mỗi phiên. Mật khẩu chỉ giữ trong bộ nhớ, không lưu ổ đĩa, keychain hay nhật ký, đưa cho sudo qua stdin, bị xóa khi ngắt kết nối, khóa app hoặc chọn Tệp → Quên mật khẩu sudo. Hộp hỏi hiện đúng lệnh `sudo …` và có nút chạy trong Terminal thay vì nhập.

## [1.12.1] - 2026-11-10

### Thêm
- An ninh: tick nhiều địa chỉ để chặn (hoặc bỏ chặn) một lần. Một hộp thoại liệt kê đúng lệnh cho từng địa chỉ; chạy lần lượt, địa chỉ nào lỗi hoặc cần mật khẩu sudo thì gõ sẵn vào Terminal.

### Đổi
- An ninh: Đăng nhập sai và Đã chặn thành bảng full chiều ngang (IP, số lần, user đã thử, lần thử cuối, trạng thái, Chặn/Bỏ chặn ngay trên dòng), chữ bôi đen copy được. Bấm một dòng thì chi tiết mở bên dưới thay vì khung bên phải.

## [1.12.0] - 2026-11-10

### Thêm
- An ninh: chặn, bỏ chặn một IP bằng ufw, ban, bỏ ban bằng fail2ban ngay trong app. Hộp thoại hiện đúng lệnh sẽ chạy, bấm xác nhận là chạy luôn (`sudo -n`, khi sudo không cần mật khẩu); nút Terminal nhỏ cạnh lệnh để gõ sẵn vào Terminal thay vì chạy. Sudo cần mật khẩu thì mở Terminal như cũ. Cảnh báo khi IP đang có phiên đăng nhập mở (có thể là chính bạn) hoặc khi ufw đang tắt.
- An ninh: copy IP từ khung chi tiết hoặc từ các bảng; IP trong bảng bôi đen được.

### Đổi
- An ninh: bố cục mục bên trái, dữ liệu bên phải (Tổng quan, Đăng nhập sai, Đã chặn, User bị thử, Đăng nhập thành công, Phiên đang mở, Tường lửa, Đăng nhập mật khẩu), có ô lọc theo IP hoặc user. Bấm vào ô thống kê thì mở danh sách tương ứng, như "Đã chặn" mở mọi địa chỉ bị chặn kèm chặn bằng ufw hay fail2ban.
- An ninh: đọc trạng thái ufw và fail2ban bằng `sudo -n` khi được, nên danh sách đã chặn hiện cả khi không đăng nhập bằng root.

### Sửa
- Cửa sổ Database và Triển khai app thiếu thanh tiêu đề (không có nút đóng, thu nhỏ, phóng to, không kéo được), nền trong suốt; các nút Dump, Restore, Delete của Database hiện như nút trình duyệt mặc định.

## [1.11.0] - 2026-11-10

### Thêm
- An ninh: tổng quan 7 ngày ở đầu app: số lần đăng nhập sai, user không tồn tại, số địa chỉ, đã ban, đăng nhập thành công; biểu đồ lần sai theo giờ (rê chuột hoặc dùng phím ←/→ để xem từng giờ); địa chỉ và user bị thử nhiều nhất; các phiên đang mở (`who`).
- Lệnh hay dùng: lưu lệnh kèm tên cho một server hoặc mọi server, một cú bấm gõ sẵn vào Terminal (không tự chạy), tìm được trong Spotlight; cảnh báo khi lệnh có vẻ chứa mật khẩu, token hay key. Nhật ký lệnh: mỗi dòng có "Gõ lại vào Terminal" và "Lưu thành lệnh hay dùng".
- Cửa sổ Tất cả server (menu server → Tất cả server…, Launchpad, Spotlight): mỗi server một thẻ với trạng thái kết nối, CPU, RAM, ổ `/`, load, thời gian chạy, mạng, số cảnh báo và cảnh báo mới nhất; kết nối hoặc chuyển sang server ngay trên thẻ. Chỉ thu số liệu khi cửa sổ mở.
- Launchpad chia app theo nhóm (Tệp & terminal, Giám sát, Web & container, Bảo mật & truy cập, Phát triển, Hệ thống) với tab cho từng nhóm; phím mũi tên đi qua các nhóm; app đã ghim luôn có dấu trên icon; nút "Khôi phục Dock mặc định" (bấm hai lần).

### Sửa
- Spotlight: tên kết quả (như "Trợ giúp: Services") không còn bị cắt còn vài chữ khi phần mô tả dài.

## [1.10.1] - 2026-11-09

### Đổi
- Thanh menu: mỗi số liệu là một cung tròn hở ở đáy, số nằm trong cung và tên nằm dưới; mỗi ô rộng cố định nên số thay đổi không làm các ô khác xê dịch. Thanh menu cao hơn một chút (34px).

## [1.10.0] - 2026-11-09

### Thêm
- Thanh menu: mỗi số liệu có vòng tròn nhỏ và nhãn; rê chuột vào hiện số lớn, chi tiết và biểu đồ 15 phút. Thêm Load, Đọc/ghi đĩa và GPU (chỉ hiện khi server có GPU NVIDIA); Swap và Thời gian chạy bật được trong Cài đặt → Thanh menu, nơi bật/tắt từng số liệu.

## [1.9.0] - 2026-11-09

### Thêm
- Giao diện tiếng Trung giản thể (简体中文): chọn trong Cài đặt → Giao diện → Ngôn ngữ; máy dùng tiếng Trung thì lần đầu Davish tự chọn. Ghi chú phát hành hiện bằng tiếng Anh.

### Đổi
- Ghi chú phát hành có cả tiếng Anh và tiếng Việt; hộp thoại cập nhật hiện bản theo ngôn ngữ giao diện.

## [1.8.2] - 2026-11-09

### Sửa
- Cập nhật: khi GitHub trả lỗi lúc tải (như sự cố 504 ngày 2026-11-09), Davish tự thử lại rồi tải qua đường GitHub API thay vì dừng ngay.
- Cập nhật không thành công giờ báo rõ ở đầu hộp thoại, đúng bước bị lỗi (kiểm tra hay tải về), kèm nút Thử lại, Tải thủ công và Chi tiết. Chi tiết hiện mã lỗi và các dòng log về cập nhật, có nút Sao chép để gửi khi báo lỗi và nút mở thư mục log.

## [1.8.1] - 2026-11-09

### Sửa
- Windows và Linux: cửa sổ Davish luôn phóng to, không kéo sang màn hình khác và không thu nhỏ được. Giờ cuối thanh menu có nút Thu nhỏ, Phóng to/Thu về và Đóng; kéo vùng trống của thanh menu (chỗ số liệu và đồng hồ) để di chuyển cửa sổ, nhấp đúp để phóng to hoặc thu về; kéo mép cửa sổ để đổi kích thước (#93).

## [1.8.0] - 2026-11-09

### Thêm
- Cài đặt → Dock: chọn kích thước Dock (Nhỏ, Vừa, Lớn) và bật Tự động ẩn. Dock ẩn xuống dưới màn hình, đưa chuột xuống cạnh dưới để hiện lại; cửa sổ phóng to hoặc snap dùng cả phần đó.

### Sửa
- Ổ đĩa: danh sách thư mục, File lớn và Dọn dẹp không cuộn được, các dòng dưới bị cắt mất; giờ cuộn được.

## [1.7.0] - 2026-11-08

### Thêm
- Docker, Nginx, Git và Database: server chưa cài chương trình thì app báo rõ và hiện lệnh cài đúng cho bản Linux của server (apt, dnf, yum, apk), kèm bật dịch vụ. Nút "Chạy trong Terminal" gõ sẵn lệnh để bạn xem rồi tự nhấn Enter. Docker được cài bằng gói của distro và thêm bạn vào nhóm `docker` (app nói rõ nhóm này ngang quyền root, có nút kết nối lại). Cài xong, quay lại cửa sổ là app tự mở (#79). Tài khoản SSH không có quyền sudo thì app chỉ báo nhờ người quản trị cài, không hiện lệnh.
- Ổ đĩa: mở một ổ là quét cả ổ một lần và hiện tổng quan: thanh chia thành thư mục đo được, file đã xóa còn bị giữ, phần chưa đo được và chỗ trống. Phần chưa đo được có lý do: thư mục user SSH không đọc được (kèm lệnh `sudo du` gõ sẵn vào Terminal), file đã xóa mà tiến trình nào còn giữ. Khi sudo không cần mật khẩu thì đo bằng root nên thấy đủ.
- Ổ đĩa: thư mục dạng cây mở tại chỗ, tab File lớn (từ 50 MB). Xóa thư mục hoặc file, làm rỗng file log, sau khi xem đúng lệnh; từ 1 GB hoặc thư mục có nội dung thì gõ tên. Thư mục hệ thống, dữ liệu Docker và database, thư mục home không xóa được từ đây.
- Dọn dẹp: Docker tách thành image không dùng, container dừng, cache build, volume (cảnh báo là dữ liệu) và log container quá 10 MB; thêm dump lỗi, file tạm cũ hơn 7 ngày, bản snap cũ, `~/.cache`, cache npm.

### Đổi
- Dock lần đầu mở chỉ ghim các app dùng hằng ngày: Terminal, Tệp, Monitor, Services, Docker, Logs, Editor, Cài đặt. App khác mở từ Launchpad hoặc Spotlight và ghim thêm được như cũ. Ai đã tự ghim hoặc bỏ ghim thì Dock giữ nguyên.

### Sửa
- Git: server chưa cài git từng hiện "Chưa có repo"; giờ báo chưa cài và đề nghị cài.
- Lệnh dài trong khung "cần quyền root" không còn bị ngắt dòng giữa chữ.
- Dọn dẹp Docker từng chạy `docker system prune -f`, chỉ xóa image treo, nên ổ vẫn đầy dù app báo lấy lại được nhiều GB. Giờ mỗi loại có lệnh riêng đúng với dung lượng hiện ra.

## [1.6.2] - 2026-11-08

### Sửa
- Terminal trên Windows: `Ctrl+Shift+V` dán hai lần; giờ dán một lần. Chuột phải trong Terminal (Windows, Linux) mở menu Sao chép, Dán, Chọn tất cả (#82).
- Terminal: bôi đen không còn đè lên các ô có nền màu riêng, như các đoạn của prompt powerline; màu chữ và màu nền của chúng giữ nguyên.
- Terminal: con trỏ là thanh dọc đứng yên như Terminal của macOS, không còn nhấp nháy và không đổi thành khối rỗng khi mất focus.

## [1.6.1] - 2026-11-08

### Sửa
- Docker: trên server chưa có container nào, app cứ hiện "Server trả lời chậm" dù mọi lệnh đã chạy xong; giờ hiện danh sách rỗng ngay. Tab Images trên server chưa có image nào cũng bị tương tự và đã được sửa (#79).

## [1.6.0] - 2026-11-07

### Thêm
- Giao diện tối. Mặc định theo chế độ sáng/tối của máy và tự đổi theo; chọn Sáng hoặc Tối trong Cài đặt → Giao diện → Chế độ để giữ cố định. Editor, biểu đồ Monitor và đồ thị Git có bảng màu riêng cho nền tối.

## [1.5.2] - 2026-11-05

### Thêm
- Lối tắt trên màn hình chính: chuột phải chọn "Tùy chỉnh…" (hoặc nhấn `F2`) để đổi tên hiển thị, chọn icon có sẵn kèm màu, hoặc dùng ảnh từ máy (PNG, JPG, WebP, tự cắt vuông). File trên server giữ nguyên và hộp thoại chạy script vẫn hiện tên thật của file.

### Sửa
- Bản 1.5.1 được đóng gói nhầm từ code của 1.5.0 nên tự nhận là 1.5.0, không có tính năng mới và cứ báo có bản cập nhật. Bản này sửa lỗi đó. Đang dùng 1.5.0 hay 1.5.1 thì cập nhật lên 1.5.2 như bình thường.

## [1.5.0] - 2026-11-05

### Thêm
- Files: tạo tệp và thư mục mới. Chuột phải vào khoảng trống, kể cả trong thư mục rỗng, để mở menu của thư mục đang xem; bấm vào khoảng trống thì bỏ chọn, như trên hệ điều hành. `⌘⇧N` tạo thư mục.

## [1.4.1] - 2026-11-05

### Sửa
- Bản 1.4.0 được đóng gói nhầm với số phiên bản cũ nên tự nhận là 1.3.4 và cứ báo có bản cập nhật. Bản này sửa lỗi đó, tính năng giữ nguyên như 1.4.0. Đang dùng 1.4.0 thì cập nhật lên 1.4.1 như bình thường.

## [1.4.0] - 2026-11-05

### Thêm
- Deploy có tiến trình: Terminal hiện từng bước đang chạy, xong hay lỗi kèm thời gian; thông báo khi deploy xong hoặc dừng ở bước lỗi; app Git liệt kê các lần deploy gần đây (commit, kết quả, thời gian).
- Hộp thoại Deploy làm lại, không còn phải tự viết lệnh: Davish xem repo (lockfile, script trong `package.json`, compose, `go.mod`, `Makefile`) và tìm app đang chạy từ thư mục đó (pm2, systemd, Docker compose), rồi đưa ra các bước bật/tắt được: lấy code mới, cài thư viện, build, khởi động lại app, thêm bước tùy chỉnh. Lệnh thật vẫn hiện dưới mỗi bước. Lựa chọn được nhớ cho lần sau.
- Bấm Deploy là chạy luôn trong Terminal; chỉ khi có bước cần sudo thì Davish gõ sẵn để bạn nhấn Enter và nhập mật khẩu.
- Lối tắt trên màn hình chính: kéo file, thư mục hoặc script từ Files ra nền (hoặc dùng menu tròn, `⌘⌥P`). Double-click để mở: thư mục vào Files, file vào Editor, script chạy trong Terminal. Mỗi server có màn hình riêng.
- Script lần đầu chạy sẽ hiện đúng lệnh; chọn "Lần sau chạy luôn" thì lần sau chạy ngay, trừ khi nội dung script đã đổi.
- Nút dừng khi đang chạy, trên dải tiến trình của Terminal và trên icon, gửi Ctrl-C vào đúng phiên đó. Deploy cũng có nút này.
- Files: chuột phải trên file hoặc thư mục để mở menu thao tác, gồm cả "Thêm vào màn hình chính".
- Terminal: nút mở thư mục đang đứng trong Files.
- Cron: tác vụ liệt kê các file và thư mục trong lệnh của nó, mở được trong Editor hoặc Files; tác vụ hệ thống mở được thư mục chứa file cron.

## [1.3.4] - 2026-11-04

### Sửa
- Giao diện khi cửa sổ hẹp: Dock không còn tràn khỏi màn hình và tự thu nhỏ icon; menu bar không xuống dòng, chỉ số tự ẩn dần khi thiếu chỗ.
- App trong cửa sổ hẹp (snap nửa màn hình chẳng hạn): nút và ô tìm kiếm không còn bị đẩy ra ngoài, sidebar thu thành dải icon, danh sách xếp trên chi tiết, bảng Docker và Monitor ẩn bớt cột phụ thay vì bắt cuộn ngang.
- Thu nhỏ cửa sổ Davish không còn làm cửa sổ con nằm ngoài tầm với.
- Tự cập nhật không còn làm hỏng app khi file của Davish đã bị thay hoặc xóa trong lúc chạy (ví dụ bản chạy từ thư mục build bị `wails dev` dọn): Davish kiểm tra app đang chạy và bản tải về đều đủ file trước khi tráo, nếu không thì dừng và nói rõ lý do. Bản dev không tự cập nhật. Thư mục tạm của lần cập nhật dở được dọn ở lần mở sau, và mọi lần cập nhật lỗi đều ghi nguyên nhân vào `davish.log`.

## [1.3.3] - 2026-11-03

### Sửa
- Mở thư mục trong Files từ app khác (ví dụ "Mở trong Files" của Git) khi Files chưa mở: cửa sổ đôi khi mở ở thư mục lần trước thay vì thư mục được chọn, phải bấm lần nữa.
- Lỗi JS khi chuyển qua lại giữa các repo trong app Git.

## [1.3.2] - 2026-10-29

### Sửa
- Desktop trắng trơn sau khi mở app hoặc mở khóa, nếu đã từng đổi dock: danh sách app ghim đọc lại sai kiểu.
- Khóa app: bấm Cancel ở hộp thoại Touch ID giờ đóng hẳn; trước đó hộp thoại hiện lại liên tục và chặn cả các app khác.
- Lỗi chỉ hiện "Đã có lỗi xảy ra" giờ ghi nguyên nhân vào `davish.log`.

## [1.3.1] - 2026-10-29

Gắn tag nhầm: giống hệt 1.3.0. Dùng 1.3.2.

## [1.3.0] - 2026-10-29

Đợt "Dựng mới": ngoài vận hành server có sẵn, Davish giờ dựng được site, container, app và database mới. Mọi file và lệnh đều hiện trước khi chạy; việc cần root được gõ sẵn vào Terminal.

### Thêm
- **Triển khai app** (Launchpad/Spotlight): trình hướng dẫn từ repo git / image / thư mục → systemd, compose, container hoặc tĩnh → site nginx → HTTPS; xem trước mọi lệnh, chạy từng bước trong Terminal, tổng kết và lệnh gỡ toàn bộ.
- Nginx → **Site mới…**: reverse proxy, tĩnh hoặc PHP-FPM; xem trước và sửa cấu hình; chỉ bật khi `nginx -t` qua, lỗi thì tắt lại. Ô reverse proxy gợi ý cổng và container đang nghe.
- Nginx → **Bật HTTPS…**: kiểm tra DNS rồi gõ sẵn `certbot --nginx` vào Terminal.
- Nginx → **Sửa nhanh…**: thêm tên miền, đổi upstream, gzip, `client_max_body_size`; sao lưu trước, `nginx -t` lỗi thì trả lại file cũ.
- Docker → **Pull image…** và **Chạy container mới…** từ form; cổng mặc định bind `127.0.0.1`, cảnh báo `0.0.0.0`.
- Docker → Compose: up, pull, restart, down theo project; **Stack compose mới…** từ mẫu.
- Docker: tạo network, volume; **Sao chép thành docker run** từ container.
- Git: **Clone repo…** (ghim host key GitHub/GitLab), **git init**, **Đặt remote…**.
- Git → **Deploy…**: công thức triển khai cho mỗi repo, lưu trên máy, chạy từng bước trong Terminal và dừng ở bước lỗi.
- Gói → **Cài thêm**: công thức cài nginx, Docker, Node (nvm), PHP-FPM, MariaDB, PostgreSQL, Redis, certbot, fail2ban, ufw; hiện toàn bộ script trước khi chạy.
- App **Database** (MariaDB/MySQL, PostgreSQL): danh sách kèm dung lượng, tạo database + user với mật khẩu ngẫu nhiên chỉ hiện một lần, xóa phải gõ tên, mở qua tunnel; **Dump** ra `~/db-backups` và **Khôi phục…** từ bản dump.
- Cổng: cột **Lộ ra Internet** (ufw + Docker), mở/đóng cổng bằng ufw (không đóng cổng SSH đang dùng), gợi ý đóng các cổng quản trị đang lộ ra.
- `SECURITY.md`: mô hình bảo mật và cách báo lỗ hổng riêng tư; README nêu "Mở là vào" và có banner.

### Sửa
- Thứ tự dock, "Bỏ qua bản này" và lựa chọn trong Logs giờ được nhớ sau khi khởi động lại (tên cài đặt có dấu `:` bị từ chối âm thầm).
- An ninh: "Chặn bằng ufw…" chèn luật lên đầu (`ufw insert 1`); trước đó luật nằm sau `ALLOW 22` nên không chặn được SSH. IP đã chặn hiện nhãn "đã chặn bằng ufw".
- Terminal: lệnh gõ sẵn hiện sau prompt thay vì trước.

## [1.2.0] - 2026-10-29

### Thêm
- Giao diện tiếng Anh (ADR 0010): chọn Tiếng Việt hoặc English trong Cài đặt → Giao diện → Ngôn ngữ; lần đầu theo ngôn ngữ của máy. Ngày giờ và số định dạng theo ngôn ngữ đã chọn. Đổi ngôn ngữ thì Davish khởi động lại, server tự kết nối lại.
- Menu bar hiện tên app in đậm cạnh menu hệ thống như macOS; bấm vào để mở menu hệ thống.

## [1.1.1] - 2026-10-29

### Sửa
- App Cổng kẹt ở "Server trả lời chậm" dù Chẩn đoán báo server bình thường: server có hai socket cùng nghe một địa chỉ và cổng (DNS của mạng Docker bridge, hoặc nhiều worker dùng chung cổng) làm danh sách có hai dòng trùng và giao diện ngừng vẽ. Các socket trùng giờ gộp thành một cổng.
- App Cổng kẹt ở "đang chờ" dù server trả lời: danh sách cổng rỗng bị gửi thành `null` làm giao diện vỡ; một số bản `ss` in thêm cột `tcp` làm không đọc được dòng nào. Lỗi giao diện từ nay cũng được ghi vào log để Sao chép chi tiết có đủ thông tin.
- Server có shell đăng nhập là zsh hay fish: lệnh của Davish viết cho `sh` nên trước đây có thể ra kết quả trống (ví dụ app Cổng không thấy cổng nào). Giờ mọi lệnh chạy dưới `sh`, shell đăng nhập là gì cũng được.
- App không còn chờ mãi khi server không trả lời (shell đăng nhập bị treo, Docker treo…). Mọi lệnh có hạn chót (mặc định 3 phút, đọc cổng 25 giây, Docker 30 giây) và trả lỗi ngay khi hết hạn, không đợi server đóng phiên; mở SFTP cũng vậy. Trước đây hạn chót chỉ yêu cầu đóng phiên nên vẫn đứng ở "Đang đọc…".
- Nút làm mới bấm được cả khi đang tải: lần tải mới thay lần đang kẹt, kết quả muộn của lần cũ bị bỏ qua.

### Thêm
- Chẩn đoán: khi một app tải lâu hơn 8 giây hoặc báo lỗi, có Thử lại, **Chẩn đoán** và **Sao chép chi tiết**. Chẩn đoán thử `echo`, `ss`, `docker ps` trên server và nói rõ nguyên nhân: mất kết nối, shell đăng nhập bị treo (kèm cách sửa ~/.bashrc), shell chậm (kèm các dòng nghi vấn), ss hoặc Docker không trả lời.
- Log của app ghi ra file JSON xoay vòng trong thư mục cache (`davish/logs`, 5 file × 5 MB): mọi lệnh chạy trên server, mã thoát, thời gian, lỗi; không bao giờ có mật khẩu, khóa hay nội dung file. Trợ giúp → "Sao chép thông tin hỗ trợ" gom phiên bản, hệ điều hành và log gần đây để gửi cho người hỗ trợ.

## [1.1.0] - 2026-10-28

### Thêm
- Docker làm lại theo kiểu Docker Desktop: bảng container (ID, image, cổng, CPU, RAM, trạng thái) gom theo project compose, thẻ CPU/RAM tổng, nút khởi động/dừng, menu ⋮ (log, shell trong container, khởi động lại, chi tiết) và xóa container ngay trên dòng; cổng publish bấm để mở trong Trình duyệt của Davish. Bảng image có thanh dung lượng; mục Volume mới (dung lượng, container đang dùng, xóa volume không dùng).
- Tự cập nhật (ADR 0009): Davish tự kiểm tra GitHub Releases lúc mở và mỗi 6 giờ (tắt được trong Cài đặt), báo bản mới kèm nhật ký thay đổi với Cập nhật / Để sau / Bỏ qua bản này. Cập nhật tải đúng file của máy, kiểm chữ ký ed25519 và SHA-256, thay bản đang chạy rồi khởi động lại; không phải gỡ và cài lại. "Kiểm tra bản cập nhật…" trong menu hệ thống và Trợ giúp.
- Chuột phải icon trên Dock như macOS: Mở/Hiện, Giữ trên Dock (bỏ chọn thì app rời Dock, app đang mở thì ở lại tới khi đóng), Đóng.
- Docker: mục Image (tất cả, không dùng) với số container đang dùng; chi tiết image gồm các layer kèm dung lượng, nền tảng, entrypoint/cmd, cổng mở, biến môi trường; chạy image trong Terminal, xóa image qua xác nhận hiện đúng lệnh `docker rmi` (không `-f`).
- App Trình duyệt (ADR 0008): mở `localhost:<cổng>` của VPS ngay trong Davish, qua proxy nghe `127.0.0.1` ở cổng ngẫu nhiên nên không đụng cổng nào trên máy; redirect về cổng khác vẫn ở lại VPS, websocket chạy, cookie đăng nhập do Davish giữ. Trang ở origin riêng, không gọi được Davish. App Cổng có nút "Mở trong Davish".
- App An ninh: lần thử đăng nhập SSH sai trong 7 ngày gom theo IP (nhiều nhất lên đầu, user đã thử), đăng nhập thành công gần đây, `last`/`lastb`, trạng thái ufw và fail2ban; chặn một IP hoặc tắt đăng nhập mật khẩu bằng lệnh sudo gõ sẵn vào Terminal (có `sshd -t` trước khi reload, tự gỡ nếu lỗi).
- App Ổ đĩa: dung lượng từng mount; đo một thư mục mỗi lần bấm (`du -x -d 1`, tối đa 30 giây, hủy được), đi sâu từng tầng, mở trong Files. Gợi ý dọn dẹp kèm dung lượng lấy lại: Docker, journal, log xoay vòng cũ, cache apt/dnf; lệnh luôn hiện trước khi chạy, cần root thì gõ sẵn vào Terminal. Thông báo ổ sắp đầy mở thẳng app ở đúng mount.
- App Cổng: cổng TCP server đang nghe (`ss -tlnp`) với process và container; tunnel một cổng về `127.0.0.1` của máy này qua kết nối SSH đang có (giữ số cổng nếu trống), mở trong trình duyệt hoặc copy địa chỉ; mở như một tên miền (ghi đè Host) cho site nginx; "Thử từ server" bằng curl. Tunnel sống qua reconnect và tắt khi đóng app.
- Menu bar thật: Tệp, Xem, Đi tới, Cửa sổ, Trợ giúp và menu của app đang mở (Git: Fetch, Pull, Thêm thư mục; Services, Docker: khởi động lại…), điều khiển bằng bàn phím, phím tắt hiện bên phải. `⌘R` làm mới cửa sổ đang mở, `⌘⇧T/F/M/S/D/L/E/G` mở từng app.
- Trợ giúp: nút `?` trên mỗi app (app làm gì, thao tác chính, phím tắt, lệnh chạy trên server), bảng phím tắt (`⌘/`), Về Davish; Spotlight tìm được trợ giúp. Hướng dẫn nhanh lần đầu, xem lại từ Trợ giúp.
- Chế độ thử trên màn hình chào: xem Davish với dữ liệu mẫu, không kết nối server nào.
- Mọi nút chỉ có icon đều có tooltip.
- Nhập server từ `~/.ssh/config` (menu chọn server hoặc màn hình chào): chọn nhiều host, host đã có được bỏ qua; Davish chỉ đọc file, không ghi. Hỗ trợ `Include`, `IdentityFile`, `ProxyJump`.
- ProxyJump: server có thể đi qua một server khác (nhiều bậc), mỗi bậc kiểm tra host key riêng; mất kết nối tới máy trung gian thì tự dựng lại cả chuỗi.
- App Git: tìm repo trên server (home, `/var/www`, `/srv`, `/opt` và thư mục thêm tay); nhánh, nhánh remote, số commit chưa đẩy/chưa kéo, file thay đổi và diff; biểu đồ commit của mọi nhánh với merge và tag. Fetch, `pull --ff-only` (chặn khi repo bẩn), đổi nhánh qua hộp xác nhận hiện đúng lệnh; cài dependency theo lockfile trong Terminal; lỗi xác thực remote dẫn tới tạo deploy key. Repo của user khác bị git từ chối thì báo rõ, không ghi đè `safe.directory`.
- App Gói: gói apt/dnf/yum/apk đã cài và có bản cập nhật, bản vá bảo mật lên đầu (thông báo khi kết nối); runtime và chương trình trên server (nginx, MariaDB, Redis, Docker, Node.js qua nvm, PHP với php-fpm, Python…) với phiên bản, file cấu hình, dịch vụ; gói npm global. Cài, nâng cấp, gỡ thì gõ sẵn lệnh `sudo` vào Terminal. Spotlight tìm được chương trình trên server.
- App Khóa SSH: quản lý `authorized_keys` của server (thêm, gỡ có sao lưu, đánh dấu key Davish đang đăng nhập), tạo deploy key trên server và thử với GitHub/GitLab (kiểm tra host key theo fingerprint họ công bố), tạo key trên máy này rồi đưa lên server và dùng để đăng nhập.
- Khóa app (ADR 0007): mật khẩu khóa riêng, mở bằng Touch ID trên macOS; tự khóa khi máy ngủ dậy, sau N phút không dùng, hoặc `⌘L` / `Ctrl+L`. Phiên SSH, log, transfer vẫn chạy khi khóa; thao tác lên server bị chặn. Quên mật khẩu thì đặt lại, đổi lại Davish xóa mật khẩu server đã lưu.
- Launchpad (ô đầu Dock hoặc `⌘⇧A`): mọi app, tìm theo tên, ghim/bỏ ghim khỏi Dock. Dock hiện thêm app đang mở chưa ghim.
- Xóa dịch vụ (Services → `Xóa…`): unit do admin tạo thì xóa file; unit của gói (như `certbot`) thì mask cùng timer kích hoạt nó; unit đã mất file thì chỉ bỏ trạng thái lỗi. Luôn hiện đúng lệnh trước khi chạy.
- Xóa site nginx (`Xóa site…`), có thể xóa luôn chứng chỉ Let's Encrypt chỉ site đó dùng (`certbot delete`).
- Kính lúp trên menu bar và mục `Tìm kiếm…` trong menu hệ thống mở Spotlight.

### Thay đổi
- Nhẹ hơn: mỗi app chỉ tải code khi mở lần đầu (JS ban đầu khoảng 96 kB gzip); CI chặn bundle vượt ngân sách. Khi Davish bị ẩn, khóa hoặc ngủ màn hình, chỉ số đọc 30 giây một lần thay vì 2 giây, cảnh báo vẫn chạy.
- Mở, đóng, thu nhỏ và khôi phục cửa sổ bằng hiệu ứng Genie như macOS: cửa sổ chảy ra từ icon của app trên Dock và chảy về đó.

### Sửa
- Cổng và Docker không còn chờ mãi khi server không trả lời (ví dụ Docker daemon bị treo): mỗi lần đọc có giới hạn thời gian và báo rõ khi quá hạn.
- Logs: nhãn trên toolbar không còn xuống dòng; cửa sổ hẹp thì ô nguồn và ô lọc co lại, hẹp hơn nữa thì toolbar xuống hàng thay vì che mất nút.
- Logs: journal của một dịch vụ im lặng 5 giây thì giải thích lý do thường gặp (dịch vụ không ghi journal, hoặc tài khoản SSH không được đọc journal hệ thống) và đưa lệnh `sudo journalctl …` để chạy trong Terminal, thay vì chờ mãi.
- Thông báo: đường dẫn dài (ví dụ đường dẫn chứng chỉ TLS) xuống dòng trong banner và bảng Thông báo thay vì tràn ra ngoài và đẩy nút đóng đi.

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

[Unreleased]: https://github.com/2hjaito/davish/compare/v1.3.4...HEAD
[1.3.4]: https://github.com/2hjaito/davish/compare/v1.3.3...v1.3.4
[1.3.3]: https://github.com/2hjaito/davish/compare/v1.3.2...v1.3.3
[1.3.2]: https://github.com/2hjaito/davish/compare/v1.3.1...v1.3.2
[1.3.1]: https://github.com/2hjaito/davish/compare/v1.3.0...v1.3.1
[1.3.0]: https://github.com/2hjaito/davish/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/2hjaito/davish/compare/v1.1.1...v1.2.0
[1.1.1]: https://github.com/2hjaito/davish/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/2hjaito/davish/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/2hjaito/davish/compare/v0.9.0...v1.0.0
[0.9.0]: https://github.com/2hjaito/davish/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/2hjaito/davish/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/2hjaito/davish/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/2hjaito/davish/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/2hjaito/davish/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/2hjaito/davish/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/2hjaito/davish/releases/tag/v0.3.0
