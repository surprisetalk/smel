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

var env = make(map[string]any)

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
		var v any
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
		"cmd/none":  "_::none ()",
		"cmd/out":   "_::out",
		"sub/every": "_::every",
		"sub/in":    "_::in",
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
		var v any
		err = cbor.Unmarshal(result, &v)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error unmarshalling file", key+":", err)
			continue
		}
		env[key] = v
	}
}

type smelUpdate func(any) (scrapscript.Flat, string, error) // TODO: -> smel.Cmd msg
type smelSub func(scrapscript.Flat) string                  // TODO: -> smel.Sub msg
type smelView func(scrapscript.Flat) string                 // TODO: -> smel.View

type platform struct {
	model  any
	update smelUpdate
	subs   []smelSub
	view   smelView
}

type model struct {
	in         string
	platform   platform
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
	return tea.Batch(
		tea.EnableMouseCellMotion,
		tea.Tick(time.Second/10, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}),
		tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return blinkMsg(t)
		}),
	)
}

type tickMsg time.Time
type blinkMsg time.Time

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

			tokens, err := scrapscript.Lex(m.in)
			if err != nil {
				m.err = err
				return m, nil
			}

			flat, err := scrapscript.Parse(tokens)
			if err != nil {
				m.err = err
				return m, nil
			}

			flat, err = scrapscript.Eval(flat, env)
			if err != nil {
				m.err = err
				return m, nil
			}

			var v any
			err = cbor.Unmarshal(flat, &v)
			if err != nil {
				m.err = err
				return m, nil
			}

			p := platform{}
			cmds := []tea.Cmd{}

			var platform map[any]any
			var ok bool

			// TODO: Need to Marshal flatscraps into golang to avoid this nonsense.
			if platform, ok = v.(map[any]any); !ok {
				m.err = fmt.Errorf("invalid platform: %v", v)
				return m, nil
			}

			var init cbor.Tag

			if init, ok = platform["init"].(cbor.Tag); !ok || init.Number != scrapscript.TagExpr {
				m.err = fmt.Errorf("invalid init: %v", v)
				return m, nil
			}

			var expr []any

			if expr, ok = init.Content.([]any); !ok {
				m.err = fmt.Errorf("invalid init: %v", v)
				return m, nil
			}

			if len(expr) != 3 {
				m.err = fmt.Errorf("invalid init: %v", v)
				return m, nil
			}

			var op cbor.Tag

			if op, ok = expr[2].(cbor.Tag); !ok || op.Number != scrapscript.TagOp || op.Content != "'" {
				m.err = fmt.Errorf("invalid init: %v", v)
				return m, nil
			}

			p.model = expr[0]

			var cmd cbor.Tag

			// TODO: if cmd/http, append to cmds.
			if cmd, ok = expr[1].(cbor.Tag); !ok || cmd.Number != scrapscript.TagExpr {
				// TODO: what to do in case of error?
			}

			if expr, ok = cmd.Content.([]any); !ok {
				m.err = fmt.Errorf("invalid cmd: %v", cmd)
				return m, nil
			}

			if len(expr) != 3 {
				m.err = fmt.Errorf("invalid cmd: %v", cmd)
				return m, nil
			}

			if op, ok = expr[2].(cbor.Tag); !ok || op.Number != scrapscript.TagOp || op.Content != " " {
				m.err = fmt.Errorf("invalid cmd: %v", cmd)
				return m, nil
			}

			var tag cbor.Tag

			if tag, ok = expr[0].(cbor.Tag); !ok || tag.Number != scrapscript.TagTag {
				m.err = fmt.Errorf("invalid cmd: %v", cmd)
				return m, nil
			}

			switch tag.Content {
			case "none":
			case "out":
				out, err := cbor.Marshal(expr[1])
				if err != nil {
					m.err = err
					return m, nil
				}
				m.out, err = scrapscript.Print(out)
				if err != nil {
					m.err = err
					return m, nil
				}
			case "err":
				m.err = fmt.Errorf("%v", expr[1])
			default:
				m.err = fmt.Errorf("invalid cmd: %v", tag.Content)
				return m, nil
			}

			// p.update = platform["update"].(scrapscript.Flat)

			// p.subs = platform["subs"].([]scrapscript.Flat)

			// p.view = platform["view"].(scrapscript.Flat)

			m.platform = p

			return m, tea.Batch(cmds...)
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
		// TODO: Handle cmd and error.
		// m.platform.update(m.platform.model)
		return m, nil
	case blinkMsg:
		m.showCursor = !m.showCursor
		return m, nil
	}
	return m, nil
}

func (m model) View() string {
	cursor := "█"
	if !m.showCursor {
		cursor = " "
	}
	s := fmt.Sprintf("> %s%s\n\n", m.in, cursor)
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
