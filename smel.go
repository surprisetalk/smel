package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	input      string
	output     string
	err        error
	showCursor bool
}

func initialModel() model {
	return model{
		showCursor: true,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tea.EnableMouseCellMotion, tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	}))
}

type tickMsg time.Time

type evalResultMsg struct {
	output string
	err    error
}

func evalScrapScript(input string) tea.Msg {
	cmd := exec.Command("python3", "scrapscript.py", "eval", "-")
	cmd.Stdin = strings.NewReader(input)
	cmd.Dir = "../scrapscript.py"
	output, err := cmd.CombinedOutput()
	return evalResultMsg{
		output: strings.TrimSpace(string(output)),
		err:    err,
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.input == "" {
				return m, nil
			}
			return m, func() tea.Msg {
				return evalScrapScript(m.input)
			}
		case tea.KeyBackspace:
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			if msg.String() != "" {
				m.input += msg.String()
			}
		}
	case tickMsg:
		m.showCursor = !m.showCursor
		return m, nil
	case evalResultMsg:
		m.output = msg.output
		m.err = msg.err
		m.input = ""
		return m, nil
	}
	return m, nil
}

func (m model) View() string {
	cursor := "█"
	if !m.showCursor {
		cursor = " "
	}

	s := "Scrapscript Interpreter\n\n"
	s += fmt.Sprintf("> %s%s\n\n", m.input, cursor)

	if m.err != nil {
		s += fmt.Sprintf("Error: %v\n", m.err)
	} else if m.output != "" {
		s += fmt.Sprintf("Result: %s\n", m.output)
	}

	s += "\nCtrl+C to quit\n"
	return s
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
	}
}
