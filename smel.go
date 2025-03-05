package main

import (
	"fmt"
	"os"
	"path/filepath"
	"smel/scrapscript"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fxamacker/cbor/v2"
)

var env = make(map[string]interface{})

// TODO: These should go in the scrapyard rather than passed via env.
func init() {
	files, err := os.ReadDir("./widgets")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error reading widgets directory:", err)
		return
	}
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".ss") {
			fmt.Fprintln(os.Stderr, "not a scrapscript file", file.Name()+":", err)
			continue
		}
		filePath := filepath.Join("./widgets", file.Name())
		in, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error reading file", file.Name()+":", err)
			continue
		}
		tokens, err := scrapscript.Lex(string(in))
		if err != nil {
			fmt.Fprintln(os.Stderr, "error lexing file", file.Name()+":", err)
			continue
		}
		flat, err := scrapscript.Parse(tokens)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error parsing file", file.Name()+":", err)
			continue
		}
		var v interface{}
		err = cbor.Unmarshal(flat, &v)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error unmarshalling file", file.Name()+":", err)
			continue
		}
		key := strings.TrimSuffix(file.Name(), ".ss")
		env[key] = v
	}

	for key, in := range map[string]string{
		// "cmd":      "#none () #out ()",
		// "cmd/none": "cmd::none ()",
		// "cmd/out":  "cmd::out",
		"cmd/none": "_::none ()",
		"cmd/out":  "_::out",
	} {
		tokens, err := scrapscript.Lex(in)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error lexing", key+":", err)
			continue
		}
		flat, err := scrapscript.Parse(tokens)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error parsing", key+":", err)
			continue
		}
		result, err := scrapscript.Eval(flat, env)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error evaluating", key+":", err)
			continue
		}
		var v interface{}
		err = cbor.Unmarshal(result, &v)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error unmarshalling file", key+":", err)
			continue
		}
		env[key] = v
	}
}

type model struct {
	in         string
	out        string
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

type evalMsg struct {
	out string
	err error
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlD, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.in == "" {
				return m, nil
			}
			return m, func() tea.Msg {
				tokens, err := scrapscript.Lex(m.in)
				if err != nil {
					return evalMsg{err: err}
				}

				flat, err := scrapscript.Parse(tokens)
				if err != nil {
					return evalMsg{err: err}
				}

				flat, err = scrapscript.Eval(flat, env)
				if err != nil {
					return evalMsg{err: err}
				}

				result, err := scrapscript.Print(flat)
				return evalMsg{
					out: strings.TrimSpace(result),
					err: err,
				}
			}
		case tea.KeyBackspace:
			if len(m.in) > 0 {
				m.in = m.in[:len(m.in)-1]
			}
		default:
			if msg.String() != "" {
				m.in += msg.String()
			}
		}
	case tickMsg:
		m.showCursor = !m.showCursor
		return m, nil
	case evalMsg:
		m.out = msg.out
		m.err = msg.err
		m.in = ""
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
	s += fmt.Sprintf("> %s%s\n\n", m.in, cursor)
	if m.err != nil {
		s += fmt.Sprintf("Error: %v\n", m.err)
	} else if m.out != "" {
		s += fmt.Sprintf("Result: %s\n", m.out)
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
