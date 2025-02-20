package main

import (
	"smel/scrapscript"
	"strings"
	"testing"
)

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
		{in: "1"},
		{in: "123"},
		{in: "-123"},
		{in: "3.14"},
		{in: "-3.14"},
		{in: "10."},
		{in: "1.0.1"},
		{in: "1 + 2"},
		{in: "1+2"},
		{in: `"hello"`},
		{in: `"hello world"`},
		{in: `"hello`},
		{in: "abc"},
		{in: "$sha1'foo"},
		{in: "[]"},
		{in: "[1,2]"},
		{in: "1\n+\t2"},
		{in: "-- 1\n2"},
		{in: ",:", error: "TODO"},
		{in: "1-2"},
		{in: "$$bills", error: "TODO"},
		{in: ".."},
		{in: "..."},
		// {in: "- ", error: "TODO"},
		{in: "-- ", error: "TODO"},
		{in: "a -> b -> a + b"},
		{in: "a->b->a+b"},
		{in: "a . b"},
		{in: "a ? b"},
		{in: "a : b"},
		{in: "~"},
		{in: "~="},
		{in: "{ x = 1, ... }"},
		{in: "{x=1,...}"},
		{in: "r@a"},
		{in: "a!b"},
		{in: "g = | 1 -> 2 | 2 -> 3"},
		{in: "1 |> f . f = a -> a + 1"},
		{in: "f >> g"},
		{in: "f << g"},
		{in: "#abc"},
		{in: "1"},
		{in: "123"},
		{in: "-123"},
		{in: "3.14"},
		{in: "-3.14"},
		{in: "10."},
		{in: "1.0.1"},
		{in: "1 + 2"},
		{in: "1+2"},
		{in: `"hello"`},
		{in: `"hello world"`},
		{in: `""`},
		{in: `"hello\"world"`},
		{in: `"hello`},
		{in: "abc"},
		{in: "$sha1'foo"},
		{in: "$$bills"},
		{in: "[]"},
		{in: "[1,2]"},
		{in: "[[1,2],[3,4]]"},
		{in: "1\n+\t2"},
		{in: "1  \n\n\t  +  \t\n  2"},
		{in: "-- 1\n2"},
		{in: "1\n-- comment1\n2\n-- comment2\n3"},
		{in: "{}"},
		{in: "{x=1}"},
		{in: "{x=1,y=2}"},
		{in: "x -> x + 1"},
		{in: "| 1 -> 2"},
		{in: "{ x, ..rest }"},
		{in: "f >> g << h"},
		{in: "0 -0 +0 0.0"},
		{in: "{x={y=1}}"},
		{in: "[[1,[2,3]],4]"},
		{in: "{x=[1,{y=2}]}"},
		{in: "1\n\n\n+\n\n\n2"},
		{in: "1 \t\n\r \n\t +\t\n \t2"},
		{in: "  \t\n\r  "},
		{in: "1\n  -- comment 1  \n  -- comment 2  \n2"},
		{in: `f = | 0 -> 1\n | n -> n * f(n-1)`},
		{in: `{ name = "test", fn = x -> x + 1, data = [1, 2, ..rest], sub = {x = 1} }`},
		{in: `data |> map(x -> x + 1) |> filter(x -> x > 0) |> reduce(acc x -> acc + x, 0)`},

		{in: "1", print: "1"},
		{in: "3.14", print: "3.14"},
		{in: "~~QUJD", print: "~~QUJD"},
		// 		{in: "~~85'K|(_", print: ""},
		// 		{in: "~~64'QUJD", print: ""},
		// 		{in: "~~32'IFBEG===", print: ""},
		// 		{in: "~~16'414243", print: ""},
		{in: "1 + 2", print: "1 + 2"},
		{in: "1 - 2", print: "1 - 2"},
		{in: `"abc" ++ "def"`, print: `"abc" ++ "def"`},
		{in: "1 >+ [2,3]", print: "1 >+ [ 2, 3 ]"},
		{in: "1 >+ 2 >+ [3,4]", print: "1 >+ 2 >+ [ 3, 4 ]"},
		{in: "[1,2] +< 3", print: "[ 1, 2 ] +< 3"},
		{in: "[1,2] +< 3 +< 4", print: "[ 1, 2 ] +< 3 +< 4"},
		{in: "[ ]", print: "[]"},
		{in: "[]", print: "[]"},
		{in: "[ 1 , 2 ]", print: "[ 1, 2 ]"},
		{in: "[ 1 + 2 , 3 + 4 ]", print: "[ 1 + 2, 3 + 4 ]"},
		{in: "a + 2 . a = 1", print: "a + 2 . a = 1"},
		{in: "a + b . a = 1 . b = 2", print: "a + b . a = 1 . b = 2"},
		{in: "a + 1 ? a == 1 . a = 1", print: "a + 1 ? a == 1 . a = 1"},
		{in: "a + b ? a == 1 ? b == 2 . a = 1 . b = 2", print: "a + b ? a == 1 ? b == 2 . a = 1 . b = 2"},
		{in: "()", print: "()"},
		{in: "(a -> b -> a + b) 3 2", print: "(a -> b -> a + b) 3 2"},
		{in: "(a -> b -> [a, b]) 3 2", print: "(a -> b -> [ a, b ]) 3 2"},
		{in: "{a = 1 + 3}", print: "{ a = 1 + 3 }"},
		{in: `rec@b . rec = { a = 1, b = "x" }`, print: `rec@b . rec = { a = 1, b = "x" }`},
		{in: "xs@1 . xs = [1, 2, 3]", print: "xs@1 . xs = [ 1, 2, 3 ]"},
		{in: "xs@y . y = 2 . xs = [1, 2, 3]", print: "xs@y . y = 2 . xs = [ 1, 2, 3 ]"},
		{in: "xs@(1+1) . xs = [1, 2, 3]", print: "xs@(1 + 1) . xs = [ 1, 2, 3 ]"},
		{in: "list_at 1 [1,2,3] . list_at = idx -> ls -> ls@idx", print: "list_at 1 [ 1, 2, 3 ] . list_at = idx -> ls -> ls@idx"},
		{in: "((a -> a + 1) >> (b -> b * 2)) 3", print: "((a -> a + 1) >> (b -> b*2)) 3"},
		{in: "((a -> a + 1) >> (x -> x) >> (b -> b * 2)) 3", print: "((a -> a + 1) >> (x -> x) >> (b -> b*2)) 3"},
		{in: "((a -> a + 1) << (b -> b * 2)) 3", print: "((a -> a + 1) << (b -> b*2)) 3"},
		{in: `inc 2 . inc = | 1 -> 2 | 2 -> 3`, print: `inc 2 . inc = | 1 -> 2 | 2 -> 3`},
		{in: `id 3 . id = | x -> x`, print: `id 3 . id = x -> x`},
		{in: `get_x rec . rec = { x = 3 } . get_x = | { x = x } -> x`, print: `get_x rec . rec = { x = 3 } . get_x = { x = x } -> x`},
		{in: `mult xs . xs = [1, 2, 3, 4, 5] . mult = | [1, x, 3, y, 5] -> x * y`, print: `mult xs . xs = [ 1, 2, 3, 4, 5 ] . mult = [ 1, x, 3, y, 5 ] -> x*y`},
		{in: "1 |> (a -> a + 2)", print: "1 |> (a -> a + 2)"},
		{in: "1 |> (a -> a + 2) |> (b -> b * 2)", print: "1 |> (a -> a + 2) |> (b -> b*2)"},
		// 		{in: "(a -> a + 2) <| 1", print: ""},
		// 		{in: "(b -> b * 2) <| (a -> a + 2) <| 1", print: ""},
		{in: `fac 5 . fac = | 0 -> 1 | 1 -> 1 | n -> n * fac (n - 1)`, print: `fac 5 . fac = | 0 -> 1 | 1 -> 1 | n -> n * fac (n - 1)`},
		{in: "6 ^ 2", print: "6^2"},
		{in: "11 % 3", print: "11 % 3"},
		{in: "# thing ()", print: "#thing ()"},
		{in: "# true ()", print: "#true ()"},
		{in: "#false ()", print: "#false ()"},
		{in: "#true () || #true () && boom", print: "#true () || #true () && boom"},
		{in: "1 < 2 && 2 < 1", print: "1 < 2 && 2 < 1"},
		{in: `f [2, 4, 6] . f = | [] -> 0 | [x, ...] -> x | c -> 1`, print: `f [ 2, 4, 6 ] . f = | [] -> 0 | [ x, ... ] -> x | c -> 1`},
		{in: `tail [1,2,3] . tail = | [first, ..rest] -> rest`, print: `tail [ 1, 2, 3 ] . tail = [ first, ..rest ] -> rest`},
		{in: `f {x = 4, y = 5} . f = | {} -> 0 | {x = a, ...} -> a | c -> 1`, print: `f { x = 4, y = 5 } . f = | {} -> 0 | { ..., x = a } -> a | c -> 1`},
		{in: `say (1 < 2) . say = | #false () -> "oh no" | #true () -> "omg"`, print: `say (1 < 2) . say = | #false () -> "oh no" | #true () -> "omg"`},
		{in: `f #add {x = 3, y = 4} . f = | # b () -> "foo" | #add {x = x, y = y} -> x + y`, print: `f #add { x = 3, y = 4 } . f = | #b () -> "foo" | #add { x = x, y = y } -> x + y`},
		{in: "1 / 2 + 3", print: "1/2 + 3"},
		{in: "{ c = 0, aaa = 0, bb = 0 }", print: "{ c = 0, bb = 0, aaa = 0 }"},
		{in: "3 * (4 + 5)", print: "3*(4 + 5)"},
		{in: "(4 + 5) * 3", print: "(4 + 5) * 3"},

		{in: "5", eval: "5"},
		{in: "3.14", eval: "3.14"},
		{in: `"xyz"`, eval: `"xyz"`},
		{in: "~~eHl6", eval: "~~eHl6"},
		{in: "no"},
		{in: "yes", env: scrapscript.Env{"yes": int64(123)}, eval: "123"},
		{in: "1 + 2", eval: "3"},
		{in: "(1 + 2) + 3", eval: "6"},
		// {in: `1 + "hello"`, wantErr: true, error: "eval Int or Float, got String"},
		{in: "1 - 2", eval: "-1"},
		{in: "2 * 3", eval: "6"},
		// {in: "3 / 9", eval: "0.3"},
		{in: "2 ^ 3", eval: "8"},
		{in: `[] ++ []`, eval: `[] ++ []`},
		{in: "10 % 4", eval: "2"},
		{in: "1 == 1", eval: "true"},
		{in: "1 == 2", eval: "false"},
		{in: "1 /= 1", eval: "false"},
		{in: "1 /= 2", eval: "true"},
		{in: `"hello" ++ " world"`, eval: `"hello world"`},
		{in: `123 ++ " world"`, error: "eval String, got Int"},
		{in: "1 >+ [2, 3]", eval: "[ 1, 2, 3 ]"},
		{in: "[1, 2] +< 3", eval: "[ 1, 2, 3 ]"},
		{in: "[1 + 2, 3 + 4]", eval: "[ 3, 7 ]"},
		{in: "x -> x", eval: "x -> x"},
		{in: "(x -> x + 1) 2", eval: "3"},
		{in: "(a -> b -> a + b) 3 2", eval: "5"},
		{in: "{a = 1 + 2}", eval: "{ a = 3 }"},
		{in: `{a = 4}@a`, eval: "4"},
		{in: `{a = 4}@b`, error: "undefined variable: b"},
		{in: "3 < 4", eval: "true"},
		{in: "3 > 4", eval: "false"},
		{in: "3 <= 4", eval: "true"},
		{in: "3 >= 4", eval: "false"},
		{in: "#true () && #false ()", eval: "false"},
		{in: "#true () || #false ()", eval: "true"},
		{in: "#abc (1 + 2)", eval: "#abc 3"},
		{in: "1.0 + 2.0", eval: "3.0"},
		{in: "1.0 / 2.0", eval: "0.5"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			lex, err := scrapscript.Lex(tt.in)
			if !strings.Contains(err.Error(), tt.error) {
				t.Fatalf("lex error containing %q, got %q", tt.error, err.Error())
			} else if err != nil {
				t.Fatalf("lex failed: %v", err)
				// } else if len(tt.lex) > 0 && tt.lex != lex {
				// 	t.Errorf("wrong lex\nwant: %#v\ngot:  %#v", tt.lex, lex)
			}

			parse, err := scrapscript.Parse(lex)
			if !strings.Contains(err.Error(), tt.error) {
				t.Fatalf("parse error containing %q, got %q", tt.error, err.Error())
			} else if err != nil {
				t.Fatalf("parse failed: %v", err)
				// } else if tt.parse != nil && tt.parse != parse {
				// t.Errorf("wrong parse\nwant: %#v\ngot:  %#v", tt.parse, parse)
			}

			print, err := scrapscript.Print(parse)
			if !strings.Contains(err.Error(), tt.error) {
				t.Fatalf("print error containing %q, got %q", tt.error, err.Error())
			} else if err != nil {
				t.Fatalf("print failed: %v", err)
			} else if tt.print != "" && tt.print != print {
				t.Errorf("wrong print\nwant: %#v\ngot:  %#v", tt.print, print)
			}

			eval_, err := scrapscript.Eval(parse, tt.env)
			if !strings.Contains(err.Error(), tt.error) {
				t.Fatalf("eval error containing %q, got %q", tt.error, err.Error())
			} else if err != nil {
				t.Fatalf("eval failed: %v", err)
			}
			eval, err := scrapscript.Print(eval_)
			if !strings.Contains(err.Error(), tt.error) {
				t.Fatalf("eval error containing %q, got %q", tt.error, err.Error())
			} else if err != nil {
				t.Fatalf("eval failed: %v", err)
			} else if tt.eval != "" && tt.eval != eval {
				t.Errorf("wrong eval\nwant: %#v\ngot:  %#v", tt.eval, eval)
			}
		})
	}
}
