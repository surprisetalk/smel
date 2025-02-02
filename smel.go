package main

import (
	"fmt"
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

func initial() model {
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
				tokens, err := Lex(m.input)
				if err != nil {
					return evalResultMsg{err: err}
				}

				flat, err := Parse(tokens)
				if err != nil {
					return evalResultMsg{err: err}
				}

				result, err := Print(flat)
				return evalResultMsg{
					output: strings.TrimSpace(result),
					err:    err,
				}
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
	s := ""
	s += fmt.Sprintf("> %s%s\n\n", m.input, cursor)
	if m.err != nil {
		s += fmt.Sprintf("Error: %v\n", m.err)
	} else if m.output != "" {
		s += fmt.Sprintf("Result: %s\n", m.output)
	} else {
		s += "\n"
	}
	return s
}

func main() {
	p := tea.NewProgram(initial())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
	}
}
