// Command gen builds the animated profile banner (assets/banner.svg) and
// refreshes the open-source section of README.md.
//
// Run from the repo root:  go run ./cmd/gen
// Set GITHUB_TOKEN to avoid API rate limits (the Action does this for you).
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// ---------- config ----------

type PR struct {
	Repo   string `json:"repo"`
	Number int    `json:"number"`
	Status string `json:"status"` // merged | in review
	URL    string `json:"url"`
	Title  string `json:"title"`
}

type Config struct {
	User        string   `json:"user"`
	Name        string   `json:"name"`
	Tagline     string   `json:"tagline"`
	Wins        int      `json:"wins"`
	Lines       []string `json:"lines"`
	Exclude     []string `json:"exclude_repos"`
	FallbackPRs []PR     `json:"fallback_prs"`
}

// ---------- GitHub API ----------

type searchResp struct {
	Items []struct {
		Title         string `json:"title"`
		Number        int    `json:"number"`
		HTMLURL       string `json:"html_url"`
		State         string `json:"state"`
		RepositoryURL string `json:"repository_url"`
		PullRequest   struct {
			MergedAt *string `json:"merged_at"`
		} `json:"pull_request"`
	} `json:"items"`
}

func fetchPRs(user string, exclude []string) ([]PR, error) {
	q := fmt.Sprintf("type:pr author:%s -user:%s", user, user)
	req, _ := http.NewRequest("GET", "https://api.github.com/search/issues?sort=updated&order=desc&per_page=50&q="+url.QueryEscape(q), nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	if t := os.Getenv("GITHUB_TOKEN"); t != "" {
		req.Header.Set("Authorization", "Bearer "+t)
	}
	res, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("github: %s", res.Status)
	}
	var out searchResp
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	skip := map[string]bool{}
	for _, r := range exclude {
		skip[strings.ToLower(r)] = true
	}
	var prs []PR
	for _, it := range out.Items {
		status := "in review"
		if it.PullRequest.MergedAt != nil {
			status = "merged"
		} else if it.State == "closed" {
			continue
		}
		repo := strings.TrimPrefix(it.RepositoryURL, "https://api.github.com/repos/")
		if skip[strings.ToLower(repo)] {
			continue
		}
		prs = append(prs, PR{repo, it.Number, status, it.HTMLURL, it.Title})
	}
	return prs, nil
}

// "rajputomsingh/Relay" -> "Relay", but keep "golang/go" recognisable.
func shortRepo(r string) string {
	if r == "golang/go" {
		return r
	}
	if i := strings.LastIndex(r, "/"); i >= 0 {
		return r[i+1:]
	}
	return r
}

// ---------- SVG ----------

const (
	W, H = 860, 380
	// character source image is 426x800 (cropped at the knees);
	// it is drawn at this scale in the banner
	charScale = 0.425
	charX     = 58
	charY     = 24
	slot      = 4.5 // seconds each dialogue line stays up
	fontPx    = 16
	charW     = fontPx * 0.6 // monospace advance
)

func esc(s string) string { return html.EscapeString(s) }

func dataURI(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b)
}

// skirt draws a pleated skirt in the character's source-pixel coordinates.
func skirt() string {
	const (
		topY           = 452.0
		topL, topR     = 123.0, 290.0
		hemL, hemR     = 104.0, 309.0
		hemY, hemCurve = 624.0, 12.0
		n              = 10
	)
	top := func(i int) (float64, float64) { return topL + float64(i)*(topR-topL)/n, topY }
	hem := func(i int) (float64, float64) {
		t := float64(i) / n
		return hemL + t*(hemR-hemL), hemY + hemCurve*math.Sin(math.Pi*t)
	}
	var b strings.Builder
	b.WriteString(`<defs>
<linearGradient id="pl" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#1b5b8c"/><stop offset="1" stop-color="#11446c"/></linearGradient>
<linearGradient id="pd" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#0f3554"/><stop offset="1" stop-color="#0a2740"/></linearGradient>
<linearGradient id="sh" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#000" stop-opacity=".45"/><stop offset=".25" stop-color="#000" stop-opacity="0"/></linearGradient>
</defs>`)
	// pivot the sway around the waist
	fmt.Fprintf(&b, `<g transform="translate(206 %g)"><g><animateTransform attributeName="transform" type="skewX" values="-1.6;1.6;-1.6" dur="3.2s" repeatCount="indefinite" calcMode="spline" keySplines=".45 0 .55 1;.45 0 .55 1"/><g transform="translate(-206 -%g)">`, topY, topY)
	var outline strings.Builder
	for i := 0; i < n; i++ {
		x0, y0 := top(i)
		x1, y1 := top(i + 1)
		x2, y2 := hem(i + 1)
		x3, y3 := hem(i)
		fill := "url(#pl)"
		if i%2 == 1 {
			fill = "url(#pd)"
		}
		fmt.Fprintf(&b, `<path d="M%.1f %.1fL%.1f %.1fL%.1f %.1fL%.1f %.1fZ" fill="%s"/>`, x0, y0, x1, y1, x2, y2, x3, y3, fill)
		if i > 0 {
			fmt.Fprintf(&b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#5DC9E2" stroke-opacity=".28" stroke-width="1"/>`, x0, y0, x3, y3)
		}
	}
	// shadow under the waistband
	fmt.Fprintf(&b, `<path d="M%g %gL%g %gL%g %gL%g %gZ" fill="url(#sh)"/>`, topL, topY, topR, topY, hemR, hemY, hemL, hemY)
	// hem trim
	outline.WriteString("M")
	for i := 0; i <= n*2; i++ {
		t := float64(i) / float64(n*2)
		x := hemL + t*(hemR-hemL)
		y := hemY + hemCurve*math.Sin(math.Pi*t)
		fmt.Fprintf(&outline, "%.1f %.1f ", x, y)
	}
	fmt.Fprintf(&b, `<path d="%s" fill="none" stroke="#00ADD8" stroke-width="4" stroke-linecap="round"/>`, strings.TrimSpace(outline.String()))
	b.WriteString(`</g></g></g>`)
	// waistband (does not sway)
	fmt.Fprintf(&b, `<path d="M%g %gQ206 %g %g %gL%g %gQ206 %g %g %gZ" fill="#0a2238"/>`,
		topL-1, topY-11, topY-15, topR+1, topY-11, topR+1, topY+1, topY-3, topL-1, topY+1)
	fmt.Fprintf(&b, `<path d="M%g %gQ206 %g %g %g" fill="none" stroke="#00ADD8" stroke-opacity=".7" stroke-width="1.5"/>`, topL, topY-10, topY-14, topR, topY-10)
	b.WriteString(`<rect x="199" y="439" width="14" height="11" rx="2" fill="#FDDD00"/><rect x="203" y="442" width="6" height="5" fill="#0a2238"/>`)
	return b.String()
}

// typing returns a text line revealed character by character during its slot.
func typing(id int, text string, x, y float64, total int, cycle float64) string {
	n := len([]rune(text))
	start := float64(id) / float64(total)
	end := float64(id+1) / float64(total)
	typeDur := math.Min(2.0, float64(n)*0.045) / cycle // fraction of cycle spent typing
	var kt, vals []string
	add := func(t, v float64) {
		kt = append(kt, fmt.Sprintf("%.4f", t))
		vals = append(vals, fmt.Sprintf("%.1f", v))
	}
	add(0, 0)
	if start > 0 {
		add(start, 0)
	}
	for c := 1; c <= n; c++ {
		add(start+typeDur*float64(c)/float64(n), float64(c)*charW+0.5)
	}
	if end < 1 {
		add(end, 0)
	}
	kt = append(kt, "1")
	vals = append(vals, vals[len(vals)-1])
	return fmt.Sprintf(`<clipPath id="c%d"><rect x="%g" y="%g" width="0" height="%d"><animate attributeName="width" dur="%gs" repeatCount="indefinite" calcMode="discrete" keyTimes="%s" values="%s"/></rect></clipPath><text x="%g" y="%g" class="dlg" clip-path="url(#c%d)">%s</text>`,
		id, x, y-fontPx, fontPx+8, cycle, strings.Join(kt, ";"), strings.Join(vals, ";"), x, y, id, esc(text))
}

func buildSVG(cfg Config, prs []PR) string {
	merged, review := 0, 0
	lastMerged, inReview := "", ""
	for _, p := range prs {
		ref := fmt.Sprintf("%s#%d", shortRepo(p.Repo), p.Number)
		if p.Status == "merged" {
			merged++
			if lastMerged == "" {
				lastMerged = ref
			}
		} else {
			review++
			if inReview == "" {
				inReview = ref
			}
		}
	}
	var lines []string
	for _, l := range cfg.Lines {
		if (strings.Contains(l, "{last_merged}") && lastMerged == "") || (strings.Contains(l, "{in_review}") && inReview == "") {
			continue
		}
		l = strings.NewReplacer("{last_merged}", lastMerged, "{in_review}", inReview).Replace(l)
		lines = append(lines, l)
	}
	cycle := slot * float64(len(lines))

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="%d" height="%d" viewBox="0 0 %d %d" role="img" aria-label="%s: %s">`, W, H, W, H, esc(cfg.Name), esc(cfg.Tagline))
	b.WriteString(`<style>
text{font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,"Liberation Mono",monospace;fill:#13232d}
.name{font-family:"Segoe UI",Inter,"Helvetica Neue",Arial,sans-serif;font-weight:800;font-size:38px;letter-spacing:-.5px;fill:#0b2533}
.tag{font-size:14px;fill:#0089ad}
.chip{font-size:12.5px}
.dlg{font-size:16px}
.dim{fill:#7d8a93;font-size:11px}
.tw{animation:tw 2.4s ease-in-out infinite}@keyframes tw{0%,100%{opacity:.15;transform:scale(.6)}50%{opacity:1;transform:scale(1)}}
.tw{transform-box:fill-box;transform-origin:center}
.arrow{animation:bob .9s ease-in-out infinite alternate}@keyframes bob{to{transform:translateY(4px)}}
.p{animation:rise linear infinite}@keyframes rise{0%{transform:translateY(0);opacity:0}15%{opacity:.9}100%{transform:translateY(-90px);opacity:0}}
</style>`)
	// background
	fmt.Fprintf(&b, `<defs><radialGradient id="glow" cx=".5" cy=".5" r=".5"><stop offset="0" stop-color="#ffffff" stop-opacity=".95"/><stop offset=".55" stop-color="#bfeaf6" stop-opacity=".55"/><stop offset="1" stop-color="#00ADD8" stop-opacity="0"/></radialGradient>
<linearGradient id="xpg" x1="0" x2="1"><stop offset="0" stop-color="#00ADD8"/><stop offset="1" stop-color="#5DE0B5"/></linearGradient>
<linearGradient id="bg" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#f3fbfe"/><stop offset="1" stop-color="#dff3fa"/></linearGradient></defs>`)
	fmt.Fprintf(&b, `<rect width="%d" height="%d" rx="16" fill="url(#bg)"/>`, W, H)
	fmt.Fprintf(&b, `<rect x=".5" y=".5" width="%d" height="%d" rx="16" fill="none" stroke="#00ADD8" stroke-opacity=".35"/>`, W-1, H-1)
	b.WriteString(`<circle cx="150" cy="190" r="180" fill="url(#glow)"><animate attributeName="r" values="170;188;170" dur="4s" repeatCount="indefinite"/></circle>`)
	b.WriteString(`<circle cx="790" cy="40" r="90" fill="#00ADD8" fill-opacity=".07"/><circle cx="700" cy="370" r="70" fill="#00ADD8" fill-opacity=".06"/><circle cx="40" cy="350" r="60" fill="#00ADD8" fill-opacity=".05"/>`)
	for i, p := range [][3]float64{{262, 60, 7}, {800, 168, 6}, {48, 120, 5}, {660, 34, 5}, {250, 300, 6}, {560, 175, 4}} {
		x, y, r := p[0], p[1], p[2]
		fmt.Fprintf(&b, `<path class="tw" style="animation-delay:-%.1fs" d="M%g %gQ%g %g %g %gQ%g %g %g %gQ%g %g %g %gQ%g %g %g %gZ" fill="#00ADD8"/>`,
			float64(i)*0.4, x, y-r, x, y, x+r, y, x, y, x, y+r, x, y, x-r, y, x, y, x, y-r)
	}

	// floating particles
	rng := rand.New(rand.NewSource(7))
	for i := 0; i < 22; i++ {
		x := 20 + rng.Float64()*260
		if i%3 == 0 {
			x = 300 + rng.Float64()*540
		}
		y := 140 + rng.Float64()*230
		r := 1 + rng.Float64()*1.8
		dur := 4 + rng.Float64()*5
		fmt.Fprintf(&b, `<circle class="p" cx="%.0f" cy="%.0f" r="%.1f" fill="#00ADD8" fill-opacity=".45" style="animation-duration:%.1fs;animation-delay:-%.1fs"/>`, x, y, r, dur, rng.Float64()*dur)
	}

	// ---- game layer ----
	xp := merged*500 + review*200 + cfg.Wins*300
	const perLevel = 750
	level := 1 + xp/perLevel
	frac := float64(xp%perLevel) / perLevel
	questSlot := -1
	for i, l := range lines {
		if lastMerged != "" && strings.Contains(l, lastMerged) {
			questSlot = i
		}
	}
	questJump, hud := "", ""
	{
		var h strings.Builder
		// hearts + level, top-left
		for i := 0; i < 5; i++ {
			x := 22 + float64(i)*20
			fmt.Fprintf(&h, `<path transform="translate(%.0f 318)" d="M8 14 1.6 7.8A3.9 3.9 0 0 1 8 2.9a3.9 3.9 0 0 1 6.4 4.9z" fill="#ff4d6d" stroke="#c9184a" stroke-width="1"/>`, x)
		}
		fmt.Fprintf(&h, `<rect x="22" y="342" width="52" height="20" rx="5" fill="#0b2533"/><text x="31" y="356.5" style="font-size:12px;font-weight:700;fill:#FDDD00">LV %d</text>`, level)
		// XP bar under the chips
		const bx, by, bw = 292.0, 162.0, 300.0
		fmt.Fprintf(&h, `<text x="%g" y="%g" style="font-size:11px;font-weight:700;fill:#0b2533">XP</text>`, bx, by+9)
		fmt.Fprintf(&h, `<rect x="%g" y="%g" width="%g" height="10" rx="5" fill="#0b2533" fill-opacity=".1"/>`, bx+22, by, bw)
		fmt.Fprintf(&h, `<rect x="%g" y="%g" width="0" height="10" rx="5" fill="url(#xpg)"><animate attributeName="width" values="0;%.1f;%.1f" keyTimes="0;.25;1" dur="9s" repeatCount="indefinite" calcMode="spline" keySplines=".2 .8 .2 1;0 0 1 1"/></rect>`, bx+22, by, bw*frac, bw*frac)
		fmt.Fprintf(&h, `<text x="%g" y="%g" style="font-size:11px;fill:#55707d">%d / %d to LV %d</text>`, bx+bw+32, by+9, xp%perLevel, perLevel, level+1)
		if questSlot >= 0 {
			n := float64(len(lines))
			t0 := float64(questSlot) / n
			t1 := t0 + 0.6/n
			kt := fmt.Sprintf("0;%.4f;%.4f;%.4f;%.4f;1", t0, t0+0.02/n, t0+0.5/n, t1)
			fmt.Fprintf(&h, `<g opacity="0"><animate attributeName="opacity" values="0;0;1;1;0;0" keyTimes="%s" dur="%gs" repeatCount="indefinite"/><animateTransform attributeName="transform" type="translate" values="0 12;0 12;0 0;0 -6;0 -10;0 -10" keyTimes="%s" dur="%gs" repeatCount="indefinite"/>`, kt, cycle, kt, cycle)
			h.WriteString(`<rect x="62" y="6" width="176" height="28" rx="8" fill="#0b2533"/><text x="74" y="25" style="font-size:12.5px;font-weight:700;fill:#FDDD00">QUEST CLEARED +500 XP</text></g>`)
			// coin burst
			for i := 0; i < 6; i++ {
				dx := -60 + float64(i)*24
				fmt.Fprintf(&h, `<g opacity="0"><animate attributeName="opacity" values="0;0;1;0;0" keyTimes="0;%.4f;%.4f;%.4f;1" dur="%gs" repeatCount="indefinite"/><animateTransform attributeName="transform" type="translate" values="0 0;0 0;%.0f -110;%.0f -70" keyTimes="0;%.4f;%.4f;1" dur="%gs" repeatCount="indefinite"/><circle cx="150" cy="200" r="6" fill="#FDDD00" stroke="#b8860b" stroke-width="1.5"/></g>`,
					t0+0.04/n, t0+0.1/n, t0+0.4/n, cycle, dx*1.4, dx*1.8, t0, t0+0.35/n, cycle)
			}
			questJump = fmt.Sprintf(`<animateTransform attributeName="transform" type="translate" additive="sum" values="0 0;0 0;0 -18;0 0;0 0" keyTimes="0;%.4f;%.4f;%.4f;1" dur="%gs" repeatCount="indefinite"/>`, t0, t0+0.08/n, t0+0.16/n, cycle)
		}
		hud = h.String()
	}

	// character: bob, skirt, blink
	fmt.Fprintf(&b, `<g transform="translate(%d %d)"><g><animateTransform attributeName="transform" type="translate" values="0 0;0 -5;0 0" dur="3.2s" repeatCount="indefinite" calcMode="spline" keySplines=".45 0 .55 1;.45 0 .55 1"/><g>%s<g transform="scale(%g)">`, charX, charY, questJump, charScale)
	fmt.Fprintf(&b, `<g transform="translate(206 150)"><g><animateTransform attributeName="transform" type="skewX" values="-2.4;2.2;-2.4" dur="2.8s" repeatCount="indefinite" calcMode="spline" keySplines=".45 0 .55 1;.45 0 .55 1"/><g transform="translate(-206 -150)"><image width="426" height="800" xlink:href="%s"/></g></g></g>`, dataURI("assets/hair.png"))
	fmt.Fprintf(&b, `<image width="426" height="800" xlink:href="%s"/>`, dataURI("assets/char.png"))
	b.WriteString(skirt())
	fmt.Fprintf(&b, `<image x="150" y="90" width="100" height="50" opacity="0" xlink:href="%s"><animate attributeName="opacity" values="0;0;1;1;0;0;1;1;0" keyTimes="0;.6;.605;.64;.645;.88;.885;.92;.925" dur="4.2s" repeatCount="indefinite"/></image>`, dataURI("assets/blink.png"))
	b.WriteString(`</g></g></g></g>`)
	b.WriteString(hud)

	// title + chips
	b.WriteString(fmt.Sprintf(`<text x="290" y="76" class="name">%s</text>`, esc(cfg.Name)))
	b.WriteString(fmt.Sprintf(`<text x="292" y="104" class="tag">%s</text>`, esc(cfg.Tagline)))
	chips := []struct{ label, color string }{
		{fmt.Sprintf("● %d PRs merged", merged), "#1a7f37"},
		{fmt.Sprintf("◆ %d in review", review), "#b35900"},
		{fmt.Sprintf("★ %d hackathon wins", cfg.Wins), "#0077a3"},
	}
	cx := 292.0
	for _, c := range chips {
		w := float64(len([]rune(c.label)))*7.6 + 22
		fmt.Fprintf(&b, `<rect x="%.0f" y="122" width="%.0f" height="26" rx="13" fill="%s" fill-opacity=".08" stroke="%s" stroke-opacity=".5"/><text x="%.0f" y="139.5" class="chip" style="fill:%s">%s</text>`, cx, w, c.color, c.color, cx+11, c.color, esc(c.label))
		cx += w + 10
	}

	// dialogue box (visual-novel style)
	b.WriteString(`<rect x="290" y="199" width="548" height="154" rx="12" fill="#0089ad" fill-opacity=".12"/><rect x="290" y="196" width="548" height="154" rx="12" fill="#ffffff" stroke="#00ADD8" stroke-width="2"/>`)
	b.WriteString(`<rect x="296" y="202" width="536" height="142" rx="8" fill="none" stroke="#00ADD8" stroke-opacity=".2"/>`)
	b.WriteString(`<rect x="308" y="182" width="118" height="30" rx="6" fill="#00ADD8"/><text x="322" y="202" style="font-weight:700;font-size:14px;fill:#ffffff;letter-spacing:2px">SHARANYA</text>`)
	for i, l := range lines {
		b.WriteString(typing(i, l, 316, 258, len(lines), cycle))
	}
	fmt.Fprintf(&b, `<text x="316" y="332" class="dim">auto-synced from GitHub · %s</text>`, time.Now().UTC().Format("02 Jan 2006"))
	b.WriteString(`<g class="arrow"><path d="M806 318l8 9 8-9z" fill="#00ADD8"/></g>`)

	b.WriteString(`</svg>`)
	return b.String()
}

// ---------- README ----------

func updateReadme(path string, prs []PR) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	const start, end = "<!--OSS:START-->", "<!--OSS:END-->"
	s := string(raw)
	i, j := strings.Index(s, start), strings.Index(s, end)
	if i < 0 || j < i {
		return nil // no open-source section in the README: nothing to update
	}
	var b strings.Builder
	b.WriteString(start + "\n")
	for k, pr := range prs {
		if k == 6 {
			break
		}
		badge := "🟡"
		if pr.Status == "merged" {
			badge = "🟢"
		}
		fmt.Fprintf(&b, "- %s **[%s#%d](%s)** · %s\n", badge, pr.Repo, pr.Number, pr.URL, pr.Title)
	}
	b.WriteString("\n<sub>🟢 merged · 🟡 in review</sub>\n" + end)
	return os.WriteFile(path, []byte(s[:i]+b.String()+s[j+len(end):]), 0o644)
}

func main() {
	raw, err := os.ReadFile("profile.json")
	if err != nil {
		log.Fatal(err)
	}
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		log.Fatal(err)
	}
	prs, err := fetchPRs(cfg.User, cfg.Exclude)
	if err != nil || len(prs) == 0 {
		log.Printf("using fallback PRs (%v)", err)
		prs = cfg.FallbackPRs
	}
	if err := os.WriteFile("assets/banner.svg", []byte(buildSVG(cfg, prs)), 0o644); err != nil {
		log.Fatal(err)
	}
	if err := updateReadme("README.md", prs); err != nil {
		log.Fatal(err)
	}
	log.Printf("ok: %d PRs", len(prs))
}
