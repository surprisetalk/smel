/*
* All of this was copied from github:tekknolagi/scrapscript using Claude.
*
* This parallel go implementation should be thrown away when the language stabilizes.
*
 */

package scrapscript

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

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
	TokenEtc
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

// validOperators contains all valid operators, organized by length for efficient lookup
var validOperators = struct {
	len1 map[string]bool
	len2 map[string]bool
	len3 map[string]bool
}{
	len1: map[string]bool{
		"+": true, "-": true, "*": true, "/": true, "^": true, "%": true,
		"<": true, ">": true, "!": true, ".": true, "=": true, ",": true,
		":": true, "?": true, "|": true, "@": true, "'": true, ";": true,
	},
	len2: map[string]bool{
		"++": true, "+<": true, ">+": true, "==": true, "/=": true, "<=": true,
		">=": true, "&&": true, "||": true, "->": true, "..": true, ">>": true,
		"<<": true, "|>": true, "::": true,
	},
	len3: map[string]bool{
		"...": true,
	},
}

func (l *lexer) readOperator() (Token, error) {
	// Try to read a 3-character operator
	if l.pos+2 < len(l.text) {
		op3 := string(l.text[l.pos]) + string(l.text[l.pos+1]) + string(l.text[l.pos+2])
		if validOperators.len3[op3] {
			l.pos += 3
			if op3 == "..." {
				return Token{Type: TokenEtc, Value: nil}, nil
			}
			return Token{Type: TokenOperator, Value: op3}, nil
		}
	}

	// Try to read a 2-character operator
	if l.pos+1 < len(l.text) {
		op2 := string(l.text[l.pos]) + string(l.text[l.pos+1])
		if validOperators.len2[op2] {
			l.pos += 2
			return Token{Type: TokenOperator, Value: op2}, nil
		}
	}

	// Try to read a 1-character operator
	op1 := string(l.text[l.pos])
	if validOperators.len1[op1] {
		l.pos++
		return Token{Type: TokenOperator, Value: op1}, nil
	}

	return Token{}, fmt.Errorf("invalid operator: %s", op1)
}

func (l *lexer) readBytes() (Token, error) {
	l.advance() // skip second ~
	var str strings.Builder
	str.WriteString(l.readWhile(func(c byte) bool {
		return !unicode.IsSpace(rune(c))
	}))
	data, err := base64.StdEncoding.DecodeString(str.String())
	if err != nil {
		return Token{}, fmt.Errorf("base64 problem: %v", err)
	}
	return Token{TokenBytesLit, data}, nil
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
	case strings.ContainsRune("+-*/<>=!&|.,:|?@^%'", rune(c)):
		return l.readOperator()
	case unicode.IsLetter(rune(c)) || c == '$' || c == '_':
		id := l.readWhile(func(c byte) bool {
			return unicode.IsLetter(rune(c)) || unicode.IsDigit(rune(c)) || c == '$' || c == '\'' || c == '_' || c == '/'
		})
		return Token{Type: TokenName, Value: id}, nil
	}

	return Token{}, fmt.Errorf("unexpected character: %c", c)
}

func Lex(input string) ([]Token, error) {
	l := &lexer{text: input}
	var Tokens []Token

	for {
		Token, err := l.nextToken()
		if err != nil {
			return nil, err
		}
		if Token.Type == TokenEOF {
			break
		}
		Tokens = append(Tokens, Token)
	}

	return Tokens, nil
}
