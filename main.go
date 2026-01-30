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

// Animation tick message
type tickMsg time.Time

func clickableLink(label, url string) string {
	return "\x1b]8;;" + url + "\x1b\\" + label + "\x1b]8;;\x1b\\"
}

// ============================================================================
// ASCII Art & Content
// ============================================================================

var asciiLogoLines = []string{
	`             __       __           __      `,
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

	baseStyle := lipgloss.NewStyle().Foreground(oniViolet).Bold(true)
	snakeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9EC5FF")).Bold(true)

	linesToShow := len(asciiLogoLines)

	// Determine the bounding box of the logo.
	maxLineLen := 0
	for i := 0; i < linesToShow; i++ {
		if len(asciiLogoLines[i]) > maxLineLen {
			maxLineLen = len(asciiLogoLines[i])
		}
	}

	if linesToShow == 0 || maxLineLen == 0 {
		return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render("")
	}

	// Add a 1-cell padding around the logo so the snake can run "around" it
	// without overwriting the text.
	pad := 1
	gridW := maxLineLen + pad*2
	gridH := linesToShow + pad*2

	// Build the base grid with the logo centered within the padding.
	baseGrid := make([][]rune, gridH)
	for y := 0; y < gridH; y++ {
		row := make([]rune, gridW)
		for x := 0; x < gridW; x++ {
			row[x] = ' '
		}
		baseGrid[y] = row
	}

	for i := 0; i < linesToShow; i++ {
		lineRunes := []rune(asciiLogoLines[i])
		for j, r := range lineRunes {
			baseGrid[pad+i][pad+j] = r
		}
	}

	// Build the perimeter path (outer border of the padded grid).
	type pt struct{ x, y int }
	path := make([]pt, 0, gridW*2+gridH*2)

	// Top edge (left -> right)
	for x := 0; x < gridW; x++ {
		path = append(path, pt{x: x, y: 0})
	}
	// Right edge (top+1 -> bottom-1)
	for y := 1; y < gridH-1; y++ {
		path = append(path, pt{x: gridW - 1, y: y})
	}
	// Bottom edge (right -> left)
	if gridH > 1 {
		for x := gridW - 1; x >= 0; x-- {
			path = append(path, pt{x: x, y: gridH - 1})
		}
	}
	// Left edge (bottom-1 -> top+1)
	if gridW > 1 {
		for y := gridH - 2; y >= 1; y-- {
			path = append(path, pt{x: 0, y: y})
		}
	}

	pathLen := len(path)
	if pathLen == 0 {
		return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render("")
	}

	// Render a short moving "snake" segment along the perimeter path.
	snakeLen := 14
	if snakeLen > pathLen {
		snakeLen = pathLen
	}
	start := sweepIndex % pathLen

	snakeGrid := make([][]bool, gridH)
	for y := 0; y < gridH; y++ {
		snakeGrid[y] = make([]bool, gridW)
	}
	for i := 0; i < snakeLen; i++ {
		idx := (start + i) % pathLen
		p := path[idx]
		snakeGrid[p.y][p.x] = true
	}

	// Merge the grids into a styled string.
	for y := 0; y < gridH; y++ {
		for x := 0; x < gridW; x++ {
			if snakeGrid[y][x] {
				result.WriteString(snakeStyle.Render("•"))
				continue
			}
			ch := string(baseGrid[y][x])
			if baseGrid[y][x] == ' ' {
				result.WriteString(ch)
			} else {
				result.WriteString(baseStyle.Render(ch))
			}
		}
		if y < gridH-1 {
			result.WriteString("\n")
		}
	}

	// Center the entire logo block
	logoBlock := result.String()
	centered := lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(logoBlock)
	return centered
}

var aboutContent = `
Hey, I'm Joe, a software developer interested in building entertaining or useful things.

Currently exploring React Internals and distributed systems.
`

var contactContent = `
Feel free to reach out!

  GitHub      %s
  Email       %s
  LinkedIn    %s
`

// ============================================================================
// Data Types
// ============================================================================

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
		Desc:    "Azure SQL VM team",
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
		Desc:    "New Glenn Rocket Software",
	},
}

// ============================================================================
// Page Types
// ============================================================================

type page int

const (
	menuPage page = iota
	aboutPage
	projectsPage
	experiencePage
	contactPage
)

var menuItems = []string{"About", "Projects", "Experience", "Contact"}

// ============================================================================
// Styles (Tokyo Night Theme)
// ============================================================================

var (
	// Tokyo Night palette
	tokyoFg      = lipgloss.Color("#C0CAF5") // primary text
	tokyoFgAlt   = lipgloss.Color("#A9B1D6") // secondary text
	tokyoMuted   = lipgloss.Color("#565F89") // muted/help text
	tokyoBlue    = lipgloss.Color("#7AA2F7") // title/border
	tokyoCyan    = lipgloss.Color("#7DCFFF") // links
	tokyoPurple  = lipgloss.Color("#BB9AF7") // selected highlight
	tokyoGreen   = lipgloss.Color("#9ECE6A") // tech tags

	// Aliases for compatibility
	oniViolet    = tokyoBlue   // titles/borders
	fujiWhite    = tokyoFg     // main text
	springGreen  = tokyoPurple // selected
	fujiGray     = tokyoMuted  // help text
	waveBlue     = tokyoCyan   // links
	surimiOrange = tokyoGreen  // tech tags
	carpYellow   = tokyoBlue   // project names

	// Styles
	titleStyle = lipgloss.NewStyle().
			Foreground(oniViolet).
			Bold(true).
			MarginBottom(1)

	menuStyle = lipgloss.NewStyle().
			Foreground(fujiWhite)

	selectedStyle = lipgloss.NewStyle().
			Foreground(springGreen).
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

// ============================================================================
// Model
// ============================================================================

type model struct {
	currentPage    page
	menuCursor     int
	projectCursor  int
	expCursor      int
	width          int
	height         int
	logoSweepIndex int
}

func initialModel() model {
	return model{
		currentPage:    menuPage,
		menuCursor:     0,
		projectCursor:  0,
		expCursor:      0,
		width:          80,
		height:         24,
		logoSweepIndex: 0,
	}
}

// ============================================================================
// Bubble Tea Interface
// ============================================================================

func (m model) Init() tea.Cmd {
	// Start the snake animation tick.
	return tickCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		if m.currentPage == menuPage {
			m.logoSweepIndex++
			return m, tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
				return tickMsg(t)
			})
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.currentPage == menuPage {
				return m, tea.Quit
			}
			m.currentPage = menuPage
			return m, tickCmd()

		case "esc", "backspace":
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

		case "enter", " ":
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
	var content string

	switch m.currentPage {
	case menuPage:
		content = m.renderMenu()
	case aboutPage:
		content = m.renderAbout()
	case projectsPage:
		content = m.renderProjects()
	case experiencePage:
		content = m.renderExperience()
	case contactPage:
		content = m.renderContact()
	}

	// Render content without an outer border
	boxWidth := min(m.width-4, 70)
	boxedContent := lipgloss.NewStyle().
		Padding(1, 2).
		Width(boxWidth).
		Render(content)

	// Center in terminal
	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		boxedContent)
}

// ============================================================================
// Page Renderers
// ============================================================================

func (m model) renderMenu() string {
	var b strings.Builder

	// Logo with animation
	b.WriteString(renderGradientLogo(60, m.logoSweepIndex))
	b.WriteString("\n\n")

	// Menu items
	for i, item := range menuItems {
		cursor := "  "
		if m.menuCursor == i {
			cursor = "→ "
		}

		line := cursor + item
		if m.menuCursor == i {
			b.WriteString(selectedStyle.Render(line))
		} else {
			b.WriteString(menuStyle.Render(line))
		}
		b.WriteString("\n")
	}

	// Help
	b.WriteString(helpStyle.Render("\n↑/↓: navigate • enter: select • esc/backspace: menu • q: quit"))

	return b.String()
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

		// Project name
		name := cursor + p.Name
		if m.projectCursor == i {
			b.WriteString(projectNameStyle.Render(name))
		} else {
			b.WriteString(menuStyle.Render(name))
		}
		b.WriteString("\n")

		// Show details for selected project
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

		// Role and company
		line := fmt.Sprintf("%s%s @ %s",
			cursor,
			roleStyle.Render(exp.Role),
			companyStyle.Render(exp.Company))
		b.WriteString(line)
		b.WriteString("\n")

		// Period
		b.WriteString("    ")
		b.WriteString(periodStyle.Render(exp.Period))
		b.WriteString("\n")

		// Description (only for selected)
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

// ============================================================================
// Main
// ============================================================================

func main() {
	// Force 256-color output for terminals that support it, even under systemd.
	lipgloss.SetColorProfile(termenv.ANSI256)

	publicKeyAuth := func(ctx ssh.Context, key ssh.PublicKey) bool {
		return true
	}
	passwordAuth := func(ctx ssh.Context, password string) bool {
		// Allow password auth as a fallback for clients without keys.
		return true
	}

	teaHandler := func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		return initialModel(), []tea.ProgramOption{tea.WithAltScreen()}
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
