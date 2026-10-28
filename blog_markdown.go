package main

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const siteURL = "https://www.joesluis.com"

type blogLineKind int

const (
	blogText blogLineKind = iota
	blogHeading
	blogCode
	blogFence
	blogQuote
	blogNote
)

// Inline styling for a single rune.
type blogAttr uint8

const (
	attrBold blogAttr = 1 << iota
	attrItalic
	attrCode
	attrLink
	attrNote
)

// A rendered markdown block (paragraph, heading, list item, code line...) before wrapping.
type blogBlock struct {
	text   []rune
	attrs  []blogAttr
	hrefs  []int // footnote number each rune links to, 0 for none
	kind   blogLineKind
	indent int // hanging indent for wrapped continuation lines
}

var (
	mdHeadingRe   = regexp.MustCompile(`^#{1,6}\s+`)
	mdListRe      = regexp.MustCompile(`^(\s*)([-*]|\d+\.)\s+`)
	mdDiagramRe   = regexp.MustCompile(`^\[\[([A-Z_]+)_ANIMATION\]\]$`)
	mdImageLineRe = regexp.MustCompile(`^!\[[^\]]*\]\([^)]*\)$`)
	mdBreakRe     = regexp.MustCompile(`(?i)<br\s*/?>`)
	mdTableSepRe  = regexp.MustCompile(`^:?-+:?$`)
)

type blogParser struct {
	slug   string
	blocks []blogBlock
	para   []string
	links  []string
}

// Converts a post's markdown into terminal-friendly blocks. Links become numbered
// footnotes, and images/diagrams become placeholders pointing at the web version.
// The returned links are indexed by footnote number minus one.
func parseBlogMarkdown(slug, md string) ([]blogBlock, []string) {
	p := &blogParser{slug: slug}
	lines := strings.Split(md, "\n")
	inCode := false
	for i := 0; i < len(lines); i++ {
		line := strings.TrimRight(strings.ReplaceAll(lines[i], "\t", "    "), " \r")
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			p.flush()
			inCode = !inCode
			fence := "```" + strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
			p.add(blogBlock{text: []rune(fence), kind: blogFence})
			continue
		}
		if inCode {
			p.add(blogBlock{text: []rune(line), kind: blogCode})
			continue
		}
		if strings.HasPrefix(trimmed, "|") {
			p.flush()
			var rows []string
			for ; i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "|"); i++ {
				rows = append(rows, strings.TrimSpace(lines[i]))
			}
			i--
			p.table(rows)
			continue
		}
		for j, part := range mdBreakRe.Split(line, -1) {
			if j > 0 {
				p.blank()
			}
			p.line(part)
		}
	}
	p.flush()

	if len(p.links) > 0 {
		p.blank()
		p.add(p.inlineBlock("## Links", blogHeading, 0))
		p.blank()
		for i, link := range p.links {
			// A non-breaking space keeps the URL on the same line as its number.
			num := fmt.Sprintf("[%d]\u00a0", i+1)
			b := blogBlock{kind: blogText, indent: utf8.RuneCountInString(num)}
			b.push(num, attrNote)
			b.pushLink(link, attrLink, i+1)
			p.add(b)
		}
	}

	for len(p.blocks) > 0 && len(p.blocks[len(p.blocks)-1].text) == 0 {
		p.blocks = p.blocks[:len(p.blocks)-1]
	}
	return p.blocks, p.links
}

func (b *blogBlock) push(s string, a blogAttr) {
	b.pushLink(s, a, 0)
}

func (b *blogBlock) pushLink(s string, a blogAttr, href int) {
	for _, c := range s {
		b.text = append(b.text, c)
		b.attrs = append(b.attrs, a)
		b.hrefs = append(b.hrefs, href)
	}
}

func (b *blogBlock) pushBlock(o blogBlock) {
	b.text = append(b.text, o.text...)
	b.attrs = append(b.attrs, o.attrs...)
	b.hrefs = append(b.hrefs, o.hrefs...)
}

func (p *blogParser) add(b blogBlock) {
	if len(b.attrs) < len(b.text) {
		b.attrs = append(b.attrs, make([]blogAttr, len(b.text)-len(b.attrs))...)
	}
	if len(b.hrefs) < len(b.text) {
		b.hrefs = append(b.hrefs, make([]int, len(b.text)-len(b.hrefs))...)
	}
	p.blocks = append(p.blocks, b)
}

// Ends the current paragraph and adds a single blank line.
func (p *blogParser) blank() {
	p.flush()
	if n := len(p.blocks); n > 0 && len(p.blocks[n-1].text) > 0 {
		p.add(blogBlock{})
	}
}

// Markdown soft line breaks are joined into one paragraph, like on the web.
func (p *blogParser) flush() {
	if len(p.para) == 0 {
		return
	}
	p.add(p.inlineBlock(strings.Join(p.para, " "), blogText, 0))
	p.para = nil
}

func (p *blogParser) line(s string) {
	trimmed := strings.TrimSpace(s)
	switch {
	case trimmed == "":
		p.blank()
	case mdDiagramRe.MatchString(trimmed):
		p.flush()
		name := strings.ToLower(strings.ReplaceAll(mdDiagramRe.FindStringSubmatch(trimmed)[1], "_", " "))
		md := fmt.Sprintf("▶ animated diagram (%s), see the [web version](%s/blog/%s)", name, siteURL, p.slug)
		p.add(p.inlineBlock(md, blogNote, 2))
	case mdImageLineRe.MatchString(trimmed):
		p.flush()
		p.add(p.inlineBlock(trimmed, blogNote, 0))
	case mdHeadingRe.MatchString(trimmed):
		p.flush()
		kind := blogHeading
		if !strings.ContainsFunc(mdHeadingRe.ReplaceAllString(trimmed, ""), func(r rune) bool {
			return unicode.IsLetter(r) || unicode.IsDigit(r)
		}) {
			// Headings used purely as decoration (e.g. a big quote mark).
			kind = blogQuote
			trimmed = mdHeadingRe.ReplaceAllString(trimmed, "")
		}
		p.add(p.inlineBlock(trimmed, kind, 0))
	case mdListRe.MatchString(s):
		p.flush()
		marker := mdListRe.FindString(s)
		b := blogBlock{kind: blogText, indent: utf8.RuneCountInString(marker)}
		b.push(marker, 0)
		b.pushBlock(p.inlineBlock(s[len(marker):], blogText, 0))
		p.add(b)
	case strings.HasPrefix(trimmed, ">"):
		p.flush()
		b := blogBlock{kind: blogQuote, indent: 2}
		b.push("│ ", attrNote)
		b.pushBlock(p.inlineBlock(strings.TrimSpace(strings.TrimPrefix(trimmed, ">")), blogQuote, 0))
		p.add(b)
	default:
		p.para = append(p.para, trimmed)
	}
}

// Tables are too wide for most terminals, so each row becomes a small list.
func (p *blogParser) table(rows []string) {
	var cells [][]string
	for _, row := range rows {
		parts := strings.Split(strings.Trim(row, "|"), "|")
		sep := true
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
			if !mdTableSepRe.MatchString(parts[i]) {
				sep = false
			}
		}
		if !sep {
			cells = append(cells, parts)
		}
	}
	if len(cells) < 2 {
		return
	}
	header := cells[0]
	for _, row := range cells[1:] {
		p.blank()
		p.add(p.inlineBlockAttr(row[0], blogText, 0, attrBold))
		for c := 1; c < len(row) && c < len(header); c++ {
			b := blogBlock{kind: blogText, indent: 4}
			b.push("  - ", 0)
			b.pushBlock(p.inlineBlockAttr(header[c]+":", blogText, 0, attrBold))
			b.push(" ", 0)
			b.pushBlock(p.inlineBlock(row[c], blogText, 0))
			p.add(b)
		}
	}
	p.blank()
}

func (p *blogParser) inlineBlock(s string, kind blogLineKind, indent int) blogBlock {
	return p.inlineBlockAttr(s, kind, indent, 0)
}

func (p *blogParser) inlineBlockAttr(s string, kind blogLineKind, indent int, base blogAttr) blogBlock {
	b := blogBlock{kind: kind, indent: indent}
	p.inline(&b, s, base)
	return b
}

func (p *blogParser) footnote(url string) int {
	if strings.HasPrefix(url, "/") {
		url = siteURL + url
	}
	for i, l := range p.links {
		if l == url {
			return i + 1
		}
	}
	p.links = append(p.links, url)
	return len(p.links)
}

// Parses `code`, **bold**, *italic*, [links](url) and ![images](src).
func (p *blogParser) inline(b *blogBlock, s string, base blogAttr) {
	for i := 0; i < len(s); {
		rest := s[i:]
		switch {
		case rest[0] == '`':
			if end := strings.IndexByte(rest[1:], '`'); end >= 0 {
				b.push(rest[1:1+end], base|attrCode)
				i += end + 2
				continue
			}
		case strings.HasPrefix(rest, "!["):
			if alt, url, n, ok := parseMdLink(rest[1:]); ok {
				if alt == "" {
					alt = "image"
				}
				num := p.footnote(url)
				b.pushLink(fmt.Sprintf("[image: %s][%d]", alt, num), base|attrNote, num)
				i += n + 1
				continue
			}
		case rest[0] == '[':
			if label, url, n, ok := parseMdLink(rest); ok {
				start := len(b.text)
				p.inline(b, label, base|attrLink)
				num := p.footnote(url)
				for k := start; k < len(b.text); k++ {
					b.hrefs[k] = num
				}
				b.pushLink(fmt.Sprintf("[%d]", num), base|attrNote, num)
				i += n
				continue
			}
		case strings.HasPrefix(rest, "**"):
			if end := strings.Index(rest[2:], "**"); end > 0 {
				p.inline(b, rest[2:2+end], base|attrBold)
				i += end + 4
				continue
			}
		case rest[0] == '*':
			if len(rest) > 1 && rest[1] != ' ' {
				if end := strings.IndexByte(rest[1:], '*'); end > 0 && rest[end] != ' ' {
					p.inline(b, rest[1:1+end], base|attrItalic)
					i += end + 2
					continue
				}
			}
		}
		r, size := utf8.DecodeRuneInString(rest)
		b.push(string(r), base)
		i += size
	}
}

// Parses "[label](url)" at the start of s, returning the consumed byte count.
func parseMdLink(s string) (label, url string, n int, ok bool) {
	mid := strings.Index(s, "](")
	if !strings.HasPrefix(s, "[") || mid < 0 || strings.Contains(s[:mid], "\n") {
		return "", "", 0, false
	}
	end := strings.IndexByte(s[mid+2:], ')')
	if end < 0 {
		return "", "", 0, false
	}
	return s[1:mid], strings.TrimSpace(s[mid+2 : mid+2+end]), mid + 3 + end, true
}

// A single display line after word wrapping.
type blogLine struct {
	text   []rune
	attrs  []blogAttr
	hrefs  []int
	kind   blogLineKind
	src    int // index of the block this came from
	srcOff int // rune offset into the block
	pad    int // leading spaces added for a hanging indent
}

func wrapBlogBlocks(blocks []blogBlock, width int) []blogLine {
	width = max(width, 1)
	var out []blogLine
	for i, b := range blocks {
		if len(b.text) == 0 {
			out = append(out, blogLine{kind: b.kind, src: i})
			continue
		}
		start := 0
		for start < len(b.text) {
			pad := 0
			if start > 0 {
				pad = min(b.indent, width/2)
			}
			avail := width - pad
			end, next := len(b.text), len(b.text)
			if end-start > avail {
				end, next = start+avail, start+avail
				// Break at the last space that fits and drop it from the display line.
				for s := start + avail; s > start; s-- {
					if b.text[s] == ' ' {
						end, next = s, s+1
						break
					}
				}
			}
			text := append([]rune(strings.Repeat(" ", pad)), b.text[start:end]...)
			attrs := append(make([]blogAttr, pad), b.attrs[start:end]...)
			hrefs := append(make([]int, pad), b.hrefs[start:end]...)
			out = append(out, blogLine{text: text, attrs: attrs, hrefs: hrefs, kind: b.kind, src: i, srcOff: start, pad: pad})
			start = next
		}
	}
	return out
}
