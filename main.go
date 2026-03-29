package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/muesli/termenv"
)

type tickMsg time.Time

func clickableLink(label, url string) string {
	return "\x1b]8;;" + url + "\x1b\\" + label + "\x1b]8;;\x1b\\"
}

var welcomeScreen = []string{"JOE SLUIS"}

var asciiLogoLines = []string{
	`              __       __           __      `,
	`            /\ \     /\ \         /\ \    `,
	`            \ \ \   /  \ \       /  \ \   `,
	`            /\ \_\ / /\ \ \     / /\ \ \  `,
	`           / /\/_// / /\ \ \   / / /\ \_\ `,
	`  __      / / /  / / /  \ \_\ / /_/_ \/_/ `,
	` /\ \    / / /  / / /   / / // /____/\    `,
	` \ \_\  / / /  / / /   / / // /\____\/    `,
	` / / /_/ / /  / / /___/ / // / /______    `,
	`/ / /__\/ /  / / /____\/ // / /_______\   `,
	`\/_______/   \/_________/ \/__________/   `,
}

func renderGradientLogo(width int, sweepIndex int) string {
	var result strings.Builder

	baseStyle := lipgloss.NewStyle().Foreground(tokyoCyan).Bold(true)
	_ = sweepIndex

	linesToShow := len(asciiLogoLines)

	maxLineLen := 0
	for i := 0; i < linesToShow; i++ {
		if len(asciiLogoLines[i]) > maxLineLen {
			maxLineLen = len(asciiLogoLines[i])
		}
	}

	if linesToShow == 0 || maxLineLen == 0 {
		return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render("")
	}

	for y := 0; y < linesToShow; y++ {
		for _, r := range asciiLogoLines[y] {
			if r == ' ' {
				result.WriteRune(r)
			} else {
				result.WriteString(baseStyle.Render(string(r)))
			}
		}
		if y < linesToShow-1 {
			result.WriteString("\n")
		}
	}

	logoBlock := result.String()
	centered := lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(logoBlock)
	return centered
}

var aboutContent = `
Hey, I'm Joe, a software developer that builds entertaining things but occasionally locks in.

Currently into database internals and distributed systems.
`

var contactContent = `
Feel free to reach out!

  GitHub      %s
  Email       %s
  LinkedIn    %s
`

type Project struct {
	Name string
	Desc string
	Tech string
	Link string
}

type Experience struct {
	Role    string
	Company string
	Period  string
	Desc    string
}

var projects = []Project{
	{
		Name: "SSH Portfolio",
		Desc: "This app",
		Tech: "Go, Bubble Tea, Wish",
		Link: "github.com/joe/ssh-portfolio",
	},
	{
		Name: "React From Scratch",
		Desc: "Built a toy React from scratch",
		Tech: "JavaScript",
		Link: "github.com/joe/react-0.5",
	},
	{
		Name: "HTTP Server From Scratch",
		Desc: "Build a HTTP server from scratch using TCP and HTTP/1.1",
		Tech: "Rust",
		Link: "github.com/joe/api-gateway",
	},
}

var experiences = []Experience{
	{
		Role:    "Software Engineer",
		Company: "Microsoft",
		Period:  "2025 - Present",
		Desc:    "Azure SQL VM team doing a lot of infra stuff",
	},
	{
		Role:    "Software Engineer Intern",
		Company: "Jenni AI",
		Period:  "2024 - 2025",
		Desc:    "Developed new product that reviews manuscripts for Jenni AI",
	},
	{
		Role:    "Software Engineer Intern",
		Company: "Blue Origin",
		Period:  "Fall 2023",
		Desc:    "Worked on the New Glenn rocket",
	},
}

type page int

const (
	splashPage page = iota
	menuPage
	aboutPage
	projectsPage
	experiencePage
	contactPage
)

var menuItems = []string{"About", "Projects", "Experience", "Contact"}

var matrixChars = []rune("ｱｲｳｴｵｶｷｸｹｺｻｼｽｾｿﾀﾁﾂﾃﾄﾅﾆﾇﾈﾉﾊﾋﾌﾍﾎﾏﾐﾑﾒﾓﾔﾕﾖﾗﾘﾙﾚﾛﾜｦﾝｧｨｩｪｫｬｭｮｯｰ<>[]{}()+-*/=$%@#")

// Theme
var (
	// Nord palette
	tokyoFg     = lipgloss.Color("#D8DEE9") // primary text
	tokyoFgAlt  = lipgloss.Color("#E5E9F0") // secondary text
	tokyoMuted  = lipgloss.Color("#4C566A") // muted/help text
	tokyoBlue   = lipgloss.Color("#81A1C1") // title/border
	tokyoCyan   = lipgloss.Color("#88C0D0") // links
	tokyoPurple = lipgloss.Color("#81A1C1") // selected highlight
	tokyoGreen  = lipgloss.Color("#8FBCBB") // tech tags

	// Aliases for compatibility
	oniViolet    = tokyoBlue   // titles/borders
	fujiWhite    = tokyoFg     // main text
	springGreen  = tokyoPurple // selected
	fujiGray     = tokyoMuted  // help text
	waveBlue     = tokyoCyan   // links
	surimiOrange = tokyoGreen  // tech tags
	carpYellow   = tokyoBlue   // project names

	titleStyle = lipgloss.NewStyle().
			Foreground(oniViolet).
			Bold(true).
			MarginBottom(1)

	menuStyle = lipgloss.NewStyle().
			Foreground(tokyoFgAlt)

	selectedStyle = lipgloss.NewStyle().
			Foreground(tokyoFgAlt).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(fujiGray).
			MarginTop(1)

	contentStyle = lipgloss.NewStyle().
			Foreground(fujiWhite)

	accentStyle = lipgloss.NewStyle().
			Foreground(waveBlue).
			Bold(true)

	subtleStyle = lipgloss.NewStyle().
			Foreground(fujiGray)

	projectNameStyle = lipgloss.NewStyle().
				Foreground(carpYellow).
				Bold(true)

	techStyle = lipgloss.NewStyle().
			Foreground(surimiOrange)

	roleStyle = lipgloss.NewStyle().
			Foreground(springGreen).
			Bold(true)

	companyStyle = lipgloss.NewStyle().
			Foreground(waveBlue)

	periodStyle = lipgloss.NewStyle().
			Foreground(fujiGray).
			Italic(true)
)

type model struct {
	currentPage    page
	menuCursor     int
	projectCursor  int
	expCursor      int
	width          int
	height         int
	mouseX         int
	mouseY         int
	mouseActive    bool
	logoSweepIndex int
	matrixFrame    int
	matrixSeed     uint64
	matrixColumns  []matrixColumn
	lifeW          int
	lifeH          int
	lifeGrid       []bool
	lifeNext       []bool
	lifeTick       int
}

type matrixColumn struct {
	active     bool
	head       int
	speed      int
	trail      int
	cooldown   int
	glyphShift int
}

func initialModel() model {
	m := model{
		currentPage:    splashPage,
		menuCursor:     0,
		projectCursor:  0,
		expCursor:      0,
		width:          80,
		height:         24,
		mouseX:         -1,
		mouseY:         -1,
		mouseActive:    false,
		logoSweepIndex: 0,
		matrixFrame:    0,
		matrixSeed:     uint64(time.Now().UnixNano()),
	}
	m.resetLife()
	return m
}

func (m model) Init() tea.Cmd {
	return tickCmd()
}

// Controls
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		switch m.currentPage {
		case splashPage:
			m.matrixFrame++
			m.advanceMatrix()
			return m, tickCmd()
		case menuPage:
			m.logoSweepIndex++
			m.lifeTick++
			if m.lifeTick%2 == 0 {
				m.stepLife()
			}
			if m.lifeTick%28 == 0 {
				m.seedLife(2)
			}
			return m, tickCmd()
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.mouseX < 0 || m.mouseX >= m.width || m.mouseY < 0 || m.mouseY >= m.height {
			m.mouseActive = false
		}
		m.ensureMatrixColumns()
		m.resetLife()
		return m, nil

	case tea.MouseMsg:
		m.mouseX = msg.X
		m.mouseY = msg.Y
		m.mouseActive = m.mouseX >= 0 && m.mouseX < m.width && m.mouseY >= 0 && m.mouseY < m.height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.currentPage == splashPage || m.currentPage == menuPage {
				return m, tea.Quit
			}
			m.currentPage = menuPage
			return m, tickCmd()

		case "esc", "backspace":
			if m.currentPage == splashPage {
				return m, tea.Quit
			}
			if m.currentPage != menuPage {
				m.currentPage = menuPage
			}
			return m, tickCmd()

		case "up", "k":
			switch m.currentPage {
			case menuPage:
				if m.menuCursor > 0 {
					m.menuCursor--
				}
			case projectsPage:
				if m.projectCursor > 0 {
					m.projectCursor--
				}
			case experiencePage:
				if m.expCursor > 0 {
					m.expCursor--
				}
			}
			return m, nil

		case "down", "j":
			switch m.currentPage {
			case menuPage:
				if m.menuCursor < len(menuItems)-1 {
					m.menuCursor++
				}
			case projectsPage:
				if m.projectCursor < len(projects)-1 {
					m.projectCursor++
				}
			case experiencePage:
				if m.expCursor < len(experiences)-1 {
					m.expCursor++
				}
			}
			return m, nil

		case "enter":
			if m.currentPage == splashPage {
				m.currentPage = menuPage
				return m, tickCmd()
			}
			if m.currentPage == menuPage {
				switch m.menuCursor {
				case 0:
					m.currentPage = aboutPage
				case 1:
					m.currentPage = projectsPage
				case 2:
					m.currentPage = experiencePage
				case 3:
					m.currentPage = contactPage
				}
			}
			return m, nil

		case " ":
			if m.currentPage == menuPage {
				switch m.menuCursor {
				case 0:
					m.currentPage = aboutPage
				case 1:
					m.currentPage = projectsPage
				case 2:
					m.currentPage = experiencePage
				case 3:
					m.currentPage = contactPage
				}
			}
			return m, nil
		}
	}
	return m, nil
}

func tickCmd() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) View() string {
	if m.currentPage == splashPage {
		return m.renderSplash()
	}

	var content string

	switch m.currentPage {
	case menuPage:
		return m.renderMenu()
	case aboutPage:
		content = m.renderAbout()
	case projectsPage:
		content = m.renderProjects()
	case experiencePage:
		content = m.renderExperience()
	case contactPage:
		content = m.renderContact()
	}

	boxWidth := min(m.width-4, 70)
	boxedContent := lipgloss.NewStyle().
		Padding(1, 2).
		Width(boxWidth).
		Render(content)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		boxedContent)
}

func (m model) renderSplash() string {
	w := max(m.width, 20)
	h := max(m.height, 8)
	m.ensureMatrixColumns()

	titleText := "J O E   S L U I S"
	prompt := "enter to continue"
	help := "q / esc to quit"

	titleTextW := len(titleText)
	titleTextH := 1
	titleTextX := max(0, (w-titleTextW)/2)
	titleTextY := max(0, h/2)

	promptY := titleTextY + titleTextH + 1
	if promptY > h-3 {
		promptY = h - 3
	}
	helpY := promptY + 2
	promptX := max(0, (w-len(prompt))/2)
	helpX := max(0, (w-len(help))/2)

	// Soften matrix density around the hero copy to improve readability without a hard box.
	focusPadX := 8
	focusPadY := 2
	contentLeft := min(titleTextX, min(promptX, helpX))
	contentRight := max(titleTextX+titleTextW-1, max(promptX+len(prompt)-1, helpX+len(help)-1))
	contentTop := min(titleTextY, min(promptY, helpY))
	contentBottom := max(titleTextY+titleTextH-1, max(promptY+1, helpY))
	focusLeft := max(0, contentLeft-focusPadX)
	focusRight := min(w-1, contentRight+focusPadX)
	focusTop := max(0, contentTop-focusPadY)
	focusBottom := min(h-1, contentBottom+focusPadY)
	feather := 6
	influenceLeft := max(0, focusLeft-feather)
	influenceRight := min(w-1, focusRight+feather)
	influenceTop := max(0, focusTop-feather)
	influenceBottom := min(h-1, focusBottom+feather)
	titleTextStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F8FAFC")).
		Bold(true)
	promptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F1F5F9")).
		Bold(true)
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#BFD7EA")).
		Bold(true)

	var b strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			insideTitleText := y == titleTextY && x >= titleTextX && x < titleTextX+len(titleText)
			if insideTitleText {
				ch := string(titleText[x-titleTextX])
				if ch == " " {
					b.WriteByte(' ')
				} else {
					b.WriteString(titleTextStyle.Render(ch))
				}
				continue
			}
			insidePromptBox := (y == promptY || y == promptY+1) &&
				x >= promptX && x < promptX+len(prompt)
			if insidePromptBox {
				if y == promptY {
					ch := string(prompt[x-promptX])
					b.WriteString(promptStyle.Render(ch))
				} else {
					b.WriteByte(' ')
				}
				continue
			}
			if y == helpY && x >= helpX && x < helpX+len(help) {
				ch := string(help[x-helpX])
				b.WriteString(helpStyle.Render(ch))
				continue
			}
			if x >= influenceLeft && x <= influenceRight && y >= influenceTop && y <= influenceBottom {
				dx := 0
				if x < focusLeft {
					dx = focusLeft - x
				} else if x > focusRight {
					dx = x - focusRight
				}
				dy := 0
				if y < focusTop {
					dy = focusTop - y
				} else if y > focusBottom {
					dy = y - focusBottom
				}

				dist := max(dx, dy)
				suppressPct := 82 - dist*11
				if suppressPct > 0 {
					noise := matrixHash(x, y, m.matrixFrame/5, int(m.matrixSeed>>16)) % 100
					if noise < suppressPct {
						b.WriteByte(' ')
						continue
					}
				}
			}

			if m.mouseActive {
				dx := float64(x - m.mouseX)
				dy := float64(y-m.mouseY) * 2.0
				distSq := dx*dx + dy*dy

				const mouseBlackRadius = 10.0
				if distSq <= mouseBlackRadius*mouseBlackRadius {
					b.WriteString(m.matrixCellBlack(x, y, w, h))
					continue
				}
			}

			b.WriteString(m.matrixCell(x, y, w, h))
		}
		if y < h-1 {
			b.WriteByte('\n')
		}
	}

	return b.String()
}

func (m model) matrixCell(x, y, w, h int) string {
	if h <= 0 || w <= 0 {
		return " "
	}

	if x < 0 || x >= len(m.matrixColumns) {
		return " "
	}
	col := m.matrixColumns[x]

	if !col.active || y > col.head || col.head-y >= col.trail {
		// Hash-based sparse glyphs avoid visible diagonal banding patterns.
		noise := matrixHash(x, y, m.matrixFrame/20, int(m.matrixSeed&0xffff))
		if noise%1000 < 4 {
			ch := string(matrixChars[noise%len(matrixChars)])
			switch noise % 3 {
			case 0:
				return lipgloss.NewStyle().Foreground(tokyoMuted).Render(ch)
			case 1:
				return lipgloss.NewStyle().Foreground(tokyoFgAlt).Render(ch)
			default:
				return lipgloss.NewStyle().Foreground(tokyoMuted).Render(ch)
			}
		}
		return " "
	}

	dist := col.head - y
	noise := matrixHash(x, y, m.matrixFrame/3, col.glyphShift)
	ch := string(matrixChars[noise%len(matrixChars)])

	switch {
	case dist == 0:
		return lipgloss.NewStyle().Foreground(tokyoCyan).Bold(true).Render(ch)
	case dist < 3:
		return lipgloss.NewStyle().Foreground(tokyoBlue).Render(ch)
	case dist < 7:
		return lipgloss.NewStyle().Foreground(tokyoFgAlt).Render(ch)
	default:
		return lipgloss.NewStyle().Foreground(tokyoMuted).Render(ch)
	}
}

func (m model) matrixCellBlack(x, y, w, h int) string {
	if h <= 0 || w <= 0 {
		return " "
	}

	if x < 0 || x >= len(m.matrixColumns) {
		return " "
	}
	col := m.matrixColumns[x]

	if !col.active || y > col.head || col.head-y >= col.trail {
		noise := matrixHash(x, y, m.matrixFrame/20, int(m.matrixSeed&0xffff))
		if noise%1000 < 4 {
			ch := string(matrixChars[noise%len(matrixChars)])
			return lipgloss.NewStyle().Foreground(lipgloss.Color("#000000")).Render(ch)
		}
		return " "
	}

	noise := matrixHash(x, y, m.matrixFrame/3, col.glyphShift)
	ch := string(matrixChars[noise%len(matrixChars)])
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#000000")).Bold(true).Render(ch)
}

func matrixHash(x, y, frame, salt int) int {
	v := uint32(x*73856093) ^ uint32(y*19349663) ^ uint32(frame*83492791) ^ uint32(salt*2654435761)
	v ^= v >> 13
	v *= 1274126177
	v ^= v >> 16
	return int(v & 0x7fffffff)
}

func (m *model) ensureMatrixColumns() {
	w := max(m.width, 20)
	if w <= 0 {
		m.matrixColumns = nil
		return
	}
	if len(m.matrixColumns) == w {
		return
	}

	prev := m.matrixColumns
	next := make([]matrixColumn, w)
	copy(next, prev)
	for i := len(prev); i < w; i++ {
		next[i] = m.newMatrixColumn()
	}
	m.matrixColumns = next
}

func (m *model) advanceMatrix() {
	m.ensureMatrixColumns()
	h := max(m.height, 8)

	for i := range m.matrixColumns {
		col := m.matrixColumns[i]
		if col.active {
			col.head += col.speed
			// Stream fully left the viewport.
			if col.head-col.trail > h {
				col.active = false
				col.cooldown = 1 + m.randN(max(8, h/3))
			}
			m.matrixColumns[i] = col
			continue
		}

		if col.cooldown > 0 {
			col.cooldown--
			m.matrixColumns[i] = col
			continue
		}

		// Randomized starts with lower frequency for a calmer rain effect.
		if m.randN(100) < 12 {
			col.active = true
			col.head = -m.randN(max(3, h/2))
			col.speed = 1 + m.randN(2)
			col.trail = 8 + m.randN(max(8, h/2))
			col.glyphShift = m.randN(len(matrixChars))
		} else {
			col.cooldown = 2 + m.randN(10)
		}
		m.matrixColumns[i] = col
	}
}

func (m *model) newMatrixColumn() matrixColumn {
	h := max(m.height, 8)
	col := matrixColumn{
		active:     m.randN(100) < 35,
		head:       -m.randN(max(3, h/2)),
		speed:      1 + m.randN(2),
		trail:      8 + m.randN(max(8, h/2)),
		cooldown:   2 + m.randN(max(10, h/2)),
		glyphShift: m.randN(len(matrixChars)),
	}
	if !col.active {
		col.head = -1
	}
	return col
}

func (m *model) randN(n int) int {
	if n <= 0 {
		return 0
	}
	m.matrixSeed = m.matrixSeed*1664525 + 1013904223
	return int((m.matrixSeed >> 16) % uint64(n))
}

func (m *model) resetLife() {
	w := max(m.width, 20)
	h := max(m.height, 8)
	m.lifeW = w
	m.lifeH = h
	size := w * h
	m.lifeGrid = make([]bool, size)
	m.lifeNext = make([]bool, size)
	m.lifeTick = 0
	m.seedLife(115)
}

func (m *model) seedLife(chancePerThousand int) {
	if m.lifeW <= 0 || m.lifeH <= 0 || len(m.lifeGrid) == 0 {
		return
	}
	for i := range m.lifeGrid {
		if m.randN(1000) < chancePerThousand {
			m.lifeGrid[i] = true
		}
	}
}

func (m *model) stepLife() {
	if m.lifeW <= 0 || m.lifeH <= 0 || len(m.lifeGrid) == 0 || len(m.lifeNext) != len(m.lifeGrid) {
		m.resetLife()
		if len(m.lifeGrid) == 0 {
			return
		}
	}

	w := m.lifeW
	h := m.lifeH
	population := 0

	for y := 0; y < h; y++ {
		up := (y - 1 + h) % h
		down := (y + 1) % h
		for x := 0; x < w; x++ {
			left := (x - 1 + w) % w
			right := (x + 1) % w
			i := y*w + x
			neighbors := 0
			if m.lifeGrid[up*w+left] {
				neighbors++
			}
			if m.lifeGrid[up*w+x] {
				neighbors++
			}
			if m.lifeGrid[up*w+right] {
				neighbors++
			}
			if m.lifeGrid[y*w+left] {
				neighbors++
			}
			if m.lifeGrid[y*w+right] {
				neighbors++
			}
			if m.lifeGrid[down*w+left] {
				neighbors++
			}
			if m.lifeGrid[down*w+x] {
				neighbors++
			}
			if m.lifeGrid[down*w+right] {
				neighbors++
			}

			alive := m.lifeGrid[i]
			nextAlive := neighbors == 3 || (alive && neighbors == 2)
			m.lifeNext[i] = nextAlive
			if nextAlive {
				population++
			}
		}
	}

	m.lifeGrid, m.lifeNext = m.lifeNext, m.lifeGrid

	total := w * h
	if total == 0 {
		return
	}
	if population < total/120 || population > total/3 {
		m.seedLife(8)
	}
}

func (m model) renderMenu() string {
	w := max(m.width, 20)
	h := max(m.height, 8)

	contentWidth := min(60, max(18, w-8))
	if contentWidth > w {
		contentWidth = w
	}

	logoLines := strings.Split(renderGradientLogo(contentWidth, m.logoSweepIndex), "\n")
	contentLines := make([]string, 0, len(logoLines)+len(menuItems)+4)
	contentLines = append(contentLines, "")
	contentLines = append(contentLines, logoLines...)
	contentLines = append(contentLines, "")

	for i, item := range menuItems {
		cursor := "  "
		if m.menuCursor == i {
			cursor = "→ "
		}

		line := cursor + item
		if m.menuCursor == i {
			line = selectedStyle.Render(line)
		} else {
			line = menuStyle.Render(line)
		}
		contentLines = append(contentLines, lipgloss.NewStyle().Width(contentWidth).Align(lipgloss.Center).Render(line))
	}

	contentLines = append(contentLines, "")
	menuHelpLineStyle := lipgloss.NewStyle().Foreground(fujiGray)
	helpLine := menuHelpLineStyle.Render("↑/↓ navigate • enter select • esc/backspace menu • q quit")
	contentLines = append(contentLines, lipgloss.NewStyle().Width(contentWidth).Align(lipgloss.Center).Render(helpLine))

	for i := range contentLines {
		contentLines[i] = lipgloss.NewStyle().Width(contentWidth).Render(contentLines[i])
	}

	contentHeight := len(contentLines)
	contentX := max(0, (w-contentWidth)/2)
	contentY := max(0, (h-contentHeight)/2)
	contentRight := min(w, contentX+contentWidth)
	contentBottom := min(h, contentY+contentHeight)

	var out strings.Builder
	for y := 0; y < h; y++ {
		insideContentRow := y >= contentY && y < contentBottom
		if !insideContentRow {
			for x := 0; x < w; x++ {
				out.WriteString(m.menuLifeCell(x, y))
			}
		} else {
			for x := 0; x < contentX; x++ {
				out.WriteString(m.menuLifeCell(x, y))
			}
			line := contentLines[y-contentY]
			out.WriteString(line)
			for x := contentRight; x < w; x++ {
				out.WriteString(m.menuLifeCell(x, y))
			}
		}

		if y < h-1 {
			out.WriteByte('\n')
		}
	}

	return out.String()
}

func (m model) menuLifeCell(x, y int) string {
	if x < 0 || y < 0 || x >= m.lifeW || y >= m.lifeH {
		return " "
	}
	i := y*m.lifeW + x
	if i < 0 || i >= len(m.lifeGrid) || !m.lifeGrid[i] {
		return " "
	}

	noise := matrixHash(x, y, m.logoSweepIndex/3, int(m.matrixSeed>>12)) % 10
	ch := "."
	if noise < 2 {
		ch = "*"
	}

	if noise < 7 {
		return lipgloss.NewStyle().Foreground(tokyoMuted).Render(ch)
	}
	return lipgloss.NewStyle().Foreground(tokyoBlue).Render(ch)
}

func (m model) renderAbout() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("━━━ About Me ━━━"))
	b.WriteString("\n")
	b.WriteString(contentStyle.Render(aboutContent))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("esc: back to menu"))

	return b.String()
}

func (m model) renderProjects() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("━━━ Projects ━━━"))
	b.WriteString("\n\n")

	for i, p := range projects {
		cursor := "  "
		if m.projectCursor == i {
			cursor = "→ "
		}

		name := cursor + p.Name
		if m.projectCursor == i {
			b.WriteString(projectNameStyle.Render(name))
		} else {
			b.WriteString(menuStyle.Render(name))
		}
		b.WriteString("\n")

		// Expands project section
		if m.projectCursor == i {
			b.WriteString(subtleStyle.Render("    " + p.Desc))
			b.WriteString("\n")
			b.WriteString("    ")
			b.WriteString(techStyle.Render(p.Tech))
			b.WriteString("\n")
			b.WriteString("    ")
			projectURL := p.Link
			if !strings.HasPrefix(projectURL, "http://") && !strings.HasPrefix(projectURL, "https://") {
				projectURL = "https://" + projectURL
			}
			b.WriteString(accentStyle.Render(clickableLink(projectURL, projectURL)))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("↑/↓: browse • esc: back to menu"))

	return b.String()
}

func (m model) renderExperience() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("━━━ Experience ━━━"))
	b.WriteString("\n\n")

	for i, exp := range experiences {
		cursor := "  "
		if m.expCursor == i {
			cursor = "→ "
		}

		line := fmt.Sprintf("%s%s @ %s",
			cursor,
			roleStyle.Render(exp.Role),
			companyStyle.Render(exp.Company))
		b.WriteString(line)
		b.WriteString("\n")

		b.WriteString("    ")
		b.WriteString(periodStyle.Render(exp.Period))
		b.WriteString("\n")

		if m.expCursor == i {
			b.WriteString("    ")
			b.WriteString(contentStyle.Render(exp.Desc))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("↑/↓: browse • esc: back to menu"))

	return b.String()
}

func (m model) renderContact() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("━━━ Contact ━━━"))
	b.WriteString("\n")

	githubURL := "https://github.com/JoeS51"
	mailtoURL := "mailto:joesluis51@gmail.com"
	linkedinURL := "https://linkedin.com/in/joesluis/"

	contact := fmt.Sprintf(
		contactContent,
		clickableLink(githubURL, githubURL),
		clickableLink("joesluis51@gmail.com", mailtoURL),
		clickableLink(linkedinURL, linkedinURL),
	)

	b.WriteString(contentStyle.Render(contact))

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("esc: back to menu"))

	return b.String()
}

func main() {
	// This supposedly fixes the color issue
	lipgloss.SetColorProfile(termenv.ANSI256)

	publicKeyAuth := func(ctx ssh.Context, key ssh.PublicKey) bool {
		return true
	}
	// This lets people see the portfolio even without a public key
	passwordAuth := func(ctx ssh.Context, password string) bool {
		return true
	}

	teaHandler := func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		return initialModel(), []tea.ProgramOption{tea.WithAltScreen(), tea.WithMouseAllMotion()}
	}

	s, err := wish.NewServer(
		wish.WithAddress("0.0.0.0:22"),
		wish.WithHostKeyPath(".ssh/host_ed25519"),
		wish.WithPublicKeyAuth(publicKeyAuth),
		wish.WithPasswordAuth(passwordAuth),
		wish.WithMiddleware(
			bubbletea.Middleware(teaHandler),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Listening on %s\n", s.Addr)
	log.Fatal(s.ListenAndServe())
}
