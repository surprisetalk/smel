package scrapscript

import (
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
