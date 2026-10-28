package main

import (
	"embed"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type BlogPost struct {
	Title       string
	Date        string
	Summary     string
	Slug        string
	ReadingTime int
}

// Post bodies live in content/blog/<slug>.md.
//
//go:embed content/blog/*.md
var blogContent embed.FS

var blogPosts = []BlogPost{
	{
		Title:       "What Is Amazon Aurora and Why Are There So Many?",
		Date:        "2026-08-22",
		Summary:     "A comparison of classic Amazon Aurora, Aurora Limitless, and Aurora DSQL, focusing on their architecture, scaling, sharding, and consistency.",
		Slug:        "what-is-amazon-aurora",
		ReadingTime: 12,
	},
	{
		Title:       "How I Rebuilt React From Scratch",
		Date:        "2026-02-10",
		Summary:     "Rebuilding React for fun",
		Slug:        "building-react-from-scratch",
		ReadingTime: 35,
	},
}

func (p BlogPost) Body() string {
	b, err := blogContent.ReadFile("content/blog/" + p.Slug + ".md")
	if err != nil {
		return "Could not load this post: " + err.Error()
	}
	return string(b)
}

// Read-only vim-style viewer state for a blog post.
type blogReader struct {
	post       int
	blocks     []blogBlock
	links      []string
	lines      []blogLine
	width      int
	cy, cx     int
	want       int // desired column for vertical motions, -1 means end of line
	top        int
	count      string
	pending    string
	cmdMode    rune // 0, ':', '/' or '?'
	cmdBuf     string
	search     string
	searchBack bool
	findOp     rune
	findCh     rune
	message    string
}

const (
	blogGutter    = 5
	blogScrollOff = 3
)

var (
	blogCursorStyle = lipgloss.NewStyle().
			Background(tokyoFg).
			Foreground(lipgloss.Color("#2E3440"))

	blogMatchStyle = lipgloss.NewStyle().
			Background(tokyoBlue).
			Foreground(lipgloss.Color("#2E3440"))

	blogQuoteStyle = lipgloss.NewStyle().
			Foreground(fujiGray).
			Italic(true)

	blogStatusStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#3B4252")).
			Foreground(tokyoFg)

	blogModeStyle = lipgloss.NewStyle().
			Background(tokyoBlue).
			Foreground(lipgloss.Color("#2E3440")).
			Bold(true)

	blogLineNrStyle = lipgloss.NewStyle().
			Foreground(tokyoMuted)

	blogCurLineNrStyle = lipgloss.NewStyle().
				Foreground(tokyoFgAlt).
				Bold(true)
)

func (m model) blogLayout() (colWidth, textWidth, viewHeight int) {
	w := max(m.width, 20)
	h := max(m.height, 8)
	colWidth = min(w-2, 88)
	textWidth = max(10, colWidth-blogGutter)
	// Title, separator, status line and command line.
	viewHeight = max(1, h-4)
	return colWidth, textWidth, viewHeight
}

func (m *model) openBlogPost(i int) {
	blocks, links := parseBlogMarkdown(blogPosts[i].Slug, blogPosts[i].Body())
	m.reader = blogReader{post: i, blocks: blocks, links: links}
	_, textW, viewH := m.blogLayout()
	m.reader.rewrap(textW)
	m.reader.scrollToCursor(viewH)
	m.currentPage = blogPostPage
}

func (r *blogReader) rewrap(width int) {
	if width == r.width && r.lines != nil {
		return
	}
	src, off := 0, 0
	if len(r.lines) > 0 {
		l := r.lines[r.cy]
		src, off = l.src, l.srcOff+max(0, r.cx-l.pad)
	}
	r.width = width
	r.lines = wrapBlogBlocks(r.blocks, width)
	r.cy, r.cx = 0, 0
	for i, l := range r.lines {
		if l.src > src {
			break
		}
		if l.src == src && l.srcOff <= off {
			r.cy = i
			r.cx = off - l.srcOff + l.pad
		}
	}
	r.cx = min(r.cx, r.maxCol(r.cy))
	r.want = r.cx
}

func clamp(v, lo, hi int) int {
	return max(lo, min(v, hi))
}

func (r *blogReader) last() int { return len(r.lines) - 1 }

func (r *blogReader) maxCol(y int) int { return max(0, len(r.lines[y].text)-1) }

func (r *blogReader) isBlank(y int) bool { return strings.TrimSpace(string(r.lines[y].text)) == "" }

func (r *blogReader) setCol(x int) {
	r.cx = clamp(x, 0, r.maxCol(r.cy))
	r.want = r.cx
}

func (r *blogReader) applyWant() {
	if r.want < 0 {
		r.cx = r.maxCol(r.cy)
	} else {
		r.cx = min(r.want, r.maxCol(r.cy))
	}
}

func (r *blogReader) moveLines(n int) {
	r.cy = clamp(r.cy+n, 0, r.last())
	r.applyWant()
}

func (r *blogReader) firstNonBlank() {
	x := 0
	for x < len(r.lines[r.cy].text) && unicode.IsSpace(r.lines[r.cy].text[x]) {
		x++
	}
	r.setCol(x)
}

func (r *blogReader) gotoLine(y int) {
	r.cy = clamp(y, 0, r.last())
	r.firstNonBlank()
}

func scrollOff(viewH int) int {
	return min(blogScrollOff, (viewH-1)/2)
}

func (r *blogReader) scrollToCursor(viewH int) {
	so := scrollOff(viewH)
	if r.cy-so < r.top {
		r.top = max(0, r.cy-so)
	}
	if r.cy+so > r.top+viewH-1 {
		r.top = min(r.cy+so-viewH+1, max(0, len(r.lines)-viewH))
		if r.cy > r.top+viewH-1 {
			r.top = r.cy - viewH + 1
		}
	}
	r.top = clamp(r.top, 0, max(0, r.last()))
}

// Moves the cursor back inside the viewport after the view itself was scrolled.
func (r *blogReader) keepCursorInView(viewH int) {
	so := scrollOff(viewH)
	lo := r.top
	if r.top > 0 {
		lo += so
	}
	hi := r.top + viewH - 1
	if hi < r.last() {
		hi -= so
	}
	hi = min(hi, r.last())
	r.cy = clamp(r.cy, min(lo, hi), hi)
	r.applyWant()
}

func (r *blogReader) scrollView(n, viewH int) {
	r.top = clamp(r.top+n, 0, max(0, r.last()))
	r.keepCursorInView(viewH)
}

func runeClass(c rune, bigWord bool) int {
	if unicode.IsSpace(c) {
		return 0
	}
	if bigWord || c == '_' || unicode.IsLetter(c) || unicode.IsDigit(c) {
		return 2
	}
	return 1
}

func (r *blogReader) wordForward(big bool) {
	y, x := r.cy, r.cx
	line := r.lines[y].text
	if x < len(line) {
		if cls := runeClass(line[x], big); cls != 0 {
			for x < len(line) && runeClass(line[x], big) == cls {
				x++
			}
		}
	}
	for {
		line = r.lines[y].text
		for x < len(line) && runeClass(line[x], big) == 0 {
			x++
		}
		if x < len(line) {
			break
		}
		if y == r.last() {
			x = max(0, len(line)-1)
			break
		}
		y++
		x = 0
		// An empty line counts as a word.
		if len(r.lines[y].text) == 0 {
			break
		}
	}
	r.cy = y
	r.setCol(x)
}

func (r *blogReader) wordEnd(big bool) {
	y, x := r.cy, r.cx+1
	for {
		line := r.lines[y].text
		for x < len(line) && runeClass(line[x], big) == 0 {
			x++
		}
		if x < len(line) {
			break
		}
		if y == r.last() {
			r.cy = y
			r.setCol(len(line) - 1)
			return
		}
		y++
		x = 0
	}
	line := r.lines[y].text
	cls := runeClass(line[x], big)
	for x+1 < len(line) && runeClass(line[x+1], big) == cls {
		x++
	}
	r.cy = y
	r.setCol(x)
}

func (r *blogReader) wordBackward(big bool) {
	y, x := r.cy, r.cx-1
	for {
		line := r.lines[y].text
		x = min(x, len(line)-1)
		for x >= 0 && runeClass(line[x], big) == 0 {
			x--
		}
		if x >= 0 {
			break
		}
		if y == 0 {
			r.cy = 0
			r.setCol(0)
			return
		}
		y--
		x = len(r.lines[y].text) - 1
		if x < 0 {
			r.cy = y
			r.setCol(0)
			return
		}
	}
	line := r.lines[y].text
	cls := runeClass(line[x], big)
	for x > 0 && runeClass(line[x-1], big) == cls {
		x--
	}
	r.cy = y
	r.setCol(x)
}

func (r *blogReader) paragraphForward() {
	y := r.cy + 1
	for y < r.last() && r.isBlank(y) {
		y++
	}
	for y < r.last() && !r.isBlank(y) {
		y++
	}
	r.cy = clamp(y, 0, r.last())
	if r.cy == r.last() && !r.isBlank(r.cy) {
		r.setCol(r.maxCol(r.cy))
	} else {
		r.setCol(0)
	}
}

func (r *blogReader) paragraphBackward() {
	y := r.cy - 1
	for y > 0 && r.isBlank(y) {
		y--
	}
	for y > 0 && !r.isBlank(y) {
		y--
	}
	r.cy = clamp(y, 0, r.last())
	r.setCol(0)
}

// f, F, t and T within the current line.
func (r *blogReader) findChar(op, ch rune, count int) {
	line := r.lines[r.cy].text
	x := r.cx
	for i := 0; i < count; i++ {
		found := -1
		switch op {
		case 'f', 't':
			start := x + 1
			if op == 't' {
				start = x + 2
			}
			for j := start; j < len(line); j++ {
				if line[j] == ch {
					found = j
					break
				}
			}
			if found >= 0 && op == 't' {
				found--
			}
		case 'F', 'T':
			start := x - 1
			if op == 'T' {
				start = x - 2
			}
			for j := start; j >= 0; j-- {
				if line[j] == ch {
					found = j
					break
				}
			}
			if found >= 0 && op == 'T' {
				found++
			}
		}
		if found < 0 {
			return
		}
		x = found
	}
	r.setCol(x)
}

// Smartcase: case-insensitive unless the pattern contains an uppercase letter.
func findAll(text []rune, pattern string) []int {
	pat := []rune(pattern)
	if len(pat) == 0 || len(pat) > len(text) {
		return nil
	}
	fold := strings.ToLower(pattern) == pattern
	var cols []int
	for i := 0; i+len(pat) <= len(text); i++ {
		ok := true
		for j, p := range pat {
			c := text[i+j]
			if fold {
				c = unicode.ToLower(c)
			}
			if c != p {
				ok = false
				break
			}
		}
		if ok {
			cols = append(cols, i)
		}
	}
	return cols
}

func (r *blogReader) searchNext(backward bool, count int) {
	if r.search == "" {
		r.message = "E35: No previous regular expression"
		return
	}
	for c := 0; c < count; c++ {
		if !r.searchOnce(backward) {
			r.message = "E486: Pattern not found: " + r.search
			return
		}
	}
}

func (r *blogReader) searchOnce(backward bool) bool {
	n := len(r.lines)
	for i := 0; i <= n; i++ {
		y := r.cy + i
		if backward {
			y = r.cy - i
		}
		wrapped := y >= n || y < 0
		y = ((y % n) + n) % n
		cols := findAll(r.lines[y].text, r.search)
		if backward {
			for k := len(cols) - 1; k >= 0; k-- {
				c := cols[k]
				if (i == 0 && c >= r.cx) || (i == n && c < r.cx) {
					continue
				}
				r.jumpToMatch(y, c, wrapped, "search hit TOP, continuing at BOTTOM")
				return true
			}
		} else {
			for _, c := range cols {
				if (i == 0 && c <= r.cx) || (i == n && c > r.cx) {
					continue
				}
				r.jumpToMatch(y, c, wrapped, "search hit BOTTOM, continuing at TOP")
				return true
			}
		}
	}
	return false
}

func (r *blogReader) jumpToMatch(y, x int, wrapped bool, wrapMsg string) {
	r.cy = y
	r.setCol(x)
	if wrapped {
		r.message = wrapMsg
	} else {
		prefix := "/"
		if r.searchBack {
			prefix = "?"
		}
		r.message = prefix + r.search
	}
}

func (r *blogReader) takeCount() (int, bool) {
	if r.count == "" {
		return 1, false
	}
	n, err := strconv.Atoi(r.count)
	r.count = ""
	if err != nil || n <= 0 {
		return 1, false
	}
	return min(n, 99999), true
}

// Returns true when the command asks to close the post.
func (r *blogReader) runCommand(mode rune, cmd string) bool {
	switch mode {
	case '/', '?':
		if cmd != "" {
			r.search = cmd
		}
		r.searchBack = mode == '?'
		r.searchNext(r.searchBack, 1)
		return false
	}

	cmd = strings.TrimSpace(cmd)
	switch cmd {
	case "":
	case "q", "q!", "wq", "wq!", "x", "qa", "qa!":
		return true
	case "w", "w!":
		r.message = "E45: 'readonly' option is set (add ! to override)"
	case "noh", "nohlsearch":
		r.search = ""
	case "h", "help":
		r.message = "hjkl w b e W B E 0 ^ $ gg G { } H M L f F t T ; , / ? n N ^d ^u ^f ^b ^e ^y zz zt zb :{n} :q"
	default:
		if n, err := strconv.Atoi(cmd); err == nil {
			r.gotoLine(n - 1)
		} else {
			r.message = "E492: Not an editor command: " + cmd
		}
	}
	return false
}

func (m model) closeBlogPost() (tea.Model, tea.Cmd) {
	m.currentPage = blogPage
	m.reader = blogReader{}
	return m, nil
}

func (m model) updateBlogPost(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	r := &m.reader
	_, _, viewH := m.blogLayout()
	key := msg.String()

	if key == "ctrl+c" {
		m.reader = blogReader{}
		m.currentPage = menuPage
		return m, tickCmd()
	}

	if r.cmdMode != 0 {
		switch msg.Type {
		case tea.KeyEsc:
			r.cmdMode, r.cmdBuf = 0, ""
		case tea.KeyEnter:
			mode, cmd := r.cmdMode, r.cmdBuf
			r.cmdMode, r.cmdBuf = 0, ""
			if r.runCommand(mode, cmd) {
				return m.closeBlogPost()
			}
			r.scrollToCursor(viewH)
		case tea.KeyBackspace:
			if r.cmdBuf == "" {
				r.cmdMode = 0
			} else {
				buf := []rune(r.cmdBuf)
				r.cmdBuf = string(buf[:len(buf)-1])
			}
		case tea.KeySpace:
			r.cmdBuf += " "
		case tea.KeyRunes:
			r.cmdBuf += string(msg.Runes)
		}
		return m, nil
	}

	r.message = ""

	if r.pending != "" {
		p := r.pending
		r.pending = ""
		cnt, hasCount := r.takeCount()
		switch p {
		case "g":
			if key == "g" {
				if hasCount {
					r.gotoLine(cnt - 1)
				} else {
					r.gotoLine(0)
				}
			}
		case "z":
			switch key {
			case "z", ".":
				r.top = max(0, r.cy-viewH/2)
			case "t", "enter":
				r.top = max(0, r.cy-scrollOff(viewH))
			case "b", "-":
				r.top = max(0, r.cy-viewH+1+scrollOff(viewH))
			}
		case "Z":
			if key == "Z" || key == "Q" {
				return m.closeBlogPost()
			}
		case "f", "F", "t", "T":
			if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
				r.findOp, r.findCh = rune(p[0]), msg.Runes[0]
				r.findChar(r.findOp, r.findCh, cnt)
			} else if msg.Type == tea.KeySpace {
				r.findOp, r.findCh = rune(p[0]), ' '
				r.findChar(r.findOp, r.findCh, cnt)
			}
		}
		r.scrollToCursor(viewH)
		return m, nil
	}

	if len(key) == 1 && key[0] >= '0' && key[0] <= '9' && !(key == "0" && r.count == "") {
		r.count += key
		return m, nil
	}

	if key == "esc" {
		r.count = ""
		return m, nil
	}

	cnt, hasCount := r.takeCount()

	switch key {
	case "q":
		return m.closeBlogPost()
	case "h", "left", "backspace":
		r.setCol(r.cx - cnt)
	case "l", "right", " ":
		r.setCol(r.cx + cnt)
	case "j", "down", "enter", "+", "ctrl+n":
		r.moveLines(cnt)
		if key == "enter" || key == "+" {
			r.firstNonBlank()
		}
	case "k", "up", "-", "ctrl+p":
		r.moveLines(-cnt)
		if key == "-" {
			r.firstNonBlank()
		}
	case "w", "W":
		for i := 0; i < cnt; i++ {
			r.wordForward(key == "W")
		}
	case "e", "E":
		for i := 0; i < cnt; i++ {
			r.wordEnd(key == "E")
		}
	case "b", "B":
		for i := 0; i < cnt; i++ {
			r.wordBackward(key == "B")
		}
	case "0", "home":
		r.setCol(0)
	case "^", "_":
		r.firstNonBlank()
	case "$", "end":
		r.moveLines(cnt - 1)
		r.cx = r.maxCol(r.cy)
		r.want = -1
	case "G":
		if hasCount {
			r.gotoLine(cnt - 1)
		} else {
			r.gotoLine(r.last())
		}
	case "}":
		for i := 0; i < cnt; i++ {
			r.paragraphForward()
		}
	case "{":
		for i := 0; i < cnt; i++ {
			r.paragraphBackward()
		}
	case "H":
		off := cnt - 1
		if r.top > 0 {
			off = max(off, scrollOff(viewH))
		}
		r.cy = min(r.top+off, r.last())
		r.firstNonBlank()
	case "L":
		bottom := min(r.top+viewH-1, r.last())
		off := cnt - 1
		if bottom < r.last() {
			off = max(off, scrollOff(viewH))
		}
		r.cy = max(bottom-off, r.top)
		r.firstNonBlank()
	case "M":
		r.cy = r.top + (min(r.top+viewH, len(r.lines))-r.top-1)/2
		r.firstNonBlank()
	case "ctrl+d", "ctrl+u":
		n := max(1, viewH/2)
		if key == "ctrl+u" {
			n = -n
		}
		r.top = clamp(r.top+n, 0, max(0, len(r.lines)-viewH))
		r.moveLines(n)
	case "ctrl+f", "pgdown":
		r.scrollView(cnt*max(1, viewH-2), viewH)
	case "ctrl+b", "pgup":
		r.scrollView(-cnt*max(1, viewH-2), viewH)
	case "ctrl+e":
		r.scrollView(cnt, viewH)
	case "ctrl+y":
		r.scrollView(-cnt, viewH)
	case "n":
		r.searchNext(r.searchBack, cnt)
	case "N":
		r.searchNext(!r.searchBack, cnt)
	case "*", "#":
		r.searchWordUnderCursor(key == "#")
	case ";", ",":
		if r.findOp != 0 {
			op := r.findOp
			if key == "," {
				op = map[rune]rune{'f': 'F', 'F': 'f', 't': 'T', 'T': 't'}[op]
			}
			r.findChar(op, r.findCh, cnt)
		}
	case "g", "z", "Z", "f", "F", "t", "T":
		r.pending = key
		if hasCount {
			r.count = strconv.Itoa(cnt)
		}
		return m, nil
	case ":", "/", "?":
		r.cmdMode = rune(key[0])
		r.cmdBuf = ""
		return m, nil
	case "i", "I", "a", "A", "o", "O", "x", "X", "d", "D", "c", "C", "s", "S", "r", "R", "p", "P", "u", "~", ">", "<", "J", "ctrl+r":
		r.message = "E21: Cannot make changes, 'modifiable' is off"
	}

	r.scrollToCursor(viewH)
	return m, nil
}

func (r *blogReader) searchWordUnderCursor(backward bool) {
	line := r.lines[r.cy].text
	if len(line) == 0 || runeClass(line[r.cx], false) != 2 {
		return
	}
	start, end := r.cx, r.cx
	for start > 0 && runeClass(line[start-1], false) == 2 {
		start--
	}
	for end < len(line) && runeClass(line[end], false) == 2 {
		end++
	}
	r.search = strings.ToLower(string(line[start:end]))
	r.searchBack = backward
	r.cx = start
	r.searchNext(backward, 1)
}

func (m model) handleBlogMouse(msg tea.MouseMsg) model {
	if msg.Action != tea.MouseActionPress {
		return m
	}
	_, _, viewH := m.blogLayout()
	switch msg.Button {
	case tea.MouseButtonWheelDown:
		m.reader.scrollView(3, viewH)
	case tea.MouseButtonWheelUp:
		m.reader.scrollView(-3, viewH)
	}
	return m
}

func (m model) renderBlog() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("━━━ Blog ━━━"))
	b.WriteString("\n\n")

	for i, p := range blogPosts {
		cursor := "  "
		if m.blogCursor == i {
			cursor = "→ "
		}

		name := cursor + p.Title
		if m.blogCursor == i {
			b.WriteString(projectNameStyle.Render(name))
		} else {
			b.WriteString(menuStyle.Render(name))
		}
		b.WriteString("\n")
		b.WriteString("    ")
		b.WriteString(periodStyle.Render(fmt.Sprintf("%s • %d min read", p.Date, p.ReadingTime)))
		b.WriteString("\n")

		if m.blogCursor == i {
			summaryWidth := max(10, min(m.width-4, 70)-4)
			b.WriteString(subtleStyle.PaddingLeft(4).Width(summaryWidth).Render(p.Summary))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("↑/↓: browse • enter: read • esc: back to menu"))

	return b.String()
}

func blogRuneStyle(kind blogLineKind, a blogAttr) lipgloss.Style {
	s := contentStyle
	switch kind {
	case blogHeading:
		s = projectNameStyle
	case blogCode:
		s = techStyle
	case blogFence:
		s = subtleStyle
	case blogQuote, blogNote:
		s = blogQuoteStyle
	}
	if a&attrBold != 0 {
		s = s.Bold(true)
	}
	if a&attrItalic != 0 {
		s = s.Italic(true)
	}
	if a&attrCode != 0 {
		s = s.Foreground(surimiOrange)
	}
	if a&attrLink != 0 {
		s = s.Foreground(waveBlue).Underline(true)
	}
	if a&attrNote != 0 {
		s = s.Foreground(fujiGray)
	}
	return s
}

func (m model) renderBlogLine(y int) string {
	r := m.reader
	line := r.lines[y]

	cursorX := -1
	if y == r.cy {
		cursorX = r.cx
	}
	if len(line.text) == 0 {
		if cursorX == 0 {
			return blogCursorStyle.Render(" ")
		}
		return ""
	}

	marks := make([]bool, len(line.text))
	patLen := len([]rune(r.search))
	for _, c := range findAll(line.text, r.search) {
		for k := c; k < c+patLen && k < len(marks); k++ {
			marks[k] = true
		}
	}

	// Runes are grouped into runs that share a style and link target.
	const (
		matchKey  = 1 << 8
		cursorKey = 1 << 9
		hrefShift = 10
	)
	keyAt := func(i int) int {
		k := int(line.attrs[i])
		switch {
		case i == cursorX:
			k = cursorKey
		case marks[i]:
			k = matchKey
		}
		return k | line.hrefs[i]<<hrefShift
	}
	render := func(k int, text string) string {
		var s string
		switch style := k & (1<<hrefShift - 1); style {
		case cursorKey:
			s = blogCursorStyle.Render(text)
		case matchKey:
			s = blogMatchStyle.Render(text)
		default:
			s = blogRuneStyle(line.kind, blogAttr(style)).Render(text)
		}
		if href := k >> hrefShift; href > 0 && href <= len(r.links) {
			s = clickableLink(s, r.links[href-1])
		}
		return s
	}

	var b strings.Builder
	start := 0
	for i := 1; i <= len(line.text); i++ {
		if i == len(line.text) || keyAt(i) != keyAt(start) {
			b.WriteString(render(keyAt(start), string(line.text[start:i])))
			start = i
		}
	}
	return b.String()
}

func (m model) renderBlogPost() string {
	w := max(m.width, 20)
	h := max(m.height, 8)
	colW, _, viewH := m.blogLayout()
	r := m.reader
	post := blogPosts[r.post]
	pad := strings.Repeat(" ", max(0, (w-colW)/2))

	rows := make([]string, 0, h)
	title := []rune(post.Title)
	if maxTitle := colW - len(post.Date) - 2; len(title) > maxTitle {
		title = append(title[:max(0, maxTitle-1)], '…')
	}
	rows = append(rows, pad+projectNameStyle.Render(string(title))+"  "+periodStyle.Render(post.Date))
	rows = append(rows, pad+subtleStyle.Render(strings.Repeat("─", colW)))

	for i := 0; i < viewH; i++ {
		y := r.top + i
		if y >= len(r.lines) {
			rows = append(rows, pad+subtleStyle.Render("~"))
			continue
		}
		var gutter string
		if y == r.cy {
			gutter = blogCurLineNrStyle.Render(fmt.Sprintf("%-4d", y+1)) + " "
		} else {
			gutter = blogLineNrStyle.Render(fmt.Sprintf("%4d", max(y-r.cy, r.cy-y))) + " "
		}
		rows = append(rows, pad+gutter+m.renderBlogLine(y))
	}

	var pos string
	switch {
	case len(r.lines) <= viewH:
		pos = "All"
	case r.top == 0:
		pos = "Top"
	case r.top+viewH >= len(r.lines):
		pos = "Bot"
	default:
		pos = fmt.Sprintf("%d%%", r.top*100/(len(r.lines)-viewH))
	}
	mode := " NORMAL "
	if r.cmdMode != 0 {
		mode = " COMMAND "
	}
	left := blogModeStyle.Render(mode) + blogStatusStyle.Render(" "+post.Slug+".md [RO] ")
	right := blogStatusStyle.Render(fmt.Sprintf(" %d:%d  %s ", r.cy+1, r.cx+1, pos))
	gap := max(0, colW-lipgloss.Width(left)-lipgloss.Width(right))
	rows = append(rows, pad+left+blogStatusStyle.Render(strings.Repeat(" ", gap))+right)

	var cmdLine string
	switch {
	case r.cmdMode != 0:
		cmdLine = contentStyle.Render(string(r.cmdMode)+r.cmdBuf) + blogCursorStyle.Render(" ")
	case r.message != "":
		cmdLine = contentStyle.Render(r.message)
	default:
		hint := subtleStyle.Render("hjkl/w/b move • ^d/^u page • / search • :help • q back")
		showCmd := contentStyle.Render(r.count + r.pending)
		if lipgloss.Width(hint)+lipgloss.Width(showCmd)+1 > colW {
			hint = subtleStyle.Render(":help • q back")
		}
		cmdLine = hint + strings.Repeat(" ", max(1, colW-lipgloss.Width(hint)-lipgloss.Width(showCmd))) + showCmd
	}
	rows = append(rows, pad+cmdLine)

	return strings.Join(rows, "\n")
}
