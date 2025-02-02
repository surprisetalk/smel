package scrapscript

import (
	"reflect"
	"strings"
	"testing"
)

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
