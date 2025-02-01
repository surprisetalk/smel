/*
* All of this was copied from github:tekknolagi/scrapscript using Claude.
*
* This parallel go implementation should be thrown away when the language stabilizes.
*
 */

package smel

import (
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

func (l *lexer) readOperator() (Token, error) {
	ops := map[string]bool{
		"+": true, "-": true, "*": true, "/": true, "^": true, "%": true,
		"++": true, "+<": true, ">+": true,
		"==": true, "/=": true, "<": true, ">": true,
		"<=": true, ">=": true, "&&": true, "||": true,
		"!": true, "->": true, ".": true, "=": true,
		",": true, ":": true, "?": true, "|": true,
		"...": true, "@": true, ">>": true, "<<": true,
		"|>": true, "<|": true,
	}

	c1 := l.peek()
	if l.hasInput() {
		l.advance()
		if l.hasInput() {
			c2 := l.peek()
			if l.hasInput() {
				c3 := l.peek()
				op3 := "" + string(c1) + string(c2) + string(c3)
				if ops[op3] {
					l.advance()
					l.advance()
					return Token{Type: TokenOperator, Value: op3}, nil
				}
			}
			op2 := "" + string(c1) + string(c2)
			if ops[op2] {
				l.advance()
				return Token{Type: TokenOperator, Value: op2}, nil
			}
		}
		op1 := string(c1)
		if ops[op1] {
			return Token{Type: TokenOperator, Value: op1}, nil
		}
	}
	return Token{}, fmt.Errorf("invalid operator: %s", string(c1))
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
	"<|": {6, 5.9},
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

func expr(flats []Flat) (Flat, error) {
	l := len(flats)
	if l == 0 {
		return nil, fmt.Errorf("empty expression")
	} else if l == 1 {
		return flats[0], nil
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
		expr, err := p.parseBinary(0)
		if err != nil {
			return nil, err
		}
		if next := p.next(); next == nil || next.Type != TokenRightParen {
			return nil, fmt.Errorf("expected )")
		}
		return expr, nil

	case TokenLeftBracket:
		list := make([]Flat, 0)
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
				item, err := p.parseBinary(precs[","].pr + 1)
				if err != nil {
					return nil, err
				}
				item_, err := expr(item)
				if err != nil {
					return nil, err
				}
				list = append(list, item_)
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
				if next.Type != TokenName {
					return nil, fmt.Errorf("expected record key")
				}
				// TODO: Handle spread.
				k := next.Value.(string)
				next = p.next()
				if next.Type != TokenOperator || next.Value != "=" {
					return nil, fmt.Errorf("expected = after record key")
				}
				v, err := p.parseBinary(precs[","].pr + 1)
				if err != nil {
					return nil, err
				}
				v_, err := expr(v)
				if err != nil {
					return nil, err
				}
				record[k] = v_
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
		return value(cbor.Marshal(record))

	case TokenOperator:
		switch token.Value {
		case "|":
			return p.parseBinary(precs["|"].pr + 1)
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
		case "...":
			return value(cbor.Marshal(cbor.Tag{TagVar, "..."}))
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

	flat, err := p.parseBinary(0)
	if err != nil {
		return nil, err
	}

	// TODO: Consider inferring types here too as a type check.

	if p.peek() != nil {
		return nil, fmt.Errorf("unexpected Tokens after expression: %v", p.peek())
	}

	return expr(flat)
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
		xs_ := []string{}
		for k, x := range xs {
			x_, err := print(x)
			if err != nil {
				return "", err
			}
			xs_ = append(xs_, fmt.Sprintf("%v = %v", k, x_))
		}
		return fmt.Sprintf("{ %v }", strings.Join(xs_, ", ")), nil
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
						if !slices.Contains([]string{"::", "@", "^", "*", "/", "//", " "}, op) {
							opStr = " " + op + " "
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

/*
type Env map[string]*Object

func Match(obj, pattern *Object) (Env, error) {
	switch pattern.Type {
	case NodeHole:
		if obj.Type == NodeHole {
			return Env{}, nil
		}
		return nil, nil

	case NodeInt:
		if obj.Type == NodeInt && obj.IntVal == pattern.IntVal {
			return Env{}, nil
		}
		return nil, nil

	case NodeFloat:
		return nil, fmt.Errorf("pattern matching is not supported for Floats")

	case NodeString:
		if obj.Type == NodeString && obj.StrVal == pattern.StrVal {
			return Env{}, nil
		}
		return nil, nil

	case NodeVar:
		return Env{pattern.Name: obj}, nil

	case NodeVariant:
		if obj.Type != NodeVariant {
			return nil, nil
		}
		if obj.Name != pattern.Name {
			return nil, nil
		}
		return Match(obj.Right, pattern.Right)

	case NodeRecord:
		if obj.Type != NodeRecord {
			return nil, nil
		}

		result := make(Env)
		useSpread := false
		seenKeys := make(map[string]bool)

		for key, patternItem := range pattern.Fields {
			if patternItem.Type == NodeSpread {
				useSpread = true
				// Handle named spread
				if patternItem.Name != "" {
					restRecord := &Object{Type: NodeRecord, Fields: make(map[string]*Object)}
					for k, v := range obj.Fields {
						if !seenKeys[k] {
							restRecord.Fields[k] = v
						}
					}
					result[patternItem.Name] = restRecord
				}
				break
			}

			seenKeys[key] = true
			objItem, ok := obj.Fields[key]
			if !ok {
				return nil, nil
			}

			part, err := Match(objItem, patternItem)
			if err != nil {
				return nil, err
			}
			if part == nil {
				return nil, nil
			}

			// Merge part into result
			for k, v := range part {
				result[k] = v
			}
		}

		if !useSpread && len(pattern.Fields) != len(obj.Fields) {
			return nil, nil
		}

		return result, nil

	case NodeList:
		if obj.Type != NodeList {
			return nil, nil
		}

		result := make(Env)
		useSpread := false

		for i, patternItem := range pattern.Params {
			if patternItem.Type == NodeSpread {
				useSpread = true
				// Handle named spread
				if patternItem.Name != "" {
					restList := &Object{Type: NodeList, Params: obj.Params[i:]}
					result[patternItem.Name] = restList
				}
				break
			}

			if i >= len(obj.Params) {
				return nil, nil
			}

			part, err := Match(obj.Params[i], patternItem)
			if err != nil {
				return nil, err
			}
			if part == nil {
				return nil, nil
			}

			// Merge part into result
			for k, v := range part {
				result[k] = v
			}
		}

		if !useSpread && len(pattern.Params) != len(obj.Params) {
			return nil, nil
		}

		return result, nil

	default:
		return nil, fmt.Errorf("match not implemented for %v", pattern.Type)
	}
}

type BinopHandler func(env Env, left, right *Object) (*Object, error)

var BINOP_HANDLERS = map[string]BinopHandler{
	"+": func(env Env, left, right *Object) (*Object, error) {
		if left.Type == NodeInt && right.Type == NodeInt {
			return &Object{Type: NodeInt, IntVal: left.IntVal + right.IntVal}, nil
		} else if left.Type == NodeFloat && right.Type == NodeFloat {
			return &Object{Type: NodeFloat, FloatVal: left.FloatVal + right.FloatVal}, nil
		}
		return nil, fmt.Errorf("cannot add %v and %v", left.Type, right.Type)
	},
	"-": func(env Env, left, right *Object) (*Object, error) {
		if left.Type == NodeInt && right.Type == NodeInt {
			return &Object{Type: NodeInt, IntVal: left.IntVal - right.IntVal}, nil
		} else if left.Type == NodeFloat && right.Type == NodeFloat {
			return &Object{Type: NodeFloat, FloatVal: left.FloatVal - right.FloatVal}, nil
		}
		return nil, fmt.Errorf("cannot subtract %v and %v", left.Type, right.Type)
	},
	"*": func(env Env, left, right *Object) (*Object, error) {
		if left.Type == NodeInt && right.Type == NodeInt {
			return &Object{Type: NodeInt, IntVal: left.IntVal * right.IntVal}, nil
		} else if left.Type == NodeFloat && right.Type == NodeFloat {
			return &Object{Type: NodeFloat, FloatVal: left.FloatVal * right.FloatVal}, nil
		}
		return nil, fmt.Errorf("cannot multiply %v and %v", left.Type, right.Type)
	},
	"/": func(env Env, left, right *Object) (*Object, error) {
		if left.Type == NodeFloat && right.Type == NodeFloat {
			return &Object{Type: NodeFloat, FloatVal: left.FloatVal / right.FloatVal}, nil
		}
		return nil, fmt.Errorf("cannot divide %v and %v", left.Type, right.Type)
	},
	"//": func(env Env, left, right *Object) (*Object, error) {
		if left.Type == NodeInt && right.Type == NodeInt {
			return &Object{Type: NodeInt, IntVal: left.IntVal / right.IntVal}, nil
		}
		return nil, fmt.Errorf("cannot floor divide %v and %v", left.Type, right.Type)
	},
	"^": func(env Env, left, right *Object) (*Object, error) {
		if left.Type == NodeInt && right.Type == NodeInt {
			return &Object{Type: NodeInt, IntVal: int64(math.Pow(float64(left.IntVal), float64(right.IntVal)))}, nil
		} else if left.Type == NodeFloat && right.Type == NodeFloat {
			return &Object{Type: NodeFloat, FloatVal: math.Pow(left.FloatVal, right.FloatVal)}, nil
		}
		return nil, fmt.Errorf("cannot exponentiate %v and %v", left.Type, right.Type)
	},
	"%": func(env Env, left, right *Object) (*Object, error) {
		if left.Type == NodeInt && right.Type == NodeInt {
			return &Object{Type: NodeInt, IntVal: left.IntVal % right.IntVal}, nil
		}
		return nil, fmt.Errorf("cannot mod %v and %v", left.Type, right.Type)
	},
	"==": func(env Env, left, right *Object) (*Object, error) {
		isEqual := reflect.DeepEqual(left, right)
		return &Object{
			Type:  NodeVariant,
			Name:  map[bool]string{true: "true", false: "false"}[isEqual],
			Right: &Object{Type: NodeHole},
		}, nil
	},
	"/=": func(env Env, left, right *Object) (*Object, error) {
		isEqual := reflect.DeepEqual(left, right)
		return &Object{
			Type:  NodeVariant,
			Name:  map[bool]string{true: "false", false: "true"}[isEqual],
			Right: &Object{Type: NodeHole},
		}, nil
	},
	"<": func(env Env, left, right *Object) (*Object, error) {
		if left.Type == NodeInt && right.Type == NodeInt {
			result := left.IntVal < right.IntVal
			return &Object{
				Type:  NodeVariant,
				Name:  map[bool]string{true: "true", false: "false"}[result],
				Right: &Object{Type: NodeHole},
			}, nil
		} else if left.Type == NodeFloat && right.Type == NodeFloat {
			result := left.FloatVal < right.FloatVal
			return &Object{
				Type:  NodeVariant,
				Name:  map[bool]string{true: "true", false: "false"}[result],
				Right: &Object{Type: NodeHole},
			}, nil
		}
		return nil, fmt.Errorf("cannot compare %v and %v", left.Type, right.Type)
	},
	">": func(env Env, left, right *Object) (*Object, error) {
		if left.Type == NodeInt && right.Type == NodeInt {
			result := left.IntVal > right.IntVal
			return &Object{
				Type:  NodeVariant,
				Name:  map[bool]string{true: "true", false: "false"}[result],
				Right: &Object{Type: NodeHole},
			}, nil
		} else if left.Type == NodeFloat && right.Type == NodeFloat {
			result := left.FloatVal > right.FloatVal
			return &Object{
				Type:  NodeVariant,
				Name:  map[bool]string{true: "true", false: "false"}[result],
				Right: &Object{Type: NodeHole},
			}, nil
		}
		return nil, fmt.Errorf("cannot compare %v and %v", left.Type, right.Type)
	},
	"<=": func(env Env, left, right *Object) (*Object, error) {
		if left.Type == NodeInt && right.Type == NodeInt {
			result := left.IntVal <= right.IntVal
			return &Object{
				Type:  NodeVariant,
				Name:  map[bool]string{true: "true", false: "false"}[result],
				Right: &Object{Type: NodeHole},
			}, nil
		} else if left.Type == NodeFloat && right.Type == NodeFloat {
			result := left.FloatVal <= right.FloatVal
			return &Object{
				Type:  NodeVariant,
				Name:  map[bool]string{true: "true", false: "false"}[result],
				Right: &Object{Type: NodeHole},
			}, nil
		}
		return nil, fmt.Errorf("cannot compare %v and %v", left.Type, right.Type)
	},
	">=": func(env Env, left, right *Object) (*Object, error) {
		if left.Type == NodeInt && right.Type == NodeInt {
			result := left.IntVal >= right.IntVal
			return &Object{
				Type:  NodeVariant,
				Name:  map[bool]string{true: "true", false: "false"}[result],
				Right: &Object{Type: NodeHole},
			}, nil
		} else if left.Type == NodeFloat && right.Type == NodeFloat {
			result := left.FloatVal >= right.FloatVal
			return &Object{
				Type:  NodeVariant,
				Name:  map[bool]string{true: "true", false: "false"}[result],
				Right: &Object{Type: NodeHole},
			}, nil
		}
		return nil, fmt.Errorf("cannot compare %v and %v", left.Type, right.Type)
	},
	"&&": func(env Env, left, right *Object) (*Object, error) {
		if left.Type == NodeVariant && left.Name == "false" {
			return left, nil // Short circuit
		}
		if left.Type != NodeVariant || (left.Name != "true" && left.Name != "false") {
			return nil, fmt.Errorf("expected boolean variant, got %v", left.Type)
		}
		if right.Type != NodeVariant || (right.Name != "true" && right.Name != "false") {
			return nil, fmt.Errorf("expected boolean variant, got %v", right.Type)
		}
		result := left.Name == "true" && right.Name == "true"
		return &Object{
			Type:  NodeVariant,
			Name:  map[bool]string{true: "true", false: "false"}[result],
			Right: &Object{Type: NodeHole},
		}, nil
	},
	"||": func(env Env, left, right *Object) (*Object, error) {
		if left.Type == NodeVariant && left.Name == "true" {
			return left, nil // Short circuit
		}
		if left.Type != NodeVariant || (left.Name != "true" && left.Name != "false") {
			return nil, fmt.Errorf("expected boolean variant, got %v", left.Type)
		}
		if right.Type != NodeVariant || (right.Name != "true" && right.Name != "false") {
			return nil, fmt.Errorf("expected boolean variant, got %v", right.Type)
		}
		result := left.Name == "true" || right.Name == "true"
		return &Object{
			Type:  NodeVariant,
			Name:  map[bool]string{true: "true", false: "false"}[result],
			Right: &Object{Type: NodeHole},
		}, nil
	},
	"++": func(env Env, left, right *Object) (*Object, error) {
		if left.Type == NodeString && right.Type == NodeString {
			return &Object{Type: NodeString, StrVal: left.StrVal + right.StrVal}, nil
		}
		return nil, fmt.Errorf("cannot concatenate %v and %v", left.Type, right.Type)
	},
	">+": func(env Env, left, right *Object) (*Object, error) {
		if right.Type != NodeList {
			return nil, fmt.Errorf("list cons requires list on right, got %v", right.Type)
		}
		params := make([]*Object, len(right.Params)+1)
		params[0] = left
		copy(params[1:], right.Params)
		return &Object{Type: NodeList, Params: params}, nil
	},
	"+<": func(env Env, left, right *Object) (*Object, error) {
		if left.Type != NodeList {
			return nil, fmt.Errorf("list append requires list on left, got %v", left.Type)
		}
		params := make([]*Object, len(left.Params)+1)
		copy(params, left.Params)
		params[len(left.Params)] = right
		return &Object{Type: NodeList, Params: params}, nil
	},
	// "!": func(env Env, left, right *Object) (*Object, error) {
	// 	return eval_exp(env, right), nil
	// },
}

func eval_exp(env Env, exp *Object) *Object {
	switch exp.Type {
	// Base cases - return the values directly
	case NodeInt, NodeFloat, NodeString, NodeBytes, NodeHole:
		return exp

	// Variable lookup
	case NodeVar:
		value, ok := env[exp.Name]
		if !ok {
			panic(fmt.Sprintf("name '%s' is not defined", exp.Name))
		}
		return value

	// Constructor cases
	case NodeVariant:
		return &Object{
			Type:  NodeVariant,
			Name:  exp.Name,                 // Tag stored in Name
			Right: eval_exp(env, exp.Right), // Value stored in Right
		}

	case NodeList:
		params := make([]*Object, len(exp.Params))
		for i, item := range exp.Params {
			params[i] = eval_exp(env, item)
		}
		return &Object{
			Type:   NodeList,
			Params: params,
		}

	case NodeRecord:
		fields := make(map[string]*Object)
		for k, v := range exp.Fields {
			fields[k] = eval_exp(env, v)
		}
		return &Object{
			Type:   NodeRecord,
			Fields: fields,
		}

	// Pattern matching and functions
	case NodeFunction:
		if exp.Left.Type != NodeVar {
			panic(fmt.Sprintf("expected variable in function definition %v", exp.Left))
		}
		// Create closure by capturing current environment
		return &Object{
			Type:   NodeFunction,
			Left:   exp.Left,  // Arg
			Right:  exp.Right, // Body
			Fields: env,       // Captured environment stored in Fields
		}

	case NodeMatchFunction:
		// Similar to function, create closure
		return &Object{
			Type:   NodeMatchFunction,
			Params: exp.Params, // Cases
			Fields: env,        // Captured environment
		}

	// Application and special forms
	case NodeApply:
		// Special case for quote
		if exp.Left.Type == NodeVar && exp.Left.Name == "$$quote" {
			return exp.Right
		}

		callee := eval_exp(env, exp.Left)
		arg := eval_exp(env, exp.Right)

		switch callee.Type {
		case NodeFunction:
			// Create new environment with captured env + arg binding
			newEnv := make(Env)
			for k, v := range callee.Fields { // Captured env
				newEnv[k] = v
			}
			newEnv[callee.Left.Name] = arg // Bind argument
			return eval_exp(newEnv, callee.Right)

		case NodeMatchFunction:
			for _, caseObj := range callee.Params {
				// Each case has pattern in Left and body in Right
				if m, _ := Match(arg, caseObj.Left); m != nil {
					newEnv := make(Env)
					for k, v := range callee.Fields {
						newEnv[k] = v
					}
					for k, v := range m {
						newEnv[k] = v
					}
					return eval_exp(newEnv, caseObj.Right)
				}
			}
			panic("no matching cases")

		default:
			panic(fmt.Sprintf("attempted to apply a non-function of type %v", callee.Type))
		}

	case NodeAccess:
		obj := eval_exp(env, exp.Left)
		switch obj.Type {
		case NodeRecord:
			if exp.Right.Type == NodeVar {
				if val, ok := obj.Fields[exp.Right.Name]; ok {
					return val
				}
				panic(fmt.Sprintf("no assignment to %s found in record", exp.Right.Name))
			}
			panic(fmt.Sprintf("cannot access record field using %v, expected a field name", exp.Right.Type))

		case NodeList:
			idx := eval_exp(env, exp.Right)
			if idx.Type == NodeInt {
				if idx.IntVal < 0 || idx.IntVal >= int64(len(obj.Params)) {
					panic(fmt.Sprintf("index %d out of bounds for list", idx.IntVal))
				}
				return obj.Params[idx.IntVal]
			}
			panic(fmt.Sprintf("cannot index into list using type %v, expected integer", idx.Type))

		default:
			panic(fmt.Sprintf("attempted to access from type %v", obj.Type))
		}

	// Environment manipulation
	case NodeAssign:
		if exp.Left.Type != NodeVar {
			panic("expected variable name in assignment")
		}

		value := eval_exp(env, exp.Right)

		// Handle function recursion by allowing function to reference itself
		if value.Type == NodeFunction || value.Type == NodeMatchFunction {
			valueCopy := *value
			newEnv := make(Env)
			for k, v := range value.Fields {
				newEnv[k] = v
			}
			newEnv[exp.Left.Name] = &valueCopy
			valueCopy.Fields = newEnv
			value = &valueCopy
		}

		newEnv := make(Env)
		for k, v := range env {
			newEnv[k] = v
		}
		newEnv[exp.Left.Name] = value
		return &Object{
			Type:   NodeRecord, // Using Record to represent EnvObject
			Fields: newEnv,
		}

	case NodeWhere:
		if exp.Right.Type == NodeAssign {
			resEnv := eval_exp(env, exp.Right)
			if resEnv.Type == NodeRecord { // EnvObject
				newEnv := make(Env)
				for k, v := range env {
					newEnv[k] = v
				}
				for k, v := range resEnv.Fields {
					newEnv[k] = v
				}
				return eval_exp(newEnv, exp.Left)
			}
		}
		panic("binding in where must be an assignment")

	case NodeAssert:
		cond := eval_exp(env, exp.Right)
		if cond.Type != NodeVariant || cond.Name != "true" {
			panic(fmt.Sprintf("condition %v failed", exp.Right))
		}
		return eval_exp(env, exp.Left)

	case NodeBinOp:
		handler, ok := BINOP_HANDLERS[exp.Op]
		if !ok {
			panic(fmt.Sprintf("no handler for %v", exp.Op))
		}
		result, err := handler(env, exp.Left, exp.Right)
		if err != nil {
			panic(err)
		}
		return result

	case NodeSpread:
		panic("cannot evaluate a spread")

	default:
		panic(fmt.Sprintf("eval_exp not implemented for %v", exp.Type))
	}
}
*/
