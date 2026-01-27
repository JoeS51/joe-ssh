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
)

// Animation tick message
type tickMsg time.Time

func clickableLink(label, url string) string {
	return "\x1b]8;;" + url + "\x1b\\" + label + "\x1b]8;;\x1b\\"
}

// ============================================================================
// ASCII Art & Content
// ============================================================================

var blockLogoLines = []string{
	"      ▄█  ▄██████▄     ▄████████ ",
	"     ███ ███    ███   ███    ███ ",
	"     ███ ███    ███   ███    █▀  ",
	"     ███ ███    ███  ▄███▄▄▄     ",
	"     ███ ███    ███ ▀▀███▀▀▀     ",
	"     ███ ███    ███   ███    █▄  ",
	"█    ███ ███    ███   ███    ███ ",
	"█▄ ▄▄███  ▀██████▀    ██████████ ",
}

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

var asciiLogoLines3 = []string{
	"     ██╗ ██████╗ ███████╗",
	"     ██║██╔═══██╗██╔════╝",
	"     ██║██║   ██║█████╗  ",
	"██   ██║██║   ██║██╔══╝  ",
	"╚█████╔╝╚██████╔╝███████╗",
	" ╚════╝  ╚═════╝ ╚══════╝",
}

func renderGradientLogo(width int, linesRevealed int) string {
	var result strings.Builder
	style := lipgloss.NewStyle().Foreground(oniViolet).Bold(true)

	// Only show revealed lines
	linesToShow := linesRevealed
	if linesToShow > len(asciiLogoLines) {
		linesToShow = len(asciiLogoLines)
	}

	for i := 0; i < len(asciiLogoLines); i++ {
		if i < linesToShow {
			result.WriteString(style.Render(asciiLogoLines[i]))
		} else {
			// Empty line to maintain spacing
			result.WriteString(strings.Repeat(" ", len(asciiLogoLines[i])))
		}
		if i < len(asciiLogoLines)-1 {
			result.WriteString("\n")
		}
	}

	// Center the entire logo block
	logoBlock := result.String()
	centered := lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(logoBlock)
	return centered
}

var aboutContent = `
Hi! I'm Joe, a software developer passionate about building
great user experiences and exploring new technologies.

I love working on:
  • Backend systems and APIs
  • Terminal applications and CLI tools
  • Cloud infrastructure and DevOps

Currently exploring Rust, React Internals and distributed systems.
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
		Desc: "This very app! A terminal-based portfolio accessible via SSH.",
		Tech: "Go, Bubble Tea, Wish",
		Link: "github.com/joe/ssh-portfolio",
	},
	{
		Name: "Task Manager CLI",
		Desc: "A command-line task manager with local storage and sync.",
		Tech: "Rust, SQLite",
		Link: "github.com/joe/task-cli",
	},
	{
		Name: "API Gateway",
		Desc: "High-performance API gateway with rate limiting and caching.",
		Tech: "Go, Redis, Docker",
		Link: "github.com/joe/api-gateway",
	},
	{
		Name: "Chat Application",
		Desc: "Real-time chat with WebSocket support and E2E encryption.",
		Tech: "TypeScript, Node.js, React",
		Link: "github.com/joe/chat-app",
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
		Desc:    "Develop new product that reviews manuscripts for Jenni AI",
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
// Styles (Kanagawa Theme)
// ============================================================================

var (
	// Kanagawa Colors - Muted, subtle, old-school terminal aesthetic
	warmWhite    = lipgloss.Color("#D8DEE9") // cool white with hint of blue (titles)
	lightGray    = lipgloss.Color("#9CA0A4") // light gray (main text)
	selectedGray = lipgloss.Color("#C8C8C8") // slightly brighter (selected)
	mutedGray    = lipgloss.Color("#727169") // muted gray (help text)
	softBlue     = lipgloss.Color("#7E9CD8") // soft blue
	mutedOrange  = lipgloss.Color("#FFA066") // muted orange
	deepRed      = lipgloss.Color("#C34043") // deep red
	softGold     = lipgloss.Color("#E6C384") // soft gold

	// Aliases for compatibility
	oniViolet    = warmWhite    // titles/borders - warm off-white
	fujiWhite    = lightGray    // main text - light gray
	springGreen  = selectedGray // selected - slightly brighter
	fujiGray     = mutedGray    // help text
	waveBlue     = softBlue     // links
	surimiOrange = mutedOrange  // tech tags
	autumnRed    = deepRed
	carpYellow   = softGold // project names

	// Styles
	logoStyle = lipgloss.NewStyle().
			Foreground(oniViolet).
			Bold(true).
			MarginBottom(1)

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
	currentPage       page
	menuCursor        int
	projectCursor     int
	expCursor         int
	width             int
	height            int
	logoLinesRevealed int
	animationDone     bool
}

func initialModel() model {
	return model{
		currentPage:       menuPage,
		menuCursor:        0,
		projectCursor:     0,
		expCursor:         0,
		width:             80,
		height:            24,
		logoLinesRevealed: 0,
		animationDone:     false,
	}
}

// ============================================================================
// Bubble Tea Interface
// ============================================================================

func (m model) Init() tea.Cmd {
	// Start the logo animation
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		if !m.animationDone && m.currentPage == menuPage {
			m.logoLinesRevealed++
			if m.logoLinesRevealed >= len(asciiLogoLines) {
				m.animationDone = true
				return m, nil
			}
			return m, tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
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
			return m, nil

		case "esc", "backspace":
			if m.currentPage != menuPage {
				m.currentPage = menuPage
			}
			return m, nil

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

	// Create bordered box
	boxWidth := min(m.width-4, 70)
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(oniViolet).
		Padding(1, 2).
		Width(boxWidth)

	boxedContent := box.Render(content)

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

	// Short welcome line before the animated logo.
	b.WriteString(subtleStyle.Render("Loading portfolio..."))
	b.WriteString("\n\n")

	// Logo with animation
	b.WriteString(renderGradientLogo(60, m.logoLinesRevealed))
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
	publicKeyAuth := func(ctx ssh.Context, key ssh.PublicKey) bool {
		return true
	}

	teaHandler := func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		return initialModel(), []tea.ProgramOption{tea.WithAltScreen()}
	}

	s, err := wish.NewServer(
		wish.WithAddress("0.0.0.0:2222"),
		wish.WithHostKeyPath(".ssh/host_ed25519"),
		wish.WithPublicKeyAuth(publicKeyAuth),
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
