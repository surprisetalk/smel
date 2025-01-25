/*
* All of this was copied from github:tekknolagi/scrapscript using Claude.
*
* This parallel go implementation should be thrown away when the language stabilizes.
*
 */

package smel_test

import (
	. "smel"
	"strings"
	"testing"
)

func TestLex(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
		wantErr  bool
		errMsg   string
	}{
		{
			name:  "tokenize single digit",
			input: "1",
			expected: []Token{
				{Type: TokenIntLit, Value: int64(1)},
			},
		},
		{
			name:  "tokenize multiple digits",
			input: "123",
			expected: []Token{
				{Type: TokenIntLit, Value: int64(123)},
			},
		},
		{
			name:  "tokenize negative int",
			input: "-123",
			expected: []Token{
				{Type: TokenOperator, Value: "-"},
				{Type: TokenIntLit, Value: int64(123)},
			},
		},
		{
			name:  "tokenize float",
			input: "3.14",
			expected: []Token{
				{Type: TokenFloatLit, Value: float64(3.14)},
			},
		},
		{
			name:  "tokenize negative float",
			input: "-3.14",
			expected: []Token{
				{Type: TokenOperator, Value: "-"},
				{Type: TokenFloatLit, Value: float64(3.14)},
			},
		},
		{
			name:  "tokenize float with no decimal part",
			input: "10.",
			expected: []Token{
				{Type: TokenFloatLit, Value: float64(10.0)},
			},
		},
		{
			name:    "tokenize float with multiple decimal points raises error",
			input:   "1.0.1",
			wantErr: true,
			errMsg:  "unexpected character '.'",
		},
		{
			name:  "tokenize binop",
			input: "1 + 2",
			expected: []Token{
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenOperator, Value: "+"},
				{Type: TokenIntLit, Value: int64(2)},
			},
		},
		{
			name:  "tokenize binop no spaces",
			input: "1+2",
			expected: []Token{
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenOperator, Value: "+"},
				{Type: TokenIntLit, Value: int64(2)},
			},
		},
		{
			name:  "tokenize string",
			input: `"hello"`,
			expected: []Token{
				{Type: TokenStringLit, Value: "hello"},
			},
		},
		{
			name:  "tokenize string with spaces",
			input: `"hello world"`,
			expected: []Token{
				{Type: TokenStringLit, Value: "hello world"},
			},
		},
		{
			name:    "tokenize string missing end quote raises error",
			input:   `"hello`,
			wantErr: true,
			errMsg:  "unterminated string",
		},
		{
			name:  "tokenize identifier",
			input: "abc",
			expected: []Token{
				{Type: TokenName, Value: "abc"},
			},
		},
		{
			name:  "tokenize identifier with special chars",
			input: "$sha1'foo",
			expected: []Token{
				{Type: TokenName, Value: "$sha1'foo"},
			},
		},
		{
			name:  "tokenize empty list",
			input: "[]",
			expected: []Token{
				{Type: TokenLeftBracket, Value: "["},
				{Type: TokenRightBracket, Value: "]"},
			},
		},
		{
			name:  "tokenize list with items",
			input: "[1,2]",
			expected: []Token{
				{Type: TokenLeftBracket, Value: "["},
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenOperator, Value: ","},
				{Type: TokenIntLit, Value: int64(2)},
				{Type: TokenRightBracket, Value: "]"},
			},
		},
		{
			name:  "ignore whitespace",
			input: "1\n+\t2",
			expected: []Token{
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenOperator, Value: "+"},
				{Type: TokenIntLit, Value: int64(2)},
			},
		},
		{
			name:  "ignore line comment",
			input: "-- 1\n2",
			expected: []Token{
				{Type: TokenIntLit, Value: int64(2)},
			},
		},
		{
			name:  "tokenize bytes base64",
			input: "~~QUJD",
			expected: []Token{
				{Type: TokenBytesLit, Value: struct {
					Base  int64
					Value string
				}{64, "QUJD"}},
			},
		},
		{
			name:  "tokenize bytes with explicit base",
			input: "~~85'K|(_",
			expected: []Token{
				{Type: TokenBytesLit, Value: struct {
					Base  int64
					Value string
				}{85, "K|(_"}},
			},
		},
		{
			name:  "tokenize two operator chars",
			input: ",:",
			expected: []Token{
				{Type: TokenOperator, Value: ","},
				{Type: TokenOperator, Value: ":"},
			},
		},
		{
			name:  "tokenize binary subtraction no spaces",
			input: "1-2",
			expected: []Token{
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenOperator, Value: "-"},
				{Type: TokenIntLit, Value: int64(2)},
			},
		},
		{
			name:  "tokenize variable with dollar prefix",
			input: "$$bills",
			expected: []Token{
				{Type: TokenName, Value: "$$bills"},
			},
		},
		/*
			{
				name:    "tokenize dot dot error",
				input:   "..",
				wantErr: true,
				errMsg:  "unexpected",
			},
		*/
		{
			name:  "tokenize spread operator",
			input: "...",
			expected: []Token{
				{Type: TokenOperator, Value: "..."},
			},
		},
		{
			name:  "tokenize with trailing whitespace",
			input: "- ",
			expected: []Token{
				{Type: TokenOperator, Value: "-"},
			},
		},
		{
			name:     "tokenize empty comment",
			input:    "-- ",
			expected: []Token{},
		},
		{
			name:  "tokenize function arrow",
			input: "a -> b -> a + b",
			expected: []Token{
				{Type: TokenName, Value: "a"},
				{Type: TokenOperator, Value: "->"},
				{Type: TokenName, Value: "b"},
				{Type: TokenOperator, Value: "->"},
				{Type: TokenName, Value: "a"},
				{Type: TokenOperator, Value: "+"},
				{Type: TokenName, Value: "b"},
			},
		},
		{
			name:  "tokenize function arrow no spaces",
			input: "a->b->a+b",
			expected: []Token{
				{Type: TokenName, Value: "a"},
				{Type: TokenOperator, Value: "->"},
				{Type: TokenName, Value: "b"},
				{Type: TokenOperator, Value: "->"},
				{Type: TokenName, Value: "a"},
				{Type: TokenOperator, Value: "+"},
				{Type: TokenName, Value: "b"},
			},
		},
		{
			name:  "tokenize where dot",
			input: "a . b",
			expected: []Token{
				{Type: TokenName, Value: "a"},
				{Type: TokenOperator, Value: "."},
				{Type: TokenName, Value: "b"},
			},
		},
		{
			name:  "tokenize assert",
			input: "a ? b",
			expected: []Token{
				{Type: TokenName, Value: "a"},
				{Type: TokenOperator, Value: "?"},
				{Type: TokenName, Value: "b"},
			},
		},
		{
			name:  "tokenize type annotation",
			input: "a : b",
			expected: []Token{
				{Type: TokenName, Value: "a"},
				{Type: TokenOperator, Value: ":"},
				{Type: TokenName, Value: "b"},
			},
		},
		{
			name:    "tokenize tilde error",
			input:   "~",
			wantErr: true,
			errMsg:  "unexpected character '~'",
		},
		{
			name:    "tokenize tilde equals error",
			input:   "~=",
			wantErr: true,
			errMsg:  "unexpected character '~'",
		},
		{
			name:  "tokenize record with spread",
			input: "{ x = 1, ... }",
			expected: []Token{
				{Type: TokenLeftBrace, Value: "{"},
				{Type: TokenName, Value: "x"},
				{Type: TokenOperator, Value: "="},
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenOperator, Value: ","},
				{Type: TokenOperator, Value: "..."},
				{Type: TokenRightBrace, Value: "}"},
			},
		},
		{
			name:  "tokenize record with spread no spaces",
			input: "{x=1,...}",
			expected: []Token{
				{Type: TokenLeftBrace, Value: "{"},
				{Type: TokenName, Value: "x"},
				{Type: TokenOperator, Value: "="},
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenOperator, Value: ","},
				{Type: TokenOperator, Value: "..."},
				{Type: TokenRightBrace, Value: "}"},
			},
		},
		{
			name:  "tokenize record access",
			input: "r@a",
			expected: []Token{
				{Type: TokenName, Value: "r"},
				{Type: TokenOperator, Value: "@"},
				{Type: TokenName, Value: "a"},
			},
		},
		{
			name:  "tokenize right eval",
			input: "a!b",
			expected: []Token{
				{Type: TokenName, Value: "a"},
				{Type: TokenOperator, Value: "!"},
				{Type: TokenName, Value: "b"},
			},
		},
		{
			name:  "tokenize match expression",
			input: "g = | 1 -> 2 | 2 -> 3",
			expected: []Token{
				{Type: TokenName, Value: "g"},
				{Type: TokenOperator, Value: "="},
				{Type: TokenOperator, Value: "|"},
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenOperator, Value: "->"},
				{Type: TokenIntLit, Value: int64(2)},
				{Type: TokenOperator, Value: "|"},
				{Type: TokenIntLit, Value: int64(2)},
				{Type: TokenOperator, Value: "->"},
				{Type: TokenIntLit, Value: int64(3)},
			},
		},
		{
			name:  "tokenize pipe operator",
			input: "1 |> f . f = a -> a + 1",
			expected: []Token{
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenOperator, Value: "|>"},
				{Type: TokenName, Value: "f"},
				{Type: TokenOperator, Value: "."},
				{Type: TokenName, Value: "f"},
				{Type: TokenOperator, Value: "="},
				{Type: TokenName, Value: "a"},
				{Type: TokenOperator, Value: "->"},
				{Type: TokenName, Value: "a"},
				{Type: TokenOperator, Value: "+"},
				{Type: TokenIntLit, Value: int64(1)},
			},
		},
		{
			name:  "tokenize compose",
			input: "f >> g",
			expected: []Token{
				{Type: TokenName, Value: "f"},
				{Type: TokenOperator, Value: ">>"},
				{Type: TokenName, Value: "g"},
			},
		},
		{
			name:  "tokenize compose reverse",
			input: "f << g",
			expected: []Token{
				{Type: TokenName, Value: "f"},
				{Type: TokenOperator, Value: "<<"},
				{Type: TokenName, Value: "g"},
			},
		},
		{
			name:  "tokenize variant with no space",
			input: "#abc",
			expected: []Token{
				{Type: TokenHash, Value: "#"},
				{Type: TokenName, Value: "abc"},
			},
		},
		// Basic literals
		{
			name:  "tokenize single digit",
			input: "1",
			expected: []Token{
				{Type: TokenIntLit, Value: int64(1)},
			},
		},
		{
			name:  "tokenize multiple digits",
			input: "123",
			expected: []Token{
				{Type: TokenIntLit, Value: int64(123)},
			},
		},
		// Numbers
		{
			name:  "tokenize negative int",
			input: "-123",
			expected: []Token{
				{Type: TokenOperator, Value: "-"},
				{Type: TokenIntLit, Value: int64(123)},
			},
		},
		{
			name:  "tokenize float",
			input: "3.14",
			expected: []Token{
				{Type: TokenFloatLit, Value: float64(3.14)},
			},
		},
		{
			name:  "tokenize negative float",
			input: "-3.14",
			expected: []Token{
				{Type: TokenOperator, Value: "-"},
				{Type: TokenFloatLit, Value: float64(3.14)},
			},
		},
		{
			name:  "tokenize float with no decimal part",
			input: "10.",
			expected: []Token{
				{Type: TokenFloatLit, Value: float64(10.0)},
			},
		},
		{
			name:    "tokenize float with multiple decimal points raises error",
			input:   "1.0.1",
			wantErr: true,
			errMsg:  "unexpected character '.'",
		},
		// Basic operations
		{
			name:  "tokenize binop",
			input: "1 + 2",
			expected: []Token{
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenOperator, Value: "+"},
				{Type: TokenIntLit, Value: int64(2)},
			},
		},
		{
			name:  "tokenize binop no spaces",
			input: "1+2",
			expected: []Token{
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenOperator, Value: "+"},
				{Type: TokenIntLit, Value: int64(2)},
			},
		},
		// Strings
		{
			name:  "tokenize string",
			input: `"hello"`,
			expected: []Token{
				{Type: TokenStringLit, Value: "hello"},
			},
		},
		{
			name:  "tokenize string with spaces",
			input: `"hello world"`,
			expected: []Token{
				{Type: TokenStringLit, Value: "hello world"},
			},
		},
		{
			name:  "tokenize empty string",
			input: `""`,
			expected: []Token{
				{Type: TokenStringLit, Value: ""},
			},
		},
		{
			name:  "tokenize string with escaped quotes",
			input: `"hello\"world"`,
			expected: []Token{
				{Type: TokenStringLit, Value: `hello"world`},
			},
		},
		{
			name:    "tokenize string missing end quote raises error",
			input:   `"hello`,
			wantErr: true,
			errMsg:  "unterminated string",
		},
		// Identifiers
		{
			name:  "tokenize identifier",
			input: "abc",
			expected: []Token{
				{Type: TokenName, Value: "abc"},
			},
		},
		{
			name:  "tokenize identifier with special chars",
			input: "$sha1'foo",
			expected: []Token{
				{Type: TokenName, Value: "$sha1'foo"},
			},
		},
		{
			name:  "tokenize dollar identifier",
			input: "$$bills",
			expected: []Token{
				{Type: TokenName, Value: "$$bills"},
			},
		},
		// Lists and brackets
		{
			name:  "tokenize empty list",
			input: "[]",
			expected: []Token{
				{Type: TokenLeftBracket, Value: "["},
				{Type: TokenRightBracket, Value: "]"},
			},
		},
		{
			name:  "tokenize list with items",
			input: "[1,2]",
			expected: []Token{
				{Type: TokenLeftBracket, Value: "["},
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenOperator, Value: ","},
				{Type: TokenIntLit, Value: int64(2)},
				{Type: TokenRightBracket, Value: "]"},
			},
		},
		{
			name:  "tokenize nested lists",
			input: "[[1,2],[3,4]]",
			expected: []Token{
				{Type: TokenLeftBracket, Value: "["},
				{Type: TokenLeftBracket, Value: "["},
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenOperator, Value: ","},
				{Type: TokenIntLit, Value: int64(2)},
				{Type: TokenRightBracket, Value: "]"},
				{Type: TokenOperator, Value: ","},
				{Type: TokenLeftBracket, Value: "["},
				{Type: TokenIntLit, Value: int64(3)},
				{Type: TokenOperator, Value: ","},
				{Type: TokenIntLit, Value: int64(4)},
				{Type: TokenRightBracket, Value: "]"},
				{Type: TokenRightBracket, Value: "]"},
			},
		},
		// Whitespace handling
		{
			name:  "ignore whitespace",
			input: "1\n+\t2",
			expected: []Token{
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenOperator, Value: "+"},
				{Type: TokenIntLit, Value: int64(2)},
			},
		},
		{
			name:  "ignore multiple whitespace",
			input: "1  \n\n\t  +  \t\n  2",
			expected: []Token{
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenOperator, Value: "+"},
				{Type: TokenIntLit, Value: int64(2)},
			},
		},
		// Comments
		{
			name:  "ignore line comment",
			input: "-- 1\n2",
			expected: []Token{
				{Type: TokenIntLit, Value: int64(2)},
			},
		},
		{
			name:  "ignore multiple comments",
			input: "1\n-- comment1\n2\n-- comment2\n3",
			expected: []Token{
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenIntLit, Value: int64(2)},
				{Type: TokenIntLit, Value: int64(3)},
			},
		},
		// Bytes literals
		{
			name:  "tokenize bytes base64",
			input: "~~QUJD",
			expected: []Token{
				{Type: TokenBytesLit, Value: struct {
					Base  int64
					Value string
				}{64, "QUJD"}},
			},
		},
		{
			name:  "tokenize bytes with base85",
			input: "~~85'K|(_",
			expected: []Token{
				{Type: TokenBytesLit, Value: struct {
					Base  int64
					Value string
				}{85, "K|(_"}},
			},
		},
		// Records
		{
			name:  "tokenize empty record",
			input: "{}",
			expected: []Token{
				{Type: TokenLeftBrace, Value: "{"},
				{Type: TokenRightBrace, Value: "}"},
			},
		},
		{
			name:  "tokenize record with one field",
			input: "{x=1}",
			expected: []Token{
				{Type: TokenLeftBrace, Value: "{"},
				{Type: TokenName, Value: "x"},
				{Type: TokenOperator, Value: "="},
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenRightBrace, Value: "}"},
			},
		},
		{
			name:  "tokenize record with multiple fields",
			input: "{x=1,y=2}",
			expected: []Token{
				{Type: TokenLeftBrace, Value: "{"},
				{Type: TokenName, Value: "x"},
				{Type: TokenOperator, Value: "="},
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenOperator, Value: ","},
				{Type: TokenName, Value: "y"},
				{Type: TokenOperator, Value: "="},
				{Type: TokenIntLit, Value: int64(2)},
				{Type: TokenRightBrace, Value: "}"},
			},
		},
		// Function syntax
		{
			name:  "tokenize function arrow",
			input: "x -> x + 1",
			expected: []Token{
				{Type: TokenName, Value: "x"},
				{Type: TokenOperator, Value: "->"},
				{Type: TokenName, Value: "x"},
				{Type: TokenOperator, Value: "+"},
				{Type: TokenIntLit, Value: int64(1)},
			},
		},
		// Pattern matching
		{
			name:  "tokenize simple pattern match",
			input: "| 1 -> 2",
			expected: []Token{
				{Type: TokenOperator, Value: "|"},
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenOperator, Value: "->"},
				{Type: TokenIntLit, Value: int64(2)},
			},
		},
		// Special operators
		{
			name:  "tokenize spread operator",
			input: "{ x, ...rest }",
			expected: []Token{
				{Type: TokenLeftBrace, Value: "{"},
				{Type: TokenName, Value: "x"},
				{Type: TokenOperator, Value: ","},
				{Type: TokenOperator, Value: "..."},
				{Type: TokenName, Value: "rest"},
				{Type: TokenRightBrace, Value: "}"},
			},
		},
		{
			name:  "tokenize compose operators",
			input: "f >> g << h",
			expected: []Token{
				{Type: TokenName, Value: "f"},
				{Type: TokenOperator, Value: ">>"},
				{Type: TokenName, Value: "g"},
				{Type: TokenOperator, Value: "<<"},
				{Type: TokenName, Value: "h"},
			},
		},
		// Edge cases
		// {
		// 	name:    "tokenize invalid sequence",
		// 	input:   "@#$%",
		// 	wantErr: true,
		// 	errMsg:  "unexpected character",
		// },
		{
			name:  "tokenize numeric edge cases",
			input: "0 -0 +0 0.0",
			expected: []Token{
				{Type: TokenIntLit, Value: int64(0)},
				{Type: TokenOperator, Value: "-"},
				{Type: TokenIntLit, Value: int64(0)},
				{Type: TokenOperator, Value: "+"},
				{Type: TokenIntLit, Value: int64(0)},
				{Type: TokenFloatLit, Value: float64(0.0)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Lex(tt.input)

			// Check error cases
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

			// Check success cases
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(tokens) != len(tt.expected) {
				t.Errorf("wrong number of tokens\nwant: %+v\ngot:  %+v", tt.expected, tokens)
				return
			}

			for i, want := range tt.expected {
				got := tokens[i]
				if got.Type != want.Type {
					t.Errorf("token[%d] wrong type\nwant: %v\ngot:  %v", i, want.Type, got.Type)
				}
				if got.Value != want.Value {
					t.Errorf("token[%d] wrong value\nwant: %v\ngot:  %v", i, want.Value, got.Value)
				}
			}
		})
	}
}

func TestOperatorCombinations(t *testing.T) {
	ops := []string{
		"+", "-", "*", "/", "^", "%",
		"==", "/=", "<", ">", "<=", ">=",
		"&&", "||", "++", ">+", "+<",
	}

	for _, op := range ops {
		// Test with spaces
		t.Run("with spaces "+op, func(t *testing.T) {
			input := "a " + op + " b"
			expected := []Token{
				{Type: TokenName, Value: "a"},
				{Type: TokenOperator, Value: op},
				{Type: TokenName, Value: "b"},
			}
			tokens, err := Lex(input)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			compareTokens(t, expected, tokens)
		})

		// Test without spaces
		t.Run("no spaces "+op, func(t *testing.T) {
			input := "a" + op + "b"
			expected := []Token{
				{Type: TokenName, Value: "a"},
				{Type: TokenOperator, Value: op},
				{Type: TokenName, Value: "b"},
			}
			tokens, err := Lex(input)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			compareTokens(t, expected, tokens)
		})
	}
}

func compareTokens(t *testing.T, expected, got []Token) {
	if len(expected) != len(got) {
		t.Errorf("wrong number of tokens\nwant: %+v\ngot:  %+v", expected, got)
		return
	}

	for i := range expected {
		if expected[i].Type != got[i].Type {
			t.Errorf("token[%d] wrong type\nwant: %v\ngot:  %v", i, expected[i].Type, got[i].Type)
		}
		if expected[i].Value != got[i].Value {
			t.Errorf("token[%d] wrong value\nwant: %v\ngot:  %v", i, expected[i].Value, got[i].Value)
		}
	}
}

// TestNestedStructures tests various combinations of nested structures
func TestNestedStructures(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "nested records",
			input: "{x={y=1}}",
			expected: []Token{
				{Type: TokenLeftBrace, Value: "{"},
				{Type: TokenName, Value: "x"},
				{Type: TokenOperator, Value: "="},
				{Type: TokenLeftBrace, Value: "{"},
				{Type: TokenName, Value: "y"},
				{Type: TokenOperator, Value: "="},
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenRightBrace, Value: "}"},
				{Type: TokenRightBrace, Value: "}"},
			},
		},
		{
			name:  "nested lists",
			input: "[[1,[2,3]],4]",
			expected: []Token{
				{Type: TokenLeftBracket, Value: "["},
				{Type: TokenLeftBracket, Value: "["},
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenOperator, Value: ","},
				{Type: TokenLeftBracket, Value: "["},
				{Type: TokenIntLit, Value: int64(2)},
				{Type: TokenOperator, Value: ","},
				{Type: TokenIntLit, Value: int64(3)},
				{Type: TokenRightBracket, Value: "]"},
				{Type: TokenRightBracket, Value: "]"},
				{Type: TokenOperator, Value: ","},
				{Type: TokenIntLit, Value: int64(4)},
				{Type: TokenRightBracket, Value: "]"},
			},
		},
		{
			name:  "mixed nesting",
			input: "{x=[1,{y=2}]}",
			expected: []Token{
				{Type: TokenLeftBrace, Value: "{"},
				{Type: TokenName, Value: "x"},
				{Type: TokenOperator, Value: "="},
				{Type: TokenLeftBracket, Value: "["},
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenOperator, Value: ","},
				{Type: TokenLeftBrace, Value: "{"},
				{Type: TokenName, Value: "y"},
				{Type: TokenOperator, Value: "="},
				{Type: TokenIntLit, Value: int64(2)},
				{Type: TokenRightBrace, Value: "}"},
				{Type: TokenRightBracket, Value: "]"},
				{Type: TokenRightBrace, Value: "}"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Lex(tt.input)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			compareTokens(t, tt.expected, tokens)
		})
	}
}

// TestComplexExpressions tests various complex language constructs
func TestComplexExpressions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name: "complex function with pattern matching",
			input: `f = | 0 -> 1
                    | n -> n * f(n-1)`,
			expected: []Token{
				{Type: TokenName, Value: "f"},
				{Type: TokenOperator, Value: "="},
				{Type: TokenOperator, Value: "|"},
				{Type: TokenIntLit, Value: int64(0)},
				{Type: TokenOperator, Value: "->"},
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenOperator, Value: "|"},
				{Type: TokenName, Value: "n"},
				{Type: TokenOperator, Value: "->"},
				{Type: TokenName, Value: "n"},
				{Type: TokenOperator, Value: "*"},
				{Type: TokenName, Value: "f"},
				{Type: TokenLeftParen, Value: "("},
				{Type: TokenName, Value: "n"},
				{Type: TokenOperator, Value: "-"},
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenRightParen, Value: ")"},
			},
		},
		{
			name: "record with complex fields",
			input: `{
                name = "test",
                fn = x -> x + 1,
                data = [1, 2, ...rest],
                sub = {x = 1}
            }`,
			expected: []Token{
				{Type: TokenLeftBrace, Value: "{"},
				{Type: TokenName, Value: "name"},
				{Type: TokenOperator, Value: "="},
				{Type: TokenStringLit, Value: "test"},
				{Type: TokenOperator, Value: ","},
				{Type: TokenName, Value: "fn"},
				{Type: TokenOperator, Value: "="},
				{Type: TokenName, Value: "x"},
				{Type: TokenOperator, Value: "->"},
				{Type: TokenName, Value: "x"},
				{Type: TokenOperator, Value: "+"},
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenOperator, Value: ","},
				{Type: TokenName, Value: "data"},
				{Type: TokenOperator, Value: "="},
				{Type: TokenLeftBracket, Value: "["},
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenOperator, Value: ","},
				{Type: TokenIntLit, Value: int64(2)},
				{Type: TokenOperator, Value: ","},
				{Type: TokenOperator, Value: "..."},
				{Type: TokenName, Value: "rest"},
				{Type: TokenRightBracket, Value: "]"},
				{Type: TokenOperator, Value: ","},
				{Type: TokenName, Value: "sub"},
				{Type: TokenOperator, Value: "="},
				{Type: TokenLeftBrace, Value: "{"},
				{Type: TokenName, Value: "x"},
				{Type: TokenOperator, Value: "="},
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenRightBrace, Value: "}"},
				{Type: TokenRightBrace, Value: "}"},
			},
		},
		{
			name: "complex pipeline",
			input: `data 
                |> map(x -> x + 1)
                |> filter(x -> x > 0)
                |> reduce(acc x -> acc + x, 0)`,
			expected: []Token{
				{Type: TokenName, Value: "data"},
				{Type: TokenOperator, Value: "|>"},
				{Type: TokenName, Value: "map"},
				{Type: TokenLeftParen, Value: "("},
				{Type: TokenName, Value: "x"},
				{Type: TokenOperator, Value: "->"},
				{Type: TokenName, Value: "x"},
				{Type: TokenOperator, Value: "+"},
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenRightParen, Value: ")"},
				{Type: TokenOperator, Value: "|>"},
				{Type: TokenName, Value: "filter"},
				{Type: TokenLeftParen, Value: "("},
				{Type: TokenName, Value: "x"},
				{Type: TokenOperator, Value: "->"},
				{Type: TokenName, Value: "x"},
				{Type: TokenOperator, Value: ">"},
				{Type: TokenIntLit, Value: int64(0)},
				{Type: TokenRightParen, Value: ")"},
				{Type: TokenOperator, Value: "|>"},
				{Type: TokenName, Value: "reduce"},
				{Type: TokenLeftParen, Value: "("},
				{Type: TokenName, Value: "acc"},
				{Type: TokenName, Value: "x"},
				{Type: TokenOperator, Value: "->"},
				{Type: TokenName, Value: "acc"},
				{Type: TokenOperator, Value: "+"},
				{Type: TokenName, Value: "x"},
				{Type: TokenOperator, Value: ","},
				{Type: TokenIntLit, Value: int64(0)},
				{Type: TokenRightParen, Value: ")"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Lex(tt.input)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			compareTokens(t, tt.expected, tokens)
		})
	}
}

// TestWhitespaceHandling tests various whitespace scenarios
func TestWhitespaceHandling(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "multiple newlines between tokens",
			input: "1\n\n\n+\n\n\n2",
			expected: []Token{
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenOperator, Value: "+"},
				{Type: TokenIntLit, Value: int64(2)},
			},
		},
		{
			name:  "mixed whitespace",
			input: "1 \t\n\r \n\t +\t\n \t2",
			expected: []Token{
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenOperator, Value: "+"},
				{Type: TokenIntLit, Value: int64(2)},
			},
		},
		{
			name:     "only whitespace",
			input:    "  \t\n\r  ",
			expected: []Token{},
		},
		{
			name:  "comments with whitespace",
			input: "1\n  -- comment 1  \n  -- comment 2  \n2",
			expected: []Token{
				{Type: TokenIntLit, Value: int64(1)},
				{Type: TokenIntLit, Value: int64(2)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Lex(tt.input)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			compareTokens(t, tt.expected, tokens)
		})
	}
}
