package main

import (
	"smel/scrapscript"
	"strings"
	"testing"
)

// Returns true if test contains expected error and is complete.
func checkTest(k string, t *testing.T, expectedError string, err error) bool {
	if err != nil {
		if expectedError != "" {
			if strings.Contains(err.Error(), expectedError) {
				return true
			} else {
				t.Fatalf("%s error containing %q, got %q", k, expectedError, err.Error())
			}
		} else {
			t.Fatalf("%s failed: %v", k, err)
		}
	}
	return false
}

func TestScrapscript(t *testing.T) {
	tests := []struct {
		in string
		// lex   []string
		parse scrapscript.Flat
		print string
		eval  string
		error string
		env   scrapscript.Env
	}{
		{in: "1.0 / 2.0", eval: "0.5"},
		{in: "1.0 + 2.0", eval: "3.0"},
		{in: "#abc (1 + 2)", eval: "#abc 3"},
		{in: "#true () || #false ()", eval: "true"},
		{in: "#true () && #false ()", eval: "false"},
		{in: "3 >= 4", eval: "false"},
		{in: "3 <= 4", eval: "true"},
		{in: "3 > 4", eval: "false"},
		{in: "3 < 4", eval: "true"},
		{in: `{a = 4}@a`, eval: "4"},
		{in: `{a = 4}@b`, eval: "()"},
		{in: "{a = 1 + 2}", eval: "{ a = 3 }"},
		{in: "(x -> x + 1) 2", eval: "3"},
		{in: "x -> x", eval: "x -> x"},
		{in: "[1 + 2, 3 + 4]", eval: "[ 3, 7 ]"},
		{in: "[1, 2] +< 3", eval: "[ 1, 2, 3 ]"},
		{in: "1 >+ [2, 3]", eval: "[ 1, 2, 3 ]"},
		{in: `123 ++ " world"`, error: "expected list"},
		{in: `"hello" ++ " world"`, eval: `"hello world"`},
		{in: "1 /= 2", eval: "true"},
		{in: "1 /= 1", eval: "false"},
		{in: "1 == 2", eval: "false"},
		{in: "1 == 1", eval: "true"},
		{in: "10 % 4", eval: "2"},
		{in: `[] ++ []`, eval: `[]`},
		{in: "2 ^ 3", eval: "8"},
		// {in: "3 / 9", eval: "0.3"},
		{in: "2 * 3", eval: "6"},
		{in: "1 - 2", eval: "-1"},
		// {in: `1 + "hello"`, wantErr: true, error: "eval Int or Float, got String"},
		{in: "(1 + 2) + 3", eval: "6"},
		{in: "1 + 2", eval: "3"},
		{in: "yes", env: scrapscript.Env{"yes": int64(123)}, eval: "123"},
		{in: "no", error: "undefined variable"},
		{in: "~~eHl6", eval: "~~eHl6"},
		{in: `"xyz"`, eval: `"xyz"`},
		{in: "3.14", eval: "3.14"},
		{in: "5", eval: "5"},

		{in: "(4 + 5) * 3", print: "(4 + 5) * 3"},
		{in: "3 * (4 + 5)", print: "3*(4 + 5)"},
		{in: "{ c = 0, aaa = 0, bb = 0 }", print: "{ c = 0, bb = 0, aaa = 0 }"},
		{in: "1.0 / 2.0 + 3.0", print: "1.0/2.0 + 3.0", eval: "3.5"},
		{in: `f (#add {x = 3, y = 4}) . f = | # b () -> "foo" | #add {x = x, y = y} -> x + y`, print: `f (#add { x = 3, y = 4 }) . f = | #b () -> "foo" | #add { x = x, y = y } -> x + y`},
		{in: `say (1 < 2) . say = | #false () -> "oh no" | #true () -> "omg"`, print: `say (1 < 2) . say = | #false () -> "oh no" | #true () -> "omg"`},
		{in: `f {x = 4, y = 5} . f = | {} -> 0 | {x = a, ...} -> a | c -> 1`, print: `f { x = 4, y = 5 } . f = | {} -> 0 | { ..., x = a } -> a | c -> 1`},
		{in: `tail [1,2,3] . tail = | [first, ..rest] -> rest`, print: `tail [ 1, 2, 3 ] . tail = [ first, ..rest ] -> rest`},
		{in: `f [2, 4, 6] . f = | [] -> 0 | [x, ...] -> x | c -> 1`, print: `f [ 2, 4, 6 ] . f = | [] -> 0 | [ x, ... ] -> x | c -> 1`},
		{in: "1 < 2 && 2 < 1", print: "1 < 2 && 2 < 1"},
		{in: "#true () || #true () && boom", print: "#true () || #true () && boom"},
		{in: "#false ()", print: "#false ()"},
		{in: "# true ()", print: "#true ()"},
		{in: "# thing ()", print: "#thing ()"},
		{in: "11 % 3", print: "11 % 3"},
		{in: "6 ^ 2", print: "6^2"},
		{in: `fac 5 . fac = | 0 -> 1 | 1 -> 1 | n -> n * fac (n - 1)`, print: `fac 5 . fac = | 0 -> 1 | 1 -> 1 | n -> n * fac (n - 1)`},
		// 		{in: "(b -> b * 2) <| (a -> a + 2) <| 1", print: ""},
		// 		{in: "(a -> a + 2) <| 1", print: ""},
		{in: "1 |> (a -> a + 2) |> (b -> b * 2)", print: "1 |> (a -> a + 2) |> (b -> b*2)"},
		{in: "1 |> (a -> a + 2)", print: "1 |> (a -> a + 2)"},
		{in: `mult xs . xs = [1, 2, 3, 4, 5] . mult = | [1, x, 3, y, 5] -> x * y`, print: `mult xs . xs = [ 1, 2, 3, 4, 5 ] . mult = [ 1, x, 3, y, 5 ] -> x*y`},
		{in: `get_x rec . rec = { x = 3 } . get_x = | { x = x } -> x`, print: `get_x rec . rec = { x = 3 } . get_x = { x = x } -> x`},
		{in: `id 3 . id = | x -> x`, print: `id 3 . id = x -> x`},
		{in: `inc 2 . inc = | 1 -> 2 | 2 -> 3`, print: `inc 2 . inc = | 1 -> 2 | 2 -> 3`},
		{in: "((a -> a + 1) << (b -> b * 2)) 3", print: "((a -> a + 1) << (b -> b*2)) 3"},
		{in: "((a -> a + 1) >> (x -> x) >> (b -> b * 2)) 3", print: "((a -> a + 1) >> (x -> x) >> (b -> b*2)) 3"},
		{in: "((a -> a + 1) >> (b -> b * 2)) 3", print: "((a -> a + 1) >> (b -> b*2)) 3"},
		{in: "list_at 1 [1,2,3] . list_at = idx -> ls -> ls@idx", print: "list_at 1 [ 1, 2, 3 ] . list_at = idx -> ls -> ls@idx"},
		{in: "xs@(1+1) . xs = [1, 2, 3]", print: "xs@(1 + 1) . xs = [ 1, 2, 3 ]"},
		{in: "xs@y . y = 2 . xs = [1, 2, 3]", print: "xs@y . y = 2 . xs = [ 1, 2, 3 ]"},
		{in: "xs@1 . xs = [1, 2, 3]", print: "xs@1 . xs = [ 1, 2, 3 ]"},
		{in: `rec@b . rec = { a = 1, b = "x" }`, print: `rec@b . rec = { a = 1, b = "x" }`},
		{in: "{a = 1 + 3}", print: "{ a = 1 + 3 }", eval: "{ a = 4 }"},
		{in: "(a -> b -> [a, b]) 3 2", print: "(a -> b -> [ a, b ]) 3 2"},
		{in: "(a -> b -> a + b) 3 2", print: "(a -> b -> a + b) 3 2", eval: "5"},
		{in: "()", print: "()"},
		{in: "a + b ? a == 1 ? b == 2 . a = 1 . b = 2", print: "a + b ? a == 1 ? b == 2 . a = 1 . b = 2"},
		{in: "a + 1 ? a == 1 . a = 1", print: "a + 1 ? a == 1 . a = 1"},
		{in: "a + b . a = 1 . b = 2", print: "a + b . a = 1 . b = 2"},
		{in: "a + 2 . a = 1", print: "a + 2 . a = 1", eval: "3"},
		{in: "[ 1 + 2 , 3 + 4 ]", print: "[ 1 + 2, 3 + 4 ]", eval: "[ 3, 7 ]"},
		{in: "[ 1 , 2 ]", print: "[ 1, 2 ]"},
		{in: "[]", print: "[]"},
		{in: "[ ]", print: "[]"},
		{in: "[1,2] +< 3 +< 4", print: "[ 1, 2 ] +< 3 +< 4"},
		{in: "[1,2] +< 3", print: "[ 1, 2 ] +< 3"},
		{in: "1 >+ 2 >+ [3,4]", print: "1 >+ 2 >+ [ 3, 4 ]"},
		{in: "1 >+ [2,3]", print: "1 >+ [ 2, 3 ]"},
		{in: `"abc" ++ "def"`, print: `"abc" ++ "def"`},
		{in: "1 - 2", print: "1 - 2"},
		{in: "1 + 2", print: "1 + 2"},
		// 		{in: "~~16'414243", print: ""},
		// 		{in: "~~32'IFBEG===", print: ""},
		// 		{in: "~~64'QUJD", print: ""},
		// 		{in: "~~85'K|(_", print: ""},
		{in: "~~QUJD", print: "~~QUJD"},
		{in: "3.14", print: "3.14"},
		{in: "1", print: "1"},

		{in: `data |> map(x -> x + 1) |> filter(x -> x > 0) |> reduce(acc x -> acc + x, 0)`},
		{in: `{ name = "test", fn = x -> x + 1, data = [1, 2, ..rest], sub = {x = 1} }`},
		{in: `f = | 0 -> 1\n | n -> n * f(n-1)`},
		{in: "1\n  -- comment 1  \n  -- comment 2  \n2"},
		{in: "  \t\n\r  "},
		{in: "1 \t\n\r \n\t +\t\n \t2"},
		{in: "1\n\n\n+\n\n\n2"},
		{in: "{x=[1,{y=2}]}"},
		{in: "[[1,[2,3]],4]"},
		{in: "{x={y=1}}"},
		{in: "0 -0 +0 0.0"},
		{in: "f >> g << h"},
		{in: "{ x, ..rest }"},
		{in: "| 1 -> 2"},
		{in: "x -> x + 1"},
		{in: "{x=1,y=2}"},
		{in: "{x=1}"},
		{in: "{}"},
		{in: "1\n-- comment1\n2\n-- comment2\n3"},
		{in: "-- 1\n2"},
		{in: "1  \n\n\t  +  \t\n  2"},
		{in: "1\n+\t2"},
		{in: "[[1,2],[3,4]]"},
		{in: "[1,2]"},
		{in: "[]"},
		{in: "$$bills"},
		{in: "$sha1'foo"},
		{in: "abc"},
		{in: `"hello`},
		{in: `"hello\"world"`},
		{in: `""`},
		{in: `"hello world"`},
		{in: `"hello"`},
		{in: "1+2"},
		{in: "1 + 2"},
		{in: "1.0.1"},
		{in: "10."},
		{in: "-3.14"},
		{in: "3.14"},
		{in: "-123"},
		{in: "123"},
		{in: "1"},
		{in: "#abc"},
		{in: "f << g"},
		{in: "f >> g"},
		{in: "1 |> f . f = a -> a + 1"},
		{in: "g . g = | 1 -> 2 | 2 -> 3"},
		// {in: "a!b"},
		{in: "r@a . r = { a = 1 }", print: "r@a . r = { a = 1 }", eval: "1"},
		{in: "123 -> {x=1,..{}}", print: "123 -> { .. {}, x = 1 }"},
		{in: "123 -> { x = 1, ..{} }", print: "123 -> { .. {}, x = 1 }"},
		{in: "a -> {x=1,..a}", print: "a -> { ..a, x = 1 }"},
		{in: "a -> { x = 1, ..a }", print: "a -> { ..a, x = 1 }"},
		{in: "{x=1,...} -> 123"},
		{in: "{ x = 1, ... } -> 123"},
		// {in: "~="},
		// {in: "~"},
		// {in: "a : b"},
		// {in: "a ? b"},
		// {in: "a . b"},
		{in: "a->b->a+b"},
		{in: "a -> b -> a + b"},
		{in: "-- ", error: "empty input"},
		// {in: "- ", error: "TODO"},
		{in: "..."},
		{in: "..", error: "unexpected Token"},
		{in: "$$bills", error: "undefined variable"},
		{in: "1-2"},
		{in: ",:", error: "unexpected Token"},
		{in: "-- 1\n2"},
		{in: "1\n+\t2"},
		{in: "[1,2]"},
		{in: "[]"},
		{in: "$sha1'foo", error: "undefined variable"},
		{in: "abc", error: "undefined variable"},
		{in: `"hello`, error: "unterminated"},
		{in: `"hello world"`},
		{in: `"hello"`},
		{in: "1+2"},
		{in: "1 + 2"},
		{in: "1.0.1", error: "unexpected character"},
		{in: "10."},
		{in: "-3.14"},
		{in: "3.14"},
		{in: "-123"},
		{in: "123"},
		{in: "1"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			lex, err := scrapscript.Lex(tt.in)
			if checkTest("lex", t, tt.error, err) {
				return
			}

			parse, err := scrapscript.Parse(lex)
			if checkTest("parse", t, tt.error, err) {
				return
			}

			print, err := scrapscript.Print(parse)
			if checkTest("print", t, tt.error, err) {
				return
			} else if tt.print != "" && tt.print != print {
				t.Errorf("wrong print\nwant: %#v\ngot:  %#v", tt.print, print)
			}

			eval_, err := scrapscript.Eval(parse, tt.env)
			if checkTest("eval", t, tt.error, err) {
				return
			}
			eval, err := scrapscript.Print(eval_)
			if checkTest("eval", t, tt.error, err) {
				return
			} else if tt.eval != "" && tt.eval != eval {
				t.Errorf("wrong eval\nwant: %#v\ngot:  %#v", tt.eval, eval)
			}

			if tt.error != "" {
				t.Errorf("expected error: %v", tt.error)
			}
		})
	}
}
