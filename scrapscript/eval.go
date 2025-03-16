package scrapscript

import (
	"fmt"
	"math"
	"slices"

	"github.com/fxamacker/cbor/v2"
)

type Env map[string]any

type closure struct {
	fn  cbor.Tag
	env Env
}

type snap struct {
	t any
	k string
	v any
}

func numOp(left, right any, intOp func(int64, int64) any, uintOp func(uint64, uint64) any, floatOp func(float64, float64) any) (any, error) {
	switch l := left.(type) {
	case uint64:
		if r, ok := right.(uint64); ok {
			return uintOp(l, r), nil
		} else if r, ok := right.(int64); ok {
			return intOp(int64(l), r), nil
		}
	case int64:
		if r, ok := right.(int64); ok {
			return intOp(l, r), nil
		} else if r, ok := right.(uint64); ok {
			return intOp(l, int64(r)), nil
		}
	case float64:
		if r, ok := right.(float64); ok {
			return floatOp(l, r), nil
		}
	}
	return nil, fmt.Errorf("invalid numeric operands: left=%v (%T), right=%v (%T)", left, left, right, right)
}

func asBool(v any) (bool, error) {
	if b, ok := v.(bool); ok {
		return b, nil
	}
	return false, fmt.Errorf("expected boolean, got %T", v)
}

func match(pattern, value any, env Env) (any, bool, error) {
	switch p := pattern.(type) {
	case int64:
		if r, ok := value.(int64); ok && p == r {
			return value, true, nil
		}
	case uint64:
		if r, ok := value.(uint64); ok && p == r {
			return value, true, nil
		}
	case snap:
		if r, ok := value.(snap); ok && p.k == r.k {
			return match(p.v, r.v, env)
		}
	case cbor.Tag:
		switch p.Number {
		case TagSym:
			env[p.Content.(string)] = value
			return value, true, nil
		case TagExpr:
			// TODO: Figure out what to do when content >= 3.
			if content, ok := p.Content.([]any); ok && len(content) == 3 {
				if opTag, ok := content[2].(cbor.Tag); ok && opTag.Number == TagOp {
					switch opTag.Content {
					case ">+":
						if rightList, ok := value.([]any); ok && len(rightList) > 0 {
							firstPattern, restPattern := content[0], content[1]
							firstElem, restElems := rightList[0], rightList[1:]
							_, firstMatched, err := match(firstPattern, firstElem, env)
							if err != nil {
								return nil, false, err
							}
							if !firstMatched {
								return nil, false, nil
							}
							_, restMatched, err := match(restPattern, restElems, env)
							if err != nil {
								return nil, false, err
							}
							if restMatched {
								return value, true, nil
							}
						}
					case " ":
						if tagPattern, ok := content[0].(cbor.Tag); ok && tagPattern.Number == TagTag {
							if tagPattern.Content == "true" && content[1] == nil {
								return true, true, nil
							}
							if tagPattern.Content == "false" && content[1] == nil {
								return false, true, nil
							}
							snapPattern := snap{nil, tagPattern.Content.(string), content[1]}
							return match(snapPattern, value, env)
						}
					}
				}
			}
		}
	case map[any]any:
		if r, ok := value.(map[any]any); ok {
			allMatched := true
			for k, patVal := range p {
				rightVal, exists := r[k]
				if !exists {
					allMatched = false
					break
				}
				if patSym, ok := patVal.(cbor.Tag); ok && patSym.Number == TagSym {
					env[patSym.Content.(string)] = rightVal
				} else if _, matched, err := match(patVal, rightVal, env); err != nil {
					return nil, false, err
				} else if !matched {
					allMatched = false
					break
				}
			}
			if allMatched {
				return value, true, nil
			}
		}
	case []any:
		if r, ok := value.([]any); ok && len(p) == len(r) {
			allMatched := true
			for j, patItem := range p {
				rightItem := r[j]
				if patSym, ok := patItem.(cbor.Tag); ok && patSym.Number == TagSym {
					env[patSym.Content.(string)] = rightItem
				} else if _, matched, err := match(patItem, rightItem, env); err != nil {
					return nil, false, err
				} else if !matched {
					allMatched = false
					break
				}
			}
			if allMatched {
				return value, true, nil
			}
		}
	}
	return nil, false, nil
}

func applyOp(op string, left, right any, env Env) (any, error) {
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
		if rec, ok := left.(map[any]any); ok {
			return rec[right.(cbor.Tag).Content.(string)], nil
		}
		return nil, fmt.Errorf("cannot access key from non-record: %v", left)
	case "+", "-", "*", "/", "%", "^":
		opFuncs := map[string]struct {
			intFn   func(int64, int64) any
			uintFn  func(uint64, uint64) any
			floatFn func(float64, float64) any
		}{
			"+": {
				func(a, b int64) any { return a + b },
				func(a, b uint64) any { return a + b },
				func(a, b float64) any { return a + b },
			},
			"-": {
				func(a, b int64) any { return a - b },
				func(a, b uint64) any { return int64(a) - int64(b) },
				func(a, b float64) any { return a - b },
			},
			"*": {
				func(a, b int64) any { return a * b },
				func(a, b uint64) any { return int64(a) * int64(b) },
				func(a, b float64) any { return a * b },
			},
			"/": {
				func(a, b int64) any { return fmt.Errorf("division not supported for integers") },
				func(a, b uint64) any { return fmt.Errorf("division not supported for integers") },
				func(a, b float64) any {
					if b == 0 {
						return fmt.Errorf("division by zero")
					}
					return a / b
				},
			},
			"%": {
				func(a, b int64) any {
					if b == 0 {
						return fmt.Errorf("modulo by zero")
					}
					return a % b
				},
				func(a, b uint64) any {
					if b == 0 {
						return fmt.Errorf("modulo by zero")
					}
					return a % b
				},
				func(a, b float64) any { return fmt.Errorf("modulo not supported for floats") },
			},
			"^": {
				func(a, b int64) any { return int64(math.Pow(float64(a), float64(b))) },
				func(a, b uint64) any { return uint64(math.Pow(float64(a), float64(b))) },
				func(a, b float64) any { return math.Pow(a, b) },
			},
		}

		funcs := opFuncs[op]
		return numOp(left, right, funcs.intFn, funcs.uintFn, funcs.floatFn)
	case "&&", "||":
		l, err := asBool(left)
		if err != nil {
			return nil, err
		}
		r, err := asBool(right)
		if err != nil {
			return nil, err
		}

		if op == "&&" {
			return l && r, nil
		}
		return l || r, nil
	case "==", "/=", "<", ">", "<=", ">=":
		if op == "==" {
			return left == right, nil
		}
		if op == "/=" {
			return left != right, nil
		}

		compFn := func(op string) (
			func(int64, int64) any,
			func(uint64, uint64) any,
			func(float64, float64) any,
		) {
			switch op {
			case "<":
				return func(a, b int64) any { return a < b },
					func(a, b uint64) any { return a < b },
					func(a, b float64) any { return a < b }
			case ">":
				return func(a, b int64) any { return a > b },
					func(a, b uint64) any { return a > b },
					func(a, b float64) any { return a > b }
			case "<=":
				return func(a, b int64) any { return a <= b },
					func(a, b uint64) any { return a <= b },
					func(a, b float64) any { return a <= b }
			case ">=":
				return func(a, b int64) any { return a >= b },
					func(a, b uint64) any { return a >= b },
					func(a, b float64) any { return a >= b }
			}
			return nil, nil, nil
		}

		intFn, uintFn, floatFn := compFn(op)
		return numOp(left, right, intFn, uintFn, floatFn)
	case ">+":
		if r, ok := right.([]any); ok {
			return append([]any{left}, r...), nil
		}
		return nil, fmt.Errorf("expected list, got %T", right)
	case "+<":
		if l, ok := left.([]any); ok {
			return append(l, right), nil
		}
		return nil, fmt.Errorf("expected list, got %T", left)
	case "++":
		if l, ok := left.(string); ok {
			if r, ok := right.(string); ok {
				return l + r, nil
			}
		}
		if l, ok := left.([]any); ok {
			if r, ok := right.([]any); ok {
				return append(l, r...), nil
			}
		}
		return nil, fmt.Errorf("expected lists or texts, got %T and %T", left, right)
	case "'":
		return cbor.Tag{Number: TagExpr, Content: []any{left, right, cbor.Tag{Number: TagOp, Content: "'"}}}, nil
	case "|>":
		val, err := eval(cbor.Tag{Number: TagExpr, Content: []any{right, left, cbor.Tag{Number: TagOp, Content: " "}}}, env)
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
			return snap{nil, tag.Content.(string), right}, nil
		}
		if closure, ok := left.(*closure); ok {
			closureEnv := make(Env)
			for k, v := range closure.env {
				closureEnv[k] = v
			}
			return applyOp(" ", closure.fn, right, closureEnv)
		}
		if fn, ok := left.(cbor.Tag); ok && fn.Number == TagFun {
			cases := fn.Content.([]any)
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

				if _, matched, err := match(pattern, right, newEnv); err != nil {
					return nil, err
				} else if matched {
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
					Content: []any{
						cbor.Tag{Number: TagSym, Content: "x"},
						cbor.Tag{
							Number: TagExpr,
							Content: []any{
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

func eval(v any, env Env) (any, error) {
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

	case snap:
		v_, err := eval(x.v, env)
		if err != nil {
			return nil, err
		}
		return snap{x.t, x.k, v_}, nil

	case []any:
		result := make([]any, len(x))
		for i, item := range x {
			val, err := eval(item, env)
			if err != nil {
				return nil, err
			}
			result[i] = val
		}
		return result, nil

	case map[any]any:
		result := make(map[any]any)
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
			if xs, ok := x.Content.([]any); ok {
				stack := []any{}

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

						if !slices.Contains([]string{"=", "::"}, op) {
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
					return nil, fmt.Errorf("invalid expression: expected 1 value on stack, got %d values: %v", len(stack), stack)
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

func clean(v any) any {
	switch x := v.(type) {
	case *closure:
		return x.fn
	case snap:
		return cbor.Tag{Number: TagExpr, Content: []any{
			cbor.Tag{Number: TagTag, Content: x.k},
			x.v,
			cbor.Tag{Number: TagOp, Content: " "},
		}}
	case []any:
		for i, item := range x {
			x[i] = clean(item)
		}
		return x
	case map[any]any:
		for k, v := range x {
			x[k] = clean(v)
		}
		return x
	case cbor.Tag:
		return cbor.Tag{Number: x.Number, Content: clean(x.Content)}
	default:
		return x
	}
}

func Eval(flat Flat, env Env) (Flat, error) {
	var v any
	err := cbor.Unmarshal(flat, &v)
	if err != nil {
		return nil, err
	}
	res, err := eval(v, env)
	if err != nil {
		return nil, err
	}
	return cbor.Marshal(clean(res))
}
