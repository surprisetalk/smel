package scrapscript

import (
	"fmt"
	"math"

	"github.com/fxamacker/cbor/v2"
)

type Env map[string]interface{}

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

func asList(v interface{}) ([]interface{}, error) {
	if l, ok := v.([]interface{}); ok {
		return l, nil
	}
	return nil, fmt.Errorf("expected list, got %T", v)
}

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
		case TagFun, TagTag:
			return x, nil

		case TagVar:
			if name, ok := x.Content.(string); ok {
				if val, ok := env[name]; ok {
					return val, nil
				}
				return nil, fmt.Errorf("undefined variable '%v'", name)
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

						right := stack[len(stack)-1]
						left := stack[len(stack)-2]
						stack = stack[:len(stack)-2]

						op := tag.Content.(string)
						switch op {
						case "+":
							result, err := numOp(left, right,
								func(a, b int64) interface{} { return a + b },
								func(a, b uint64) interface{} { return a + b },
								func(a, b float64) interface{} { return a + b })
							if err != nil {
								return nil, err
							}
							stack = append(stack, result)
							continue
						case "-":
							result, err := numOp(left, right,
								func(a, b int64) interface{} { return a - b },
								func(a, b uint64) interface{} { return int64(a) - int64(b) },
								func(a, b float64) interface{} { return a - b })
							if err != nil {
								return nil, err
							}
							stack = append(stack, result)
							continue
						case "*":
							result, err := numOp(left, right,
								func(a, b int64) interface{} { return a * b },
								func(a, b uint64) interface{} { return int64(a) * int64(b) },
								func(a, b float64) interface{} { return a * b })
							if err != nil {
								return nil, err
							}
							stack = append(stack, result)
							continue
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
							stack = append(stack, result)
							continue
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
							stack = append(stack, result)
							continue
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
							stack = append(stack, result)
							continue
						case "&&":
							l, err := asBool(left)
							if err != nil {
								return nil, err
							}
							r, err := asBool(right)
							if err != nil {
								return nil, err
							}
							stack = append(stack, l && r)
							continue
						case "||":
							l, err := asBool(left)
							if err != nil {
								return nil, err
							}
							r, err := asBool(right)
							if err != nil {
								return nil, err
							}
							stack = append(stack, l || r)
							continue
						case ">+":
							r, err := asList(right)
							if err != nil {
								return nil, err
							}
							stack = append(stack, append([]interface{}{left}, r...))
							continue
						case "++":
							// TODO: For now, just return the expression unevaluated
							stack = append(stack, cbor.Tag{
								Number:  TagExpr,
								Content: []interface{}{left, right, "++"},
							})
							continue
						case "==":
							// TODO: Compare as bytes.
							stack = append(stack, left == right)
							continue
						case "/=":
							// TODO: Compare as bytes.
							stack = append(stack, left != right)
							continue
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
							stack = append(stack, result)
							continue
						case "'":
							val, err := eval(cbor.Tag{Number: TagExpr, Content: []interface{}{left, right, cbor.Tag{Number: TagTag, Content: "pair"}}}, env)
							if err != nil {
								return nil, err
							}
							stack = append(stack, val)
							continue
						case "|>":
							val, err := eval(cbor.Tag{Number: TagExpr, Content: []interface{}{right, left, cbor.Tag{Number: TagOp, Content: " "}}}, env)
							if err != nil {
								return nil, err
							}
							stack = append(stack, val)
							continue
						case "::":
							stack = append(stack, cbor.Tag{Number: TagTag, Content: right.(cbor.Tag).Content})
							continue
						case " ":
							l, err := eval(left, env)
							if err != nil {
								return nil, err
							}
							left = l
							if tag, ok := left.(cbor.Tag); ok && tag.Number == TagTag {
								val, err := eval(right, env)
								if err != nil {
									return nil, err
								}
								if tag.Content == "true" && right == nil {
									stack = append(stack, true)
									continue
								}
								if tag.Content == "false" && right == nil {
									stack = append(stack, false)
									continue
								}
								stack = append(stack, cbor.Tag{Number: TagExpr, Content: []interface{}{tag, val, cbor.Tag{Number: TagOp, Content: " "}}})
								continue
							}
							if fn, ok := left.(cbor.Tag); ok && fn.Number == TagFun {
								isMatch := false
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

									if pat, ok := pattern.(cbor.Tag); ok && pat.Number == TagVar {
										newEnv[pat.Content.(string)] = right
										result, err := eval(body, newEnv)
										if err != nil {
											return nil, err
										}
										stack = append(stack, result)
										isMatch = true
										break
									}
								}
								if !isMatch {
									return nil, fmt.Errorf("unmatched function application")
								}
								continue
							}
							return nil, fmt.Errorf("invalid function application: %v", left)
						default:
							return nil, fmt.Errorf("unimplemented operator: %v", op)
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
	return cbor.Marshal(res)
}

/*
def match(obj: Object, pattern: Object) -> Optional[Env]:
    if isinstance(pattern, Hole):
        return {} if isinstance(obj, Hole) else None
    if isinstance(pattern, Int):
        return {} if isinstance(obj, Int) and obj.value == pattern.value else None
    if isinstance(pattern, Float):
        raise MatchError("pattern matching is not supported for Floats")
    if isinstance(pattern, String):
        return {} if isinstance(obj, String) and obj.value == pattern.value else None
    if isinstance(pattern, Var):
        return {pattern.name: obj}
    if isinstance(pattern, Variant):
        if not isinstance(obj, Variant):
            return None
        if obj.tag != pattern.tag:
            return None
        return match(obj.value, pattern.value)
    if isinstance(pattern, Record):
        if not isinstance(obj, Record):
            return None
        result: Env = {}
        use_spread = False
        seen_keys: set[str] = set()
        for key, pattern_item in pattern.data.items():
            if isinstance(pattern_item, Spread):
                use_spread = True
                if pattern_item.name is not None:
                    assert isinstance(result, dict)  # for .update()
                    rest_keys = set(obj.data.keys()) - seen_keys
                    result.update({pattern_item.name: Record({key: obj.data[key] for key in rest_keys})})
                break
            seen_keys.add(key)
            obj_item = obj.data.get(key)
            if obj_item is None:
                return None
            part = match(obj_item, pattern_item)
            if part is None:
                return None
            assert isinstance(result, dict)  # for .update()
            result.update(part)
        if not use_spread and len(pattern.data) != len(obj.data):
            return None
        return result
    if isinstance(pattern, List):
        if not isinstance(obj, List):
            return None
        result: Env = {}  # type: ignore
        use_spread = False
        for i, pattern_item in enumerate(pattern.items):
            if isinstance(pattern_item, Spread):
                use_spread = True
                if pattern_item.name is not None:
                    assert isinstance(result, dict)  # for .update()
                    result.update({pattern_item.name: List(obj.items[i:])})
                break
            if i >= len(obj.items):
                return None
            obj_item = obj.items[i]
            part = match(obj_item, pattern_item)
            if part is None:
                return None
            assert isinstance(result, dict)  # for .update()
            result.update(part)
        if not use_spread and len(pattern.items) != len(obj.items):
            return None
        return result
    raise NotImplementedError(f"match not implemented for {type(pattern).__name__}")


def eval_exp(env: Env, exp: Object) -> Object:
    logger.debug(exp)
    if isinstance(exp, (Int, Float, String, Bytes, Hole, Closure, NativeFunction)):
        return exp
    if isinstance(exp, Variant):
        return Variant(exp.tag, eval_exp(env, exp.value))
    if isinstance(exp, Var):
        value = env.get(exp.name)
        if value is None:
            raise NameError(f"name '{exp.name}' is not defined")
        return value
    if isinstance(exp, Binop):
        handler = BINOP_HANDLERS.get(exp.op)
        if handler is None:
            raise NotImplementedError(f"no handler for {exp.op}")
        return handler(env, exp.left, exp.right)
    if isinstance(exp, List):
        return List([eval_exp(env, item) for item in exp.items])
    if isinstance(exp, Record):
        return Record({k: eval_exp(env, exp.data[k]) for k in exp.data})
    if isinstance(exp, Assign):
        # TODO(max): Rework this. There's something about matching that we need
        # to figure out and implement.
        assert isinstance(exp.name, Var)
        value = eval_exp(env, exp.value)
        if isinstance(value, Closure):
            # We want functions to be able to call themselves without using the
            # Y combinator or similar, so we bind functions (and only
            # functions) using a letrec-like strategy. We augment their
            # captured environment with a binding to themselves.
            assert isinstance(value.env, dict)
            value.env[exp.name.name] = value
            # We still improve_closure here even though we also did it on
            # Closure creation because the Closure might not need a binding for
            # itself (it might not be recursive).
            value = improve_closure(value)
        return EnvObject({**env, exp.name.name: value})
    if isinstance(exp, Where):
        assert isinstance(exp.binding, Assign)
        res_env = eval_exp(env, exp.binding)
        assert isinstance(res_env, EnvObject)
        new_env = {**env, **res_env.env}
        return eval_exp(new_env, exp.body)
    if isinstance(exp, Assert):
        cond = eval_exp(env, exp.cond)
        if cond != TRUE:
            raise AssertionError(f"condition {exp.cond} failed")
        return eval_exp(env, exp.value)
    if isinstance(exp, Function):
        if not isinstance(exp.arg, Var):
            raise RuntimeError(f"expected variable in function definition {exp.arg}")
        value = Closure(env, exp)
        value = improve_closure(value)
        return value
    if isinstance(exp, MatchFunction):
        value = Closure(env, exp)
        value = improve_closure(value)
        return value
    if isinstance(exp, Apply):
        if isinstance(exp.func, Var) and exp.func.name == "$$quote":
            return exp.arg
        callee = eval_exp(env, exp.func)
        arg = eval_exp(env, exp.arg)
        if isinstance(callee, NativeFunction):
            return callee.func(arg)
        if not isinstance(callee, Closure):
            raise TypeError(f"attempted to apply a non-closure of type {type(callee).__name__}")
        if isinstance(callee.func, Function):
            assert isinstance(callee.func.arg, Var)
            new_env = {**callee.env, callee.func.arg.name: arg}
            return eval_exp(new_env, callee.func.body)
        elif isinstance(callee.func, MatchFunction):
            for case in callee.func.cases:
                m = match(arg, case.pattern)
                if m is None:
                    continue
                return eval_exp({**callee.env, **m}, case.body)
            raise MatchError("no matching cases")
        else:
            raise TypeError(f"attempted to apply a non-function of type {type(callee.func).__name__}")
    if isinstance(exp, Access):
        obj = eval_exp(env, exp.obj)
        if isinstance(obj, Record):
            if not isinstance(exp.at, Var):
                raise TypeError(f"cannot access record field using {type(exp.at).__name__}, expected a field name")
            if exp.at.name not in obj.data:
                raise NameError(f"no assignment to {exp.at.name} found in record")
            return obj.data[exp.at.name]
        elif isinstance(obj, List):
            access_at = eval_exp(env, exp.at)
            if not isinstance(access_at, Int):
                raise TypeError(f"cannot index into list using type {type(access_at).__name__}, expected integer")
            if access_at.value < 0 or access_at.value >= len(obj.items):
                raise ValueError(f"index {access_at.value} out of bounds for list")
            return obj.items[access_at.value]
        raise TypeError(f"attempted to access from type {type(obj).__name__}")
    elif isinstance(exp, Spread):
        raise RuntimeError("cannot evaluate a spread")
    raise NotImplementedError(f"eval_exp not implemented for {exp}")
*/
