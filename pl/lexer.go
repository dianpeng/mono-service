package pl

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"unicode"
)

const (
	tkId = iota
	tkGId
	tkRId
	tkInt
	tkReal
	tkStr
	tkDollar
	tkLPar
	tkRPar
	tkLSqr
	tkRSqr
	tkLBra
	tkRBra
	tkArrow

	// used specifically for the string interpolation
	tkDRBra
	tkDot

	tkComma
	tkAssign
	tkColon
	tkScope
	tkSemicolon
	tkQuest
	tkSharp

	// unary
	tkNot

	// arithmetic
	tkAdd
	tkSub
	tkMul
	tkDiv
	tkMod
	tkPow

	// comparison
	tkLt
	tkLe
	tkGt
	tkGe
	tkEq
	tkNe

	// regex
	tkRegexpMatch
	tkRegexpNMatch
	tkRegex

	// multiple line string
	tkMStr

	// logical
	tkAnd
	tkOr

	// keyword
	tkTrue
	tkFalse
	tkNull
	tkSession
	tkLet
	tkWhen
	tkImport
	tkIf
	tkElif
	tkElse
	tkTry
	tkReturn
	tkContinue
	tkBreak
	tkNext

	// intrinsic keywords, used for special builtin functionalities
	tkTemplate

	tkEof
	tkError
)

type lexer struct {
	input  []rune
	cursor int
	token  int

	// lexeme
	valueInt  int64
	valueReal float64
	valueText string
}

func newLexer(input string) *lexer {
	return &lexer{
		input: []rune(input),
	}
}

func getTokenName(tk int) string {
	switch tk {
	case tkId:
		return "id"
	case tkGId:
		return "gid"
	case tkRId:
		return "rid"
	case tkInt:
		return "int"
	case tkReal:
		return "real"
	case tkStr:
		return "str"
	case tkMStr:
		return "mstr"
	case tkDollar:
		return "dollar"
	case tkLPar:
		return "("
	case tkRPar:
		return ")"
	case tkLSqr:
		return "["
	case tkRSqr:
		return "]"
	case tkLBra:
		return "{"
	case tkRBra:
		return "}"

	case tkDRBra:
		return "}}"

	case tkDot:
		return "."
	case tkAssign:
		return "="
	case tkArrow:
		return "=>"
	case tkColon:
		return ":"
	case tkScope:
		return "::"
	case tkSemicolon:
		return ";"
	case tkQuest:
		return "?"
	case tkSharp:
		return "#"

	case tkAdd:
		return "+"
	case tkSub:
		return "-"
	case tkMul:
		return "*"
	case tkPow:
		return "**"
	case tkDiv:
		return "/"
	case tkMod:
		return "%"
	case tkLt:
		return "<"
	case tkLe:
		return "<="
	case tkGt:
		return ">"
	case tkGe:
		return ">="
	case tkEq:
		return "=="
	case tkNe:
		return "!="
	case tkNot:
		return "!"

	case tkTrue:
		return "true"
	case tkFalse:
		return "false"
	case tkNull:
		return "null"
	case tkLet:
		return "let"
	case tkSession:
		return "session"
	case tkWhen:
		return "when"
	case tkImport:
		return "import"
	case tkTry:
		return "try"
	case tkIf:
		return "if"
	case tkElif:
		return "elif"
	case tkElse:
		return "else"

	case tkContinue:
		return "continue"
	case tkBreak:
		return "break"
	case tkNext:
		return "next"
	case tkReturn:
		return "return"

	case tkTemplate:
		return "template"

	case tkError:
		return "<error>"
	default:
		return "<none>"
	}
}

func (t *lexer) yield(tk int, offset int) int {
	t.cursor += offset
	t.token = tk
	return tk
}

func (t *lexer) pos() (int, int) {
	l := 1
	c := 1

	for i := 0; i < t.cursor; i++ {
		char := t.input[i]
		if char == '\n' {
			l++
			c = 1
		} else {
			c++
		}
	}
	return l, c
}

func (t *lexer) err(msg string) int {
	prefix := t.position()
	t.valueText = fmt.Sprintf("%s: %s", prefix, msg)
	t.token = tkError
	return tkError
}

func (t *lexer) position() string {
	line, col := t.pos()
	// get a string piece around the cursor
	var start, end int
	if t.cursor >= 32 {
		start = t.cursor - 32
	} else {
		start = 0
	}

	if t.cursor+32 < len(t.input) {
		end = t.cursor + 32
	} else {
		end = len(t.input)
	}

	return fmt.Sprintf("around (%d, %d)@(```%s```)", line, col, string(t.input[start:end]))
}

func (t *lexer) e(err error) int {
	return t.err(err.Error())
}

func (t *lexer) toError() error {
	if t.token != tkError {
		log.Fatalf("invalid toError, current token is not error")
	}
	return errors.New(t.valueText)
}

func (t *lexer) expectCurrent(tk int) bool {
	tt := t.token
	if tt == tk {
		return true
	} else {
		t.err(fmt.Sprintf("expect token %s, but got %s",
			getTokenName(tk), getTokenName(t.token)))
		return false
	}
}

func (t *lexer) expect(tk int) bool {
	t.next()
	return t.expectCurrent(tk)
}

func (t *lexer) scanStr() int {
	var buffer bytes.Buffer

	singleQuote := true
	done := false

	if t.input[t.cursor] == '"' {
		singleQuote = false
	}

	for t.cursor++; t.cursor < len(t.input); t.cursor++ {
		c := t.input[t.cursor]

		if (singleQuote && c == '\'') || (!singleQuote && c == '"') {
			done = true
			break
		} else if c == '\\' {
			ncursor := t.cursor + 1
			if ncursor < len(t.input) {
				nc := t.input[ncursor]
				switch nc {
				case '\\':
					buffer.WriteRune('\\')
					break

				case '\'':
					buffer.WriteRune('\'')
					break

				case 't':
					buffer.WriteRune('\t')
					break

				case 'r':
					buffer.WriteRune('\r')
					break

				case 'n':
					buffer.WriteRune('\n')
					break

				case 'b':
					buffer.WriteRune('\b')
					break

				case 'v':
					buffer.WriteRune('\v')
					break

				default:
					return t.err("invalid character escape")
				}

				t.cursor++
				continue
			} else {
				return t.err("early termination of string literal")
			}
		} else {
			buffer.WriteRune(c)
		}
	}

	if done {
		t.valueText = buffer.String()
		t.cursor++
		t.token = tkStr
		return tkStr
	} else {
		return t.err("string not closed properly")
	}
}

// this scanning process is kind of simple since we just check floating point
// representation of real number instead of scientific representation. We may
// add that support in the future
func (t *lexer) scanNum() int {
	var buffer bytes.Buffer
	hasDot := false
	hasExp := false

	if t.input[t.cursor] == '-' {
		t.cursor++
		buffer.WriteRune('-')
	}

	for ; t.cursor < len(t.input); t.cursor++ {
		c := t.input[t.cursor]

		if unicode.IsDigit(c) {
			buffer.WriteRune(c)
		} else if c == '.' {
			if hasDot {
				break
			} else {
				hasDot = true
			}
			buffer.WriteRune('.')
		} else if c == 'e' || c == 'E' {
			if hasExp {
				break
			} else {
				hasExp = true
				// looking for a + or - afterwards
				if t.cursor+1 < len(t.input) {
					nc := t.input[t.cursor+1]
					if nc == '+' || nc == '-' {
						t.cursor++
						buffer.WriteRune(nc)
						continue
					}
				}
				return t.err("numeric number is invalid, Ee must follow a '+' or '-'")
			}
		} else {
			break
		}
	}

	if hasDot || hasExp {
		i, err := strconv.ParseFloat(buffer.String(), 64)
		if err != nil {
			return t.e(err)
		}
		t.valueReal = i
		t.token = tkReal
		return tkReal
	} else {
		i, err := strconv.ParseInt(buffer.String(), 10, 64)
		if err != nil {
			return t.e(err)
		}
		t.valueInt = i
		t.token = tkInt
		return tkInt
	}
}

func (t *lexer) tryPrefixString(c rune) (int, bool) {
	// Prefix string's prefix checking
	switch c {
	case 'R', 'r':
		break

	default:
		return 0, false
	}

	tk := tkError

	if t.cursor+1 < len(t.input) {
		nc := t.input[t.cursor+1]
		switch nc {
		case '\'', '"':
			tk = tkRegex
			break
		default:
			return 0, false
		}
	}

	// Scan the rest of the string into valueText, and the scanStr will mark the
	// token to be tkStr, which will be replaced accordingly later on
	t.cursor++
	x := t.scanStr()
	if x != tkStr {
		return x, true
	}

	t.token = tk
	return tk, true
}

func (t *lexer) scanRId() int {
	must(t.input[t.cursor] == '@', "must be @")
	t.cursor++
	must(t.input[t.cursor] == '\'' ||
		t.input[t.cursor] == '"', "must be quoted string")
	x := t.scanStr()
	if x != tkStr {
		return x
	} else {
		t.token = tkRId
		return tkRId
	}
}

func (t *lexer) scanIdOrKeywordOrPrefixString(c rune) int {
	if tk, ok := t.tryPrefixString(c); ok {
		return tk
	}

	idType := tkId

	// @ prefixed token
	if c == '@' {
		nc := t.input[t.cursor+1]
		if nc == '"' || nc == '\'' {
			return t.scanRId()
		}
		t.cursor++
		idType = tkGId
	} else if !unicode.IsLetter(c) && c != '_' {
		return t.err("unrecognized token here, expect keyword or identifier")
	}

	var buffer bytes.Buffer

	for ; t.cursor < len(t.input); t.cursor++ {
		c := t.input[t.cursor]
		if unicode.IsLetter(c) || c == '_' || unicode.IsDigit(c) {
			buffer.WriteRune(c)
		} else {
			break
		}
	}

	idOrKeyword := buffer.String()

	switch idOrKeyword {
	case "true":
		t.token = tkTrue
		return tkTrue
	case "false":
		t.token = tkFalse
		return tkFalse
	case "null":
		t.token = tkNull
		return tkNull
	case "let":
		t.token = tkLet
		return tkLet
	case "session":
		t.token = tkSession
		return tkSession
	case "when":
		t.token = tkWhen
		return tkWhen
	case "import":
		t.token = tkImport
		return tkImport
	case "try":
		t.token = tkTry
		return tkTry
	case "if":
		t.token = tkIf
		return tkIf
	case "elif":
		t.token = tkElif
		return tkElif
	case "else":
		t.token = tkElse
		return tkElse
	case "continue":
		t.token = tkContinue
		return tkContinue
	case "break":
		t.token = tkBreak
		return tkBreak
	case "next":
		t.token = tkNext
		return tkNext
	case "return":
		t.token = tkReturn
		return tkReturn

		// intrinsic
	case "template":
		t.token = tkTemplate
		return tkTemplate

	default:
		break
	}

	t.valueText = idOrKeyword
	t.token = idType
	return tkId
}

func (t *lexer) scanComment() {
	for ; t.cursor < len(t.input); t.cursor++ {
		c := t.input[t.cursor]
		if c == '\n' {
			t.cursor++
			break
		}
	}
}

func (t *lexer) scanCommentBlock() bool {
	for ; t.cursor < len(t.input); t.cursor++ {
		c := t.input[t.cursor]
		if c == '*' && ((t.cursor + 1) < len(t.input)) {
			nc := t.input[t.cursor+1]
			if nc == '/' {
				t.cursor += 2
				return true
			}
		}
	}
	t.err("block comment must be closed by */")
	return false
}

func (t *lexer) p2(t0, t1 int, lh rune) int {
	if t.cursor+1 < len(t.input) {
		if t.input[t.cursor+1] == lh {
			return t.yield(t1, 2)
		}
	}
	return t.yield(t0, 1)
}

func (t *lexer) pp(tk int, lh rune) int {
	if t.cursor+1 < len(t.input) {
		if t.input[t.cursor+1] == lh {
			return t.yield(tk, 2)
		}
	}
	return t.err(fmt.Sprintf("unknown token, expect one more %c", lh))
}

func (t *lexer) pp2(tk0, tk1, tk2 int, l0, l1 rune) int {
	if t.cursor+1 < len(t.input) {
		c := t.input[t.cursor+1]
		switch c {
		case l0:
			return t.yield(tk1, 2)
		case l1:
			return t.yield(tk2, 2)
		default:
			break
		}
	}
	return t.yield(tk0, 1)
}

// try to find out the multiple line string's marker
func (t *lexer) mulstrMarker() (string, error) {
	if t.cursor+2 >= len(t.input) {
		return "", fmt.Errorf("multiple line string start marker invalid")
	}

	if t.input[t.cursor+1] != '`' || t.input[t.cursor+2] != '`' {
		return "", fmt.Errorf("multiple line string start marker incomplete")
	}

	// optionally looking for a here document end marker, otherwise the end
	// marker will be \n````

	t.cursor += 3
	markerStart := t.cursor
	for ; t.cursor < len(t.input); t.cursor++ {
		c := t.input[t.cursor]
		if c == '\n' {
			break
		}
	}

	// now t.cursor points to the first linebreak after the ```
	var end string
	if markerStart == t.cursor {
		end = "\n```"
	} else {
		end = "\n" + string(t.input[markerStart:t.cursor]) + "```"
	}

	// now the cursor points to the start of the valid content, ie skipping the
	// linebreak previously found
	t.cursor++

	// then we just need to relentlessly search for the end tag to learn where
	// to finish the multiple line string parsing ...
	return end, nil
}

func (t *lexer) scanMStr() int {
	endTag, err := t.mulstrMarker()
	if err != nil {
		return t.e(err)
	}

	// now just searching for the end
	tagPos := strings.Index(string(t.input[t.cursor:]), endTag)
	if tagPos == -1 {
		return t.err("the multiple line string is not closed properly")
	}

	startPos := t.cursor
	endPos := t.cursor + tagPos

	t.valueText = string(t.input[startPos:endPos])
	t.cursor = endPos + len(endTag)
	t.token = tkMStr
	return tkMStr
}

func (t *lexer) next() int {
	for t.cursor < len(t.input) {
		c := t.input[t.cursor]
		switch c {
		case ' ', '\t', '\r', '\n', '\v':
			t.cursor++
			continue

		case '+':
			return t.yield(tkAdd, 1)
		case '-':
			return t.yield(tkSub, 1)
		case '*':
			return t.p2(tkMul, tkPow, '*')
		case '/':
			if t.cursor+1 < len(t.input) {
				nc := t.input[t.cursor+1]
				if nc == '/' {
					t.cursor += 2
					t.scanComment()
					continue
				} else if nc == '*' {
					t.cursor += 2
					if !t.scanCommentBlock() {
						return t.token
					} else {
						continue
					}
				}
			}
			return t.yield(tkDiv, 1)

		case '%':
			return t.yield(tkMod, 1)

		case '=':
			return t.pp2(tkAssign, tkArrow, tkEq, '>', '=')

		case '`':
			return t.scanMStr()

		case '&':
			return t.pp(tkAnd, '&')

		case '|':
			return t.pp(tkOr, '|')

		case '!':
			return t.pp2(tkNot, tkNe, tkRegexpNMatch, '=', '~')
		case '~':
			return t.yield(tkRegexpMatch, 1)

		case '>':
			return t.p2(tkGt, tkGe, '=')
		case '<':
			return t.p2(tkLt, tkLe, '=')

		case '(':
			return t.yield(tkLPar, 1)
		case ')':
			return t.yield(tkRPar, 1)
		case '[':
			return t.yield(tkLSqr, 1)
		case ']':
			return t.yield(tkRSqr, 1)
		case '{':
			return t.yield(tkLBra, 1)
		case '}':
			return t.p2(tkRBra, tkDRBra, '}')
		case '.':
			return t.yield(tkDot, 1)
		case ',':
			return t.yield(tkComma, 1)
		case '$':
			return t.yield(tkDollar, 1)
		case '?':
			return t.yield(tkQuest, 1)
		case '#':
			return t.yield(tkSharp, 1)
		case ';':
			return t.yield(tkSemicolon, 1)
		case ':':
			return t.p2(tkColon, tkScope, ':')
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			return t.scanNum()
		case '"', '\'':
			return t.scanStr()

		default:
			return t.scanIdOrKeywordOrPrefixString(c)
		}
	}

	return t.yield(tkEof, 0)
}
