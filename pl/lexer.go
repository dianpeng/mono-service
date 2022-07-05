package pl

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strconv"
	"unicode"
)

const (
	tkId = iota

	// qualify id
	tkSId
	tkGId
	tkDId
	tkRId
	tkEId

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
	tkAt
	tkSharp
	tkPipe
	tkLExprBra

	// unary
	tkNot

	// arithmetic
	tkAdd
	tkSub
	tkMul
	tkDiv
	tkMod
	tkPow

	// agg-arithmetic
	tkAddAssign
	tkSubAssign
	tkMulAssign
	tkDivAssign
	tkModAssign
	tkPowAssign
	tkInc
	tkDec

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
	tkDynamic
	tkGlobal
	tkSession
	tkLet
	tkConst
	tkSwitch
	tkCase
	tkModule
	tkImport
	tkExtern
	tkIf
	tkElif
	tkElse
	tkTry
	tkReturn
	tkFor
	tkContinue
	tkBreak
	tkFunction
	tkConfig

	tkRule
	tkEmit

	// generator
	tkIterator
	tkYield

	// intrinsic keywords, used for special builtin functionalities
	tkTemplate

	tkEof
	tkError
)

const (
	minLexerDCursorOffset = 64
)

type dcursorqueue struct {
	q [minLexerDCursorOffset]int
	c int
}

func (d *dcursorqueue) index(x int) int {
	return x % minLexerDCursorOffset
}

func (d *dcursorqueue) add(o int) {
	d.q[d.index(d.c)] = o
	d.c++
}

func (d *dcursorqueue) frontIndex(off int) int {
	if d.c <= minLexerDCursorOffset {
		return d.index(0 + off)
	} else {
		return d.index(d.c - minLexerDCursorOffset + off)
	}
}

func (d *dcursorqueue) backIndex(off int) int {
	if d.c <= off {
		return -1
	}
	return d.index(d.c - off)
}

func (d *dcursorqueue) front(off int) int {
	return d.q[d.frontIndex(off)]
}

func (d *dcursorqueue) back(off int) int {
	idx := d.backIndex(1 + off)
	if idx == -1 {
		return -1
	}
	return d.q[idx]
}

func (d *dcursorqueue) allowedOffset(off int) bool {
	if off <= d.c {
		return true
	} else {
		return off < minLexerDCursorOffset
	}
}

type lexer struct {
	input  []rune
	cursor int
	token  int

	// for error reporting
	dCursor int
	dq      dcursorqueue

	// lexeme
	valueInt  int64
	valueReal float64
	valueText string

	// option
	allowDRBra bool

	// internal shit
	cursorStart int
}

func isaggassign(tk int) bool {
	switch tk {
	case tkAddAssign, tkSubAssign, tkMulAssign, tkDivAssign, tkPowAssign, tkModAssign, tkInc, tkDec:
		return true
	default:
		return false
	}
}

func isassign(tk int) bool {
	return tk == tkAssign || isaggassign(tk)
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

	case tkSId:
		return "session_id"
	case tkDId:
		return "dynamic_id"
	case tkRId:
		return "resource_id"
	case tkEId:
		return "extern_id"
	case tkGId:
		return "global_id"

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
	case tkAt:
		return "@"
	case tkSharp:
		return "#"
	case tkPipe:
		return "|"
	case tkLExprBra:
		return "|{"

	case tkAdd:
		return "+"
	case tkAddAssign:
		return "+="
	case tkSub:
		return "-"
	case tkSubAssign:
		return "-="
	case tkMul:
		return "*"
	case tkMulAssign:
		return "*="
	case tkPow:
		return "**"
	case tkPowAssign:
		return "**="
	case tkDiv:
		return "/"
	case tkDivAssign:
		return "/="
	case tkMod:
		return "%"
	case tkModAssign:
		return "%="
	case tkInc:
		return "++"
	case tkDec:
		return "--"
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
	case tkConst:
		return "const"
	case tkGlobal:
		return "global"
	case tkDynamic:
		return "dynamic"
	case tkSession:
		return "session"
	case tkExtern:
		return "extern"
	case tkSwitch:
		return "switch"
	case tkCase:
		return "case"
	case tkModule:
		return "module"
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

	case tkFor:
		return "for"
	case tkContinue:
		return "continue"
	case tkBreak:
		return "break"
	case tkFunction:
		return "fn"
	case tkRule:
		return "rule"

	case tkIterator:
		return "iter"
	case tkYield:
		return "yield"

	case tkEmit:
		return "emit"
	case tkReturn:
		return "return"

	case tkConfig:
		return "config"

	case tkTemplate:
		return "template"

	case tkError:
		return "<error>"
	default:
		return "<none>"
	}
}

// this function just does one thing, it tries to update dCursor field. The
// dCursor field is a token boundary that offset from current cursor around
// a predefined value. The reason for this is when we start to report an
// diagnostic error, what we do is we start the substring from the dcursor
// which is a valid reporting boundary. This is better than the current way
// for reporting anyway
func (t *lexer) saveDCursor(start int) {
	if t.token != tkError {
		t.dq.add(start)
		offset := 0
		ncursor := t.dCursor

		for ncursor+minLexerDCursorOffset < t.cursor {
			if !t.dq.allowedOffset(offset) {
				break
			}
			bump := t.dq.front(offset)
			ncursor = bump
			offset++
		}

		t.dCursor = ncursor
	}
}

func (t *lexer) yield(tk int, offset int) int {
	t.cursor += offset
	t.token = tk
	return tk
}

// generate a source location field for debugging reporting purpose
func (t *lexer) dbg() sourceloc {
	line, column := t.pos()
	return sourceloc{
		source: string(t.input),
		offset: t.cursor,
		line:   line,
		column: column,
	}
}

func (t *lexer) pos() (int, int) {
	l := 1
	c := 1

	clampedSize := t.cursor
	if clampedSize >= len(t.input) {
		clampedSize = len(t.input)
	}

	for i := 0; i < clampedSize; i++ {
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

func (t *lexer) nextLineBreak(where int) int {
	for i := where; i < len(t.input); i++ {
		if t.input[i] == '\n' {
			return i
		}
	}
	return len(t.input) - 1
}

func (t *lexer) paddingSize(where int) int {
	i := where - 1
	for ; i >= 0; i-- {
		if t.input[i] == '\n' {
			return where - i - 1
		}
	}

	return where
}

func (t *lexer) position() string {
	line, col := t.pos()

	// get a string piece around the cursor
	var start, end int
	start = t.dCursor

	if t.cursor+minLexerDCursorOffset < len(t.input) {
		end = t.cursor + minLexerDCursorOffset
	} else {
		end = len(t.input)
	}

	t1 := t.dq.back(0)
	if t1 == -1 {
		t1 = 0
	}

	lb := t.nextLineBreak(t1)

	if end < lb {
		end = lb
	}

	prefix := string(t.input[start:lb])
	after := string(t.input[lb+1 : end])

	p0 := fmt.Sprintf(
		"around line %d and column %d, near source code:\n%s",
		line,
		col,
		prefix,
	)

	// generate padding before the highlighter
	var padding string
	{
		b := new(bytes.Buffer)
		off := t.paddingSize(t1)
		for i := 0; i < off; i++ {
			b.WriteRune(' ')
		}
		padding = b.String()
	}

	// generate highlighter in the output
	var highlight string
	{
		hsize := t.cursor - t1
		b := new(bytes.Buffer)
		for i := 0; i < hsize; i++ {
			b.WriteRune('^')
		}
		highlight = b.String()
	}

	// assemble everything together to formulate the final output
	return fmt.Sprintf(
		"%s\n%s%s\n%s\n",
		p0,
		padding,
		highlight,
		after,
	)
}

func (t *lexer) tokenName() string {
	return getTokenName(t.token)
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
		t.err(fmt.Sprintf("expect token %s, but got %s", getTokenName(tk), t.tokenName()))
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

func (t *lexer) scanRId() int {
	must(t.input[t.cursor] == '@', "must be @")
	t.cursor++
	if t.cursor == len(t.input) {
		return t.err("early terminate of resource identifier")
	}

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
	if t.cursor == len(t.input) {
		return t.err("early terminate of prefixed identifiers"), false
	}

	x := t.scanStr()
	if x != tkStr {
		return x, true
	}

	t.token = tk
	return tk, true
}

var lexerkeyword = map[string]int{
	"true":  tkTrue,
	"false": tkFalse,
	"null":  tkNull,

	"dynamic": tkDynamic,
	"global":  tkGlobal,
	"session": tkSession,
	"extern":  tkExtern,

	"module": tkModule,
	"import": tkImport,

	"let":   tkLet,
	"const": tkConst,

	/* when case */
	"switch": tkSwitch,
	"case":   tkCase,

	/* if else branch */
	"if":   tkIf,
	"elif": tkElif, // we have real else if, instead of nested else + if
	"else": tkElse,

	/* for loops */
	"for":      tkFor,
	"continue": tkContinue,
	"break":    tkBreak,

	/* other control flow */
	"try":    tkTry,
	"return": tkReturn,

	/* reserve 2 keywords for function definition, this may not be a good idea though */
	"fn": tkFunction,

	/* generator */
	"iter":  tkIterator,
	"yield": tkYield,

	/* rule */
	"rule": tkRule,
	"emit": tkEmit,

	"config": tkConfig,

	/* intrinsic keywords */
	"template": tkTemplate,
}

func (t *lexer) tryQualifyId(keyword int) int {
	switch keyword {
	case tkSession,
		tkGlobal,
		tkDynamic,
		tkExtern:
		break
	default:
		return -1
	}

	cursor := t.cursor

	// we just peek forward 2 tokens otherwise we just don't know what happened

	ntk1 := t.next()
	if ntk1 != tkScope {
		t.cursor = cursor
		return -1
	}

	ntk2 := t.next()
	if ntk2 != tkId {
		t.cursor = cursor
		return -1
	}

	// notes since it is id, its lexeme value has been put inplace, so nothing
	// need to done here, just return the token we have

	switch keyword {
	case tkSession:
		return tkSId
	case tkGlobal:
		return tkGId
	case tkDynamic:
		return tkDId
	default:
		return tkEId
	}
}

func (t *lexer) scanIdOrKeywordOrPrefixString(c rune) int {
	if tk, ok := t.tryPrefixString(c); ok {
		return tk
	}

	idType := tkId
	hasPrefix := false

	if !unicode.IsLetter(c) && c != '_' {
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

	if !hasPrefix {
		id, ok := lexerkeyword[idOrKeyword]
		if ok {
			// FIXME(dpeng): here we hack the lexer to support a grammar basically
			//   which should be supported by parser instead of lexer. But this makes
			//   our parser code easier to maintain. Basically we just want to separate
			//   some speical naming thing, ie qualifier
			if tk := t.tryQualifyId(id); tk != -1 {
				t.token = tk
				return tk
			}

			t.token = id
			return id
		}
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
	hasLB := false
	for ; t.cursor < len(t.input); t.cursor++ {
		c := t.input[t.cursor]
		if c == '\n' {
			hasLB = true
			break
		}
	}
	if !hasLB {
		return "", fmt.Errorf("multiple line string expect a linebreak after ```")
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

	// check whether we have the end tag or not
	if t.cursor+3 > len(t.input) {
		return t.err("the multiple line string is not terminated properly")
	}

	// notes, you cannot search via strings.Index due to the invalid rune inside
	// of the sequences. The fuzzer finds this bug
	runeTag := []rune(endTag)
	tagPos := -1
	for i := t.cursor; i < len(t.input); i++ {
		if i+len(runeTag) > len(t.input) {
			break
		}

		found := true
		for idx, x := range runeTag {
			if t.input[i+idx] != x {
				found = false
				break
			}
		}
		if found {
			tagPos = (i - t.cursor)
			break
		}
	}

	if tagPos == -1 {
		return t.err("the multiple line string is not closed properly")
	}

	startPos := t.cursor
	endPos := t.cursor + tagPos
	fmt.Printf(":: %d:%d:%d\n", t.cursor, tagPos, endPos)

	t.valueText = string(t.input[startPos:endPos])
	t.cursor = endPos + len(endTag)
	t.token = tkMStr
	return tkMStr
}

func (t *lexer) next() int {
	startDCursor := t.cursor
	pDCursor := &startDCursor
	defer func() {
		t.saveDCursor(*pDCursor)
	}()

	for t.cursor < len(t.input) {
		c := t.input[t.cursor]
		switch c {
		case ' ', '\t', '\r', '\n', '\v':
			t.cursor++
			startDCursor = t.cursor
			continue
		case '+':
			return t.pp2(tkAdd, tkAddAssign, tkInc, '=', '+')

		case '-':
			return t.pp2(tkSub, tkSubAssign, tkDec, '=', '-')

		case '*':
			if t.cursor+1 < len(t.input) {
				nc := t.input[t.cursor+1]
				switch nc {
				case '=':
					return t.yield(tkMulAssign, 2)

				case '*':
					if t.cursor+2 < len(t.input) {
						nnc := t.input[t.cursor+2]
						if nnc == '=' {
							return t.yield(tkPowAssign, 3)
						}
					}
					return t.yield(tkPow, 2)

				default:
					break
				}
			}
			return t.yield(tkMul, 1)

		case '/':
			if t.cursor+1 < len(t.input) {
				nc := t.input[t.cursor+1]
				switch nc {
				case '/':
					t.cursor += 2
					t.scanComment()
					startDCursor = t.cursor
					continue

				case '*':
					t.cursor += 2
					if !t.scanCommentBlock() {
						return t.token
					} else {
						startDCursor = t.cursor
						continue
					}
					break

				case '=':
					return t.yield(tkDivAssign, 2)
				default:
					break
				}
			}
			return t.yield(tkDiv, 1)

		case '%':
			return t.p2(tkMod, tkModAssign, '=')

		case '=':
			return t.pp2(tkAssign, tkArrow, tkEq, '>', '=')

		case '`':
			return t.scanMStr()

		case '&':
			return t.pp(tkAnd, '&')

		case '|':
			return t.pp2(tkPipe, tkOr, tkLExprBra, '|', '{')

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
			if t.allowDRBra {
				return t.p2(tkRBra, tkDRBra, '}')
			} else {
				return t.yield(tkRBra, 1)
			}
		case '.':
			return t.yield(tkDot, 1)
		case ',':
			return t.yield(tkComma, 1)
		case '$':
			return t.yield(tkDollar, 1)
		case '?':
			return t.yield(tkQuest, 1)
		case '@':
			if t.cursor+1 < len(t.input) {
				nc := t.input[t.cursor+1]
				if nc == '"' || nc == '\'' {
					return t.scanRId()
				}
			}
			return t.yield(tkAt, 1)

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
