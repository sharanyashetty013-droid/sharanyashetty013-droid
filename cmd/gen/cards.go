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
	cardBG     = "#1c2b4a"
	cardBorder = "#5d6652"
	ink        = "#fff3c4"
	inkSoft    = "#b8c4d9"
	accent     = "#ffe066"
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
	fmt.Fprintf(b, `<rect x="%g" y="%g" width="%g" height="%g" fill="#0a1226"/>`, x+6, y+6, w-6, h-6)
	fmt.Fprintf(b, `<rect x="%g" y="%g" width="%g" height="%g" fill="%s"/>`, x, y, w-6, h-6, cardBorder)
	fmt.Fprintf(b, `<rect x="%g" y="%g" width="%g" height="%g" fill="%s"/>`, x+4, y+4, w-14, h-14, fill)
	for _, c := range [][2]float64{{x, y}, {x + w - 10, y}, {x, y + h - 10}, {x + w - 10, y + h - 10}} {
		fmt.Fprintf(b, `<rect x="%g" y="%g" width="4" height="4" fill="#0a1226"/>`, c[0], c[1])
	}
}

func sparkle(b *strings.Builder, x, y float64, c string) {
	fmt.Fprintf(b, `<rect x="%g" y="%g" width="4" height="12" fill="%s"/><rect x="%g" y="%g" width="12" height="4" fill="%s"/>`, x+4, y, c, x, y+4, c)
}

func headerSVG(title string) string {
	var b strings.Builder
	svgOpen(&b, 880, 84, title)
	frame(&b, 0, 4, 880, 76, "#1a2b4f")
	b.WriteString(`<rect x="22" y="26" width="16" height="16" fill="#ffe066"/><rect x="26" y="30" width="8" height="8" fill="#5d6652"/>`)
	fmt.Fprintf(&b, `<text x="54" y="50" class="px" style="font-size:22px">%s</text>`, esc(strings.ToUpper(title)))
	sparkle(&b, 800, 22, "#5d6652")
	sparkle(&b, 828, 40, "#c9b98a")
	b.WriteString(`</svg>`)
	return b.String()
}

func introSVG(line1, line2 string) string {
	var b strings.Builder
	svgOpen(&b, 880, 130, line1)
	frame(&b, 0, 0, 880, 130, "#13213d")
	fmt.Fprintf(&b, `<rect x="4" y="4" width="866" height="10" fill="%s"/>`, cardBG)
	fmt.Fprintf(&b, `<text x="437" y="62" text-anchor="middle" style="font-weight:800;font-size:28px">%s</text>`, esc(line1))
	fmt.Fprintf(&b, `<text x="437" y="98" text-anchor="middle" style="font-weight:700;font-size:21px;fill:%s">%s</text>`, inkSoft, esc(line2))
	sparkle(&b, 30, 30, "#5d6652")
	sparkle(&b, 828, 84, "#5d6652")
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
	lines := wrap(p.Desc, 62)
	h := 118 + float64(len(lines))*30 + 52
	var b strings.Builder
	svgOpen(&b, 880, h, p.Name)
	frame(&b, 0, 0, 880, h, "#13213d")
	fmt.Fprintf(&b, `<rect x="4" y="4" width="866" height="10" fill="%s"/>`, cardBG)
	fmt.Fprintf(&b, `<text x="34" y="66" style="font-weight:800;font-size:34px">%s</text>`, esc(p.Name))
	fmt.Fprintf(&b, `<text x="840" y="62" text-anchor="end" class="px" style="font-size:12px;fill:%s">OPEN ></text>`, accent)
	for i, l := range lines {
		fmt.Fprintf(&b, `<text x="34" y="%g" style="font-weight:700;font-size:22px;fill:%s">%s</text>`, 108+float64(i)*30, inkSoft, esc(l))
	}
	x := 34.0
	ty := 108 + float64(len(lines))*30 + 4
	for _, t := range p.Tags {
		w := float64(len([]rune(t)))*10.5 + 28
		fmt.Fprintf(&b, `<rect x="%g" y="%g" width="%g" height="34" rx="17" fill="%s"/><text x="%g" y="%g" style="font-weight:800;font-size:17px;fill:%s">%s</text>`, x, ty, w, cardBG, x+14, ty+23, accent, esc(t))
		x += w + 10
	}
	b.WriteString(`</svg>`)
	return b.String()
}

func prSVG(p PR) string {
	lines := wrap(p.Title, 58)
	if len(lines) > 2 {
		lines = append(lines[:1], lines[1]+"…")
	}
	h := 86 + float64(len(lines))*28
	var b strings.Builder
	svgOpen(&b, 880, h, p.Repo)
	frame(&b, 0, 0, 880, h, "#13213d")
	b.WriteString(`<rect x="26" y="26" width="28" height="28" rx="6" fill="#ffe066"/><path d="M33 34h6v12h-6zM45 34l6 6-6 6" fill="none" stroke="#13213d" stroke-width="3"/>`)
	fmt.Fprintf(&b, `<text x="70" y="50" style="font-weight:800;font-size:26px">%s</text>`, esc(p.Repo))
	for i, l := range lines {
		fmt.Fprintf(&b, `<text x="70" y="%g" style="font-weight:700;font-size:21px;fill:%s">%s</text>`, 84+float64(i)*28, inkSoft, esc(l))
	}
	b.WriteString(`</svg>`)
	return b.String()
}

func winsSVG(wins []Win) string {
	h := 40 + float64(len(wins))*74
	var b strings.Builder
	svgOpen(&b, 880, h, "Hackathon wins")
	frame(&b, 0, 0, 880, h, "#13213d")
	medal := map[string][2]string{"gold": {"#f5c542", "#b8860b"}, "silver": {"#cfd8dc", "#8d9aa1"}, "bronze": {"#e0a370", "#9a5b2c"}}
	for i, w := range wins {
		y := 24 + float64(i)*74
		m := medal[w.Place]
		fmt.Fprintf(&b, `<circle cx="56" cy="%g" r="22" fill="%s" stroke="%s" stroke-width="4"/>`, y+30, m[0], m[1])
		fmt.Fprintf(&b, `<text x="56" y="%g" text-anchor="middle" class="px" style="font-size:14px;fill:%s">%s</text>`, y+36, m[1], map[string]string{"gold": "1", "silver": "2", "bronze": "3"}[w.Place])
		fmt.Fprintf(&b, `<text x="98" y="%g" style="font-weight:800;font-size:26px">%s</text>`, y+28, esc(w.Event))
		fmt.Fprintf(&b, `<text x="98" y="%g" style="font-weight:700;font-size:20px;fill:%s">%s</text>`, y+54, inkSoft, esc(w.Note))
		if i < len(wins)-1 {
			fmt.Fprintf(&b, `<rect x="34" y="%g" width="806" height="2" fill="%s"/>`, y+70, cardBG)
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
	frame(&b, 0, 0, W, h, "#13213d")
	fmt.Fprintf(&b, `<rect x="4" y="4" width="866" height="10" fill="%s"/>`, cardBG)
	for _, t := range titles {
		fmt.Fprintf(&b, `<text x="%g" y="%g" class="px" style="font-size:14px;fill:%s">%s</text>`, padL, t.y, inkSoft, esc(strings.ToUpper(t.name)))
	}
	// glossy chips: gradient body, white shine on the top half, soft rim
	b.WriteString(`<defs><linearGradient id="chipA" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="` + chipTop + `"/><stop offset="1" stop-color="` + chipBottom + `"/></linearGradient></defs>`)
	for _, c := range chips {
		fmt.Fprintf(&b, `<rect x="%g" y="%g" width="%g" height="38" rx="19" fill="url(#chipA)" stroke="%s" stroke-width="1.5"/>`, c.x, c.y, c.w, chipRim)
		fmt.Fprintf(&b, `<rect x="%g" y="%g" width="%g" height="15" rx="7.5" fill="#13213d" fill-opacity=".38"/>`, c.x+6, c.y+3, c.w-12)
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
	var md strings.Builder
	for i, p := range prs {
		name := fmt.Sprintf("pr-%d.svg", i)
		writeFile("assets/cards/"+name, prSVG(p))
		fmt.Fprintf(&md, "<a href=\"%s\"><img src=\"./assets/cards/%s\" width=\"100%%\" alt=\"%s\"/></a>\n", p.URL, name, esc(p.Repo+" · "+p.Title))
	}
	return md.String()
}
