package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModel_Update(t *testing.T) {
	tests := []struct {
		input  string
		output string
	}{
		{
			input:  "always (1 + 2 * 3 - 4)",
			output: "3",
		},
		{
			input:  "always 123",
			output: "123",
		},
		{
			input:  "123 + 1 |> always",
			output: "124",
		},
		{
			input:  "every 1000",
			output: "TODO",
		},
		{
			input:  "every 1000 |> random |> plot",
			output: "TODO",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			m := model{
				input: tt.input,
			}
			_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			msg := cmd().(evalResultMsg)
			if msg.err != nil {
				t.Errorf("error: %v", msg.err)
			} else if msg.output != tt.output {
				t.Errorf("expected output %s, got %s", tt.output, msg.output)
			}
		})
	}
}
