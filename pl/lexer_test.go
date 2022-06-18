package pl

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TextLexerComment(t *testing.T) {
	assert := assert.New(t)
	{
		l := newLexer("/* abcd */")
		assert.Equal(l.next(), tkEof, "c1")
	}
	{
		l := newLexer("/* abcd\na */")
		assert.Equal(l.next(), tkEof, "c2.2")
	}
	{
		l := newLexer("/* // abcd */")
		assert.Equal(l.next(), tkEof, "c3.2")
	}
	{
		l := newLexer("/*a\n#abcd */")
		assert.Equal(l.next(), tkId, "c4.1")
		assert.Equal(l.next(), tkEof, "c4.2")
	}

	{
		l := newLexer("//abcd")
		assert.Equal(l.next(), tkEof, "cc1")
	}
	{
		l := newLexer("//abcd\na")
		assert.Equal(l.next(), tkId, "cc2.1")
		assert.Equal(l.next(), tkEof, "cc2.2")
	}
	{
		l := newLexer("a//abcd")
		assert.Equal(l.next(), tkId, "cc3.1")
		assert.Equal(l.next(), tkEof, "cc3.2")
	}
	{
		l := newLexer("a\n//abcd")
		assert.Equal(l.next(), tkId, "cc4.1")
		assert.Equal(l.next(), tkEof, "cc4.2")
	}
}

func TestLexerBasic(t *testing.T) {
	assert := assert.New(t)
	{
		l := newLexer("")
		assert.Equal(l.cursor, 0, "cursor")
		assert.Equal(l.token, 0, "token")
	}

	{
		l := newLexer("")
		assert.Equal(l.next(), tkEof, "eof")
	}

	{
		l := newLexer("$()[]{};=.,:}}")
		l.allowDRBra = true
		assert.Equal(l.next(), tkDollar, "$")
		assert.Equal(l.next(), tkLPar, "(")
		assert.Equal(l.next(), tkRPar, ")")
		assert.Equal(l.next(), tkLSqr, "[")
		assert.Equal(l.next(), tkRSqr, "]")
		assert.Equal(l.next(), tkLBra, "{")
		assert.Equal(l.next(), tkRBra, "}")
		assert.Equal(l.next(), tkSemicolon, ";")
		assert.Equal(l.next(), tkAssign, "=")
		assert.Equal(l.next(), tkDot, ".")
		assert.Equal(l.next(), tkComma, ",")
		assert.Equal(l.next(), tkColon, ":")
		assert.Equal(l.next(), tkDRBra, "}}")
		assert.Equal(l.next(), tkEof, "<eof>")
	}
	{
		l := newLexer("}}}")
		l.allowDRBra = true
		assert.Equal(l.next(), tkDRBra, "}}")
		assert.Equal(l.next(), tkRBra, "}")
	}
}

func TestLexerBasic2(t *testing.T) {
	assert := assert.New(t)
	{
		l := newLexer("+-***/>>=<<=!==== || &&")
		assert.True(l.next() == tkAdd)
		assert.True(l.next() == tkSub)
		assert.True(l.next() == tkPow)
		assert.True(l.next() == tkMul)
		assert.True(l.next() == tkDiv)
		assert.True(l.next() == tkGt)
		assert.True(l.next() == tkGe)
		assert.True(l.next() == tkLt)
		assert.True(l.next() == tkLe)
		assert.True(l.next() == tkNe)
		assert.True(l.next() == tkEq)
		assert.True(l.next() == tkAssign)
		assert.True(l.next() == tkOr)
		assert.True(l.next() == tkAnd)
		assert.True(l.next() == tkEof)
	}
}

func TestLexerNum(t *testing.T) {
	assert := assert.New(t)
	{
		l := newLexer("0 1 0.1 0.1")
		assert.Equal(l.next(), tkInt, "tkInt")
		assert.Equal(l.valueInt, int64(0), "int(0)")

		assert.Equal(l.next(), tkInt, "tkInt")
		assert.Equal(l.valueInt, int64(1), "int(1)")

		assert.Equal(l.next(), tkReal, "tkReal")
		assert.Equal(l.valueReal, 0.1, "real(0.1)")

		assert.Equal(l.next(), tkReal, "tkReal")
		assert.Equal(l.valueReal, 0.1, "real(0.1)")

		assert.Equal(l.next(), tkEof, "tkEof")
	}
}

func TestLexerKeywordOrId(t *testing.T) {
	assert := assert.New(t)
	{
		l := newLexer("if if2 elif elif2 else else3 when_")
		assert.True(l.next() == tkIf)
		assert.True(l.next() == tkId)
		assert.True(l.next() == tkElif)
		assert.True(l.next() == tkId)
		assert.True(l.next() == tkElse)
		assert.True(l.next() == tkId)
		assert.True(l.next() == tkId)
		assert.True(l.next() == tkEof)
	}
	{
		l := newLexer("true true_ false false2 null null_ let let_")
		assert.Equal(l.next(), tkTrue, "true")

		assert.Equal(l.next(), tkId, "id")
		assert.Equal(l.valueText, "true_", "true_")

		assert.Equal(l.next(), tkFalse, "false")

		assert.Equal(l.next(), tkId, "id")
		assert.Equal(l.valueText, "false2", "false2")

		assert.Equal(l.next(), tkNull, "null")

		assert.Equal(l.next(), tkId, "id")
		assert.Equal(l.valueText, "null_", "null_")

		assert.Equal(l.next(), tkLet, "let")

		assert.Equal(l.next(), tkId, "id")
		assert.Equal(l.valueText, "let_", "let_")

		assert.Equal(l.next(), tkEof, "tkEof")
	}
}
