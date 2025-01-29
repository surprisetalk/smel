/*
* All of this was copied from github:tekknolagi/scrapscript using Claude.
*
* This parallel go implementation should be thrown away when the language stabilizes.
*
 */

package smel_test

import (
	"reflect"
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

func TestParseBasicLiterals(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Object
	}{
		{
			name:  "parse single digit",
			input: "1",
			expected: &Object{
				Type:   NodeInt,
				IntVal: 1,
			},
		},
		{
			name:  "parse multiple digits",
			input: "123",
			expected: &Object{
				Type:   NodeInt,
				IntVal: 123,
			},
		},
		{
			name:  "parse negative int",
			input: "-123",
			expected: &Object{
				Type:   NodeInt,
				IntVal: -123,
			},
		},
		{
			name:  "parse decimal",
			input: "3.14",
			expected: &Object{
				Type:     NodeFloat,
				FloatVal: 3.14,
			},
		},
		{
			name:  "parse negative decimal",
			input: "-3.14",
			expected: &Object{
				Type:     NodeFloat,
				FloatVal: -3.14,
			},
		},
		{
			name:  "parse string",
			input: `"hello"`,
			expected: &Object{
				Type:   NodeString,
				StrVal: "hello",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Lex(tt.input)
			if err != nil {
				t.Fatalf("failed to tokenize input: %v", err)
			}

			got, err := Parse(tokens)
			if err != nil {
				t.Fatalf("failed to parse tokens: %v", err)
			}

			if !objectsEqual(got, tt.expected) {
				t.Errorf("\nwant: %+v\ngot:  %+v", tt.expected, got)
			}
		})
	}
}

func TestParseVariables(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Object
	}{
		{
			name:  "parse simple variable",
			input: "abc_123",
			expected: &Object{
				Type: NodeVar,
				Name: "abc_123",
			},
		},
		{
			name:  "parse sha variable",
			input: "$sha1'abc",
			expected: &Object{
				Type: NodeVar,
				Name: "$sha1'abc",
			},
		},
		{
			name:  "parse sha variable without quote",
			input: "$sha1abc",
			expected: &Object{
				Type: NodeVar,
				Name: "$sha1abc",
			},
		},
		{
			name:  "parse dollar variable",
			input: "$",
			expected: &Object{
				Type: NodeVar,
				Name: "$",
			},
		},
		{
			name:  "parse double dollar variable",
			input: "$$",
			expected: &Object{
				Type: NodeVar,
				Name: "$$",
			},
		},
		{
			name:  "parse double dollar variable with name",
			input: "$$bills",
			expected: &Object{
				Type: NodeVar,
				Name: "$$bills",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Lex(tt.input)
			if err != nil {
				t.Fatalf("failed to tokenize input: %v", err)
			}

			got, err := Parse(tokens)
			if err != nil {
				t.Fatalf("failed to parse tokens: %v", err)
			}

			if !objectsEqual(got, tt.expected) {
				t.Errorf("\nwant: %+v\ngot:  %+v", tt.expected, got)
			}
		})
	}
}

func TestParseBinaryOperations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Object
	}{
		{
			name:  "parse binary add",
			input: "1 + 2",
			expected: &Object{
				Type: NodeBinOp,
				Op:   "+",
				Left: &Object{
					Type:   NodeInt,
					IntVal: 1,
				},
				Right: &Object{
					Type:   NodeInt,
					IntVal: 2,
				},
			},
		},
		{
			name:  "parse binary subtract",
			input: "1 - 2",
			expected: &Object{
				Type: NodeBinOp,
				Op:   "-",
				Left: &Object{
					Type:   NodeInt,
					IntVal: 1,
				},
				Right: &Object{
					Type:   NodeInt,
					IntVal: 2,
				},
			},
		},
		{
			name:  "parse chained add",
			input: "1 + 2 + 3",
			expected: &Object{
				Type: NodeBinOp,
				Op:   "+",
				Left: &Object{
					Type: NodeBinOp,
					Op:   "+",
					Left: &Object{
						Type:   NodeInt,
						IntVal: 1,
					},
					Right: &Object{
						Type:   NodeInt,
						IntVal: 2,
					},
				},
				Right: &Object{
					Type:   NodeInt,
					IntVal: 3,
				},
			},
		},
		{
			name:  "parse multiply binds tighter than add",
			input: "1 + 2 * 3",
			expected: &Object{
				Type: NodeBinOp,
				Op:   "+",
				Left: &Object{
					Type:   NodeInt,
					IntVal: 1,
				},
				Right: &Object{
					Type: NodeBinOp,
					Op:   "*",
					Left: &Object{
						Type:   NodeInt,
						IntVal: 2,
					},
					Right: &Object{
						Type:   NodeInt,
						IntVal: 3,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Lex(tt.input)
			if err != nil {
				t.Fatalf("failed to tokenize input: %v", err)
			}

			got, err := Parse(tokens)
			if err != nil {
				t.Fatalf("failed to parse tokens: %v", err)
			}

			if !objectsEqual(got, tt.expected) {
				t.Errorf("\nwant: %+v\ngot:  %+v", tt.expected, got)
			}
		})
	}
}

func TestParseListsAndRecords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Object
	}{
		{
			name:  "parse empty list",
			input: "[]",
			expected: &Object{
				Type:   NodeList,
				Params: []*Object{},
			},
		},
		{
			name:  "parse list with items",
			input: "[1, 2]",
			expected: &Object{
				Type: NodeList,
				Params: []*Object{
					{Type: NodeInt, IntVal: 1},
					{Type: NodeInt, IntVal: 2},
				},
			},
		},
		{
			name:  "parse empty record",
			input: "{}",
			expected: &Object{
				Type:   NodeRecord,
				Fields: map[string]*Object{},
			},
		},
		{
			name:  "parse record with single field",
			input: "{x = 1}",
			expected: &Object{
				Type: NodeRecord,
				Fields: map[string]*Object{
					"x": {Type: NodeInt, IntVal: 1},
				},
			},
		},
		{
			name:  "parse record with multiple fields",
			input: "{x = 1, y = 2}",
			expected: &Object{
				Type: NodeRecord,
				Fields: map[string]*Object{
					"x": {Type: NodeInt, IntVal: 1},
					"y": {Type: NodeInt, IntVal: 2},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Lex(tt.input)
			if err != nil {
				t.Fatalf("failed to tokenize input: %v", err)
			}

			got, err := Parse(tokens)
			if err != nil {
				t.Fatalf("failed to parse tokens: %v", err)
			}

			if !objectsEqual(got, tt.expected) {
				t.Errorf("\nwant: %+v\ngot:  %+v", tt.expected, got)
			}
		})
	}
}

func TestParseFunctions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Object
	}{
		{
			name:  "parse simple function",
			input: "x -> x + 1",
			expected: &Object{
				Type: NodeFunction,
				Left: &Object{Type: NodeVar, Name: "x"},
				Right: &Object{
					Type:  NodeBinOp,
					Op:    "+",
					Left:  &Object{Type: NodeVar, Name: "x"},
					Right: &Object{Type: NodeInt, IntVal: 1},
				},
			},
		},
		{
			name:  "parse function with two args",
			input: "a -> b -> a + b",
			expected: &Object{
				Type: NodeFunction,
				Left: &Object{Type: NodeVar, Name: "a"},
				Right: &Object{
					Type: NodeFunction,
					Left: &Object{Type: NodeVar, Name: "b"},
					Right: &Object{
						Type:  NodeBinOp,
						Op:    "+",
						Left:  &Object{Type: NodeVar, Name: "a"},
						Right: &Object{Type: NodeVar, Name: "b"},
					},
				},
			},
		},
		{
			name:  "parse function application",
			input: "f a",
			expected: &Object{
				Type:  NodeApply,
				Left:  &Object{Type: NodeVar, Name: "f"},
				Right: &Object{Type: NodeVar, Name: "a"},
			},
		},
		{
			name:  "parse function application two args",
			input: "f a b",
			expected: &Object{
				Type: NodeApply,
				Left: &Object{
					Type:  NodeApply,
					Left:  &Object{Type: NodeVar, Name: "f"},
					Right: &Object{Type: NodeVar, Name: "a"},
				},
				Right: &Object{Type: NodeVar, Name: "b"},
			},
		},
		{
			name:  "parse function assignment",
			input: "id = x -> x",
			expected: &Object{
				Type: NodeAssign,
				Name: "id",
				Right: &Object{
					Type:  NodeFunction,
					Left:  &Object{Type: NodeVar, Name: "x"},
					Right: &Object{Type: NodeVar, Name: "x"},
				},
			},
		},
	}

	runParseTests(t, tests)
}

func TestParseComposition(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Object
	}{
		{
			name:  "parse forward compose",
			input: "f >> g",
			expected: &Object{
				Type: NodeFunction,
				Left: &Object{Type: NodeVar, Name: "$v0"},
				Right: &Object{
					Type: NodeApply,
					Left: &Object{Type: NodeVar, Name: "g"},
					Right: &Object{
						Type:  NodeApply,
						Left:  &Object{Type: NodeVar, Name: "f"},
						Right: &Object{Type: NodeVar, Name: "$v0"},
					},
				},
			},
		},
		{
			name:  "parse backward compose",
			input: "f << g",
			expected: &Object{
				Type: NodeFunction,
				Left: &Object{Type: NodeVar, Name: "$v0"},
				Right: &Object{
					Type: NodeApply,
					Left: &Object{Type: NodeVar, Name: "f"},
					Right: &Object{
						Type:  NodeApply,
						Left:  &Object{Type: NodeVar, Name: "g"},
						Right: &Object{Type: NodeVar, Name: "$v0"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// resetGensym() // Reset symbol counter before each test
			tokens, err := Lex(tt.input)
			if err != nil {
				t.Fatalf("failed to tokenize input: %v", err)
			}

			got, err := Parse(tokens)
			if err != nil {
				t.Fatalf("failed to parse tokens: %v", err)
			}

			if !objectsEqual(got, tt.expected) {
				t.Errorf("\nwant: %+v\ngot:  %+v", tt.expected, got)
			}
		})
	}
}

func TestParsePipes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Object
	}{
		{
			name:  "parse forward pipe",
			input: "1 |> f",
			expected: &Object{
				Type:  NodeApply,
				Left:  &Object{Type: NodeVar, Name: "f"},
				Right: &Object{Type: NodeInt, IntVal: 1},
			},
		},
		{
			name:  "parse backward pipe",
			input: "f <| 1",
			expected: &Object{
				Type:  NodeApply,
				Left:  &Object{Type: NodeVar, Name: "f"},
				Right: &Object{Type: NodeInt, IntVal: 1},
			},
		},
		{
			name:  "parse nested forward pipe",
			input: "1 |> f |> g",
			expected: &Object{
				Type: NodeApply,
				Left: &Object{Type: NodeVar, Name: "g"},
				Right: &Object{
					Type:  NodeApply,
					Left:  &Object{Type: NodeVar, Name: "f"},
					Right: &Object{Type: NodeInt, IntVal: 1},
				},
			},
		},
	}

	runParseTests(t, tests)
}

func TestParseVariants(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Object
	}{
		{
			name:  "parse simple variant",
			input: "#abc 1",
			expected: &Object{
				Type:  NodeVariant,
				Name:  "abc",
				Right: &Object{Type: NodeInt, IntVal: 1},
			},
		},
		{
			name:  "parse variant with expression",
			input: "#some (1 + 2)",
			expected: &Object{
				Type: NodeVariant,
				Name: "some",
				Right: &Object{
					Type:  NodeBinOp,
					Op:    "+",
					Left:  &Object{Type: NodeInt, IntVal: 1},
					Right: &Object{Type: NodeInt, IntVal: 2},
				},
			},
		},
	}

	runParseTests(t, tests)
}

func TestParseSpecialOperators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Object
	}{
		{
			name:  "parse where",
			input: "a . b",
			expected: &Object{
				Type:  NodeWhere,
				Left:  &Object{Type: NodeVar, Name: "a"},
				Right: &Object{Type: NodeVar, Name: "b"},
			},
		},
		{
			name:  "parse nested where",
			input: "a . b . c",
			expected: &Object{
				Type: NodeWhere,
				Left: &Object{
					Type:  NodeWhere,
					Left:  &Object{Type: NodeVar, Name: "a"},
					Right: &Object{Type: NodeVar, Name: "b"},
				},
				Right: &Object{Type: NodeVar, Name: "c"},
			},
		},
		{
			name:  "parse assert",
			input: "a ? b",
			expected: &Object{
				Type:  NodeAssert,
				Left:  &Object{Type: NodeVar, Name: "a"},
				Right: &Object{Type: NodeVar, Name: "b"},
			},
		},
		{
			name:  "parse record access",
			input: "r@a",
			expected: &Object{
				Type:  NodeAccess,
				Left:  &Object{Type: NodeVar, Name: "r"},
				Right: &Object{Type: NodeVar, Name: "a"},
			},
		},
	}

	runParseTests(t, tests)
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:    "empty input",
			input:   "",
			wantErr: "empty input",
		},
		{
			name:    "unterminated string",
			input:   `"hello`,
			wantErr: "unterminated string",
		},
		{
			name:    "invalid record assignment",
			input:   "{1 = 2}",
			wantErr: "expected variable",
		},
		{
			name:    "trailing comma in list",
			input:   "[1,]",
			wantErr: "expected",
		},
		{
			name:    "missing variant name",
			input:   "#",
			wantErr: "unexpected end",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Lex(tt.input)
			if err != nil {
				// If the error occurs during tokenization, that's fine
				if tt.wantErr != "" && contains(err.Error(), tt.wantErr) {
					return
				}
				t.Fatalf("unexpected tokenization error: %v", err)
			}

			_, err = Parse(tokens)
			if err == nil {
				t.Error("expected error but got none")
				return
			}
			if !contains(err.Error(), tt.wantErr) {
				t.Errorf("want error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

// Helper functions

func runParseTests(t *testing.T, tests []struct {
	name     string
	input    string
	expected *Object
},
) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Lex(tt.input)
			if err != nil {
				t.Fatalf("failed to tokenize input: %v", err)
			}

			got, err := Parse(tokens)
			if err != nil {
				t.Fatalf("failed to parse tokens: %v", err)
			}

			if !objectsEqual(got, tt.expected) {
				t.Errorf("\nwant: %+v\ngot:  %+v", tt.expected, got)
			}
		})
	}
}

func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// Helper function to compare two AST objects
func objectsEqual(a, b *Object) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	if a.Type != b.Type {
		return false
	}

	switch a.Type {
	case NodeInt:
		return a.IntVal == b.IntVal
	case NodeFloat:
		return a.FloatVal == b.FloatVal
	case NodeString:
		return a.StrVal == b.StrVal
	case NodeVar:
		return a.Name == b.Name
	case NodeBinOp:
		return a.Op == b.Op && objectsEqual(a.Left, b.Left) && objectsEqual(a.Right, b.Right)
	case NodeList:
		if len(a.Params) != len(b.Params) {
			return false
		}
		for i := range a.Params {
			if !objectsEqual(a.Params[i], b.Params[i]) {
				return false
			}
		}
		return true
	case NodeRecord:
		if len(a.Fields) != len(b.Fields) {
			return false
		}
		for k, v1 := range a.Fields {
			v2, ok := b.Fields[k]
			if !ok || !objectsEqual(v1, v2) {
				return false
			}
		}
		return true
	default:
		return a.Name == b.Name && objectsEqual(a.Left, b.Left) && objectsEqual(a.Right, b.Right)
	}
}

func TestMatch(t *testing.T) {
	tests := []struct {
		name    string
		obj     *Object
		pattern *Object
		want    Env
		wantErr bool
		errStr  string
	}{
		{
			name:    "match hole with non-hole returns none",
			obj:     &Object{Type: NodeInt, IntVal: 1},
			pattern: &Object{Type: NodeHole},
			want:    nil,
		},
		{
			name:    "match hole with hole returns empty dict",
			obj:     &Object{Type: NodeHole},
			pattern: &Object{Type: NodeHole},
			want:    Env{},
		},
		{
			name:    "match with equal ints returns empty dict",
			obj:     &Object{Type: NodeInt, IntVal: 1},
			pattern: &Object{Type: NodeInt, IntVal: 1},
			want:    Env{},
		},
		{
			name:    "match with inequal ints returns none",
			obj:     &Object{Type: NodeInt, IntVal: 2},
			pattern: &Object{Type: NodeInt, IntVal: 1},
			want:    nil,
		},
		{
			name:    "match int with non-int returns none",
			obj:     &Object{Type: NodeString, StrVal: "abc"},
			pattern: &Object{Type: NodeInt, IntVal: 1},
			want:    nil,
		},
		{
			name:    "match with equal floats raises match error",
			obj:     &Object{Type: NodeFloat, FloatVal: 1},
			pattern: &Object{Type: NodeFloat, FloatVal: 1},
			wantErr: true,
			errStr:  "pattern matching is not supported for Floats",
		},
		{
			name:    "match with equal strings returns empty dict",
			obj:     &Object{Type: NodeString, StrVal: "a"},
			pattern: &Object{Type: NodeString, StrVal: "a"},
			want:    Env{},
		},
		{
			name:    "match with inequal strings returns none",
			obj:     &Object{Type: NodeString, StrVal: "b"},
			pattern: &Object{Type: NodeString, StrVal: "a"},
			want:    nil,
		},
		{
			name:    "match string with non-string returns none",
			obj:     &Object{Type: NodeInt, IntVal: 1},
			pattern: &Object{Type: NodeString, StrVal: "abc"},
			want:    nil,
		},
		{
			name:    "match var returns dict with var name",
			obj:     &Object{Type: NodeString, StrVal: "abc"},
			pattern: &Object{Type: NodeVar, Name: "a"},
			want:    Env{"a": &Object{Type: NodeString, StrVal: "abc"}},
		},
		{
			name: "match record with non-record returns none",
			obj:  &Object{Type: NodeInt, IntVal: 2},
			pattern: &Object{
				Type: NodeRecord,
				Fields: map[string]*Object{
					"x": {Type: NodeVar, Name: "x"},
					"y": {Type: NodeVar, Name: "y"},
				},
			},
			want: nil,
		},
		{
			name: "match record with vars returns dict with keys",
			obj: &Object{
				Type: NodeRecord,
				Fields: map[string]*Object{
					"x": {Type: NodeInt, IntVal: 1},
					"y": {Type: NodeInt, IntVal: 2},
				},
			},
			pattern: &Object{
				Type: NodeRecord,
				Fields: map[string]*Object{
					"x": {Type: NodeVar, Name: "x"},
					"y": {Type: NodeVar, Name: "y"},
				},
			},
			want: Env{
				"x": &Object{Type: NodeInt, IntVal: 1},
				"y": &Object{Type: NodeInt, IntVal: 2},
			},
		},
		{
			name: "match list with vars returns dict with keys",
			obj: &Object{
				Type: NodeList,
				Params: []*Object{
					{Type: NodeInt, IntVal: 1},
					{Type: NodeInt, IntVal: 2},
				},
			},
			pattern: &Object{
				Type: NodeList,
				Params: []*Object{
					{Type: NodeVar, Name: "x"},
					{Type: NodeVar, Name: "y"},
				},
			},
			want: Env{
				"x": &Object{Type: NodeInt, IntVal: 1},
				"y": &Object{Type: NodeInt, IntVal: 2},
			},
		},
		{
			name: "match list with spread returns empty dict",
			obj: &Object{
				Type: NodeList,
				Params: []*Object{
					{Type: NodeInt, IntVal: 1},
					{Type: NodeInt, IntVal: 2},
					{Type: NodeInt, IntVal: 3},
				},
			},
			pattern: &Object{
				Type: NodeList,
				Params: []*Object{
					{Type: NodeInt, IntVal: 1},
					{Type: NodeSpread},
				},
			},
			want: Env{},
		},
		{
			name: "match list with named spread",
			obj: &Object{
				Type: NodeList,
				Params: []*Object{
					{Type: NodeInt, IntVal: 1},
					{Type: NodeInt, IntVal: 2},
					{Type: NodeInt, IntVal: 3},
					{Type: NodeInt, IntVal: 4},
				},
			},
			pattern: &Object{
				Type: NodeList,
				Params: []*Object{
					{Type: NodeVar, Name: "a"},
					{Type: NodeInt, IntVal: 2},
					{Type: NodeSpread, Name: "rest"},
				},
			},
			want: Env{
				"a": &Object{Type: NodeInt, IntVal: 1},
				"rest": &Object{
					Type: NodeList,
					Params: []*Object{
						{Type: NodeInt, IntVal: 3},
						{Type: NodeInt, IntVal: 4},
					},
				},
			},
		},
		{
			name: "match record with spread returns empty dict",
			obj: &Object{
				Type: NodeRecord,
				Fields: map[string]*Object{
					"a": {Type: NodeInt, IntVal: 1},
					"b": {Type: NodeInt, IntVal: 2},
				},
			},
			pattern: &Object{
				Type: NodeRecord,
				Fields: map[string]*Object{
					"a":   {Type: NodeInt, IntVal: 1},
					"...": {Type: NodeSpread},
				},
			},
			want: Env{},
		},
		{
			name: "match variant with equal tag returns empty dict",
			obj: &Object{
				Type:  NodeVariant,
				Name:  "abc",
				Right: &Object{Type: NodeHole},
			},
			pattern: &Object{
				Type:  NodeVariant,
				Name:  "abc",
				Right: &Object{Type: NodeHole},
			},
			want: Env{},
		},
		{
			name: "match variant with inequal tag returns none",
			obj: &Object{
				Type:  NodeVariant,
				Name:  "def",
				Right: &Object{Type: NodeHole},
			},
			pattern: &Object{
				Type:  NodeVariant,
				Name:  "abc",
				Right: &Object{Type: NodeHole},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Match(tt.obj, tt.pattern)
			if tt.wantErr {
				if err == nil {
					t.Errorf("match() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if err.Error() != tt.errStr {
					t.Errorf("match() error = %v, wantErr %v", err, tt.errStr)
					return
				}
				return
			}
			if err != nil {
				t.Errorf("match() unexpected error: %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("match() = %v, want %v", got, tt.want)
			}
		})
	}
}
