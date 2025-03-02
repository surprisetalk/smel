package scrapscript

import (
	"fmt"
	"math"
	"slices"

	"github.com/fxamacker/cbor/v2"
)

type Env map[string]interface{}

type closure struct {
	fn  cbor.Tag
	env Env
}

func numOp(left, right interface{}, intOp func(int64, int64) interface{}, uintOp func(uint64, uint64) interface{}, floatOp func(float64, float64) interface{}) (interface{}, error) {
	if l, ok := left.(uint64); ok {
		if r, ok := right.(uint64); ok {
			return uintOp(l, r), nil
		}
		if r, ok := right.(int64); ok {
			return intOp(int64(l), r), nil
		}
	}
	if l, ok := left.(int64); ok {
		if r, ok := right.(int64); ok {
			return intOp(l, r), nil
		}
		if r, ok := right.(uint64); ok {
			return intOp(l, int64(r)), nil
		}
	}
	if l, ok := left.(float64); ok {
		if r, ok := right.(float64); ok {
			return floatOp(l, r), nil
		}
	}
	return nil, fmt.Errorf("invalid numeric operands: left=%v (%T), right=%v (%T)", left, left, right, right)
}

func asBool(v interface{}) (bool, error) {
	if b, ok := v.(bool); ok {
		return b, nil
	}
	return false, fmt.Errorf("expected boolean, got %T", v)
}

func applyOp(op string, left, right interface{}, env Env) (interface{}, error) {
	switch op {
	case "?":
		r, err := asBool(right)
		if err != nil {
			return nil, err
		}
		if !r {
			return nil, fmt.Errorf("assertion failed")
		}
		return left, nil
	case ".":
		if right != nil {
			return nil, fmt.Errorf("expected env, got %v", right)
		}
		return left, nil
	case "=":
		if pat, ok := left.(cbor.Tag); ok && pat.Number == TagSym {
			if env == nil {
				return nil, fmt.Errorf("cannot assign to variable %v (no environment)", pat.Content)
			}
			env[pat.Content.(string)] = right
			return nil, nil
		}
		return nil, fmt.Errorf("cannot assign %v = %v", left, right)
	case "@":
		if rec, ok := left.(map[interface{}]interface{}); ok {
			return rec[right.(cbor.Tag).Content.(string)], nil
		}
		return nil, fmt.Errorf("cannot access key from non-record: %v", left)
	case "+":
		result, err := numOp(left, right,
			func(a, b int64) interface{} { return a + b },
			func(a, b uint64) interface{} { return a + b },
			func(a, b float64) interface{} { return a + b })
		if err != nil {
			return nil, err
		}
		return result, nil
	case "-":
		result, err := numOp(left, right,
			func(a, b int64) interface{} { return a - b },
			func(a, b uint64) interface{} { return int64(a) - int64(b) },
			func(a, b float64) interface{} { return a - b })
		if err != nil {
			return nil, err
		}
		return result, nil
	case "*":
		result, err := numOp(left, right,
			func(a, b int64) interface{} { return a * b },
			func(a, b uint64) interface{} { return int64(a) * int64(b) },
			func(a, b float64) interface{} { return a * b })
		if err != nil {
			return nil, err
		}
		return result, nil
	case "/":
		result, err := numOp(left, right,
			func(a, b int64) interface{} {
				return fmt.Errorf("division not supported for integers")
			},
			func(a, b uint64) interface{} {
				return fmt.Errorf("division not supported for integers")
			},
			func(a, b float64) interface{} {
				if b == 0 {
					return fmt.Errorf("division by zero")
				}
				return a / b
			})
		if err != nil {
			return nil, err
		}
		return result, nil
	case "%":
		result, err := numOp(left, right,
			func(a, b int64) interface{} {
				if b == 0 {
					return fmt.Errorf("modulo by zero")
				}
				return a % b
			},
			func(a, b uint64) interface{} {
				if b == 0 {
					return fmt.Errorf("modulo by zero")
				}
				return a % b
			},
			func(a, b float64) interface{} {
				return fmt.Errorf("modulo not supported for floats")
			})
		if err != nil {
			return nil, err
		}
		return result, nil
	case "^":
		result, err := numOp(left, right,
			func(a, b int64) interface{} {
				return int64(math.Pow(float64(a), float64(b)))
			},
			func(a, b uint64) interface{} {
				return uint64(math.Pow(float64(a), float64(b)))
			},
			func(a, b float64) interface{} {
				return math.Pow(a, b)
			})
		if err != nil {
			return nil, err
		}
		return result, nil
	case "&&":
		l, err := asBool(left)
		if err != nil {
			return nil, err
		}
		r, err := asBool(right)
		if err != nil {
			return nil, err
		}
		return l && r, nil
	case "||":
		l, err := asBool(left)
		if err != nil {
			return nil, err
		}
		r, err := asBool(right)
		if err != nil {
			return nil, err
		}
		return l || r, nil
	case ">+":
		if r, ok := right.([]interface{}); ok {
			return append([]interface{}{left}, r...), nil
		}
		return nil, fmt.Errorf("expected list, got %T", right)
	case "+<":
		if l, ok := left.([]interface{}); ok {
			return append(l, right), nil
		}
		return nil, fmt.Errorf("expected list, got %T", left)
	case "++":
		if l, ok := left.(string); ok {
			if r, ok := right.(string); ok {
				return l + r, nil
			}
		}
		if l, ok := left.([]interface{}); ok {
			if r, ok := right.([]interface{}); ok {
				return append(l, r...), nil
			}
		}
		return nil, fmt.Errorf("expected lists or texts, got %T and %T", left, right)
	case "==":
		// TODO: Compare as bytes.
		return left == right, nil
	case "/=":
		// TODO: Compare as bytes.
		return left != right, nil
	case "<", ">", "<=", ">=":
		result, err := numOp(left, right,
			func(a, b int64) interface{} {
				switch op {
				case "<":
					return a < b
				case ">":
					return a > b
				case "<=":
					return a <= b
				case ">=":
					return a >= b
				}
				return nil
			},
			func(a, b uint64) interface{} {
				switch op {
				case "<":
					return a < b
				case ">":
					return a > b
				case "<=":
					return a <= b
				case ">=":
					return a >= b
				}
				return nil
			},
			func(a, b float64) interface{} {
				switch op {
				case "<":
					return a < b
				case ">":
					return a > b
				case "<=":
					return a <= b
				case ">=":
					return a >= b
				}
				return nil
			})
		if err != nil {
			return nil, err
		}
		return result, nil
	case "'":
		val, err := eval(cbor.Tag{Number: TagExpr, Content: []interface{}{left, right, cbor.Tag{Number: TagSym, Content: "pair"}}}, env)
		if err != nil {
			return nil, err
		}
		return val, nil
	case "|>":
		val, err := eval(cbor.Tag{Number: TagExpr, Content: []interface{}{right, left, cbor.Tag{Number: TagOp, Content: " "}}}, env)
		if err != nil {
			return nil, err
		}
		return val, nil
	case "::":
		return cbor.Tag{Number: TagTag, Content: right.(cbor.Tag).Content}, nil
	case " ":
		if tag, ok := left.(cbor.Tag); ok && tag.Number == TagTag {
			if tag.Content == "true" && right == nil {
				return true, nil
			}
			if tag.Content == "false" && right == nil {
				return false, nil
			}
			return cbor.Tag{Number: TagExpr, Content: []interface{}{tag, right, cbor.Tag{Number: TagOp, Content: " "}}}, nil
		}

		if closure, ok := left.(*closure); ok {
			closureEnv := make(Env)
			for k, v := range closure.env {
				closureEnv[k] = v
			}
			return applyOp(" ", closure.fn, right, closureEnv)
		}

		if fn, ok := left.(cbor.Tag); ok && fn.Number == TagFun {
			cases := fn.Content.([]interface{})
			if len(cases) == 0 {
				return nil, fmt.Errorf("empty function")
			}

			for i := 0; i < len(cases); i += 2 {
				pattern := cases[i]
				body := cases[i+1]

				newEnv := make(Env)
				for k, v := range env {
					newEnv[k] = v
				}

				if pat, ok := pattern.(cbor.Tag); ok && pat.Number == TagSym {
					newEnv[pat.Content.(string)] = right
					result, err := eval(body, newEnv)
					if err != nil {
						return nil, err
					}
					if fn, ok := result.(cbor.Tag); ok && fn.Number == TagFun {
						return &closure{fn: fn, env: newEnv}, nil
					}
					return result, nil
				}
			}
			l, _ := print(left)
			r, _ := print(right)
			return nil, fmt.Errorf("unmatched function application: (%v) %v", l, r)
		}
		return nil, fmt.Errorf("invalid function application: %v", left)
	case ">>":
		if leftFn, ok := left.(*closure); ok {
			if rightFn, ok := right.(*closure); ok {
				composedFn := cbor.Tag{
					Number: TagFun,
					Content: []interface{}{
						cbor.Tag{Number: TagSym, Content: "x"},
						cbor.Tag{
							Number: TagExpr,
							Content: []interface{}{
								rightFn,
								leftFn,
								cbor.Tag{Number: TagSym, Content: "x"},
								cbor.Tag{Number: TagOp, Content: " "},
								cbor.Tag{Number: TagOp, Content: " "},
							},
						},
					},
				}
				return &closure{fn: composedFn, env: env}, nil
			}
			if rightTag, ok := right.(cbor.Tag); ok && rightTag.Number == TagFun {
				rightClosure := &closure{fn: rightTag, env: env}
				return applyOp(">>", leftFn, rightClosure, env)
			}
		}
		if leftTag, ok := left.(cbor.Tag); ok && leftTag.Number == TagFun {
			leftClosure := &closure{fn: leftTag, env: env}
			return applyOp(">>", leftClosure, right, env)
		}
		return nil, fmt.Errorf("function composition requires two functions, got %T and %T", left, right)
	default:
		return nil, fmt.Errorf("unimplemented operator: %v", op)
	}
}

func eval(v interface{}, env Env) (interface{}, error) {
	if v == nil {
		return nil, nil
	}

	if env == nil {
		env = make(Env)
	}

	switch x := v.(type) {
	case *closure:
		return x, nil

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
		case TagFun, TagType, TagTag:
			return x, nil

		case TagSym:
			if name, ok := x.Content.(string); ok {
				if val, ok := env[name]; ok {
					return val, nil
				}
				return nil, fmt.Errorf("undefined variable: %v", name)
			}
			return nil, fmt.Errorf("invalid variable name: %v", x.Content)

		case TagExpr:
			if xs, ok := x.Content.([]interface{}); ok {
				stack := []interface{}{}

				for _, x := range xs {
					if tag, ok := x.(cbor.Tag); ok && tag.Number == TagOp {
						if len(stack) < 2 {
							return nil, fmt.Errorf("insufficient operands for operator '%v' (need 2, have %d)", tag.Content, len(stack))
						}

						op := tag.Content.(string)

						right := stack[len(stack)-1]
						left := stack[len(stack)-2]
						stack = stack[:len(stack)-2]

						if op == "." {
							tmp := right
							right = left
							left = tmp
						}

						if !slices.Contains([]string{"=", "@", "::"}, op) {
							r, err := eval(right, env)
							if err != nil {
								return nil, err
							}
							right = r
						}

						if !slices.Contains([]string{"="}, op) {
							l, err := eval(left, env)
							if err != nil {
								return nil, err
							}
							left = l
						}

						res, err := applyOp(op, left, right, env)
						if err != nil {
							return nil, err
						}

						stack = append(stack, res)

					} else {
						stack = append(stack, x)
					}
				}

				if len(stack) != 1 {
					return nil, fmt.Errorf("invalid expression: expected 1 value on stack, got %d values", len(stack))
				}
				return stack[0], nil
			}
			return nil, fmt.Errorf("invalid expression structure: %v", x.Content)

		default:
			return nil, fmt.Errorf("unsupported tag number: %v", x.Number)
		}

	default:
		return nil, fmt.Errorf("unsupported type: %v (%T)", v, v)
	}
}

func Eval(flat Flat, env Env) (Flat, error) {
	var v interface{}
	err := cbor.Unmarshal(flat, &v)
	if err != nil {
		return nil, err
	}
	res, err := eval(v, env)
	if err != nil {
		return nil, err
	}
	if closure, ok := res.(*closure); ok {
		return cbor.Marshal(closure.fn)
	}
	return cbor.Marshal(res)
}
