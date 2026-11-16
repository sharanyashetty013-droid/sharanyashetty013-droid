// Command gen builds the animated "live HTTP server" SVG for the GitHub
// profile and refreshes the open-source section of README.md.
//
// Run from the repo root:  go run ./cmd/gen
// Set GITHUB_TOKEN to avoid API rate limits (the Action does this for you).
package main

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

// ---------- config ----------

type Endpoint struct {
	Path string      `json:"path"`
	MS   int         `json:"ms"`
	Body [][2]string `json:"body"`
}

type PR struct {
	Repo   string `json:"repo"`
	Number int    `json:"number"`
	Status string `json:"status"` // merged | in review | closed
	URL    string `json:"url"`
	Title  string `json:"title"`
}

type Config struct {
	User        string     `json:"user"`
	Port        int        `json:"port"`
	Uptime      string     `json:"uptime"`
	Endpoints   []Endpoint `json:"endpoints"`
	FallbackPRs []PR       `json:"fallback_prs"`
}

// ---------- GitHub API ----------

type searchResp struct {
	TotalCount int `json:"total_count"`
	Items      []struct {
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

func search(q string, perPage int) (*searchResp, error) {
	u := "https://api.github.com/search/issues?sort=updated&order=desc&per_page=" +
		fmt.Sprint(perPage) + "&q=" + url.QueryEscape(q)
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	if t := os.Getenv("GITHUB_TOKEN"); t != "" {
		req.Header.Set("Authorization", "Bearer "+t)
	}
	c := &http.Client{Timeout: 15 * time.Second}
	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("github: %s", res.Status)
	}
	var out searchResp
	return &out, json.NewDecoder(res.Body).Decode(&out)
}

// fetchPRs returns PRs to other people's repos, newest first, plus totals.
func fetchPRs(user string) (prs []PR, total, merged int, err error) {
	all, err := search(fmt.Sprintf("type:pr author:%s -user:%s", user, user), 20)
	if err != nil {
		return nil, 0, 0, err
	}
	m, err := search(fmt.Sprintf("type:pr is:merged author:%s -user:%s", user, user), 1)
	if err != nil {
		return nil, 0, 0, err
	}
	for _, it := range all.Items {
		status := "in review"
		switch {
		case it.PullRequest.MergedAt != nil:
			status = "merged"
		case it.State == "closed":
			continue // closed without merge: don't show
		}
		repo := strings.TrimPrefix(it.RepositoryURL, "https://api.github.com/repos/")
		prs = append(prs, PR{repo, it.Number, status, it.HTMLURL, it.Title})
	}
	return prs, all.TotalCount, m.TotalCount, nil
}

// ---------- pixel character ----------
// One char = one pixel. '.' is transparent.
var palette = map[byte]string{
	'H': "#2b2422", // hair
	'h': "#4d3f3a", // hair shine
	'P': "#2b2422", // ponytail (own layer, it sways)
	'r': "#FDDD00", // hair tie
	'S': "#c58c64", // skin
	'E': "#1b1b1b", // eyes
	'C': "#e08f78", // blush
	'M': "#8a3f32", // mouth
	'T': "#00ADD8", // crop top
	't': "#5DC9E2", // crop top highlight
	'k': "#0e2e48", // skirt waistband
	'K': "#1f4e79", // skirt
	'B': "#e6edf3", // shoes
}

var body = []string{
	"....HHHHHH......",
	"...HhhHHHHH.....",
	"..HHHHHHHHHHr...",
	"..HHSSSSSSHH....",
	"..HSSSSSSSSH....",
	"..HSSESSESSH....",
	"..HSCSSSSCSH....",
	"..HSSSMMSSSH....",
	"....SSSSSS......",
	"......SS........",
	"..TTTTTTTTTT....",
	"..TtTTTTTTTT....",
	"...TTTTTTTT.....",
	"....SSSSSS......",
	"....kkkkkk......",
	"...KKKKKKKK.....",
	"..KKKKKKKKKK....",
	"..KKKKKKKKKK....",
	"....SS..SS......",
	"....SS..SS......",
	"....SS..SS......",
	"...BBB..BBB.....",
}

var ponytail = []string{
	"................",
	"................",
	".............P..",
	".............PP.",
	".............PP.",
	"..............PP",
	"...............P",
	"...............P",
}

// arms in three poses
var armsIdle = []string{
	"", "", "", "", "", "", "", "", "", "", "", "",
	"..S........S....",
	"..S........S....",
	"..S........S....",
	"..S........S....",
}
var armsCatch = []string{
	"", "", "", "", "", "",
	".S..........S...",
	".S..........S...",
	".S..........S...",
	".S..........S...",
}
var armsThrow = []string{
	"", "", "", "", "", "", "", "", "", "", "",
	"............SSS.",
	"..S.............",
	"..S.............",
	"..S.............",
	"..S.............",
}

const px = 7 // size of one pixel

func pixels(grid []string, ox, oy int) string {
	var b strings.Builder
	for y, row := range grid {
		for x := 0; x < len(row); {
			c := row[x]
			if c == '.' {
				x++
				continue
			}
			run := 1
			for x+run < len(row) && row[x+run] == c {
				run++
			}
			fmt.Fprintf(&b, `<rect x="%d" y="%d" width="%d" height="%d" fill="%s"/>`,
				ox+x*px, oy+y*px, run*px, px, palette[c])
			x += run
		}
	}
	return b.String()
}

// ---------- JSON-ish rendering ----------

var keyRe = regexp.MustCompile(`^(\s*)"([^"]+)": (.*?)(,?)$`)

func esc(s string) string { return html.EscapeString(s) }

func renderLine(line string, x, y int) string {
	m := keyRe.FindStringSubmatch(line)
	if m == nil {
		return fmt.Sprintf(`<text x="%d" y="%d" class="p">%s</text>`, x, y, esc(line))
	}
	ind, key, val, comma := m[1], m[2], m[3], m[4]
	vcls := "s"
	switch {
	case strings.HasPrefix(val, "["):
		vcls = "a"
	case strings.Contains(val, "merged"):
		vcls = "ok"
	case strings.Contains(val, "review"):
		vcls = "y"
	}
	return fmt.Sprintf(`<text x="%d" y="%d" xml:space="preserve"><tspan class="p">%s"</tspan><tspan class="k">%s</tspan><tspan class="p">": </tspan><tspan class="%s">%s</tspan><tspan class="p">%s</tspan></text>`,
		x, y, ind, esc(key), vcls, esc(val), comma)
}

func bodyLines(ep Endpoint, prs []PR, lastMerged string) []string {
	var kv [][2]string
	for _, p := range ep.Body {
		switch {
		case p[0] == "$prs":
			for i, pr := range prs {
				if i == 4 {
					break
				}
				kv = append(kv, [2]string{fmt.Sprintf("%s#%d", shortRepo(pr.Repo), pr.Number), pr.Status})
			}
		case p[1] == "$last_merged":
			kv = append(kv, [2]string{p[0], lastMerged})
		default:
			kv = append(kv, p)
		}
	}
	lines := []string{"{"}
	for i, p := range kv {
		comma := ","
		if i == len(kv)-1 {
			comma = ""
		}
		v := p[1]
		if !strings.HasPrefix(v, "[") {
			v = `"` + v + `"`
		}
		lines = append(lines, fmt.Sprintf(`  "%s": %s%s`, p[0], v, comma))
	}
	return append(lines, "}")
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
	W, H   = 860, 340
	slot   = 3.0 // seconds per request
	charX  = 300
	charY  = 78
	panelX = 488
)

func buildSVG(cfg Config, prs []PR, total, merged int, lastMerged string) string {
	n := len(cfg.Endpoints)
	cycle := slot * float64(n)
	share := 100.0 / float64(n) // percent of the full cycle one slot takes
	at := func(f float64) string { return fmt.Sprintf("%.2f%%", f*share) } // f = fraction of a slot

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img" aria-label="Sharanya's live HTTP server: a pixel character catching requests and throwing back responses">`, W, H, W, H)

	// ----- styles -----
	b.WriteString(`<style>
text{font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,"Liberation Mono",monospace;font-size:12px;fill:#e6edf3}
.k{fill:#00ADD8}.s{fill:#9be9ff}.p{fill:#6e7681}.a{fill:#FDDD00}.ok{fill:#3fb950}.y{fill:#FDDD00}
.dim{fill:#6e7681}.blue{fill:#00ADD8}.t{font-size:12.5px}
.live{animation:blink 1.2s steps(2) infinite}@keyframes blink{50%{opacity:.15}}
.cur{animation:blink 1s steps(2) infinite}
`)
	// character poses (3s loop, aligned to each slot)
	fmt.Fprintf(&b, `.idle{animation:idle %[1]gs infinite}.catch{opacity:0;animation:catch %[1]gs infinite}.throw{opacity:0;animation:throw %[1]gs infinite}
@keyframes idle{0%%,27%%{opacity:1}28%%,62%%{opacity:0}63%%,100%%{opacity:1}}
@keyframes catch{0%%,27%%{opacity:0}28%%,47%%{opacity:1}48%%,100%%{opacity:0}}
@keyframes throw{0%%,47%%{opacity:0}48%%,62%%{opacity:1}63%%,100%%{opacity:0}}
.hop{animation:hop %[1]gs infinite}@keyframes hop{0%%,27%%{transform:translateY(0)}33%%{transform:translateY(-8px)}40%%,100%%{transform:translateY(0)}}
.tail{transform-box:fill-box;transform-origin:0 0;animation:sway 1.5s ease-in-out infinite alternate}@keyframes sway{from{transform:rotate(-6deg)}to{transform:rotate(9deg)}}
.resp{opacity:0;animation:resp %[1]gs infinite}
@keyframes resp{0%%,49%%{opacity:0;transform:translate(0,0)}50%%{opacity:1;transform:translate(0,0)}62%%{transform:translate(48px,-46px)}74%%{opacity:1;transform:translate(96px,0)}75%%,100%%{opacity:0;transform:translate(96px,0)}}
.boot{animation:boot %[2]gs forwards}@keyframes boot{0%%,99%%{opacity:1}100%%{opacity:0}}
`, slot, slot*0.75)
	// per-request + per-panel animations (full cycle, staggered)
	fmt.Fprintf(&b, `.req{opacity:0;animation:req %gs infinite}
@keyframes req{0%%{opacity:1;transform:translateX(0)}%s{opacity:1;transform:translateX(160px)}%s,100%%{opacity:0;transform:translateX(160px)}}
.pan{opacity:0;animation:pan %gs infinite}
@keyframes pan{0%%,%s{opacity:0}%s,%s{opacity:1}%s,100%%{opacity:0}}
`, cycle, at(0.28), at(0.29), cycle, at(0.74), at(0.75), at(1.74), at(1.75))
	for i := range cfg.Endpoints {
		fmt.Fprintf(&b, ".d%d{animation-delay:%gs}\n", i, float64(i)*slot)
	}
	b.WriteString(`</style>`)

	// ----- frame -----
	fmt.Fprintf(&b, `<rect width="%d" height="%d" rx="12" fill="#0d1117"/>`, W, H)
	fmt.Fprintf(&b, `<rect x=".5" y=".5" width="%d" height="%d" rx="12" fill="none" stroke="#00ADD8" stroke-opacity=".35"/>`, W-1, H-1)
	// dotted backdrop
	b.WriteString(`<defs><pattern id="g" width="20" height="20" patternUnits="userSpaceOnUse"><rect width="2" height="2" fill="#00ADD8" opacity=".09"/></pattern></defs>`)
	fmt.Fprintf(&b, `<rect x="1" y="34" width="%d" height="%d" fill="url(#g)"/>`, W-2, H-68)

	// title bar
	b.WriteString(`<circle cx="20" cy="17" r="5" fill="#ff5f56"/><circle cx="37" cy="17" r="5" fill="#ffbd2e"/><circle cx="54" cy="17" r="5" fill="#27c93f"/>`)
	fmt.Fprintf(&b, `<text x="76" y="21" class="t"><tspan class="dim">~/</tspan>sharanya.go <tspan class="dim">— listening on</tspan> <tspan class="blue">:%d</tspan></text>`, cfg.Port)
	fmt.Fprintf(&b, `<g class="live"><circle cx="%d" cy="17" r="4.5" fill="#3fb950"/></g><text x="%d" y="21" class="t" fill="#3fb950" style="fill:#3fb950">LIVE</text>`, W-60, W-50)
	fmt.Fprintf(&b, `<line x1="1" y1="34" x2="%d" y2="34" stroke="#00ADD8" stroke-opacity=".25"/>`, W-1)

	// client side label
	b.WriteString(`<text x="22" y="62" class="dim">// incoming</text>`)

	// requests
	for i, ep := range cfg.Endpoints {
		label := "GET " + ep.Path
		w := len(label)*7 + 22
		fmt.Fprintf(&b, `<g class="req d%d"><rect x="%d" y="118" width="%d" height="24" rx="4" fill="#0b2a36" stroke="#00ADD8"/><text x="%d" y="134"><tspan class="y">GET</tspan> %s</text></g>`,
			i, 285-160-w, w, 285-160-w+11, esc(ep.Path))
	}

	// floor
	floorY := charY + len(body)*px
	for x := 16; x < panelX-12; x += 14 {
		fmt.Fprintf(&b, `<rect x="%d" y="%d" width="12" height="6" fill="#00ADD8" opacity=".22"/>`, x, floorY+2)
	}

	// character
	fmt.Fprintf(&b, `<g class="hop">`)
	// name plate
	fmt.Fprintf(&b, `<text x="%d" y="%d" class="dim" style="font-size:11px">▼ sharanya</text>`, charX+12, charY-14)
	fmt.Fprintf(&b, `<g class="tail">%s</g>`, pixels(ponytail, charX, charY))
	b.WriteString(pixels(body, charX, charY))
	fmt.Fprintf(&b, `<g class="idle">%s</g>`, pixels(armsIdle, charX, charY))
	fmt.Fprintf(&b, `<g class="catch">%s</g>`, pixels(armsCatch, charX, charY))
	fmt.Fprintf(&b, `<g class="throw">%s</g>`, pixels(armsThrow, charX, charY))
	b.WriteString(`</g>`)

	// thrown response
	fmt.Fprintf(&b, `<g class="resp"><rect x="%d" y="148" width="62" height="22" rx="4" fill="#0f2a17" stroke="#3fb950"/><text x="%d" y="163" style="fill:#3fb950">200 OK</text></g>`, charX+16*px-20, charX+16*px-12)

	// response panel
	pw := W - panelX - 18
	fmt.Fprintf(&b, `<rect x="%d" y="50" width="%d" height="%d" rx="8" fill="#0a1520" stroke="#00ADD8" stroke-opacity=".4"/>`, panelX, pw, H-50-46)
	fmt.Fprintf(&b, `<text x="%d" y="70" class="dim">// response</text>`, panelX+14)
	fmt.Fprintf(&b, `<g class="boot"><text x="%d" y="100" class="dim">waiting for requests<tspan class="cur">_</tspan></text></g>`, panelX+14)
	for i, ep := range cfg.Endpoints {
		fmt.Fprintf(&b, `<g class="pan d%d">`, i)
		fmt.Fprintf(&b, `<text x="%d" y="96"><tspan class="dim">→</tspan> <tspan class="y">GET</tspan> %s <tspan class="dim">HTTP/1.1</tspan></text>`, panelX+14, esc(ep.Path))
		fmt.Fprintf(&b, `<text x="%d" y="114"><tspan class="dim">←</tspan> <tspan class="ok">200 OK</tspan> <tspan class="dim">· %dms · application/json</tspan></text>`, panelX+14, ep.MS)
		for j, l := range bodyLines(ep, prs, lastMerged) {
			b.WriteString(renderLine(l, panelX+14, 140+j*18))
		}
		b.WriteString(`</g>`)
	}

	// status bar
	fmt.Fprintf(&b, `<line x1="1" y1="%d" x2="%d" y2="%d" stroke="#00ADD8" stroke-opacity=".25"/>`, H-34, W-1, H-34)
	fmt.Fprintf(&b, `<text x="20" y="%d" class="t"><tspan class="dim">uptime</tspan> %s  <tspan class="dim">·  PRs shipped</tspan> <tspan class="blue">%d</tspan>  <tspan class="dim">·  merged</tspan> <tspan class="ok">%d</tspan>  <tspan class="dim">·  last sync</tspan> %s</text>`,
		H-13, esc(cfg.Uptime), total, merged, time.Now().UTC().Format("02 Jan 2006 15:04 UTC"))

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
		return fmt.Errorf("markers %s / %s not found in %s", start, end, path)
	}
	var b strings.Builder
	b.WriteString(start + "\n")
	b.WriteString("| request | repo | what | status |\n|---|---|---|---|\n")
	for k, pr := range prs {
		if k == 6 {
			break
		}
		badge := "🟡 in review"
		if pr.Status == "merged" {
			badge = "🟢 merged"
		}
		fmt.Fprintf(&b, "| `POST /pulls` | [%s#%d](%s) | %s | %s |\n", pr.Repo, pr.Number, pr.URL, strings.ReplaceAll(pr.Title, "|", "\\|"), badge)
	}
	b.WriteString(end)
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

	prs, total, merged, err := fetchPRs(cfg.User)
	if err != nil || len(prs) == 0 {
		log.Printf("using fallback PRs (%v)", err)
		prs = cfg.FallbackPRs
		total, merged = len(prs), 0
		for _, p := range prs {
			if p.Status == "merged" {
				merged++
			}
		}
	}

	lastMerged := "soon"
	for _, p := range prs {
		if p.Status == "merged" {
			lastMerged = fmt.Sprintf("%s#%d", shortRepo(p.Repo), p.Number)
			break
		}
	}

	if err := os.WriteFile("assets/server.svg", []byte(buildSVG(cfg, prs, total, merged, lastMerged)), 0o644); err != nil {
		log.Fatal(err)
	}
	if err := updateReadme("README.md", prs); err != nil {
		log.Fatal(err)
	}
	log.Printf("ok: %d PRs (%d merged)", total, merged)
}
