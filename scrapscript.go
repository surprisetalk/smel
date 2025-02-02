/*
* All of this was copied from github:tekknolagi/scrapscript using Claude.
*
* This parallel go implementation should be thrown away when the language stabilizes.
*
 */

package main

import (
	"cmp"
	"encoding/base64"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/fxamacker/cbor/v2"
)

//// FLAT

type Flat = cbor.RawMessage

type TagType = uint64

const (
	TagExpr TagType = ' '
	TagOp   TagType = '+'
	TagVar  TagType = '='
	TagTag  TagType = '#'
	TagDict TagType = '\''
	TagFun  TagType = '|'
	TagEtc  TagType = '.'
)

func tagOp(op string) Flat {
	// TODO: Do NOT store this as a string! So inefficient.
	op_, err := cbor.Marshal(cbor.Tag{TagOp, op})
	if err != nil {
		panic(err)
	}
	return op_
}

//// LEX

type TokenType int

const (
	TokenEOF TokenType = iota
	TokenHash
	TokenLeftParen
	TokenRightParen
	TokenLeftBrace
	TokenRightBrace
	TokenLeftBracket
	TokenRightBracket
	TokenOperator
	TokenName
	TokenStringLit
	TokenIntLit
	TokenFloatLit
	TokenBytesLit
	TokenEtc
)

type Token struct {
	Type  TokenType
	Value interface{}
}

type lexer struct {
	text string
	pos  int
}

func (l *lexer) hasInput() bool {
	return l.pos < len(l.text)
}

func (l *lexer) peek() byte {
	if !l.hasInput() {
		return 0
	}
	return l.text[l.pos]
}

func (l *lexer) advance() {
	l.pos++
}

func (l *lexer) readWhile(pred func(byte) bool) string {
	var result strings.Builder
	for l.hasInput() && pred(l.peek()) {
		result.WriteByte(l.peek())
		l.advance()
	}
	return result.String()
}

// validOperators contains all valid operators, organized by length for efficient lookup
var validOperators = struct {
	len1 map[string]bool
	len2 map[string]bool
	len3 map[string]bool
}{
	len1: map[string]bool{
		"+": true, "-": true, "*": true, "/": true, "^": true, "%": true,
		"<": true, ">": true, "!": true, ".": true, "=": true, ",": true,
		":": true, "?": true, "|": true, "@": true,
	},
	len2: map[string]bool{
		"++": true, "+<": true, ">+": true, "==": true, "/=": true, "<=": true,
		">=": true, "&&": true, "||": true, "->": true, "..": true, ">>": true,
		"<<": true, "|>": true,
	},
	len3: map[string]bool{
		"...": true,
	},
}

func (l *lexer) readOperator() (Token, error) {
	// Try to read a 3-character operator
	if l.pos+2 < len(l.text) {
		op3 := string(l.text[l.pos]) + string(l.text[l.pos+1]) + string(l.text[l.pos+2])
		if validOperators.len3[op3] {
			l.pos += 3
			if op3 == "..." {
				return Token{Type: TokenEtc, Value: nil}, nil
			}
			return Token{Type: TokenOperator, Value: op3}, nil
		}
	}

	// Try to read a 2-character operator
	if l.pos+1 < len(l.text) {
		op2 := string(l.text[l.pos]) + string(l.text[l.pos+1])
		if validOperators.len2[op2] {
			l.pos += 2
			return Token{Type: TokenOperator, Value: op2}, nil
		}
	}

	// Try to read a 1-character operator
	op1 := string(l.text[l.pos])
	if validOperators.len1[op1] {
		l.pos++
		return Token{Type: TokenOperator, Value: op1}, nil
	}

	return Token{}, fmt.Errorf("invalid operator: %s", op1)
}

func (l *lexer) readBytes() (Token, error) {
	l.advance() // skip second ~
	var str strings.Builder
	str.WriteString(l.readWhile(func(c byte) bool {
		return !unicode.IsSpace(rune(c))
	}))
	data, err := base64.StdEncoding.DecodeString(str.String())
	if err != nil {
		return Token{}, fmt.Errorf("base64 problem: %v", err)
	}
	return Token{TokenBytesLit, data}, nil
}

func (l *lexer) nextToken() (Token, error) {
	l.readWhile(func(c byte) bool {
		return c == ' ' || c == '\t' || c == '\n' || c == '\r'
	})

	if !l.hasInput() {
		return Token{Type: TokenEOF}, nil
	}

	c := l.peek()

	switch {
	case c == '"':
		l.advance() // skip opening quote
		var str strings.Builder

		for l.hasInput() && l.peek() != '"' {
			if l.hasInput() && l.peek() == '\\' {
				l.advance()
				custom := map[byte]byte{
					'"': '"',
					'n': '\n',
					't': '\t',
					'r': '\r',
				}
				if _, ok := custom[l.peek()]; !ok {
					return Token{}, fmt.Errorf("TODO")
				}
				str.WriteByte(custom[l.peek()])
				l.advance()
			} else {
				str.WriteByte(l.peek())
				l.advance()
			}
		}

		if !l.hasInput() {
			return Token{}, fmt.Errorf("unterminated string")
		}
		l.advance() // skip closing quote
		return Token{Type: TokenStringLit, Value: str.String()}, nil
	case c >= '0' && c <= '9':
		var num strings.Builder
		isFloat := false

		for l.hasInput() {
			c := l.peek()
			if c == '.' {
				if isFloat {
					return Token{}, fmt.Errorf("unexpected character '.'")
				}
				isFloat = true
				num.WriteByte(c)
				l.advance()
			} else if c >= '0' && c <= '9' {
				num.WriteByte(c)
				l.advance()
			} else {
				break
			}
		}

		str := num.String()
		if isFloat {
			val, err := strconv.ParseFloat(str, 64)
			if err != nil {
				return Token{}, err
			}
			return Token{Type: TokenFloatLit, Value: val}, nil
		}

		val, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return Token{}, err
		}
		return Token{Type: TokenIntLit, Value: val}, nil

	case c == '#':
		l.advance()
		return Token{Type: TokenHash, Value: "#"}, nil
	case c == '~':
		l.advance()
		if l.hasInput() && l.peek() == '~' {
			return l.readBytes()
		}
		return Token{}, fmt.Errorf("unexpected character '~'")
	case c == '-':
		if l.pos+1 < len(l.text) && l.text[l.pos+1] == '-' {
			// Skip comment until newline
			l.advance()
			l.advance()
			for l.hasInput() && l.peek() != '\n' {
				l.advance()
			}
			return l.nextToken()
		}
		return l.readOperator()
	case strings.ContainsRune("()[]{}", rune(c)):
		l.advance()
		custom := map[byte]TokenType{
			'(': TokenLeftParen,
			')': TokenRightParen,
			'{': TokenLeftBrace,
			'}': TokenRightBrace,
			'[': TokenLeftBracket,
			']': TokenRightBracket,
		}
		return Token{Type: custom[c], Value: string(c)}, nil
	case strings.ContainsRune("+-*/<>=!&|.,:|?@^%", rune(c)):
		return l.readOperator()
	case unicode.IsLetter(rune(c)) || c == '$' || c == '_':
		id := l.readWhile(func(c byte) bool {
			return unicode.IsLetter(rune(c)) || unicode.IsDigit(rune(c)) || c == '$' || c == '\'' || c == '_'
		})
		return Token{Type: TokenName, Value: id}, nil
	}

	return Token{}, fmt.Errorf("unexpected character: %c", c)
}

func Lex(input string) ([]Token, error) {
	l := &lexer{text: input}
	var Tokens []Token

	for {
		Token, err := l.nextToken()
		if err != nil {
			return nil, err
		}
		if Token.Type == TokenEOF {
			break
		}
		Tokens = append(Tokens, Token)
	}

	return Tokens, nil
}

//// PARSE

type prec struct {
	pl float64
	pr float64
}

var precs = map[string]prec{
	"::": {2000, 1999.9},
	"..": {1500, 1500.1},
	"@":  {1002, 1002.1},
	" ":  {1000, 1000.1},
	">>": {14, 13.9},
	"<<": {14, 13.9},
	"^":  {13, 13.1},
	"*":  {12, 12.1},
	"/":  {12, 12.1},
	"//": {12, 11.9},
	"%":  {12, 11.9},
	"+":  {11, 10.9},
	"-":  {11, 10.9},
	">*": {10, 10.1},
	"++": {10, 10.1},
	">+": {10, 9.9},
	"+<": {10, 10.1},
	"==": {9, 9},
	"/=": {9, 9},
	"<":  {9, 9},
	">":  {9, 9},
	"<=": {9, 9},
	">=": {9, 9},
	"&&": {8, 8.1},
	"||": {7, 7.1},
	"|>": {6, 6.1},
	"#":  {5.5, 5.4},
	"->": {5, 4.9},
	"|":  {4.5, 4.6},
	":":  {4.5, 4.4},
	"=":  {4, 4.1},
	"!":  {3, 2.9},
	".":  {3, 3.1},
	"?":  {3, 3.1},
	// ",":  {1, 0},
}

const highestPrec = 100.0

var symCounter = -1

func gensym() string {
	symCounter++
	return fmt.Sprintf("$v%d", symCounter)
}

func resetGensym() {
	symCounter = -1
}

type parser struct {
	Tokens []Token
	pos    int
}

func (p *parser) peek() *Token {
	if p.pos >= len(p.Tokens) {
		return nil
	}
	return &p.Tokens[p.pos]
}

func (p *parser) next() *Token {
	if p.pos >= len(p.Tokens) {
		return nil
	}
	token := &p.Tokens[p.pos]
	p.pos++
	return token
}

func value(flat Flat, err error) ([]Flat, error) {
	return []Flat{flat}, err
}

func expr(flats []Flat, err error) (Flat, error) {
	if err != nil {
		return nil, err
	}
	if flats == nil {
		return nil, fmt.Errorf("something went wrong")
	}
	l := len(flats)
	if l == 0 {
		return nil, fmt.Errorf("empty expression")
	} else if l == 1 {
		return flats[0], err
	}
	return cbor.Marshal(cbor.Tag{TagExpr, flats})
}

func (p *parser) parseUnary(prec float64) ([]Flat, error) {
	token := p.next()
	if token == nil {
		return nil, fmt.Errorf("unexpected end of input")
	}

	switch token.Type {
	case TokenIntLit, TokenFloatLit, TokenStringLit, TokenBytesLit:
		return value(cbor.Marshal(token.Value))

	case TokenName:
		if token.Value == "true" {
			return value(cbor.Marshal(true))
		}
		if token.Value == "false" {
			return value(cbor.Marshal(false))
		}
		return value(cbor.Marshal(cbor.Tag{TagVar, token.Value}))

	case TokenHash:
		tag := p.next()
		if tag == nil {
			return nil, fmt.Errorf("unexpected end")
		}
		if tag.Type != TokenName {
			return nil, fmt.Errorf("expected name after #")
		}
		return value(cbor.Marshal(cbor.Tag{TagTag, tag.Value}))

	case TokenLeftParen:
		if next := p.peek(); next != nil && next.Type == TokenRightParen {
			p.next() // consume )
			return value(cbor.Marshal(nil))
		}
		ex, err := p.parseBinary(0)
		if err != nil {
			return nil, err
		}
		if next := p.next(); next == nil || next.Type != TokenRightParen {
			return nil, fmt.Errorf("expected )")
		}
		return ex, nil

	case TokenLeftBracket:
		list := make([]Flat, 0)
		// TODO: Handle spread.
		for {
			{
				next := p.peek()
				if next == nil {
					return nil, fmt.Errorf("unfinished list")
				}
				if next.Type == TokenRightBracket {
					p.next()
					break
				}
				item, err := expr(p.parseBinary(precs[","].pr + 1))
				if err != nil {
					return nil, err
				}
				list = append(list, item)
			}
			{
				next := p.next()
				if next.Type == TokenRightBracket {
					break
				}
				if next.Type != TokenOperator || next.Value != "," {
					return nil, fmt.Errorf("expected , between list items but received: %v", next.Value)
				}
			}
		}
		return value(cbor.Marshal(list))

	case TokenLeftBrace:
		record := make(map[string]Flat)
		for {
			{
				next := p.next()
				if next == nil {
					return nil, fmt.Errorf("expected , or }")
				}
				if next.Type == TokenRightBrace {
					break
				}
				if next.Type == TokenEtc {
					v, err := cbor.Marshal(cbor.Tag{TagEtc, ""})
					if err != nil {
						return nil, err
					}
					record[""] = v
				} else if next.Type == TokenOperator && next.Value == ".." {
					next := p.next()
					if next == nil {
						return nil, fmt.Errorf("unexpected end during spread")
					}
					if next.Type != TokenName {
						return nil, fmt.Errorf("expected spread variable")
					}
					v, err := cbor.Marshal(cbor.Tag{TagEtc, next.Value.(string)})
					if err != nil {
						return nil, err
					}
					record[""] = v
				} else {
					if next.Type != TokenName {
						return nil, fmt.Errorf("expected record key")
					}
					k := next.Value.(string)
					next = p.next()
					if next.Type != TokenOperator || next.Value != "=" {
						return nil, fmt.Errorf("expected = after record key")
					}
					v, err := expr(p.parseBinary(precs[","].pr + 1))
					if err != nil {
						return nil, err
					}
					record[k] = v
				}
			}
			{
				next := p.next()
				if next.Type == TokenRightBrace {
					break
				}
				if next.Type != TokenOperator || next.Value != "," {
					return nil, fmt.Errorf("expected , between record items but received: %v", next.Value)
				}
			}
		}
		em, err := cbor.CanonicalEncOptions().EncMode()
		if err != nil {
			return nil, err
		}
		return value(em.Marshal(record))

	case TokenEtc:
		return value(cbor.Marshal(cbor.Tag{TagEtc, ""}))

	case TokenOperator:
		switch token.Value {
		case "|":
			fun := []Flat{}
			for {
				x, err := expr(p.parseBinary(precs["->"].pr + 0.2))
				if err != nil {
					return nil, err
				}
				next := p.next()
				if next == nil {
					return nil, fmt.Errorf("expected ->")
				}
				y, err := expr(p.parseBinary(precs["|"].pr + 0.2))
				if err != nil {
					return nil, err
				}
				fun = append(fun, x, y)
				next = p.peek()
				if next == nil || next.Type != TokenOperator || next.Value != "|" {
					break
				}
				p.next()
			}
			return value(cbor.Marshal(cbor.Tag{TagFun, fun}))
		case "-":
			op := p.peek()
			switch op.Type {
			case TokenIntLit:
				op.Value = -op.Value.(int)
				return p.parseUnary(highestPrec + 1)

			case TokenFloatLit:
				op.Value = -op.Value.(float64)
				return p.parseUnary(highestPrec + 1)

			default:
				right, err := p.parseUnary(highestPrec + 1)
				// TODO: 0 - right
				return right, err

			}
		case "..":
			next := p.next()
			if next == nil {
				return nil, fmt.Errorf("unexpected end during spread")
			}
			if next.Type != TokenName {
				return nil, fmt.Errorf("expected spread variable")
			}
			return value(cbor.Marshal(cbor.Tag{TagEtc, next.Value.(string)}))
		}
	}

	return nil, fmt.Errorf("unexpected Token %v", token)
}

func (p *parser) parseBinary(prec float64) ([]Flat, error) {
	left, err := p.parseUnary(prec)
	if err != nil {
		return nil, err
	}
	if p.peek() == nil {
		return left, nil
	}

	expr := left
	for {
		op := p.peek()
		if op == nil {
			break
		}

		if op.Type == TokenRightParen || op.Type == TokenRightBracket || op.Type == TokenRightBrace {
			break
		}

		if op.Type != TokenOperator {
			opPrec := precs[" "]
			if opPrec.pl < prec {
				break
			}
			right, err := p.parseBinary(opPrec.pr)
			if err != nil {
				return nil, err
			}
			expr = append(expr, append(right, tagOp(" "))...)
			continue
		}

		opPrec, ok := precs[op.Value.(string)]
		if !ok || opPrec.pl < prec {
			break
		}
		p.next()

		// TODO: Look for more parse errors here.
		switch op.Value {
		case "=":
			if TagType(left[0][0]) == TagVar {
				return nil, fmt.Errorf("expected variable name before =")
			}
		case "|":
			return nil, fmt.Errorf("bad match case")
		}

		right, err := p.parseBinary(opPrec.pr)
		if err != nil {
			return nil, err
		}
		expr = append(expr, append(right, tagOp(op.Value.(string)))...)
	}

	return expr, nil
}

func Parse(Tokens []Token) (Flat, error) {
	if len(Tokens) == 0 {
		return nil, fmt.Errorf("empty input")
	}

	p := &parser{Tokens: Tokens}
	resetGensym()

	flat, err := expr(p.parseBinary(0))
	if err != nil {
		return nil, err
	}

	// TODO: Consider inferring types here too as a type check.

	if p.peek() != nil {
		return nil, fmt.Errorf("unexpected Tokens after expression: %v", p.peek())
	}

	return flat, nil
}

//// PRINT

func print(v interface{}) (string, error) {
	if v == nil {
		return "()", nil
	}
	if x, ok := (v).(bool); ok {
		return fmt.Sprintf("%v", x), nil
	}
	if x, ok := (v).(uint64); ok {
		return fmt.Sprintf("%v", x), nil
	}
	if x, ok := (v).(int64); ok {
		return fmt.Sprintf("%v", x), nil
	}
	if x, ok := (v).(float64); ok {
		return fmt.Sprintf("%v", x), nil
	}
	if x, ok := (v).([]byte); ok {
		return fmt.Sprintf("~~%v", base64.StdEncoding.EncodeToString(x)), nil
	}
	if x, ok := (v).(string); ok {
		return fmt.Sprintf(`"%v"`, x), nil
	}
	if xs, ok := (v).([]interface{}); ok {
		if len(xs) == 0 {
			return "[]", nil
		}
		xs_ := []string{}
		for _, x := range xs {
			x_, err := print(x)
			if err != nil {
				return "", err
			}
			xs_ = append(xs_, x_)
		}
		return fmt.Sprintf("[ %v ]", strings.Join(xs_, ", ")), nil
	}
	if xs, ok := (v).(map[interface{}]interface{}); ok {
		if len(xs) == 0 {
			return "{}", nil
		}
		type kv struct {
			k string
			v string
		}
		xs_ := []kv{}
		for k, x := range xs {
			v, err := print(x)
			if err != nil {
				return "", err
			}
			xs_ = append(xs_, kv{k.(string), v})
		}
		slices.SortFunc(xs_, func(a, b kv) int {
			if len(a.k) == len(b.k) {
				return cmp.Compare(a.k, b.k)
			}
			return cmp.Compare(len(a.k), len(b.k))
		})
		xs__ := []string{}
		for _, x_ := range xs_ {
			if x_.k == "" {
				xs__ = append(xs__, fmt.Sprintf("%v", x_.v))
			} else {
				xs__ = append(xs__, fmt.Sprintf("%v = %v", x_.k, x_.v))
			}
		}
		return fmt.Sprintf("{ %v }", strings.Join(xs__, ", ")), nil
	}
	if x, ok := (v).(cbor.Tag); ok {
		switch x.Number {
		case TagExpr:
			if xs, ok := x.Content.([]interface{}); ok {
				s := []struct {
					text string
					prec prec
				}{}

				for _, x := range xs {
					if x_, ok := x.(cbor.Tag); ok && x_.Number == TagOp {
						if len(s) < 2 {
							return "", fmt.Errorf("insufficient operands for operator")
						}

						op := x_.Content.(string)
						pp, ok := precs[op]
						if !ok {
							return "", fmt.Errorf("unrecognized operator: %v", op)
						}

						right := s[len(s)-1]
						left := s[len(s)-2]
						s = s[:len(s)-2]

						opStr := op
						if op != " " {
							if !slices.Contains([]string{"::", "@", "^", "*", "/", "//", " "}, op) || left.prec.pr < pp.pl || left.prec.pl == precs[" "].pl || right.prec.pl == precs[" "].pl {
								opStr = " " + op + " "
							}
						}

						leftStr := left.text
						if left.prec.pr < pp.pl {
							leftStr = "(" + leftStr + ")"
						}

						rightStr := right.text
						if right.prec.pl < pp.pr {
							rightStr = "(" + rightStr + ")"
						}

						s = append(s, struct {
							text string
							prec prec
						}{
							leftStr + opStr + rightStr,
							pp,
						})
					} else {
						text, err := print(x)
						if err != nil {
							return "", err
						}
						s = append(s, struct {
							text string
							prec prec
						}{text, prec{10000, 10000}})
					}
				}

				if len(s) != 1 {
					return "", fmt.Errorf("invalid expression: too many operands")
				}
				return s[0].text, nil
			}
			return "", fmt.Errorf("expected list of flats: %v", x.Content)
		case TagFun:
			if xs, ok := x.Content.([]interface{}); ok {
				if len(xs) == 0 {
					return "", fmt.Errorf("empty matcher")
				}
				if len(xs)%2 == 1 {
					return "", fmt.Errorf("unfinished matcher: %v", x.Content)
				}
				s := ""
				for i, x := range xs {
					if i != 0 {
						s += " "
					}
					x_, err := print(x)
					if err != nil {
						return "", err
					}
					if i%2 == 0 {
						if len(xs) == 2 {
							s += x_
						} else {
							s += fmt.Sprintf("| %v", x_)
						}
					} else {
						s += fmt.Sprintf("-> %v", x_)
					}
				}
				return s, nil
			}
			return "", fmt.Errorf("expected list of flats: %v", x.Content)
		case TagOp:
			if s, ok := x.Content.(string); ok {
				return s, nil
			}
			return "", fmt.Errorf("non-string operator")
		case TagVar:
			if s, ok := x.Content.(string); ok {
				return s, nil
			}
			return "", fmt.Errorf("non-string variable")
		case TagTag:
			if s, ok := x.Content.(string); ok {
				return "#" + s, nil
			}
			return "", fmt.Errorf("non-string tag")
		case TagEtc:
			if s, ok := x.Content.(string); ok {
				if s == "" {
					return "...", nil
				} else {
					return ".." + s, nil
				}
			}
			return "", fmt.Errorf("non-string tag")
		}
		return "", fmt.Errorf("unsupported cbor tag %v", x.Number)
	}

	return "", fmt.Errorf("unrecognized flat %v", v)
}

func Print(flat Flat) (string, error) {
	var v interface{}
	err := cbor.Unmarshal(flat, &v)
	if err != nil {
		return "", err
	}
	return print(v)
}

//// EVAL

type Env map[string]interface{}

func eval(v interface{}, env Env) (interface{}, error) {
	if v == nil {
		return nil, nil
	}

	switch x := v.(type) {
	case bool, uint64, int64, float64, []byte, string:
		return x, nil

	case []interface{}:
		result := make([]interface{}, len(x))
		for i, item := range x {
			val, err := eval(item, env)
			if err != nil {
				return nil, err
			}
			result[i] = val
		}
		return result, nil

	case map[interface{}]interface{}:
		result := make(map[interface{}]interface{})
		for k, v := range x {
			val, err := eval(v, env)
			if err != nil {
				return nil, err
			}
			result[k] = val
		}
		return result, nil

	case cbor.Tag:
		switch x.Number {
		case TagExpr:
			if xs, ok := x.Content.([]interface{}); ok {
				stack := []interface{}{}

				for _, x := range xs {
					if tag, ok := x.(cbor.Tag); ok && tag.Number == TagOp {
						if len(stack) < 2 {
							return nil, fmt.Errorf("insufficient operands for operator")
						}

						right := stack[len(stack)-1]
						left := stack[len(stack)-2]
						stack = stack[:len(stack)-2]

						op := tag.Content.(string)
						switch op {
						case "+":
							if l, ok := left.(int64); ok {
								if r, ok := right.(int64); ok {
									stack = append(stack, l+r)
									continue
								}
							}
							return nil, fmt.Errorf("invalid operands for +")
						case " ":
							// Function application
							if fn, ok := left.(cbor.Tag); ok && fn.Number == TagFun {
								cases := fn.Content.([]interface{})
								for i := 0; i < len(cases); i += 2 {
									pattern := cases[i]
									body := cases[i+1]

									newEnv := make(Env)
									for k, v := range env {
										newEnv[k] = v
									}

									// Basic pattern matching
									if pat, ok := pattern.(cbor.Tag); ok && pat.Number == TagVar {
										newEnv[pat.Content.(string)] = right
										result, err := eval(body, newEnv)
										if err != nil {
											return nil, err
										}
										stack = append(stack, result)
										break
									}
								}
								continue
							}
							return nil, fmt.Errorf("invalid function application")
						default:
							return nil, fmt.Errorf("unsupported operator: %v", op)
						}
					} else {
						val, err := eval(x, env)
						if err != nil {
							return nil, err
						}
						stack = append(stack, val)
					}
				}

				if len(stack) != 1 {
					return nil, fmt.Errorf("invalid expression: wrong number of values on stack")
				}
				return stack[0], nil
			}
			return nil, fmt.Errorf("invalid expression")

		case TagVar:
			if name, ok := x.Content.(string); ok {
				if val, ok := env[name]; ok {
					return val, nil
				}
				return nil, fmt.Errorf("undefined variable: %v", name)
			}
			return nil, fmt.Errorf("invalid variable name")

		case TagFun, TagTag:
			return x, nil

		default:
			return nil, fmt.Errorf("unsupported tag: %v", x.Number)
		}

	default:
		return nil, fmt.Errorf("unsupported type: %T", v)
	}
}

func Eval(flat Flat) (interface{}, error) {
	var v interface{}
	err := cbor.Unmarshal(flat, &v)
	if err != nil {
		return nil, err
	}
	return eval(v, make(Env))
}
