package scrapscript

import (
	"fmt"

	"github.com/fxamacker/cbor/v2"
)

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
							if l, ok := left.(uint64); ok {
								if r, ok := right.(uint64); ok {
									stack = append(stack, l+r)
									continue
								}
							}
							if l, ok := left.(float64); ok {
								if r, ok := right.(float64); ok {
									stack = append(stack, l+r)
									continue
								}
							}
							return nil, fmt.Errorf("invalid operands for +")
						case "/":
							if l, ok := left.(float64); ok {
								if r, ok := right.(float64); ok {
									stack = append(stack, l/r)
									continue
								}
							}
							return nil, fmt.Errorf("invalid operands for /")
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
