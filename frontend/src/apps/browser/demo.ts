// What the Browser app shows in the plain browser preview and the demo
// mode, where there is no server: a small web app as if on the VPS.
import type { Address } from './address'

const css = `
  *{box-sizing:border-box}body{margin:0;font:14px/1.45 -apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;color:#1d2433;background:#f4f6fb}
  header{display:flex;align-items:center;gap:12px;padding:14px 24px;background:#1f2a44;color:#fff}
  header b{font-size:16px}header span{opacity:.6;font-size:12px}nav{margin-left:auto;display:flex;gap:18px;font-size:13px;opacity:.85}
  main{padding:22px 24px;display:grid;gap:16px}
  .cards{display:grid;grid-template-columns:repeat(4,1fr);gap:14px}
  .card{background:#fff;border-radius:12px;padding:14px 16px;box-shadow:0 1px 2px #0001}
  .card small{color:#6b7385;font-size:12px}.card div{font-size:22px;font-weight:650;margin-top:4px}
  .up{color:#16a067;font-size:12px}
  table{width:100%;border-collapse:collapse;background:#fff;border-radius:12px;overflow:hidden;box-shadow:0 1px 2px #0001}
  th,td{text-align:left;padding:10px 14px;border-bottom:1px solid #eef0f5;font-size:13px}th{color:#6b7385;font-weight:500;background:#fafbfd}
  .tag{display:inline-block;padding:1px 8px;border-radius:10px;font-size:12px}.paid{background:#e3f6ec;color:#16794a}.new{background:#e6effd;color:#2457c5}.ship{background:#fdf1e1;color:#a5610b}
  .hot{position:fixed;bottom:12px;right:14px;font-size:11px;color:#6b7385;background:#fff;border-radius:8px;padding:4px 8px;box-shadow:0 1px 3px #0002}
`

const orders: [string, string, string, string, string][] = [
  ['#10482', 'Nguyễn Minh Anh', '1.250.000 ₫', 'paid', 'Đã thanh toán'],
  ['#10481', 'Trần Quốc Bảo', '640.000 ₫', 'ship', 'Đang giao'],
  ['#10480', 'Lê Thu Hà', '2.180.000 ₫', 'new', 'Mới'],
  ['#10479', 'Phạm Gia Huy', '315.000 ₫', 'paid', 'Đã thanh toán'],
  ['#10478', 'Võ Ngọc Lan', '980.000 ₫', 'ship', 'Đang giao'],
]

function shop(): string {
  const rows = orders
    .map(
      ([id, who, total, kind, label]) =>
        `<tr><td>${id}</td><td>${who}</td><td>${total}</td><td><span class="tag ${kind}">${label}</span></td></tr>`,
    )
    .join('')
  return `<!doctype html><html lang="vi"><head><meta charset="utf-8"><title>Davitek Shop · Admin</title><style>${css}</style></head><body>
  <header><b>Davitek Shop</b><span>admin · shop.davitek.com</span><nav><a>Tổng quan</a><a>Đơn hàng</a><a>Sản phẩm</a><a>Khách hàng</a></nav></header>
  <main>
    <div class="cards">
      <div class="card"><small>Doanh thu hôm nay</small><div>18,4 tr ₫</div><span class="up">▲ 12% so với hôm qua</span></div>
      <div class="card"><small>Đơn mới</small><div>42</div><span class="up">▲ 5</span></div>
      <div class="card"><small>Đang giao</small><div>17</div></div>
      <div class="card"><small>Khách quay lại</small><div>63%</div></div>
    </div>
    <table><thead><tr><th>Đơn</th><th>Khách</th><th>Tổng</th><th>Trạng thái</th></tr></thead><tbody>${rows}</tbody></table>
  </main>
  <div class="hot">⚡ Vite HMR · đã kết nối</div>
</body></html>`
}

function nginx(): string {
  return `<!doctype html><html><head><meta charset="utf-8"><title>Welcome to nginx!</title><style>body{width:35em;margin:40px auto;font-family:Tahoma,Verdana,Arial,sans-serif}</style></head><body>
  <h1>Welcome to nginx!</h1><p>If you see this page, the nginx web server is successfully installed and working.</p></body></html>`
}

/** The page at a demo address. */
export function demoPage(a: Address): { html: string; title: string } {
  return a.port === 80
    ? { html: nginx(), title: 'Welcome to nginx!' }
    : { html: shop(), title: 'Davitek Shop · Admin' }
}
