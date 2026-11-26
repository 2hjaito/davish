// What people said in public, word for word, with who said it and where.
// The English is as written (typos included); vi and zh are translations,
// and the page shows its own language with the English one click away.
// Shown on the landing page; the first three are open, the rest fold.
import type { Locale } from '../i18n'

export interface Voice {
  /** The handle, as on the site it came from. */
  who: string
  /** The post or comment it was said in, when known. */
  url?: string
  source: 'Reddit' | 'GitHub' | 'daily.dev' | 'X'
  /** The language it was written in; English when left out. */
  lang?: 'en' | 'zh'
  /** The words as written, in the language above. */
  text: string
  /** The words in the page's own language, by locale. */
  tr: Partial<Record<Locale, string>>
}

export const voices: Voice[] = [
  {
    who: 'kevinpirnie',
    url: 'https://www.reddit.com/r/SelfHosting/s/uzTNhQx7hJ',
    source: 'Reddit',
    text: 'Nice job, definately nice job, and the more I mess around with it, the more I want to use it.',
    tr: {
      vi: 'Làm tốt lắm, thật sự rất tốt, và càng nghịch tôi càng muốn dùng nó.',
      zh: '干得漂亮，真的很棒，而且我越用越想用下去。',
    },
  },
  {
    who: 'tian2992',
    url: 'https://github.com/2hjaito/davish/issues/159#issuecomment-6941449365',
    source: 'GitHub',
    text: 'PR #184 provides a great experience on Intel IGP. With perfectly smooth performance on the balanced profile.',
    tr: {
      vi: 'PR #184 mang lại trải nghiệm rất tốt trên card đồ họa tích hợp Intel, mượt hoàn hảo ở cấu hình cân bằng.',
      zh: 'PR #184 在 Intel 集成显卡上体验很好，使用“平衡”档时非常流畅。',
    },
  },
  {
    who: 'bbotea',
    url: 'https://github.com/2hjaito/davish/issues/248#issuecomment-7131308993',
    source: 'GitHub',
    text: 'Wow, this is even better than what I had originally envisioned! … Keep up the fantastic work!',
    tr: {
      vi: 'Wow, còn hay hơn cả những gì tôi hình dung ban đầu! … Cứ tiếp tục làm tốt nhé!',
      zh: '哇，比我最初设想的还要好！……继续保持出色的工作！',
    },
  },
  {
    who: 'DismissedFetus',
    source: 'Reddit',
    text: "Happy with the memory footprint, pretty decent UI/UX from what I've fiddled with. Hope to see it developed long term!",
    tr: {
      vi: 'Hài lòng với mức dùng bộ nhớ, giao diện và trải nghiệm khá ổn theo những gì tôi đã thử. Mong thấy nó được phát triển lâu dài!',
      zh: '对内存占用很满意，从我试过的部分来看界面和体验相当不错。希望它能长期发展下去！',
    },
  },
  {
    who: 'sureshbhardwaj',
    source: 'daily.dev',
    text: 'I am using it from last 2 hours and finding it very useful.',
    tr: {
      vi: 'Tôi dùng nó được 2 giờ rồi và thấy rất hữu ích.',
      zh: '我已经用了两个小时，觉得非常有用。',
    },
  },
  {
    who: 'QingQ77',
    url: 'https://x.com/QingQ77/status/2119581232182984974',
    source: 'X',
    lang: 'zh',
    text: '不想在服务器上再装一套面板的人，可以用它把终端、文件、服务、Docker、日志这些日常操作塞进一个桌面客户端，连服务器走 SSH 就够。',
    tr: {
      en: 'If you don’t want to install yet another panel on your server, you can use it to put everyday work like the terminal, files, services, Docker and logs into one desktop client. Connecting over SSH is all it needs.',
      vi: 'Ai không muốn cài thêm một bảng điều khiển lên server thì có thể dùng nó để gom terminal, file, dịch vụ, Docker, log vào một ứng dụng desktop, kết nối server chỉ cần SSH.',
    },
  },
]

/**
 * The face next to a name: GitHub's own for GitHub, and a picture kept in
 * public/voices/ for the others (their sites give no stable address).
 */
export function avatar(v: Voice): string {
  return v.source === 'GitHub' ? `https://github.com/${v.who}.png?size=96` : `/voices/${v.who}.jpg`
}

/** The name as the site it came from writes it: Reddit's are u/name, X's @name. */
export function handle(v: Voice): string {
  return v.source === 'Reddit' ? `u/${v.who}` : v.source === 'X' ? `@${v.who}` : v.who
}

/** How many are open before "See more". */
export const OPEN = 3
