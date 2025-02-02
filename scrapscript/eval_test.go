package scrapscript

import (
	"strings"
	"testing"
)

func TestEval(t *testing.T) {
	tests := []struct {
		input    string
		env      Env
		expected string
		wantErr  bool
		errMsg   string
	}{
		// Basic literal evaluation
		{
			input:    "5",
			expected: "5",
		},
		{
			input:    "3.14",
			expected: "3.14",
		},
		{
			input:    `"xyz"`,
			expected: "xyz",
		},
		{
			input:    "~~eHl6", // base64 for "xyz"
			expected: "~~eHl6",
		},

		// Variable evaluation
		{
			input:   "no",
			wantErr: true,
		},
		{
			input:    "yes",
			env:      Env{"yes": int64(123)},
			expected: "123",
		},

		// Binary operations
		{
			input:    "1 + 2",
			expected: "3",
		},
		{
			input:    "(1 + 2) + 3",
			expected: "6",
		},
		{
			input:   `1 + "hello"`,
			wantErr: true,
			errMsg:  "expected Int or Float, got String",
		},
		{
			input:    "1 - 2",
			expected: "-1",
		},
		{
			input:    "2 * 3",
			expected: "6",
		},
		{
			input:    "3 / 9",
			expected: "0.3",
		},
		{
			input:    "2 ^ 3",
			expected: "8",
		},
		{
			input:    "10 % 4",
			expected: "2",
		},

		// Comparison operations
		{
			input:    "1 == 1",
			expected: "true",
		},
		{
			input:    "1 == 2",
			expected: "false",
		},
		{
			input:    "1 /= 1",
			expected: "false",
		},
		{
			input:    "1 /= 2",
			expected: "true",
		},

		// String operations
		{
			input:    `"hello" ++ " world"`,
			expected: `"hello world"`,
		},
		{
			input:   `123 ++ " world"`,
			wantErr: true,
			errMsg:  "expected String, got Int",
		},

		// List operations
		{
			input:    "1 >+ [2, 3]",
			expected: "[ 1, 2, 3 ]",
		},
		{
			input:    "[1, 2] +< 3",
			expected: "[ 1, 2, 3 ]",
		},
		{
			input:    "[1 + 2, 3 + 4]",
			expected: "[ 3, 7 ]",
		},

		// Function evaluation
		{
			input:    "x -> x",
			expected: "x -> x",
		},
		{
			input:    "(x -> x + 1) 2",
			expected: "3",
		},
		{
			input:    "(a -> b -> a + b) 3 2",
			expected: "5",
		},

		// Record operations
		{
			input:    "{a = 1 + 2}",
			expected: "{ a = 3 }",
		},
		{
			input:    `{a = 4}@a`,
			expected: "4",
		},
		{
			input:   `{a = 4}@b`,
			wantErr: true,
			errMsg:  "no assignment to b found in record",
		},

		// Boolean operations
		{
			input:    "3 < 4",
			expected: "true",
		},
		{
			input:    "3 > 4",
			expected: "false",
		},
		{
			input:    "3 <= 4",
			expected: "true",
		},
		{
			input:    "3 >= 4",
			expected: "false",
		},
		{
			input:    "#true () && #false ()",
			expected: "false",
		},
		{
			input:    "#true () || #false ()",
			expected: "true",
		},

		// Variants
		{
			input:    "#abc (1 + 2)",
			expected: "#abc 3",
		},

		// Float operations
		{
			input:    "1.0 + 2.0",
			expected: "3.0",
		},
		// {
		// 	input:    "1 + 2.0",
		// 	expected: "invalid operands for +",
		// },
		{
			input:    "1.0 / 2.0",
			expected: "0.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens, err := Lex(tt.input)
			if err != nil {
				t.Fatalf("Lex failed: %v", err)
			}

			ast, err := Parse(tokens)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			result, err := Eval(ast)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Eval failed: %v", err)
			}

			ans, err := Print(result)
			if err != nil {
				t.Fatalf("Print failed: %v", result)
			}

			if ans != tt.expected {
				t.Errorf("wrong result\nwant: %#v\ngot:  %#v", tt.expected, result)
			}
		})
	}
}
