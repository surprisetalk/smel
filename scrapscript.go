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

/*

def parse_assign(tokens: Token[], p: float = 0) -> "Assign":
    assign = parse_binary(tokens, p)
    if isinstance(assign, Spread):
        return Assign(Var("..."), assign)
    if not isinstance(assign, Assign):
        raise ParseError("failed to parse variable assignment in record constructor")
    return assign


def gensym() -> str:
    gensym.counter += 1  # type: ignore
    return f"$v{gensym.counter}"  # type: ignore


def gensym_reset() -> None:
    gensym.counter = -1  # type: ignore


gensym_reset()


def parse_unary(tokens: Token[], p: float) -> "Object":
    token = next(tokens)
    l: Object
    if isinstance(token, IntLit):
        return Int(token.value)
    elif isinstance(token, FloatLit):
        return Float(token.value)
    elif isinstance(token, Name):
        # TODO: Handle kebab case vars
        return Var(token.value)
    elif isinstance(token, Hash):
        if isinstance(variant := next(tokens), Name):
            # It needs to be higher than the precedence of the -> operator so that
            # we can match variants in MatchFunction
            # It needs to be higher than the precedence of the && operator so that
            # we can use #true() and #false() in boolean expressions
            # It needs to be higher than the precedence of juxtaposition so that
            # f #true() #false() is parsed as f(TRUE)(FALSE)
            return Variant(variant.value, parse_binary(tokens, PS[""].pr + 1))
        else:
            raise UnexpectedTokenError(variant)
    elif isinstance(token, BytesLit):
        base = token.base
        if base == 85:
            l = Bytes(base64.b85decode(token.value))
        elif base == 64:
            l = Bytes(base64.b64decode(token.value))
        elif base == 32:
            l = Bytes(base64.b32decode(token.value))
        elif base == 16:
            l = Bytes(base64.b16decode(token.value))
        else:
            raise ParseError(f"unexpected base {base!r} in {token!r}")
        return l
    elif isinstance(token, StringLit):
        return String(token.value)
    elif token == Operator("..."):
        try:
            if isinstance(tokens.peek(), Name):
                return Spread(next(tokens).value)
            else:
                return Spread()
        except StopIteration:
            return Spread()
    elif token == Operator("|"):
        expr = parse_binary(tokens, PS["|"].pr)  # TODO: make this work for larger arities
        if not isinstance(expr, Function):
            raise ParseError(f"expected function in match expression {expr!r}")
        cases = [MatchCase(expr.arg, expr.body)]
        while True:
            try:
                if tokens.peek() != Operator("|"):
                    break
            except StopIteration:
                break
            next(tokens)
            expr = parse_binary(tokens, PS["|"].pr)  # TODO: make this work for larger arities
            if not isinstance(expr, Function):
                raise ParseError(f"expected function in match expression {expr!r}")
            cases.append(MatchCase(expr.arg, expr.body))
        return MatchFunction(cases)
    elif isinstance(token, LeftParen):
        if isinstance(tokens.peek(), RightParen):
            l = Hole()
        else:
            l = parse(tokens)
        next(tokens)
        return l
    elif isinstance(token, LeftBracket):
        l = List([])
        token = tokens.peek()
        if isinstance(token, RightBracket):
            next(tokens)
        else:
            l.items.append(parse_binary(tokens, 2))
            while not isinstance(next(tokens), RightBracket):
                if isinstance(l.items[-1], Spread):
                    raise ParseError("spread must come at end of list match")
                # TODO: Implement .. operator
                l.items.append(parse_binary(tokens, 2))
        return l
    elif isinstance(token, LeftBrace):
        l = Record({})
        token = tokens.peek()
        if isinstance(token, RightBrace):
            next(tokens)
        else:
            assign = parse_assign(tokens, 2)
            l.data[assign.name.name] = assign.value
            while not isinstance(next(tokens), RightBrace):
                if isinstance(assign.value, Spread):
                    raise ParseError("spread must come at end of record match")
                # TODO: Implement .. operator
                assign = parse_assign(tokens, 2)
                l.data[assign.name.name] = assign.value
        return l
    elif token == Operator("-"):
        # Unary minus
        # Precedence was chosen to be higher than binary ops so that -a op
        # b is (-a) op b and not -(a op b).
        # Precedence was chosen to be higher than function application so that
        # -a b is (-a) b and not -(a b).
        r = parse_binary(tokens, HIGHEST_PREC + 1)
        if isinstance(r, Int):
            assert r.value >= 0, "Tokens should never have negative values"
            return Int(-r.value)
        if isinstance(r, Float):
            assert r.value >= 0, "Tokens should never have negative values"
            return Float(-r.value)
        return Binop(BinopKind.SUB, Int(0), r)
    else:
        raise UnexpectedTokenError(token)


def parse_binary(tokens: Token[], p: float) -> "Object":
    l: Object = parse_unary(tokens, p)
    while True:
        op: Token
        try:
            op = tokens.peek()
        except StopIteration:
            break
        if isinstance(op, (RightParen, RightBracket, RightBrace)):
            break
        if not isinstance(op, Operator):
            prec = PS[""]
            pl, pr = prec.pl, prec.pr
            if pl < p:
                break
            l = Apply(l, parse_binary(tokens, pr))
            continue
        prec = PS[op.value]
        pl, pr = prec.pl, prec.pr
        if pl < p:
            break
        next(tokens)
        if op == Operator("="):
            if not isinstance(l, Var):
                raise ParseError(f"expected variable in assignment {l!r}")
            l = Assign(l, parse_binary(tokens, pr))
        elif op == Operator("->"):
            l = Function(l, parse_binary(tokens, pr))
        elif op == Operator("|>"):
            l = Apply(parse_binary(tokens, pr), l)
        elif op == Operator("<|"):
            l = Apply(l, parse_binary(tokens, pr))
        elif op == Operator(">>"):
            r = parse_binary(tokens, pr)
            varname = gensym()
            l = Function(Var(varname), Apply(r, Apply(l, Var(varname))))
        elif op == Operator("<<"):
            r = parse_binary(tokens, pr)
            varname = gensym()
            l = Function(Var(varname), Apply(l, Apply(r, Var(varname))))
        elif op == Operator("."):
            l = Where(l, parse_binary(tokens, pr))
        elif op == Operator("?"):
            l = Assert(l, parse_binary(tokens, pr))
        elif op == Operator("@"):
            # TODO: revisit whether to use @ or . for field access
            l = Access(l, parse_binary(tokens, pr))
        else:
            assert isinstance(op, Operator)
            l = Binop(BinopKind.from_str(op.value), l, parse_binary(tokens, pr))
    return l


def parse(tokens: Token[]) -> "Object":
    try:
        return parse_binary(tokens, 0)
    except StopIteration:
        raise UnexpectedEOFError("unexpected end of input")

*/

//// ENCODE

/*
tags = [
    TYPE_SHORT := b"i",  # fits in 64 bits
    TYPE_LONG := b"l",  # bignum
    TYPE_FLOAT := b"d",
    TYPE_STRING := b"s",
    TYPE_REF := b"r",
    TYPE_LIST := b"[",
    TYPE_RECORD := b"{",
    TYPE_VARIANT := b"#",
    TYPE_VAR := b"v",
    TYPE_FUNCTION := b"f",
    TYPE_MATCH_FUNCTION := b"m",
    TYPE_CLOSURE := b"c",
    TYPE_BYTES := b"b",
    TYPE_HOLE := b"(",
    TYPE_ASSIGN := b"=",
    TYPE_BINOP := b"+",
    TYPE_APPLY := b" ",
    TYPE_WHERE := b".",
    TYPE_ACCESS := b"@",
    TYPE_SPREAD := b"S",
    TYPE_NAMED_SPREAD := b"R",
]
FLAG_REF = 0x80


DIGIT_MASK = (1 << 64) - 1

def ref(tag: bytes) -> bytes:
    return (tag[0] | FLAG_REF).to_bytes(1, "little")

tags = tags + [ref(v) for v in tags]
assert len(tags) == len(set(tags)), "Duplicate tags"
assert all(len(v) == 1 for v in tags), "Tags must be 1 byte"
assert all(isinstance(v, bytes) for v in tags)


def zigzag_encode(val: int) -> int:
    if val < 0:
        return -2 * val - 1
    return 2 * val


def zigzag_decode(val: int) -> int:
    if val & 1 == 1:
        return -val // 2
    return val // 2


@dataclass
class Serializer:
    refs: typing.List[Object] = dataclasses.field(default_factory=list)
    output: bytearray = dataclasses.field(default_factory=bytearray)

    def ref(self, obj: Object) -> Optional[int]:
        for idx, ref in enumerate(self.refs):
            if ref is obj:
                return idx
        return None

    def add_ref(self, ty: bytes, obj: Object) -> int:
        assert len(ty) == 1
        assert self.ref(obj) is None
        self.emit(ref(ty))
        result = len(self.refs)
        self.refs.append(obj)
        return result

    def emit(self, obj: bytes) -> None:
        self.output.extend(obj)

    def _fits_in_nbits(self, obj: int, nbits: int) -> bool:
        return -(1 << (nbits - 1)) <= obj < (1 << (nbits - 1))

    def _short(self, number: int) -> bytes:
        # From Peter Ruibal, https://github.com/fmoo/python-varint
        number = zigzag_encode(number)
        buf = bytearray()
        while True:
            towrite = number & 0x7F
            number >>= 7
            if number:
                buf.append(towrite | 0x80)
            else:
                buf.append(towrite)
                break
        return bytes(buf)

    def _long(self, number: int) -> bytes:
        digits = []
        number = zigzag_encode(number)
        while number:
            digits.append(number & DIGIT_MASK)
            number >>= BITS_PER_DIGIT
        buf = bytearray(self._short(len(digits)))
        for digit in digits:
            buf.extend(digit.to_bytes(BYTES_PER_DIGIT, "little"))
        return bytes(buf)

    def _string(self, obj: str) -> bytes:
        encoded = obj.encode("utf-8")
        return self._short(len(encoded)) + encoded

    def serialize(self, obj: Object) -> None:
        assert isinstance(obj, Object), type(obj)
        if (ref := self.ref(obj)) is not None:
            return self.emit(TYPE_REF + self._short(ref))
        if isinstance(obj, Int):
            if self._fits_in_nbits(obj.value, 64):
                self.emit(TYPE_SHORT)
                self.emit(self._short(obj.value))
                return
            self.emit(TYPE_LONG)
            self.emit(self._long(obj.value))
            return
        if isinstance(obj, String):
            return self.emit(TYPE_STRING + self._string(obj.value))
        if isinstance(obj, List):
            self.add_ref(TYPE_LIST, obj)
            self.emit(self._short(len(obj.items)))
            for item in obj.items:
                self.serialize(item)
            return
        if isinstance(obj, Variant):
            # TODO(max): Determine if this should be a ref
            self.emit(TYPE_VARIANT)
            # TODO(max): String pool (via refs) for strings longer than some length?
            self.emit(self._string(obj.tag))
            return self.serialize(obj.value)
        if isinstance(obj, Record):
            # TODO(max): Determine if this should be a ref
            self.emit(TYPE_RECORD)
            self.emit(self._short(len(obj.data)))
            for key, value in obj.data.items():
                self.emit(self._string(key))
                self.serialize(value)
            return
        if isinstance(obj, Var):
            return self.emit(TYPE_VAR + self._string(obj.name))
        if isinstance(obj, Function):
            self.emit(TYPE_FUNCTION)
            self.serialize(obj.arg)
            return self.serialize(obj.body)
        if isinstance(obj, MatchFunction):
            self.emit(TYPE_MATCH_FUNCTION)
            self.emit(self._short(len(obj.cases)))
            for case in obj.cases:
                self.serialize(case.pattern)
                self.serialize(case.body)
            return
        if isinstance(obj, Closure):
            self.add_ref(TYPE_CLOSURE, obj)
            self.serialize(obj.func)
            self.emit(self._short(len(obj.env)))
            for key, value in obj.env.items():
                self.emit(self._string(key))
                self.serialize(value)
            return
        if isinstance(obj, Bytes):
            self.emit(TYPE_BYTES)
            self.emit(self._short(len(obj.value)))
            self.emit(obj.value)
            return
        if isinstance(obj, Float):
            self.emit(TYPE_FLOAT)
            self.emit(struct.pack("<d", obj.value))
            return
        if isinstance(obj, Hole):
            self.emit(TYPE_HOLE)
            return
        if isinstance(obj, Assign):
            self.emit(TYPE_ASSIGN)
            self.serialize(obj.name)
            self.serialize(obj.value)
            return
        if isinstance(obj, Binop):
            self.emit(TYPE_BINOP)
            self.emit(self._string(BinopKind.to_str(obj.op)))
            self.serialize(obj.left)
            self.serialize(obj.right)
            return
        if isinstance(obj, Apply):
            self.emit(TYPE_APPLY)
            self.serialize(obj.func)
            self.serialize(obj.arg)
            return
        if isinstance(obj, Where):
            self.emit(TYPE_WHERE)
            self.serialize(obj.body)
            self.serialize(obj.binding)
            return
        if isinstance(obj, Access):
            self.emit(TYPE_ACCESS)
            self.serialize(obj.obj)
            self.serialize(obj.at)
            return
        if isinstance(obj, Spread):
            if obj.name is not None:
                self.emit(TYPE_NAMED_SPREAD)
                self.emit(self._string(obj.name))
                return
            self.emit(TYPE_SPREAD)
            return
        raise NotImplementedError(type(obj))


*/

//// DECODE

/*

@dataclass
class Deserializer:
    flat: Union[bytes, memoryview]
    idx: int = 0
    refs: typing.List[Object] = dataclasses.field(default_factory=list)

    def __post_init__(self) -> None:
        if isinstance(self.flat, bytes):
            self.flat = memoryview(self.flat)

    def read(self, size: int) -> memoryview:
        result = memoryview(self.flat[self.idx : self.idx + size])
        self.idx += size
        return result

    def read_tag(self) -> Tuple[bytes, bool]:
        tag = self.read(1)[0]
        is_ref = bool(tag & FLAG_REF)
        return (tag & ~FLAG_REF).to_bytes(1, "little"), is_ref

    def _string(self) -> str:
        length = self._short()
        encoded = self.read(length)
        return str(encoded, "utf-8")

    def _short(self) -> int:
        # From Peter Ruibal, https://github.com/fmoo/python-varint
        shift = 0
        result = 0
        while True:
            i = self.read(1)[0]
            result |= (i & 0x7F) << shift
            shift += 7
            if not (i & 0x80):
                break
        return zigzag_decode(result)

    def _long(self) -> int:
        num_digits = self._short()
        digits = []
        for _ in range(num_digits):
            digit = int.from_bytes(self.read(BYTES_PER_DIGIT), "little")
            digits.append(digit)
        result = 0
        for digit in reversed(digits):
            result <<= BITS_PER_DIGIT
            result |= digit
        return zigzag_decode(result)

    def parse(self) -> Object:
        ty, is_ref = self.read_tag()
        if ty == TYPE_REF:
            idx = self._short()
            return self.refs[idx]
        if ty == TYPE_SHORT:
            assert not is_ref
            return Int(self._short())
        if ty == TYPE_LONG:
            assert not is_ref
            return Int(self._long())
        if ty == TYPE_STRING:
            assert not is_ref
            return String(self._string())
        if ty == TYPE_LIST:
            length = self._short()
            result_list = List([])
            assert is_ref
            self.refs.append(result_list)
            for i in range(length):
                result_list.items.append(self.parse())
            return result_list
        if ty == TYPE_RECORD:
            assert not is_ref
            length = self._short()
            result_rec = Record({})
            for i in range(length):
                key = self._string()
                value = self.parse()
                result_rec.data[key] = value
            return result_rec
        if ty == TYPE_VARIANT:
            assert not is_ref
            tag = self._string()
            value = self.parse()
            return Variant(tag, value)
        if ty == TYPE_VAR:
            assert not is_ref
            return Var(self._string())
        if ty == TYPE_FUNCTION:
            assert not is_ref
            arg = self.parse()
            body = self.parse()
            return Function(arg, body)
        if ty == TYPE_MATCH_FUNCTION:
            assert not is_ref
            length = self._short()
            result_matchfun = MatchFunction([])
            for i in range(length):
                pattern = self.parse()
                body = self.parse()
                result_matchfun.cases.append(MatchCase(pattern, body))
            return result_matchfun
        if ty == TYPE_CLOSURE:
            func = self.parse()
            length = self._short()
            assert isinstance(func, (Function, MatchFunction))
            result_closure = Closure({}, func)
            assert is_ref
            self.refs.append(result_closure)
            for i in range(length):
                key = self._string()
                value = self.parse()
                assert isinstance(result_closure.env, dict)  # For mypy
                result_closure.env[key] = value
            return result_closure
        if ty == TYPE_BYTES:
            assert not is_ref
            length = self._short()
            return Bytes(self.read(length))
        if ty == TYPE_FLOAT:
            assert not is_ref
            return Float(struct.unpack("<d", self.read(8))[0])
        if ty == TYPE_HOLE:
            assert not is_ref
            return Hole()
        if ty == TYPE_ASSIGN:
            assert not is_ref
            name = self.parse()
            value = self.parse()
            assert isinstance(name, Var)
            return Assign(name, value)
        if ty == TYPE_BINOP:
            assert not is_ref
            op = BinopKind.from_str(self._string())
            left = self.parse()
            right = self.parse()
            return Binop(op, left, right)
        if ty == TYPE_APPLY:
            assert not is_ref
            func = self.parse()
            arg = self.parse()
            return Apply(func, arg)
        if ty == TYPE_WHERE:
            assert not is_ref
            body = self.parse()
            binding = self.parse()
            return Where(body, binding)
        if ty == TYPE_ACCESS:
            assert not is_ref
            obj = self.parse()
            at = self.parse()
            return Access(obj, at)
        if ty == TYPE_SPREAD:
            return Spread()
        if ty == TYPE_NAMED_SPREAD:
            return Spread(self._string())
        raise NotImplementedError(bytes(ty))


*/

//// DECODE

/*

def unpack_number(obj: Object) -> Union[int, float]:
    if not isinstance(obj, (Int, Float)):
        raise TypeError(f"expected Int or Float, got {type(obj).__name__}")
    return obj.value


def eval_number(env: Env, exp: Object) -> Union[int, float]:
    result = eval_exp(env, exp)
    return unpack_number(result)


def eval_str(env: Env, exp: Object) -> str:
    result = eval_exp(env, exp)
    if not isinstance(result, String):
        raise TypeError(f"expected String, got {type(result).__name__}")
    return result.value


def eval_bool(env: Env, exp: Object) -> bool:
    result = eval_exp(env, exp)
    if not isinstance(result, Variant):
        raise TypeError(f"expected #true or #false, got {type(result).__name__}")
    if result.tag not in ("true", "false"):
        raise TypeError(f"expected #true or #false, got {type(result).__name__}")
    return result.tag == "true"


def eval_list(env: Env, exp: Object) -> typing.List[Object]:
    result = eval_exp(env, exp)
    if not isinstance(result, List):
        raise TypeError(f"expected List, got {type(result).__name__}")
    return result.items


def make_bool(x: bool) -> Object:
    return TRUE if x else FALSE


def wrap_inferred_number_type(x: Union[int, float]) -> Object:
    # TODO: Since this is intended to be a reference implementation
    # we should avoid relying heavily on Python's implementation of
    # arithmetic operations, type inference, and multiple dispatch.
    # Update this to make the interpreter more language agnostic.
    if isinstance(x, int):
        return Int(x)
    return Float(x)

class MatchError(Exception):
    pass

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


def free_in(exp: Object) -> Set[str]:
    if isinstance(exp, (Int, Float, String, Bytes, Hole, NativeFunction)):
        return set()
    if isinstance(exp, Variant):
        return free_in(exp.value)
    if isinstance(exp, Var):
        return {exp.name}
    if isinstance(exp, Spread):
        if exp.name is not None:
            return {exp.name}
        return set()
    if isinstance(exp, Binop):
        return free_in(exp.left) | free_in(exp.right)
    if isinstance(exp, List):
        if not exp.items:
            return set()
        return set.union(*(free_in(item) for item in exp.items))
    if isinstance(exp, Record):
        if not exp.data:
            return set()
        return set.union(*(free_in(value) for key, value in exp.data.items()))
    if isinstance(exp, Function):
        assert isinstance(exp.arg, Var)
        return free_in(exp.body) - {exp.arg.name}
    if isinstance(exp, MatchFunction):
        if not exp.cases:
            return set()
        return set.union(*(free_in(case) for case in exp.cases))
    if isinstance(exp, MatchCase):
        return free_in(exp.body) - free_in(exp.pattern)
    if isinstance(exp, Apply):
        return free_in(exp.func) | free_in(exp.arg)
    if isinstance(exp, Access):
        # For records, y is not free in x@y; it is a field name.
        # For lists, y *is* free in x@y; it is an index expression (could be a
        # var).
        # For now, we'll assume it might be an expression and mark it as a
        # (possibly extra) freevar.
        return free_in(exp.obj) | free_in(exp.at)
    if isinstance(exp, Where):
        assert isinstance(exp.binding, Assign)
        return (free_in(exp.body) - {exp.binding.name.name}) | free_in(exp.binding)
    if isinstance(exp, Assign):
        return free_in(exp.value)
    if isinstance(exp, Closure):
        # TODO(max): Should this remove the set of keys in the closure env?
        return free_in(exp.func)
    raise NotImplementedError(("free_in", type(exp)))


def improve_closure(closure: Closure) -> Closure:
    freevars = free_in(closure.func)
    env = {boundvar: value for boundvar, value in closure.env.items() if boundvar in freevars}
    return Closure(env, closure.func)


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
