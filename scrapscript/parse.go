package scrapscript

import (
	"fmt"

	"github.com/fxamacker/cbor/v2"
)

var em, cborErr = cbor.CanonicalEncOptions().EncMode()

func init() {
	if cborErr != nil {
		panic(cborErr)
	}
}

type Flat = cbor.RawMessage

type TagN = uint64

// TODO: Consider adding a TagExbr (backwards expr) for efficiency reasons, i.e. loading assignments into memory early.
const (
	TagExpr TagN = ' '
	TagOp   TagN = '+'
	TagSym  TagN = '='
	TagTag  TagN = '#'
	TagEtc  TagN = '.'
	TagFun  TagN = '|'  // e.g. | a -> 0 | _ -> 1
	TagDict TagN = '\'' // e.g. dict/from [ "a"' 1, "b"' 1 ]
	TagType TagN = ':'  // e.g. #a int #b int
)

func tagOp(op string) Flat {
	// TODO: Do NOT store this as a string! So inefficient.
	op_, err := tag(TagOp, op)
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
	"'":  {6.5, 6.4},
	"|>": {6, 6.1},
	"#":  {5.5, 2000}, // TODO: This should bind tighter than " " on one side? So that (#a #b) and (#a () #b ()) both work?
	"->": {5, 4.9},
	"|":  {4.5, 4.6},
	":":  {4.5, 4.4},
	"=":  {4, 4.1},
	"?":  {3, 3.1},
	"!":  {2, 1.9},
	".":  {1, 1.1},
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

func tag(t TagN, content interface{}) (Flat, error) {
	return em.Marshal(cbor.Tag{Number: t, Content: content})
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
	return tag(TagExpr, flats)
}

// TODO: Use prec?
func (p *parser) unary(prec float64) ([]Flat, error) {
	token := p.next()
	if token == nil {
		return nil, fmt.Errorf("unexpected end of input")
	}
	switch token.Type {
	case TokenIntLit, TokenFloatLit, TokenStringLit, TokenBytesLit:
		return value(em.Marshal(token.Value))

	case TokenName:
		switch token.Value {
		case "true":
			return value(em.Marshal(true))
		case "false":
			return value(em.Marshal(false))
		default:
			return value(tag(TagSym, token.Value))
		}

	case TokenLeftParen:
		if next := p.peek(); next != nil && next.Type == TokenRightParen {
			p.next() // consume )
			return value(em.Marshal(nil))
		}
		ex, err := p.binary(0)
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
				item, err := expr(p.binary(precs[","].pr + 1))
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
		return value(em.Marshal(list))

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
					v, err := tag(TagSym, "_")
					if err != nil {
						return nil, err
					}
					record[""] = v
				} else if next.Type == TokenOperator && next.Value == ".." {
					if p.peek() == nil {
						return nil, fmt.Errorf("unexpected end during spread")
					}
					next, err := expr(p.unary(prec))
					if err != nil {
						return nil, err
					}
					v, err := tag(TagEtc, next)
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
					if next.Type == TokenRightBrace || next.Value == "," {
						v, err := tag(TagSym, token.Value)
						if err != nil {
							return nil, err
						}
						record[k] = v
						if next.Type == TokenRightBrace {
							break
						}
						continue
					}
					if next.Type != TokenOperator || next.Value != "=" {
						return nil, fmt.Errorf("expected = after record key")
					}
					v, err := expr(p.binary(precs[","].pr + 1))
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
		return value(em.Marshal(record))

	case TokenEtc:
		v, err := value(tag(TagSym, "_"))
		if err != nil {
			return nil, err
		}
		return value(tag(TagEtc, v))

	case TokenOperator:
		switch token.Value {
		case "|":
			fun := []Flat{}
			for {
				x, err := expr(p.binary(precs["->"].pr + 0.2))
				if err != nil {
					return nil, err
				}
				next := p.next()
				if next == nil {
					return nil, fmt.Errorf("expected ->")
				}
				y, err := expr(p.binary(precs["|"].pr + 0.2))
				if err != nil {
					return nil, err
				}
				fun = append(fun, x, y)
				if next = p.peek(); next == nil || next.Type != TokenOperator || next.Value != "|" {
					break
				}
				p.next()
			}
			return value(tag(TagFun, fun))
		case "-":
			op := p.peek()
			switch op.Type {
			case TokenIntLit:
				op.Value = -op.Value.(int64)
				return p.unary(highestPrec + 1)

			case TokenFloatLit:
				op.Value = -op.Value.(float64)
				return p.unary(highestPrec + 1)

			default:
				right, err := p.unary(highestPrec + 1)
				// TODO: 0 - right
				return right, err

			}
		case "#":
			next := p.next()
			if next == nil {
				return nil, fmt.Errorf("unexpected end during tag")
			}
			if next.Type != TokenName {
				return nil, fmt.Errorf("expected tag name")
			}
			return value(tag(TagTag, next.Value.(string)))
		}
	}

	return nil, fmt.Errorf("unexpected Token %v", token)
}

func (p *parser) binary(prec float64) ([]Flat, error) {
	left, err := p.unary(prec)
	if err != nil {
		return nil, err
	}
	if p.peek() == nil {
		return left, nil
	}

	exps := left
	for {
		op := p.peek()
		if op == nil || op.Type == TokenRightParen || op.Type == TokenRightBracket || op.Type == TokenRightBrace {
			break
		}

		if op.Type != TokenOperator {
			opPrec := precs[" "]
			if opPrec.pl < prec {
				break
			}
			right, err := p.binary(opPrec.pr)
			if err != nil {
				return nil, err
			}
			exps = append(exps, append(right, tagOp(" "))...)
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
			if TagN(left[0][0]) == TagSym {
				return nil, fmt.Errorf("expected variable name before =")
			}
		case "|":
			return nil, fmt.Errorf("bad match case")
		}

		right, err := p.binary(opPrec.pr)
		if err != nil {
			return nil, err
		}

		if op.Value.(string) == "->" {
			l, err := expr(exps, nil)
			if err != nil {
				return nil, err
			}
			r, err := expr(right, nil)
			if err != nil {
				return nil, err
			}
			// TODO: Shouldn't we be able to do `cbor.Tag{TagFun, []cbor.Tag{{TagExpr, exps}, {TagExpr, right}}}`? It's not working for some reason.
			exp, err := tag(TagFun, []Flat{l, r})
			if err != nil {
				return nil, err
			}
			exps = []Flat{exp}
		} else if op.Value.(string) == "." {
			exps = append(right, append(exps, tagOp(op.Value.(string)))...)
		} else {
			exps = append(exps, append(right, tagOp(op.Value.(string)))...)
		}
	}

	return exps, nil
}

func Parse(Tokens []Token) (Flat, error) {
	if len(Tokens) == 0 {
		return nil, fmt.Errorf("empty input")
	}

	p := &parser{Tokens: Tokens}
	resetGensym()

	flat, err := expr(p.binary(0))
	if err != nil {
		return nil, err
	}

	// TODO: Consider inferring types here too as a type check.

	if p.peek() != nil {
		return nil, fmt.Errorf("unexpected Tokens after expression: %v", p.peek())
	}

	return flat, nil
}
