package main

// Light-blue "template" cards: every README section is a matching SVG card, so the
// whole profile reads as one light-blue page (GitHub doesn't allow page colours).

import (
	"fmt"
	"os"
	"strings"
)

type Project struct {
	Name string   `json:"name"`
	URL  string   `json:"url"`
	Desc string   `json:"desc"`
	Tags []string `json:"tags"`
	Slug string   `json:"slug"`
}

type Win struct {
	Place string `json:"place"` // gold | silver | bronze
	Event string `json:"event"`
	Note  string `json:"note"`
}

const (
	cardBG     = "#e4ecf8"
	cardBorder = "#7d8a9e"
	ink        = "#14234a"
	inkSoft    = "#4a5b7a"
	accent     = "#b07d14"
)

func svgOpen(b *strings.Builder, w, h float64, label string) {
	fmt.Fprintf(b, `<svg xmlns="http://www.w3.org/2000/svg" width="%g" height="%g" viewBox="0 0 %g %g" role="img" aria-label="%s">`, w, h, w, h, esc(label))
	b.WriteString(`<style>`)
	b.WriteString(font("Pixel", "assets/fonts/pixel.woff2", 400))
	b.WriteString(font("Nunito", "assets/fonts/nunito-800.woff2", 800))
	b.WriteString(font("Nunito", "assets/fonts/nunito-700.woff2", 700))
	b.WriteString(`text{font-family:Nunito,"Segoe UI",Arial,sans-serif;fill:` + ink + `}.px{font-family:Pixel,monospace}</style>`)
}

// pixel-style frame: soft drop, blue border, light fill, notched corners
func frame(b *strings.Builder, x, y, w, h float64, fill string) {
	fmt.Fprintf(b, `<rect x="%g" y="%g" width="%g" height="%g" fill="#c3cfe0"/>`, x+6, y+6, w-6, h-6)
	fmt.Fprintf(b, `<rect x="%g" y="%g" width="%g" height="%g" fill="%s"/>`, x, y, w-6, h-6, cardBorder)
	fmt.Fprintf(b, `<rect x="%g" y="%g" width="%g" height="%g" fill="%s"/>`, x+4, y+4, w-14, h-14, fill)
	for _, c := range [][2]float64{{x, y}, {x + w - 10, y}, {x, y + h - 10}, {x + w - 10, y + h - 10}} {
		fmt.Fprintf(b, `<rect x="%g" y="%g" width="4" height="4" fill="#c3cfe0"/>`, c[0], c[1])
	}
}

func sparkle(b *strings.Builder, x, y float64, c string) {
	fmt.Fprintf(b, `<rect x="%g" y="%g" width="4" height="12" fill="%s"/><rect x="%g" y="%g" width="12" height="4" fill="%s"/>`, x+4, y, c, x, y+4, c)
}

func headerSVG(title string) string {
	var b strings.Builder
	svgOpen(&b, 1320, 64, title)
	frame(&b, 0, 2, 1320, 60, "#dde6f4")
	b.WriteString(`<rect x="22" y="20" width="14" height="14" fill="#b07d14"/><rect x="26" y="24" width="6" height="6" fill="#7d8a9e"/>`)
	fmt.Fprintf(&b, `<text x="50" y="38" class="px" style="font-size:20px">%s</text>`, esc(strings.ToUpper(title)))
	sparkle(&b, 1250, 16, "#7d8a9e")
	sparkle(&b, 1276, 32, "#e8c25a")
	b.WriteString(`</svg>`)
	return b.String()
}

func introSVG(line1, line2 string) string {
	var b strings.Builder
	svgOpen(&b, 1320, 100, line1)
	frame(&b, 0, 0, 1320, 100, "#f6f8fc")
	fmt.Fprintf(&b, `<text x="657" y="44" text-anchor="middle" style="font-weight:800;font-size:28px">%s</text>`, esc(line1))
	fmt.Fprintf(&b, `<text x="657" y="76" text-anchor="middle" style="font-weight:700;font-size:21px;fill:#4a5b7a">%s</text>`, esc(line2))
	sparkle(&b, 30, 24, "#7d8a9e")
	sparkle(&b, 1270, 62, "#7d8a9e")
	b.WriteString(`</svg>`)
	return b.String()
}

func wrap(s string, n int) []string {
	var lines []string
	cur := ""
	for _, w := range strings.Fields(s) {
		if len(cur)+len(w)+1 > n && cur != "" {
			lines = append(lines, cur)
			cur = w
		} else if cur == "" {
			cur = w
		} else {
			cur += " " + w
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

func projectSVG(p Project) string {
	const W, H = 440.0, 300.0
	lines := wrap(p.Desc, 30)
	if len(lines) > 4 {
		lines = append(lines[:3], lines[3]+"…")
	}
	var b strings.Builder
	svgOpen(&b, W, H, p.Name)
	frame(&b, 0, 0, W, H, "#f6f8fc")
	fmt.Fprintf(&b, `<rect x="4" y="4" width="%g" height="8" fill="#e4ecf8"/>`, W-14)
	fmt.Fprintf(&b, `<text x="26" y="56" style="font-weight:800;font-size:32px">%s</text>`, esc(p.Name))
	for i, l := range lines {
		fmt.Fprintf(&b, `<text x="26" y="%g" style="font-weight:700;font-size:20px;fill:#4a5b7a">%s</text>`, 96+float64(i)*28, esc(l))
	}
	x := 26.0
	for _, t := range p.Tags {
		w := float64(len([]rune(t)))*10 + 24
		if x+w > W-30 {
			break
		}
		fmt.Fprintf(&b, `<rect x="%g" y="236" width="%g" height="32" rx="16" fill="#e4ecf8"/><text x="%g" y="258" style="font-weight:800;font-size:16px;fill:#b07d14">%s</text>`, x, w, x+12, esc(t))
		x += w + 8
	}
	b.WriteString(`</svg>`)
	return b.String()
}

// RepoGroup is every PR to one repo; PRs[0] is the most recently updated.
type RepoGroup struct {
	Repo   string // owner/name
	PRs    []PR
	Merged int
}

// groupPRs folds PRs into one entry per repo: featured repos first (in the given order),
// then by number of PRs, then by most recent activity. At most max groups.
func groupPRs(prs []PR, featured []string, max int) (shown []RepoGroup, total int) {
	idx := map[string]int{}
	var groups []RepoGroup
	for _, p := range prs {
		repo := p.Repo
		if i := strings.LastIndex(repo, "#"); i >= 0 {
			repo = repo[:i]
		}
		k := strings.ToLower(repo)
		if _, ok := idx[k]; !ok {
			idx[k] = len(groups)
			groups = append(groups, RepoGroup{Repo: repo})
		}
		g := &groups[idx[k]]
		g.PRs = append(g.PRs, p)
		if p.Status != "open" {
			g.Merged++
		}
	}
	rank := map[string]int{}
	for i, f := range featured {
		rank[strings.ToLower(f)] = i + 1
	}
	order := make([]int, len(groups)) // original index = recency order
	for i := range order {
		order[i] = i
	}
	less := func(a, b int) bool {
		ra, rb := rank[strings.ToLower(groups[a].Repo)], rank[strings.ToLower(groups[b].Repo)]
		switch {
		case ra != 0 && rb != 0:
			return ra < rb
		case ra != 0 || rb != 0:
			return ra != 0
		case len(groups[a].PRs) != len(groups[b].PRs):
			return len(groups[a].PRs) > len(groups[b].PRs)
		}
		return a < b
	}
	for i := 1; i < len(order); i++ { // insertion sort: tiny list, stable
		for j := i; j > 0 && less(order[j], order[j-1]); j-- {
			order[j], order[j-1] = order[j-1], order[j]
		}
	}
	for _, i := range order {
		if len(shown) == max {
			break
		}
		shown = append(shown, groups[i])
	}
	return shown, len(groups)
}

func repoLabel(repo string) string {
	if i := strings.Index(repo, "/"); i >= 0 && repo[:i] != "golang" {
		return repo[i+1:]
	}
	return repo
}

func repoSVG(g RepoGroup) string {
	const W, H = 440.0, 200.0
	latest := g.PRs[0]
	lines := wrap(latest.Title, 32)
	if len(lines) > 2 {
		lines = append(lines[:1], lines[1]+"…")
	}
	var b strings.Builder
	svgOpen(&b, W, H, g.Repo)
	frame(&b, 0, 0, W, H, "#f6f8fc")
	b.WriteString(`<rect x="24" y="24" width="28" height="28" rx="6" fill="#b07d14"/><path d="M31 32h6v12h-6zM43 32l6 6-6 6" fill="none" stroke="#f6f8fc" stroke-width="3"/>`)
	fmt.Fprintf(&b, `<text x="64" y="47" style="font-weight:800;font-size:25px">%s</text>`, esc(repoLabel(g.Repo)))
	// status pill: "3 PRs · 3 merged" / "merged" / "in review"
	pill, color := "merged", "#2e8b57"
	if n := len(g.PRs); n > 1 {
		pill = fmt.Sprintf("%d PRs · %d merged", n, g.Merged)
		if g.Merged == 0 {
			pill, color = fmt.Sprintf("%d PRs · in review", n), accent
		}
	} else if latest.Status == "open" {
		pill, color = "in review", accent
	}
	pw := float64(len([]rune(pill)))*9.5 + 24
	fmt.Fprintf(&b, `<rect x="26" y="66" width="%g" height="28" rx="14" fill="#e4ecf8"/><text x="38" y="86" style="font-weight:800;font-size:15px;fill:%s">%s</text>`, pw, color, esc(pill))
	for i, l := range lines {
		fmt.Fprintf(&b, `<text x="26" y="%g" style="font-weight:700;font-size:20px;fill:#4a5b7a">%s</text>`, 128+float64(i)*28, esc(l))
	}
	b.WriteString(`</svg>`)
	return b.String()
}

func seeAllSVG(shown, total int) string {
	var b strings.Builder
	label := "See all contributions →"
	if total > shown {
		label = fmt.Sprintf("See all contributions · +%d more repos →", total-shown)
	}
	svgOpen(&b, 1320, 64, label)
	frame(&b, 0, 2, 1320, 60, "#f6f8fc")
	fmt.Fprintf(&b, `<text x="657" y="40" text-anchor="middle" class="px" style="font-size:16px;fill:%s">%s</text>`, accent, esc(strings.ToUpper(label)))
	b.WriteString(`</svg>`)
	return b.String()
}

func winsSVG(wins []Win) string {
	const W, H = 1320.0, 150.0
	var b strings.Builder
	svgOpen(&b, W, H, "Hackathon wins")
	frame(&b, 0, 0, W, H, "#f6f8fc")
	medal := map[string][2]string{"gold": {"#f5c542", "#b8860b"}, "silver": {"#cfd8dc", "#8d9aa1"}, "bronze": {"#e0a370", "#9a5b2c"}}
	colW := (W - 20) / float64(max(len(wins), 1))
	for i, w := range wins {
		x := 10 + float64(i)*colW
		m := medal[w.Place]
		fmt.Fprintf(&b, `<circle cx="%g" cy="72" r="26" fill="%s" stroke="%s" stroke-width="4"/>`, x+50, m[0], m[1])
		fmt.Fprintf(&b, `<text x="%g" y="79" text-anchor="middle" class="px" style="font-size:16px;fill:%s">%s</text>`, x+50, m[1], map[string]string{"gold": "1", "silver": "2", "bronze": "3"}[w.Place])
		ev := w.Event
		if len(ev) > 22 {
			ev = strings.Split(ev, " · ")[0]
		}
		fmt.Fprintf(&b, `<text x="%g" y="66" style="font-weight:800;font-size:25px">%s</text>`, x+92, esc(ev))
		fmt.Fprintf(&b, `<text x="%g" y="96" style="font-weight:700;font-size:19px;fill:#4a5b7a">%s</text>`, x+92, esc(w.Note))
		if i > 0 {
			fmt.Fprintf(&b, `<rect x="%g" y="30" width="2" height="90" fill="#e4ecf8"/>`, x-4)
		}
	}
	b.WriteString(`</svg>`)
	return b.String()
}

type StackGroup struct {
	Name  string   `json:"name"`
	Color string   `json:"color"`
	Items []string `json:"items"`
}

// stackSVG lays chips out row by row, wrapping inside each category.
var chipTop, chipBottom, chipRim, chipText = "#f2fbff", "#bfe7fb", "#8fcff2", "#0f4c75"

func stackSVG(groups []StackGroup) string {
	const W, padL, maxX = 880.0, 34.0, 840.0
	type chip struct {
		x, y, w     float64
		text, color string
	}
	var chips []chip
	var titles []struct {
		y    float64
		name string
	}
	y := 40.0
	for _, g := range groups {
		titles = append(titles, struct {
			y    float64
			name string
		}{y + 18, g.Name})
		y += 34
		x := padL
		for _, it := range g.Items {
			w := float64(len([]rune(it)))*11 + 32
			if x+w > maxX {
				x = padL
				y += 48
			}
			chips = append(chips, chip{x, y, w, it, g.Color})
			x += w + 10
		}
		y += 64
	}
	h := y + 6
	var b strings.Builder
	svgOpen(&b, W, h, "Tech stack")
	frame(&b, 0, 0, W, h, "#f6f8fc")
	fmt.Fprintf(&b, `<rect x="4" y="4" width="866" height="10" fill="%s"/>`, cardBG)
	for _, t := range titles {
		fmt.Fprintf(&b, `<text x="%g" y="%g" class="px" style="font-size:14px;fill:%s">%s</text>`, padL, t.y, inkSoft, esc(strings.ToUpper(t.name)))
	}
	// glossy chips: gradient body, white shine on the top half, soft rim
	b.WriteString(`<defs><linearGradient id="chipA" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="` + chipTop + `"/><stop offset="1" stop-color="` + chipBottom + `"/></linearGradient></defs>`)
	for _, c := range chips {
		fmt.Fprintf(&b, `<rect x="%g" y="%g" width="%g" height="38" rx="19" fill="url(#chipA)" stroke="%s" stroke-width="1.5"/>`, c.x, c.y, c.w, chipRim)
		fmt.Fprintf(&b, `<rect x="%g" y="%g" width="%g" height="15" rx="7.5" fill="#f6f8fc" fill-opacity=".38"/>`, c.x+6, c.y+3, c.w-12)
		fmt.Fprintf(&b, `<text x="%g" y="%g" style="font-weight:800;font-size:19px;fill:%s">%s</text>`, c.x+16, c.y+26, chipText, esc(c.text))
	}
	b.WriteString(`</svg>`)
	return b.String()
}

func writeFile(path, s string) {
	if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
		panic(err)
	}
}

// writeCards renders every template card and returns the README blocks for the dynamic parts.
func writeCards(cfg Config, prs []PR) string {
	_ = os.MkdirAll("assets/cards", 0o755)
	for _, h := range []struct{ slug, title string }{{"built", "Things I've built"}, {"oss", "Open source"}, {"wins", "Hackathon wins"}, {"stats", "Stats"}, {"find", "Find me"}} {
		writeFile("assets/cards/h-"+h.slug+".svg", headerSVG(h.title))
	}
	writeFile("assets/cards/intro.svg", introSVG("2nd-year CSE @ NMAMIT", "building backends in Go and smart contracts in Solidity"))
	for _, p := range cfg.ProjectList {
		writeFile("assets/cards/p-"+p.Slug+".svg", projectSVG(p))
	}
	writeFile("assets/cards/wins.svg", winsSVG(cfg.Wins2))
	writeFile("assets/cards/h-stack.svg", headerSVG("Tech stack"))
	// remove stale PR cards, then write current ones
	if old, err := os.ReadDir("assets/cards"); err == nil {
		for _, f := range old {
			if strings.HasPrefix(f.Name(), "pr-") {
				_ = os.Remove("assets/cards/" + f.Name())
			}
		}
	}
	max := cfg.OSSMax
	if max <= 0 {
		max = 6
	}
	groups, total := groupPRs(prs, cfg.OSSFeatured, max)
	var md strings.Builder
	for i, g := range groups {
		name := fmt.Sprintf("pr-%d.svg", i)
		writeFile("assets/cards/"+name, repoSVG(g))
		link := g.PRs[0].URL
		if len(g.PRs) > 1 { // several PRs: link to all of mine in that repo
			link = fmt.Sprintf("https://github.com/%s/pulls?q=is%%3Apr+author%%3A%s", g.Repo, cfg.User)
		}
		fmt.Fprintf(&md, "<a href=\"%s\"><img src=\"./assets/cards/%s\" width=\"32%%\" alt=\"%s\"/></a>\n", link, name, esc(fmt.Sprintf("%s · %d PR(s) · %s", g.Repo, len(g.PRs), g.PRs[0].Title)))
	}
	writeFile("assets/cards/pr-all.svg", seeAllSVG(len(groups), total))
	all := fmt.Sprintf("https://github.com/search?q=is%%3Apr+author%%3A%s+-user%%3A%s&type=pullrequests&s=updated", cfg.User, cfg.User)
	fmt.Fprintf(&md, "<a href=\"%s\"><img src=\"./assets/cards/pr-all.svg\" width=\"100%%\" alt=\"See all my open-source contributions\"/></a>\n", all)
	return md.String()
}
