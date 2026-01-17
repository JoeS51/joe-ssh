package main

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/bubbletea"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF79C6")).
			MarginBottom(1)

	menuStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8BE9FD"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#50FA7B")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4")).
			MarginTop(1)
)

// Model holds the application state
type model struct {
	choices  []string
	cursor   int
	selected map[int]struct{}
}

// Initial model
func initialModel() model {
	return model{
		choices: []string{
			"About Me",
			"Projects",
			"Experience",
			"Contact",
		},
		selected: make(map[int]struct{}),
	}
}

// Init is called when the program starts
func (m model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter", " ":
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		}
	}
	return m, nil
}

// View renders the UI
func (m model) View() string {
	s := titleStyle.Render("✨ Welcome to My Portfolio ✨") + "\n\n"

	for i, choice := range m.choices {
		cursor := "  "
		if m.cursor == i {
			cursor = "→ "
		}

		checked := " "
		if _, ok := m.selected[i]; ok {
			checked = "●"
		}

		line := cursor + "[" + checked + "] " + choice
		if m.cursor == i {
			s += selectedStyle.Render(line) + "\n"
		} else {
			s += menuStyle.Render(line) + "\n"
		}
	}

	s += helpStyle.Render("\n↑/↓ or j/k: navigate • enter: select • q: quit")

	return s
}

func main() {
	// Accept any public key (open to all)
	publicKeyAuth := func(ctx ssh.Context, key ssh.PublicKey) bool {
		return true
	}

	// Create a Bubble Tea handler for SSH sessions
	teaHandler := func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		return initialModel(), []tea.ProgramOption{tea.WithAltScreen()}
	}

	s, err := wish.NewServer(
		wish.WithAddress("127.0.0.1:23234"),
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
