package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
)

const (
	host = "0.0.0.0"
	port = "22"
)

func main() {
	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithMiddleware(
			bubbletea.Middleware(teaHandler),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)

	if err != nil {
		log.Error("Could not create server", "error", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Starting SSH server", "host", host, "port", port)
	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not listen on address", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("Shutting down SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("Could not shutdown server", "error", err)
	}
}

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	pty, _, _ := s.Pty()

	renderer := bubbletea.MakeRenderer(s)
	mainStyle := renderer.NewStyle().MarginLeft(2)
	checkboxStyle := renderer.NewStyle().Bold(false).Foreground(lipgloss.Color("213"))
	aboutStyle := renderer.NewStyle().Bold(true).Foreground(lipgloss.Color("246"))
	aboutNameStyle := renderer.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	subtleStyle := renderer.NewStyle().Foreground(lipgloss.Color("241"))
	dotStyle := renderer.NewStyle().Foreground(lipgloss.Color("236")).Render("•")

	m := model{
		Width:          pty.Window.Width,
		Height:         pty.Window.Height,
		Choice:         0,
		Chosen:         false,
		mainStyle:      mainStyle,
		checkboxStyle:  checkboxStyle,
		aboutStyle:     aboutStyle,
		aboutNameStyle: aboutNameStyle,
		subtleStyle:    subtleStyle,
		dotStyle:       dotStyle,
		sess:           s,
		runtime:        "",
	}

	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

type model struct {
	Width          int
	Height         int
	Choice         int
	Chosen         bool
	mainStyle      lipgloss.Style
	checkboxStyle  lipgloss.Style
	aboutStyle     lipgloss.Style
	aboutNameStyle lipgloss.Style
	subtleStyle    lipgloss.Style
	dotStyle       string
	sess           ssh.Session
	runtime        string
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) View() string {
	about := m.aboutStyle.Render(fmt.Sprintf(strings.TrimSpace(`
Hi I'm %s,

A self taught developer specialized in many software domains
including MacOS Apps, Web, Backend, Gen AI, and more.

I'm fluent in TypeScript, Python, Java and more.
	`), m.aboutNameStyle.Render("Eduardo Monteiro")))

	tpl := m.subtleStyle.Render("Hint: Q or Ctrl+C to quit")

	choices := fmt.Sprintf(
		"%s\n%s",
		m.subtleStyle.Copy().Foreground(lipgloss.Color("13")).Render("GitHub         https://github.com/edu-wmv"),
		m.subtleStyle.Copy().Foreground(lipgloss.Color("33")).Render("Linkedin       https://linkedin.com/in/eduardomonteiro-ss"),
	)

	s := fmt.Sprintf("%s\n\n%s\n\n%s", about, choices, tpl)
	return m.mainStyle.Render("\n" + s + "\n\n")
}
