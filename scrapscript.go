/*
* All of this was copied from github:tekknolagi/scrapscript using Claude.
*
* This parallel go implementation should be thrown away when the language stabilizes.
*
 */

package smel

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/fxamacker/cbor/v2"
	"github.com/gammazero/deque"
)

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
	base := int64(64) // default base

	str.WriteString(l.readWhile(func(c byte) bool {
		return !unicode.IsSpace(rune(c))
	}))

	parts := strings.Split(str.String(), "'")
	if len(parts) > 1 {
		var err error
		base, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return Token{}, fmt.Errorf("invalid base in bytes literal: %s", parts[0])
		}
		return Token{Type: TokenBytesLit, Value: struct {
			Base  int64
			Value string
		}{base, parts[1]}}, nil
	}

	return Token{Type: TokenBytesLit, Value: struct {
		Base  int64
		Value string
	}{base, str.String()}}, nil
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
	var tokens []Token

	for {
		token, err := l.nextToken()
		if err != nil {
			return nil, err
		}
		if token.Type == TokenEOF {
			break
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}

//// PARSE

type TagType = uint64

const (
	TagExpr TagType = iota
	TagOp
	TagVar
	TagTag
)

type prec struct {
	pl float64
	pr float64
}

var precs = map[string]prec{
	" ":  {0, 1},   // Default/juxtaposition
	"=":  {2, 1},   // Assignment
	"->": {3, 2},   // Function arrow
	"|>": {4, 5},   // Forward pipe
	"<|": {5, 4},   // Backward pipe
	">>": {6, 7},   // Forward compose
	"<<": {7, 6},   // Backward compose
	".":  {8, 9},   // Where
	"?":  {10, 11}, // Assert
	"@":  {12, 13}, // Access
	"+":  {20, 21}, // Add
	"-":  {20, 21}, // Subtract
	"*":  {30, 31}, // Multiply
	"/":  {30, 31}, // Divide
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
	tokens []Token
	pos    int
}

func newParser(tokens []Token) *parser {
	return &parser{tokens: tokens}
}

func (p *parser) peek() *Token {
	if p.pos >= len(p.tokens) {
		return nil
	}
	return &p.tokens[p.pos]
}

func (p *parser) next() *Token {
	if p.pos >= len(p.tokens) {
		return nil
	}
	token := &p.tokens[p.pos]
	p.pos++
	return token
}

func (p *parser) parseUnary(prec float64) (cbor.RawMessage, error) {
	token := p.next()
	if token == nil {
		return nil, fmt.Errorf("unexpected end of input")
	}

	switch token.Type {
	case TokenIntLit, TokenFloatLit, TokenName, TokenStringLit:
		return cbor.Marshal(token.Value)

	case TokenHash:
		tag := p.next()
		if tag == nil {
			return nil, fmt.Errorf("unexpected end")
		}
		if tag.Type != TokenName {
			return nil, fmt.Errorf("expected name after #")
		}
		right, err := p.parseBinary(precs[" "].pr + 1)
		if err != nil {
			return nil, err
		}
		return cbor.Marshal(cbor.Tag{TagTag, right})

	case TokenLeftParen:
		if next := p.peek(); next != nil && next.Type == TokenRightParen {
			p.next() // consume )
			return cbor.Marshal(nil)
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
		list := make([]cbor.RawMessage, 0)
		for {
			next := p.next()
			if next == nil {
				return nil, fmt.Errorf("expected , or ]")
			}
			if next.Type == TokenRightBracket {
				break
			}
			if next.Type != TokenOperator || next.Value != "," {
				return nil, fmt.Errorf("expected , between list items")
			}
			item, err := p.parseBinary(2)
			if err != nil {
				return nil, err
			}
			list = append(list, item)
		}
		return cbor.Marshal(list)

	case TokenLeftBrace:
		record := make(map[string]cbor.RawMessage)
		for {
			{
				next := p.next()
				if next == nil {
					return nil, fmt.Errorf("expected , or }")
				}
				if next.Type == TokenRightBrace {
					break
				}
				if next.Type != TokenOperator || next.Value != "," {
					return nil, fmt.Errorf("expected , between record fields")
				}
			}
			{
				l, err := p.parseUnary(prec)
				if err != nil {
					return nil, err
				}
				var k string
				err = cbor.Unmarshal(l, &k)
				if err != nil {
					return nil, err
				}
				// TODO: Handle spread.
				next := p.next()
				if next == nil {
					return nil, fmt.Errorf("expected =")
				}
				if next.Type != TokenOperator || next.Value != "=" {
					return nil, fmt.Errorf("expected = after record key")
				}
				r, err := p.parseUnary(prec)
				if err != nil {
					return nil, err
				}
				record[k] = r
			}
		}
		return cbor.Marshal(record)

	case TokenOperator:
		switch token.Value {
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
			return cbor.Marshal(cbor.Tag{TagVar, "..."})
		}
	}

	return nil, fmt.Errorf("unexpected token %v", token)
}

func tagOp(op string) cbor.RawMessage {
	op_, err := cbor.Marshal(cbor.RawTag{TagOp, []byte(op)})
	if err != nil {
		panic(err)
	}
	return op_
}

func (p *parser) parseBinary(prec float64) (cbor.RawMessage, error) {
	left, err := p.parseUnary(prec)
	if err != nil {
		return nil, err
	}

	expr := new(deque.Deque[cbor.RawMessage])
	expr.PushFront(left)
	// TODO: expr.Grow(todo)
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
			expr.PushFront(tagOp(" "))
			expr.PushBack(right)
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
			if TagType(left[0]) == TagVar {
				return nil, fmt.Errorf("expected variable name before =")
			}
		}

		right, err := p.parseBinary(opPrec.pr)
		if err != nil {
			return nil, err
		}
		expr.PushFront(tagOp(op.Value.(string)))
		expr.PushBack(right)
	}

	return cbor.Marshal(cbor.Tag{TagExpr, expr})
}

func Parse(tokens []Token) ([]byte, error) {
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty input")
	}

	p := newParser(tokens)
	resetGensym()

	flat, err := p.parseBinary(0)
	if err != nil {
		return nil, err
	}

	// TODO: Consider inferring types here too as a type check.

	if p.peek() != nil {
		return nil, fmt.Errorf("unexpected tokens after expression")
	}

	return flat, nil
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
