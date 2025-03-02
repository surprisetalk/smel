package scrapscript

import (
	"cmp"
	"encoding/base64"
	"fmt"
	"slices"
	"strings"

	"github.com/fxamacker/cbor/v2"
)

func print(v interface{}) (string, error) {
	if v == nil {
		return "()", nil
	}

	switch x := v.(type) {
	case bool, uint64, int64:
		return fmt.Sprintf("%v", x), nil
	case float64:
		s := fmt.Sprintf("%f", x)
		s = strings.TrimRight(s, "0")
		if strings.HasSuffix(s, ".") {
			s = s + "0"
		}
		return s, nil
	case []byte:
		return fmt.Sprintf("~~%v", base64.StdEncoding.EncodeToString(x)), nil
	case string:
		return fmt.Sprintf(`"%v"`, x), nil
	case snap:
		return fmt.Sprintf("#%v (%v)", x.k, x.v), nil
	case []interface{}:
		if len(x) == 0 {
			return "[]", nil
		}
		xs_ := []string{}
		for _, item := range x {
			item_, err := print(item)
			if err != nil {
				return "", err
			}
			xs_ = append(xs_, item_)
		}
		return fmt.Sprintf("[ %v ]", strings.Join(xs_, ", ")), nil
	case map[interface{}]interface{}:
		if len(x) == 0 {
			return "{}", nil
		}
		type kv struct {
			k string
			v string
		}
		xs_ := []kv{}
		for k, item := range x {
			v, err := print(item)
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
				if x_.v == "_" {
					xs__ = append(xs__, "...")
				} else {
					xs__ = append(xs__, fmt.Sprintf("..%v", x_.v))
				}
			} else {
				xs__ = append(xs__, fmt.Sprintf("%v = %v", x_.k, x_.v))
			}
		}
		return fmt.Sprintf("{ %v }", strings.Join(xs__, ", ")), nil
	case cbor.Tag:
		switch x.Number {
		case TagExpr:
			if xs, ok := x.Content.([]interface{}); ok {
				s := []struct {
					text string
					prec prec
				}{}
				suffix := ""

				for _, x := range xs {
					var text string
					var p prec = prec{10000, 10000} // Default precedence for most elements

					x_, ok := x.(cbor.Tag)
					if !ok {
						_, err := print(x)
						if err != nil {
							return "", err
						}
					}

					switch x_.Number {
					case TagOp:
						op := x_.Content.(string)
						if op == "#" {
							if len(s) < 1 {
								return "", fmt.Errorf("insufficient operands for #")
							}
							s[len(s)-1].text = "#" + s[len(s)-1].text
							continue
						}
						if len(s) < 2 {
							return "", fmt.Errorf("insufficient operands for operator")
						}

						pp, ok := precs[op]
						if !ok {
							return "", fmt.Errorf("unrecognized operator: %v", op)
						}

						left, right := s[len(s)-2], s[len(s)-1]
						if op == "." {
							left, right = right, left
						}

						s = s[:len(s)-2]

						opStr := op
						if op != " " {
							if !slices.Contains([]string{"::", "@", "^", "*", "/", "//", " "}, op) ||
								left.prec.pr < pp.pl || left.prec.pl == precs[" "].pl || right.prec.pl == precs[" "].pl {
								opStr = " " + op + " "
							}
						}

						leftStr, rightStr := left.text, right.text
						if left.prec.pr < pp.pl {
							leftStr = "(" + leftStr + ")"
						}
						if right.prec.pl < pp.pr {
							rightStr = "(" + rightStr + ")"
						}
						text = leftStr + opStr + rightStr
						p = pp

					case TagFun:
						var err error
						text, err = print(x)
						if err != nil {
							return "", err
						}
						p = prec{5, 4.9}

					default:
						var err error
						text, err = print(x)
						if err != nil {
							return "", err

						}
					}

					if text != "" {
						s = append(s, struct {
							text string
							prec prec
						}{text, p})
					}
				}

				if len(s) != 1 {
					return "", fmt.Errorf("invalid expression: too many operands")
				}
				return s[0].text + suffix, nil
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
		case TagSym:
			if s, ok := x.Content.(string); ok {
				return s, nil
			}
			return "", fmt.Errorf("non-string variable")
		case TagTag:
			if s, ok := x.Content.(string); ok {
				return "#" + s, nil
			}
			return "", fmt.Errorf("non-string tag")
		default:
			return "", fmt.Errorf("unsupported cbor tag %v", x.Number)
		}
	default:
		return "", fmt.Errorf("unrecognized flat %v", v)
	}
}

func Print(flat Flat) (string, error) {
	var v interface{}
	err := cbor.Unmarshal(flat, &v)
	if err != nil {
		return "", err
	}
	return print(v)
}
