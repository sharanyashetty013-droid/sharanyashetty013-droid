// Command gen pulls live stats (GitHub contributions + streak, LeetCode) and writes:
//
//	assets/stats.svg       – the "player stats" card
//
// Run from the repo root:  go run ./cmd/gen
// Set GITHUB_TOKEN so the GraphQL contribution query works (the Action does this).
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type LC struct {
	Easy   int `json:"easy"`
	Medium int `json:"medium"`
	Hard   int `json:"hard"`
}

type Stats struct {
	Contributions int `json:"contributions"`
	Streak        int `json:"streak"`
	Longest       int `json:"longest_streak"`
	LeetCode      LC  `json:"leetcode"`
}

type Clue struct {
	Title string `json:"title"`
	Sub   string `json:"sub"`
}

type Config struct {
	User        string       `json:"user"`
	LeetCode    string       `json:"leetcode"`
	Name        string       `json:"name"`
	Tagline     string       `json:"tagline"`
	Wins        int          `json:"wins"`
	Projects    int          `json:"projects"`
	Lines       []string     `json:"lines"`
	Clues       []Clue       `json:"clues"`
	Fallback    Stats        `json:"fallback"`
	Exclude     []string     `json:"exclude_repos"`
	OSS         []PR         `json:"fallback_merged"`
	ProjectList []Project    `json:"project_cards"`
	Wins2       []Win        `json:"win_list"`
	Stack       []StackGroup `json:"stack"`
}

var client = &http.Client{Timeout: 20 * time.Second}

func postJSON(url string, body any, hdr map[string]string, out any) error {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("%s: %s", url, res.Status)
	}
	return json.NewDecoder(res.Body).Decode(out)
}

// contributions returns this year's total and the current / longest daily streaks.
func contributions(user string) (total, streak, longest int, err error) {
	tok := os.Getenv("GITHUB_TOKEN")
	if tok == "" {
		return 0, 0, 0, fmt.Errorf("GITHUB_TOKEN not set")
	}
	var out struct {
		Data struct {
			User struct {
				CC struct {
					Restricted int `json:"restrictedContributionsCount"`
					Cal        struct {
						Total int `json:"totalContributions"`
						Weeks []struct {
							Days []struct {
								Count int `json:"contributionCount"`
							} `json:"contributionDays"`
						} `json:"weeks"`
					} `json:"contributionCalendar"`
				} `json:"contributionsCollection"`
			} `json:"user"`
		} `json:"data"`
	}
	q := `query($u:String!){user(login:$u){contributionsCollection{restrictedContributionsCount contributionCalendar{totalContributions weeks{contributionDays{contributionCount}}}}}}`
	if err := postJSON("https://api.github.com/graphql", map[string]any{"query": q, "variables": map[string]string{"u": user}},
		map[string]string{"Authorization": "Bearer " + tok}, &out); err != nil {
		return 0, 0, 0, err
	}
	var days []int
	for _, w := range out.Data.User.CC.Cal.Weeks {
		for _, d := range w.Days {
			days = append(days, d.Count)
		}
	}
	if len(days) == 0 {
		return 0, 0, 0, fmt.Errorf("graphql: no contribution data")
	}
	run := 0
	for _, c := range days {
		if c > 0 {
			run++
			longest = max(longest, run)
		} else {
			run = 0
		}
	}
	// current streak: today may still be empty, so start from yesterday in that case
	i := len(days) - 1
	if days[i] == 0 {
		i--
	}
	for ; i >= 0 && days[i] > 0; i-- {
		streak++
	}
	// with "include private contributions" on, the calendar total already counts them; don't add Restricted again
	log.Printf("calendar total %d (restricted, already included: %d)", out.Data.User.CC.Cal.Total, out.Data.User.CC.Restricted)
	// also log this calendar year's numbers, to compare with "N contributions in YYYY" on the profile
	var y struct {
		Data struct {
			User struct {
				CC struct {
					Restricted int `json:"restrictedContributionsCount"`
					Cal        struct {
						Total int `json:"totalContributions"`
					} `json:"contributionCalendar"`
				} `json:"contributionsCollection"`
			} `json:"user"`
		} `json:"data"`
	}
	from := time.Date(time.Now().UTC().Year(), 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	qy := `query($u:String!,$f:DateTime!){user(login:$u){contributionsCollection(from:$f){restrictedContributionsCount contributionCalendar{totalContributions}}}}`
	if err := postJSON("https://api.github.com/graphql", map[string]any{"query": qy, "variables": map[string]string{"u": user, "f": from}},
		map[string]string{"Authorization": "Bearer " + tok}, &y); err == nil {
		log.Printf("this year: calendar %d, restricted %d", y.Data.User.CC.Cal.Total, y.Data.User.CC.Restricted)
	}
	return out.Data.User.CC.Cal.Total, streak, longest, nil
}

func leetcode(user string) (LC, error) {
	var out struct {
		Data struct {
			MatchedUser struct {
				Stats struct {
					AC []struct {
						Difficulty string `json:"difficulty"`
						Count      int    `json:"count"`
					} `json:"acSubmissionNum"`
				} `json:"submitStatsGlobal"`
			} `json:"matchedUser"`
		} `json:"data"`
	}
	q := `query($u:String!){matchedUser(username:$u){submitStatsGlobal{acSubmissionNum{difficulty count}}}}`
	if err := postJSON("https://leetcode.com/graphql", map[string]any{"query": q, "variables": map[string]string{"u": user}},
		map[string]string{"Referer": "https://leetcode.com", "User-Agent": "Mozilla/5.0"}, &out); err != nil {
		return LC{}, err
	}
	var lc LC
	for _, a := range out.Data.MatchedUser.Stats.AC {
		switch a.Difficulty {
		case "Easy":
			lc.Easy = a.Count
		case "Medium":
			lc.Medium = a.Count
		case "Hard":
			lc.Hard = a.Count
		}
	}
	if lc.Easy+lc.Medium+lc.Hard == 0 {
		return LC{}, fmt.Errorf("leetcode: empty stats")
	}
	return lc, nil
}

func level(s Stats, c Config) int { return 1 + s.Contributions/100 + c.Wins/3 }

// ---------- stats card ----------

func font(name, path string, weight int) string {
	b, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	return fmt.Sprintf(`@font-face{font-family:%s;font-weight:%d;src:url(data:font/woff2;base64,%s) format("woff2")}`, name, weight, base64.StdEncoding.EncodeToString(b))
}

var icons = map[string]string{
	"bolt":  `<path d="M3 -11 -7 2h6l-3 10 10-13H0z" fill="#fff"/>`,
	"flame": `<path d="M0 -12c4 5 9 8 9 14a9 9 0 0 1-18 0c0-4 2-6 4-8 0 3 2 4 3 4-1-4 0-7 2-10z" fill="#fff"/>`,
	"star":  `<path d="M0 -11 3.2 -3.4 11 -3.4 4.8 1.6 7 9.4 0 4.8 -7 9.4 -4.8 1.6 -11 -3.4 -3.2 -3.4z" fill="#fff"/>`,
	"code":  `<path d="M-4 -7-10 0l6 7M4 -7l6 7-6 7M2 -10l-4 20" fill="none" stroke="#fff" stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round"/>`,
	"block": `<rect x="-8" y="-8" width="16" height="16" rx="2" fill="#fff"/><rect x="-4" y="-4" width="8" height="8" fill="#0089ad"/>`,
}

func esc(s string) string { return html.EscapeString(s) }

func tile(b *strings.Builder, x, y, w, h float64, icon, color, label, value, sub string) {
	fmt.Fprintf(b, `<rect x="%g" y="%g" width="%g" height="%g" rx="14" fill="#fff" stroke="#bfeaf6" stroke-width="2"/>`, x, y, w, h)
	fmt.Fprintf(b, `<g transform="translate(%g %g)"><rect x="-20" y="-20" width="40" height="40" rx="10" fill="%s"/>%s</g>`, x+38, y+h/2, color, icons[icon])
	fmt.Fprintf(b, `<text x="%g" y="%g" class="lbl">%s</text>`, x+72, y+30, esc(label))
	fmt.Fprintf(b, `<text x="%g" y="%g" class="num">%s</text>`, x+72, y+66, esc(value))
	if sub != "" {
		fmt.Fprintf(b, `<text x="%g" y="%g" class="sub">%s</text>`, x+72, y+86, esc(sub))
	}
}

func statsSVG(c Config, s Stats) string {
	const W, H = 880.0, 330.0
	lv := level(s, c)
	solved := s.LeetCode.Easy + s.LeetCode.Medium + s.LeetCode.Hard
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%g" height="%g" viewBox="0 0 %g %g" role="img" aria-label="Player stats">`, W, H, W, H)
	b.WriteString(`<style>`)
	b.WriteString(font("Pixel", "assets/fonts/pixel.woff2", 400))
	b.WriteString(font("Nunito", "assets/fonts/nunito-800.woff2", 800))
	b.WriteString(font("Nunito", "assets/fonts/nunito-700.woff2", 700))
	b.WriteString(`text{font-family:Nunito,"Segoe UI",Arial,sans-serif;fill:#0b2533}
.px{font-family:Pixel,monospace}.lbl{font-family:Pixel,monospace;font-size:9px;fill:#55707d}
.num{font-weight:800;font-size:30px}.sub{font-weight:700;font-size:13px;fill:#7d8a93}</style>`)
	b.WriteString(`<defs><linearGradient id="bg" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#f3fbfe"/><stop offset="1" stop-color="#dff3fa"/></linearGradient>
<linearGradient id="xp" x1="0" x2="1"><stop offset="0" stop-color="#00ADD8"/><stop offset="1" stop-color="#5DE0B5"/></linearGradient></defs>`)
	fmt.Fprintf(&b, `<rect x="1" y="1" width="%g" height="%g" rx="18" fill="url(#bg)" stroke="#00ADD8" stroke-opacity=".5" stroke-width="2"/>`, W-2, H-2)
	// header
	b.WriteString(`<path d="M30 30l12 8-12 8z" fill="#00ADD8"/><text x="52" y="46" class="px" style="font-size:16px">PLAYER STATS</text>`)
	fmt.Fprintf(&b, `<rect x="604" y="24" width="64" height="30" rx="7" fill="#0b2533"/><text x="614" y="45" class="px" style="font-size:12px;fill:#FDDD00">LV%d</text>`, lv)
	frac := float64(s.Contributions%100) / 100
	fmt.Fprintf(&b, `<rect x="680" y="33" width="172" height="12" rx="6" fill="#0b2533" fill-opacity=".1"/><rect x="680" y="33" width="%.1f" height="12" rx="6" fill="url(#xp)"/>`, 172*frac)
	// tiles
	const g, tw, th = 14.0, 266.0, 100.0
	x0, y0 := 28.0, 70.0
	y1 := y0 + th + g
	tile(&b, x0, y0, tw, th, "bolt", "#00ADD8", "CONTRIBUTIONS", fmt.Sprint(s.Contributions), "last 12 months")
	tile(&b, x0+tw+g, y0, tw, th, "flame", "#ff7b39", "STREAK", fmt.Sprintf("%d days", s.Streak), fmt.Sprintf("best %d days", s.Longest))
	tile(&b, x0+2*(tw+g), y0, tw, th, "star", "#e0a800", "HACKATHON WINS", fmt.Sprint(c.Wins), "2nd · 3rd · 2nd")
	// leetcode: wide tile with difficulty bars
	tile(&b, x0, y1, 2*tw+g, th, "code", "#ffa116", "LEETCODE", fmt.Sprint(solved), "solved")
	bars := []struct {
		name  string
		n     int
		color string
	}{{"EASY", s.LeetCode.Easy, "#1cbaba"}, {"MED", s.LeetCode.Medium, "#ffb700"}, {"HARD", s.LeetCode.Hard, "#f63737"}}
	maxN := max(s.LeetCode.Easy, s.LeetCode.Medium, s.LeetCode.Hard, 1)
	for i, br := range bars {
		by := y1 + 24 + float64(i)*24
		bx := x0 + 200
		fmt.Fprintf(&b, `<text x="%g" y="%g" class="lbl">%s</text>`, bx, by+9, br.name)
		fmt.Fprintf(&b, `<rect x="%g" y="%g" width="220" height="10" rx="5" fill="#0b2533" fill-opacity=".08"/>`, bx+46, by)
		fmt.Fprintf(&b, `<rect x="%g" y="%g" width="%.1f" height="10" rx="5" fill="%s"/>`, bx+46, by, 220*float64(br.n)/float64(maxN), br.color)
		fmt.Fprintf(&b, `<text x="%g" y="%g" style="font-weight:800;font-size:13px">%d</text>`, bx+274, by+10, br.n)
	}
	tile(&b, x0+2*(tw+g), y1, tw, th, "block", "#0089ad", "PROJECTS", fmt.Sprint(c.Projects), "shipped")
	fmt.Fprintf(&b, `<text x="28" y="%g" class="sub" style="font-size:11px">auto-synced · %s</text>`, H-14, time.Now().UTC().Format("02 Jan 2006"))
	b.WriteString(`</svg>`)
	return b.String()
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
	s := cfg.Fallback
	if tot, cur, best, err := contributions(cfg.User); err != nil {
		log.Printf("contributions: using fallback (%v)", err)
	} else {
		s.Contributions, s.Streak, s.Longest = tot, cur, best
	}
	if lc, err := leetcode(cfg.LeetCode); err != nil {
		log.Printf("leetcode: using fallback (%v)", err)
	} else {
		s.LeetCode = lc
	}

	prs, err := mergedPRs(cfg.User, cfg.Exclude)
	if err != nil || len(prs) == 0 {
		log.Printf("merged PRs: using fallback (%v)", err)
		prs = cfg.OSS
	}
	if err := updateOSS(writeCards(cfg, prs)); err != nil {
		log.Fatal(err)
	}
	sd := map[string]any{"contributions": s.Contributions, "streak": s.Streak, "best": s.Longest, "wins": cfg.Wins, "level": level(s, cfg),
		"lc": map[string]int{"easy": s.LeetCode.Easy, "medium": s.LeetCode.Medium, "hard": s.LeetCode.Hard}}
	sj, _ := json.MarshalIndent(sd, "", "  ")
	if err := os.WriteFile("assets/stats-data.json", append(sj, '\n'), 0o644); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("assets/stats.svg", []byte(statsSVG(cfg, s)), 0o644); err != nil {
		log.Fatal(err)
	}
	log.Printf("ok: %d contributions, %d-day streak, %d leetcode solved", s.Contributions, s.Streak, s.LeetCode.Easy+s.LeetCode.Medium+s.LeetCode.Hard)
}

// ---------- merged open-source PRs ----------

type PR struct{ Repo, URL, Title string }

func mergedPRs(user string, exclude []string) ([]PR, error) {
	q := fmt.Sprintf("type:pr author:%s -user:%s", user, user)
	req, _ := http.NewRequest("GET", "https://api.github.com/search/issues?sort=updated&order=desc&per_page=30&q="+url.QueryEscape(q), nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	if t := os.Getenv("GITHUB_TOKEN"); t != "" {
		req.Header.Set("Authorization", "Bearer "+t)
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("github search: %s", res.Status)
	}
	var out struct {
		Items []struct {
			Title         string `json:"title"`
			Number        int    `json:"number"`
			HTMLURL       string `json:"html_url"`
			RepositoryURL string `json:"repository_url"`
			State         string `json:"state"`
			PullRequest   struct {
				MergedAt *string `json:"merged_at"`
			} `json:"pull_request"`
		} `json:"items"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	skip := map[string]bool{}
	for _, r := range exclude {
		skip[strings.ToLower(r)] = true
	}
	var prs []PR
	for _, it := range out.Items {
		repo := strings.TrimPrefix(it.RepositoryURL, "https://api.github.com/repos/")
		if skip[strings.ToLower(repo)] || (it.State == "closed" && it.PullRequest.MergedAt == nil) {
			continue // closed without merging: leave it out
		}
		prs = append(prs, PR{fmt.Sprintf("%s#%d", repo, it.Number), it.HTMLURL, it.Title})
	}
	return prs, nil
}

// updateOSS rewrites the block between the OSS markers in README.md.
func updateOSS(block string) error {
	raw, err := os.ReadFile("README.md")
	if err != nil {
		return err
	}
	const start, end = "<!--OSS:START-->", "<!--OSS:END-->"
	s := string(raw)
	i, j := strings.Index(s, start), strings.Index(s, end)
	if i < 0 || j < i {
		return nil
	}
	var b strings.Builder
	b.WriteString(start + "\n")
	b.WriteString(block)
	b.WriteString(end)
	return os.WriteFile("README.md", []byte(s[:i]+b.String()+s[j+len(end):]), 0o644)
}
