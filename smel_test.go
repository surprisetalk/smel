package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModel_Update(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{
			in:  "always (1 + 2 * 3 - 4)",
			out: "3",
		},
		{
			in:  "always 123",
			out: "123",
		},
		{
			in:  "123 + 1 |> always",
			out: "124",
		},
		{
			in:  "every 1000",
			out: "TODO",
		},
		{
			in:  "every 1000 |> random |> plot",
			out: "TODO",
		},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			m := model{
				in: tt.in,
			}
			_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			msg := cmd().(evalMsg)
			if msg.err != nil {
				t.Errorf("error: %v", msg.err)
			} else if msg.out != tt.out {
				t.Errorf("expected out %s, got %s", tt.out, msg.out)
			}
		})
	}
}
