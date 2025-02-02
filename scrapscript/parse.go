package scrapscript

import (
	"fmt"

	"github.com/fxamacker/cbor/v2"
)

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
