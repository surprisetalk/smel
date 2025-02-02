package scrapscript

import (
	"reflect"
	"strings"
	"testing"
)

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
