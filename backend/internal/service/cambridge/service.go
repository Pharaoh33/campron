package cambridge

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"campron_enterprise/backend/internal/config"
)

/*
Service 层：专注 Cambridge 抓取与解析
- FetchEntryHTML：下载词条页面 HTML
- ExtractMP3URLs：从 HTML 提取 uk/us 的 mp3 地址
- ExtractIPA：从 HTML 提取 uk/us 的 IPA 音标（尽量保持 Cambridge 原样，例如 /ækˈtɪv.ə.ti/）
- DownloadMP3：下载 mp3 并保存到本地（带 .part 临时文件、简单大小校验）
*/

type Service struct {
	baseHost  string
	userAgent string
	client    *http.Client
}

type MP3 struct {
	Accent string // uk/us
	URL    string
}

var (
	// mp3 地址一般形如：
	// /media/english/us_pron/e/eus/eus70/eus70064.mp3
	// /media/english/uk_pron/u/uka/ukact/ukactiv007.mp3
	reMP3 = regexp.MustCompile(`(?i)(https?://dictionary\.cambridge\.org)?(/media/english/(uk|us)_pron/[^"'\s>]+?\.mp3)`)

	// IPA span 一般形如：
	// <span class="ipa dipa lpr-2 lpl-1">/ækˈtɪv.ə.ti/</span>
	// 注意：这里只抓 span 内的文本（不跨标签），避免过度贪婪。
	reIPA = regexp.MustCompile(`(?is)<span[^>]*class="[^"]*\bipa\b[^"]*"[^>]*>([^<]+)</span>`)
)

func New(cfg *config.Config) *Service {
	return &Service{
		baseHost:  cfg.Cambridge.BaseHost,
		userAgent: cfg.Cambridge.UserAgent,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *Service) entryURL(word string) string {
	// Cambridge 词条 URL 通常是 /dictionary/english/<word>
	// 空格替换成 '-' 能提高命中率（例如 multi word）
	w := strings.TrimSpace(word)
	w = strings.ReplaceAll(w, " ", "-")
	return strings.TrimRight(s.baseHost, "/") + "/dictionary/english/" + url.PathEscape(w)
}

// FetchEntryHTML 抓取词条页面 HTML
func (s *Service) FetchEntryHTML(word string) (pageURL string, html []byte, err error) {
	pageURL = s.entryURL(word)

	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return pageURL, nil, err
	}
	// 简单伪装浏览器，减少 403
	req.Header.Set("User-Agent", s.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := s.client.Do(req)
	if err != nil {
		return pageURL, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return pageURL, nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	html, err = io.ReadAll(resp.Body)
	if err != nil {
		return pageURL, nil, err
	}
	return pageURL, html, nil
}

// ExtractMP3URLs 从 HTML 中提取指定口音的 mp3 URL
func (s *Service) ExtractMP3URLs(html []byte, accent string) ([]MP3, error) {
	want := map[string]bool{}
	switch accent {
	case "us":
		want["us"] = true
	case "uk":
		want["uk"] = true
	case "both":
		want["uk"] = true
		want["us"] = true
	default:
		want["us"] = true
	}

	matches := reMP3.FindAllSubmatch(html, -1)

	seen := map[string]bool{}
	found := map[string][]string{}

	for _, m := range matches {
		path := string(m[2])                 // /media/...
		acc := strings.ToLower(string(m[3])) // uk/us
		if !want[acc] {
			continue
		}
		u := strings.TrimRight(s.baseHost, "/") + path // 补全绝对 URL
		key := acc + "|" + u
		if seen[key] {
			continue
		}
		seen[key] = true
		found[acc] = append(found[acc], u)
	}

	out := make([]MP3, 0, len(want))
	for acc := range want {
		arr := found[acc]
		if len(arr) == 0 {
			continue
		}
		// 经验：更短的 URL 往往是主词条发音（而不是派生/变体）
		sort.SliceStable(arr, func(i, j int) bool {
			if len(arr[i]) != len(arr[j]) {
				return len(arr[i]) < len(arr[j])
			}
			return i < j
		})
		out = append(out, MP3{Accent: acc, URL: arr[0]})
	}

	// 输出顺序固定：uk 在前，us 在后（便于前端展示）
	sort.SliceStable(out, func(i, j int) bool { return out[i].Accent < out[j].Accent })
	return out, nil
}

// ExtractIPA 返回每个口音对应的 IPA 文本（尽量与 Cambridge 页面一致）
func (s *Service) ExtractIPA(html []byte, accent string) map[string]string {
	want := map[string]bool{}
	switch accent {
	case "us":
		want["us"] = true
	case "uk":
		want["uk"] = true
	case "both":
		want["uk"] = true
		want["us"] = true
	default:
		want["us"] = true
	}

	txt := string(html)
	out := map[string]string{}
	if want["uk"] {
		out["uk"] = extractIPAForRegion(txt, "uk")
	}
	if want["us"] {
		out["us"] = extractIPAForRegion(txt, "us")
	}
	return out
}

func ensureIPASlash(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	if !strings.HasPrefix(s, "/") {
		s = "/" + s
	}
	if !strings.HasSuffix(s, "/") {
		s = s + "/"
	}
	return s
}

// extractIPAForRegion：启发式解析
// 1) 优先找 region 标记：<span class="region dreg">uk</span> 或 us
// 2) 在其后截取一个窗口，找到第一个 ipa span（通常就是红框里的那个）
// 3) 兜底：全页第一个 ipa
func extractIPAForRegion(html, region string) string {
	reRegion := regexp.MustCompile(`(?is)<span[^>]*class="[^"]*\bregion\b[^"]*\bdreg\b[^"]*"[^>]*>` + regexp.QuoteMeta(region) + `</span>`)
	loc := reRegion.FindStringIndex(html)
	if loc != nil {
		start := loc[1]
		end := start + 6000
		if end > len(html) {
			end = len(html)
		}
		window := html[start:end]
		if m := reIPA.FindStringSubmatch(window); len(m) == 2 {
			// 这里不去掉 / /，保持页面原样（例如 /ækˈtɪv.ə.ti/）
			ipa := strings.TrimSpace(htmlEntityDecode(m[1]))
			return ensureIPASlash(ipa)
		}
	}

	// fallback：取全页第一个 IPA（不保证区分 uk/us，但比空好）
	if m := reIPA.FindStringSubmatch(html); len(m) == 2 {
		return strings.TrimSpace(htmlEntityDecode(m[1]))
	}
	return ""
}

// htmlEntityDecode：最小化解码（IPA 一般不会太多实体字符）
func htmlEntityDecode(s string) string {
	repl := strings.NewReplacer(
		"&nbsp;", " ",
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
	)
	return repl.Replace(s)
}

// DownloadMP3 下载 mp3 并保存到 dstPath
func (s *Service) DownloadMP3(mp3URL, referer, dstPath string) error {
	req, err := http.NewRequest("GET", mp3URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", s.userAgent)
	req.Header.Set("Accept", "audio/mpeg,audio/*;q=0.9,*/*;q=0.8")
	// 加上 Referer 更像浏览器行为，降低被拦概率
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	// 目标目录不存在则创建
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}

	// 写入临时文件，成功后 rename（避免中断导致半截文件）
	tmp := dstPath + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
		_ = os.Remove(tmp)
	}()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	// 简单校验：Cambridge 发音 mp3 通常 > 1KB
	fi, err := os.Stat(tmp)
	if err != nil {
		return err
	}
	if fi.Size() < 1024 {
		return fmt.Errorf("downloaded file too small (%d bytes), may be blocked", fi.Size())
	}
	return os.Rename(tmp, dstPath)
}
