import type { Dict } from './en'

export const vi: Dict = {
  lang: 'vi',
  title: 'Davish: desktop cho server Linux của bạn',
  description:
    'Terminal, file, Docker, log, database và nhiều hơn nữa qua SSH, trong một app desktop cho macOS, Windows và Linux. Không cài gì lên server.',
  nav: { features: 'Tính năng', security: 'Bảo mật', download: 'Tải về', docs: 'Tài liệu', theme: 'Sáng hoặc tối', changelog: 'Thay đổi' },
  hero: {
    title: 'Một desktop cho server Linux của bạn.',
    lead: 'Terminal, file, dịch vụ, Docker, log, database và git qua SSH, gom vào một nơi dùng như một hệ điều hành.',
    tags: ['Không web panel', 'Không đăng nhập', 'Không agent'],
    download: 'Tải cho',
    other: 'Hệ điều hành khác',
    github: 'Xem trên GitHub',
    version: (v: string) => `Phiên bản ${v} · Miễn phí, mã nguồn mở (GPL-3.0)`,
    shotAlt: 'Desktop của Davish: Terminal, Files và Monitor trên một server',
    badge: (v: string) => `Có gì mới trong ${v}`,
    docs: 'Đọc tài liệu',
    platforms: 'Chạy trên macOS, Windows và Linux',
    hint: 'Chọn một app để xem thử',
    tabs: { desktop: 'Desktop', servers: 'Tất cả server', docker: 'Docker', database: 'Database', git: 'Git', security: 'An ninh' },
  },
  why: {
    title: 'Vì sao là Davish',
    items: [
      {
        title: 'Mở là vào',
        text: 'Không màn hình đăng nhập, không web panel. Davish kết nối bằng SSH key và đưa bạn vào thẳng desktop của server. Không có gì mới lắng nghe trên server.',
      },
      {
        title: 'Không cần agent',
        text: 'Chỉ cần sshd và SFTP. Mọi thứ chạy từ máy bạn; không chép, không cài, không để lại gì trên server.',
      },
      {
        title: 'Thấy trước mọi lệnh',
        text: 'Thao tác phá hủy luôn hiện đúng lệnh sẽ chạy. Nhật ký lệnh ghi lại tất cả và xuất thay đổi thành script.',
      },
      {
        title: 'Nhiều server, một desktop',
        text: 'Mỗi server là một workspace nhớ cửa sổ của nó. Kéo file từ server này sang server khác.',
      },
      {
        title: 'Dùng luôn ssh config của bạn',
        text: 'Nhập host từ ~/.ssh/config hoặc Ansible inventory, kèm key và ProxyJump.',
      },
      {
        title: 'Tự cập nhật',
        text: 'Có bản mới thì hiện thay đổi; Davish tự tải, kiểm chữ ký rồi khởi động lại vào bản mới.',
      },
    ],
  },
  features: {
    title: 'Mọi thứ server cần, trong những cửa sổ bạn đã quen',
    lead: 'Hơn hai mươi app chung một desktop: kéo, snap và chuyển qua lại như trên máy của chính bạn.',
    show: [
      {
        key: 'servers',
        title: 'Mọi server trong một cái nhìn',
        text: 'Sức khỏe và cảnh báo của mọi server trong một cửa sổ, kèm bản đồ server đang phục vụ gì: tên miền, nginx, cổng, container.',
      },
      {
        key: 'multi-run',
        title: 'Một lệnh, nhiều server',
        text: 'Chạy một lệnh trên nhiều server cùng lúc, mỗi server một tab terminal, có nút dừng dừng được thật.',
      },
      {
        key: 'git',
        title: 'Git ngay trên server',
        text: 'Trạng thái, diff và đồ thị commit của các repo trên server, kèm fetch và pull.',
      },
      {
        key: 'docker',
        title: 'Docker rõ ràng',
        text: 'Container theo compose project, image và các layer, volume, log và shell chỉ một cú bấm.',
      },
      {
        key: 'sqlite-rows',
        title: 'Database không cần web panel',
        text: 'MariaDB, PostgreSQL, MongoDB và SQLite: tạo, dump, khôi phục, và xem bảng SQLite theo từng trang.',
      },
      {
        key: 'security',
        title: 'Ai đang gõ cửa',
        text: 'Đăng nhập thất bại theo giờ và theo địa chỉ, phiên đang mở, ufw và fail2ban, không cần rời desktop.',
      },
    ],
    more: 'Còn có',
    list: [
      'Terminal',
      'Files (SFTP)',
      'Editor',
      'Logs',
      'Monitor',
      'Dịch vụ',
      'Cron',
      'Nginx',
      'Gói phần mềm',
      'Cổng & tunnel',
      'Trình duyệt',
      'Ổ đĩa',
      'SSH key',
      'Lệnh hay dùng',
      'Nhật ký lệnh',
      'Spotlight & Launchpad',
    ],
    all: 'Xem mọi tính năng',
  },
  security: {
    title: 'Đủ tin cậy để giao quyền root',
    items: [
      'Mật khẩu và passphrase chỉ nằm trong keychain của hệ điều hành.',
      'Mật khẩu sudo không bao giờ được lưu: hỏi một lần mỗi phiên, giữ trong bộ nhớ, quên sau 15 phút.',
      'Tự khóa bằng mật khẩu riêng hoặc Touch ID khi bạn rời máy.',
      'Bản cập nhật được kiểm chứng: trình cập nhật kiểm chữ ký bản phát hành trước khi cài.',
    ],
    more: 'Đọc ghi chú bảo mật',
  },
  download: {
    title: 'Tải Davish',
    lead: 'Miễn phí cho macOS, Windows và Linux. Server chỉ cần OpenSSH có SFTP.',
    platforms: {
      macos: { name: 'macOS', note: 'macOS 12 trở lên, Apple silicon và Intel' },
      windows: { name: 'Windows', note: 'Windows 10 trở lên, 64-bit' },
      linux: { name: 'Linux', note: 'GTK 3 và WebKitGTK 4.1, x86-64' },
    },
    installer: 'Bộ cài',
    archive: 'Bản portable',
    checksums: 'Checksum',
    notes: 'Ghi chú phát hành',
    unsigned:
      'Davish chưa được ký số nên macOS và Windows sẽ cảnh báo lần đầu. Hướng dẫn cài trên GitHub chỉ cách mở.',
    guide: 'Hướng dẫn cài',
  },
  stats: {
    downloads: 'lượt tải app',
    apps: 'app trên một desktop',
    version: (date: string) => `phiên bản mới nhất, ${date}`,
    stars: 'sao trên GitHub',
    free: 'Miễn phí',
    freeNote: 'mã nguồn mở, GPL-3.0',
  },
  docs: {
    title: 'Mười phút là dùng được',
    lead: 'Mỗi app có một trang riêng trong tài liệu: làm gì, chạy lệnh gì trên server, và dữ liệu của bạn nằm ở đâu.',
    start: 'Bắt đầu',
    links: [
      { label: 'Cài Davish', href: 'start/install/' },
      { label: 'Kết nối server đầu tiên', href: 'start/first-server/' },
      { label: 'Server cần gì', href: 'start/requirements/' },
      { label: 'Phím tắt', href: 'start/shortcuts/' },
    ],
  },
  community: { discord: 'Tham gia Discord', discordNote: 'Hỏi đáp, góp ý và bản dùng thử sớm.' },
  thanks: {
    title: 'Được làm cùng những người dùng nó',
    lead: 'Code, báo lỗi, thử trên máy của chính mình và góp ý: cảm ơn các bạn.',
    code: 'code',
    issue: 'issue',
    discussion: 'thảo luận',
    join: 'Cùng tham gia trên GitHub',
  },
  footer: {
    license: 'Davish là phần mềm tự do theo giấy phép GPL-3.0.',
    changelog: 'Nhật ký thay đổi',
    issues: 'Báo lỗi',
    language: 'Ngôn ngữ',
  },
}
