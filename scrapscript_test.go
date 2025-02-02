/*
* All of this was copied from github:tekknolagi/scrapscript using Claude.
*
* This parallel go implementation should be thrown away when the language stabilizes.
*
 */

package main

import (
	"reflect"
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
		// {
		// 	name:     "tokenize bytes base64",
		// 	input:    "~~QUJD",
		// 	expected: []Token{{TokenBytesLit, []byte("abc")}},
		// },
		// {
		// 	name:     "tokenize bytes with explicit base",
		// 	input:    "~~85'K|(_",
		// 	expected: []Token{{TokenBytesLit, []byte("abc")}},
		// },
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
				{Type: TokenEtc, Value: nil},
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
				{Type: TokenEtc, Value: nil},
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
				{Type: TokenEtc, Value: nil},
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
		// // Bytes literals
		// {
		// 	name:     "tokenize bytes base64",
		// 	input:    "~~QUJD",
		// 	expected: []Token{{TokenBytesLit, []byte("abc")}},
		// },
		// {
		// 	name:     "tokenize bytes with base85",
		// 	input:    "~~85'K|(_",
		// 	expected: []Token{{TokenBytesLit, []byte("abc")}},
		// },
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
			input: "{ x, ..rest }",
			expected: []Token{
				{Type: TokenLeftBrace, Value: "{"},
				{Type: TokenName, Value: "x"},
				{Type: TokenOperator, Value: ","},
				{Type: TokenOperator, Value: ".."},
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
                data = [1, 2, ..rest],
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
				{Type: TokenOperator, Value: ".."},
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

func TestPrint(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"int", "1", "1"},
		{"float", "3.14", "3.14"},
		{"bytes", "~~QUJD", "~~QUJD"},
		// 		{"bytes_base85", "~~85'K|(_", ""},
		// 		{"bytes_base64", "~~64'QUJD", ""},
		// 		{"bytes_base32", "~~32'IFBEG===", ""},
		// 		{"bytes_base16", "~~16'414243", ""},
		{"int_addition", "1 + 2", "1 + 2"},
		{"int_subtraction", "1 - 2", "1 - 2"},
		{"string_concat", `"abc" ++ "def"`, `"abc" ++ "def"`},
		{"list_cons", "1 >+ [2,3]", "1 >+ [ 2, 3 ]"},
		{"list_cons_nested", "1 >+ 2 >+ [3,4]", "1 >+ 2 >+ [ 3, 4 ]"},
		{"list_append", "[1,2] +< 3", "[ 1, 2 ] +< 3"},
		{"list_append_nested", "[1,2] +< 3 +< 4", "[ 1, 2 ] +< 3 +< 4"},
		{"empty_list", "[ ]", "[]"},
		{"empty_list_no_spaces", "[]", "[]"},
		{"list_of_ints", "[ 1 , 2 ]", "[ 1, 2 ]"},
		{"list_of_exprs", "[ 1 + 2 , 3 + 4 ]", "[ 1 + 2, 3 + 4 ]"},
		{"where", "a + 2 . a = 1", "a + 2 . a = 1"},
		{"nested_where", "a + b . a = 1 . b = 2", "a + b . a = 1 . b = 2"},
		{"assert", "a + 1 ? a == 1 . a = 1", "a + 1 ? a == 1 . a = 1"},
		{"nested_assert", "a + b ? a == 1 ? b == 2 . a = 1 . b = 2", "a + b ? a == 1 ? b == 2 . a = 1 . b = 2"},
		{"hole", "()", "()"},
		{"function_app_two_args", "(a -> b -> a + b) 3 2", "(a -> b -> a + b) 3 2"},
		{"function_create_list", "(a -> b -> [a, b]) 3 2", "(a -> b -> [ a, b ]) 3 2"},
		{"create_record", "{a = 1 + 3}", "{ a = 1 + 3 }"},
		{"access_record", `rec@b . rec = { a = 1, b = "x" }`, `rec@b . rec = { a = 1, b = "x" }`},
		{"access_list", "xs@1 . xs = [1, 2, 3]", "xs@1 . xs = [ 1, 2, 3 ]"},
		{"access_list_var", "xs@y . y = 2 . xs = [1, 2, 3]", "xs@y . y = 2 . xs = [ 1, 2, 3 ]"},
		{"access_list_expr", "xs@(1+1) . xs = [1, 2, 3]", "xs@(1 + 1) . xs = [ 1, 2, 3 ]"},
		{"access_list_closure", "list_at 1 [1,2,3] . list_at = idx -> ls -> ls@idx", "list_at 1 [ 1, 2, 3 ] . list_at = idx -> ls -> ls@idx"},
		{"compose", "((a -> a + 1) >> (b -> b * 2)) 3", "((a -> a + 1) >> (b -> b*2)) 3"},
		{"double_compose", "((a -> a + 1) >> (x -> x) >> (b -> b * 2)) 3", "((a -> a + 1) >> (x -> x) >> (b -> b*2)) 3"},
		{"reverse_compose", "((a -> a + 1) << (b -> b * 2)) 3", "((a -> a + 1) << (b -> b*2)) 3"},
		{"simple_match", `inc 2 . inc = | 1 -> 2 | 2 -> 3`, `inc 2 . inc = | 1 -> 2 | 2 -> 3`},
		{"match_var", `id 3 . id = | x -> x`, `id 3 . id = x -> x`},
		{"match_record", `get_x rec . rec = { x = 3 } . get_x = | { x = x } -> x`, `get_x rec . rec = { x = 3 } . get_x = { x = x } -> x`},
		{"match_list", `mult xs . xs = [1, 2, 3, 4, 5] . mult = | [1, x, 3, y, 5] -> x * y`, `mult xs . xs = [ 1, 2, 3, 4, 5 ] . mult = [ 1, x, 3, y, 5 ] -> x*y`},
		{"pipe", "1 |> (a -> a + 2)", "1 |> (a -> a + 2)"},
		{"pipe_nested", "1 |> (a -> a + 2) |> (b -> b * 2)", "1 |> (a -> a + 2) |> (b -> b*2)"},
		// 		{"reverse_pipe", "(a -> a + 2) <| 1", ""},
		// 		{"reverse_pipe_nested", "(b -> b * 2) <| (a -> a + 2) <| 1", ""},
		{"factorial", `fac 5 . fac = | 0 -> 1 | 1 -> 1 | n -> n * fac (n - 1)`, `fac 5 . fac = | 0 -> 1 | 1 -> 1 | n -> n * fac (n - 1)`},
		{"exponentiation", "6 ^ 2", "6^2"},
		{"modulus", "11 % 3", "11 % 3"},
		{"variant", "# thing ()", "#thing ()"},
		{"variant_true", "# true ()", "#true ()"},
		{"variant_false", "#false ()", "#false ()"},
		{"boolean_ops", "#true () || #true () && boom", "#true () || #true () && boom"},
		{"compare_ops", "1 < 2 && 2 < 1", "1 < 2 && 2 < 1"},
		{"match_list_spread", `f [2, 4, 6] . f = | [] -> 0 | [x, ...] -> x | c -> 1`, `f [ 2, 4, 6 ] . f = | [] -> 0 | [ x, ... ] -> x | c -> 1`},
		{"match_list_named_spread", `tail [1,2,3] . tail = | [first, ..rest] -> rest`, `tail [ 1, 2, 3 ] . tail = [ first, ..rest ] -> rest`},
		{"match_record_spread", `f {x = 4, y = 5} . f = | {} -> 0 | {x = a, ...} -> a | c -> 1`, `f { x = 4, y = 5 } . f = | {} -> 0 | { ..., x = a } -> a | c -> 1`},
		{"match_variant", `say (1 < 2) . say = | #false () -> "oh no" | #true () -> "omg"`, `say (1 < 2) . say = | #false () -> "oh no" | #true () -> "omg"`},
		{"match_variant_record", `f #add {x = 3, y = 4} . f = | # b () -> "foo" | #add {x = x, y = y} -> x + y`, `f #add { x = 3, y = 4 } . f = | #b () -> "foo" | #add { x = x, y = y } -> x + y`},
		{"division", "1 / 2 + 3", "1/2 + 3"},
		{"division", "{ c = 0, aaa = 0, bb = 0 }", "{ c = 0, bb = 0, aaa = 0 }"},
		{"print_parens_r", "3 * (4 + 5)", "3*(4 + 5)"},
		{"print_parens_l", "(4 + 5) * 3", "(4 + 5) * 3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.input)

			tokens, err := Lex(tt.input)
			if err != nil {
				t.Fatalf("Lex failed: %v", err)
			}
			// t.Log(tokens)

			flat, err := Parse(tokens)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			// t.Log(flat)

			output, err := Print(flat)
			if err != nil {
				t.Fatalf("Print failed: %v", err)
			}

			if output != tt.expected {
				t.Fatalf("\nexpected: %v\nreceived: %v", tt.expected, output)
			}
		})
	}
}

func TestEval(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		env      Env
		expected interface{}
		wantErr  bool
		errMsg   string
	}{
		// Basic literal evaluation
		{
			name:     "eval int returns int",
			input:    "5",
			expected: int64(5),
		},
		{
			name:     "eval float returns float",
			input:    "3.14",
			expected: float64(3.14),
		},
		{
			name:     "eval string returns string",
			input:    `"xyz"`,
			expected: "xyz",
		},
		{
			name:     "eval bytes returns bytes",
			input:    "~~eHl6", // base64 for "xyz"
			expected: []byte("xyz"),
		},

		// Variable evaluation
		{
			name:    "eval non-existent var raises error",
			input:   "no",
			wantErr: true,
			errMsg:  "name 'no' is not defined",
		},
		{
			name:     "eval bound var returns value",
			input:    "yes",
			env:      Env{"yes": int64(123)},
			expected: int64(123),
		},

		// Binary operations
		{
			name:     "eval binop add returns sum",
			input:    "1 + 2",
			expected: int64(3),
		},
		{
			name:     "eval nested binop",
			input:    "(1 + 2) + 3",
			expected: int64(6),
		},
		{
			name:    "eval binop add with int string raises error",
			input:   `1 + "hello"`,
			wantErr: true,
			errMsg:  "expected Int or Float, got String",
		},
		{
			name:     "eval binop sub",
			input:    "1 - 2",
			expected: int64(-1),
		},
		{
			name:     "eval binop mul",
			input:    "2 * 3",
			expected: int64(6),
		},
		{
			name:     "eval binop div",
			input:    "3 / 10",
			expected: float64(0.3),
		},
		{
			name:     "eval binop exp",
			input:    "2 ^ 3",
			expected: int64(8),
		},
		{
			name:     "eval binop mod",
			input:    "10 % 4",
			expected: int64(2),
		},

		// Comparison operations
		{
			name:     "eval equal with equal returns true",
			input:    "1 == 1",
			expected: true,
		},
		{
			name:     "eval equal with inequal returns false",
			input:    "1 == 2",
			expected: false,
		},
		{
			name:     "eval not equal with equal returns false",
			input:    "1 /= 1",
			expected: false,
		},
		{
			name:     "eval not equal with inequal returns true",
			input:    "1 /= 2",
			expected: true,
		},

		// String operations
		{
			name:     "eval string concat returns string",
			input:    `"hello" ++ " world"`,
			expected: "hello world",
		},
		{
			name:    "eval string concat with int raises error",
			input:   `123 ++ " world"`,
			wantErr: true,
			errMsg:  "expected String, got Int",
		},

		// List operations
		{
			name:     "eval list cons returns list",
			input:    "1 >+ [2, 3]",
			expected: []interface{}{int64(1), int64(2), int64(3)},
		},
		{
			name:     "eval list append",
			input:    "[1, 2] +< 3",
			expected: []interface{}{int64(1), int64(2), int64(3)},
		},
		{
			name:     "eval list evaluates elements",
			input:    "[1 + 2, 3 + 4]",
			expected: []interface{}{int64(3), int64(7)},
		},

		// Function evaluation
		{
			name:     "eval function returns closure",
			input:    "x -> x",
			env:      Env{"a": int64(1), "b": int64(2)},
			expected: "closure",
		},
		{
			name:     "eval function application one arg",
			input:    "(x -> x + 1) 2",
			expected: int64(3),
		},
		{
			name:     "eval function application two args",
			input:    "(a -> b -> a + b) 3 2",
			expected: int64(5),
		},

		// Record operations
		{
			name:     "eval record evaluates expressions",
			input:    "{a = 1 + 2}",
			expected: map[string]interface{}{"a": int64(3)},
		},
		{
			name:     "eval record access",
			input:    `{a = 4}@a`,
			expected: int64(4),
		},
		{
			name:    "eval record access invalid field",
			input:   `{a = 4}@b`,
			wantErr: true,
			errMsg:  "no assignment to b found in record",
		},

		// Boolean operations
		{
			name:     "eval less returns bool",
			input:    "3 < 4",
			expected: true,
		},
		{
			name:     "eval greater returns bool",
			input:    "3 > 4",
			expected: false,
		},
		{
			name:     "eval less equal returns bool",
			input:    "3 <= 4",
			expected: true,
		},
		{
			name:     "eval greater equal returns bool",
			input:    "3 >= 4",
			expected: false,
		},
		{
			name:     "eval boolean and",
			input:    "#true () && #false ()",
			expected: false,
		},
		{
			name:     "eval boolean or",
			input:    "#true () || #false ()",
			expected: true,
		},

		// Variants
		{
			name:     "eval variant returns variant",
			input:    "#abc (1 + 2)",
			expected: map[string]interface{}{"abc": int64(3)},
		},

		// Float operations
		{
			name:     "eval float and float addition",
			input:    "1.0 + 2.0",
			expected: float64(3.0),
		},
		// {
		// 	name:     "eval int and float addition",
		// 	input:    "1 + 2.0",
		// 	expected: "invalid operands for +",
		// },
		{
			name:     "eval float division",
			input:    "1.0 / 2.0",
			expected: float64(0.5),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			// // Special case for closures since we can't compare functions directly
			// if tt.expected == "closure" {
			// 	if _, ok := result.(Closure); !ok {
			// 		t.Errorf("expected Closure, got %T", result)
			// 	}
			// 	return
			// }

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("wrong result\nwant: %#v\ngot:  %#v", tt.expected, result)
			}
		})
	}
}
