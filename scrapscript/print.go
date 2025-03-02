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
		s := fmt.Sprintf("%f", x)
		s = strings.TrimRight(s, "0")
		if strings.HasSuffix(s, ".") {
			s = s + "0"
		}
		return s, nil
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
	}
	if x, ok := (v).(cbor.Tag); ok {
		switch x.Number {
		case TagExpr:
			if xs, ok := x.Content.([]interface{}); ok {
				s := []struct {
					text string
					prec prec
				}{}
				suffix := ""

				for _, x := range xs {
					if x_, ok := x.(cbor.Tag); ok && x_.Number == TagOp {
						if x_.Content.(string) == "#" {
							if len(s) < 1 {
								return "", fmt.Errorf("insufficient operands for #")
							}
							s[len(s)-1] = struct {
								text string
								prec prec
							}{"#" + s[len(s)-1].text, s[len(s)-1].prec}
							continue
						}

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

						if op == "." {
							tmp := right
							right = left
							left = tmp
						}

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
					} else if x_, ok := x.(cbor.Tag); ok && x_.Number == TagFun {
						text, err := print(x)
						if err != nil {
							return "", err
						}
						s = append(s, struct {
							text string
							prec prec
						}{text, prec{5, 4.9}})
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
