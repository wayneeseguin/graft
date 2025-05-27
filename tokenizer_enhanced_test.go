package graft

import (
	"testing"
	. "github.com/smartystreets/goconvey/convey"
)

func TestEnhancedTokenizer(t *testing.T) {
	Convey("Enhanced Tokenizer", t, func() {
		Convey("Basic Tokenization", func() {
			Convey("should tokenize simple operator call", func() {
				tokens := TokenizeExpression(`grab defaults.password`)
				So(len(tokens), ShouldEqual, 2)
				So(tokens[0].Type, ShouldEqual, TokenOperator)
				So(tokens[0].Value, ShouldEqual, "grab")
				So(tokens[1].Type, ShouldEqual, TokenReference)
				So(tokens[1].Value, ShouldEqual, "defaults.password")
			})

			Convey("should tokenize quoted strings", func() {
				tokens := TokenizeExpression(`"hello world" reference`)
				So(len(tokens), ShouldEqual, 2)
				So(tokens[0].Type, ShouldEqual, TokenLiteral)
				So(tokens[0].Value, ShouldEqual, `"hello world"`)
				So(tokens[1].Type, ShouldEqual, TokenReference)
				So(tokens[1].Value, ShouldEqual, "reference")
			})

			Convey("should tokenize environment variables", func() {
				tokens := TokenizeExpression(`$ENV_VAR $ANOTHER_VAR`)
				So(len(tokens), ShouldEqual, 2)
				So(tokens[0].Type, ShouldEqual, TokenEnvVar)
				So(tokens[0].Value, ShouldEqual, "$ENV_VAR")
				So(tokens[1].Type, ShouldEqual, TokenEnvVar)
				So(tokens[1].Value, ShouldEqual, "$ANOTHER_VAR")
			})

			Convey("should tokenize special literals", func() {
				tokens := TokenizeExpression(`nil true false null`)
				So(len(tokens), ShouldEqual, 4)
				for _, tok := range tokens {
					So(tok.Type, ShouldEqual, TokenLiteral)
				}
			})

			Convey("should tokenize numbers", func() {
				tokens := TokenizeExpression(`42 3.14 -17 +2.5`)
				So(len(tokens), ShouldEqual, 6) // -17 is tokenized as - and 17, +2.5 as + and 2.5
				So(tokens[0].Type, ShouldEqual, TokenLiteral)
				So(tokens[0].Value, ShouldEqual, "42")
				So(tokens[1].Type, ShouldEqual, TokenLiteral)
				So(tokens[1].Value, ShouldEqual, "3.14")
				So(tokens[2].Type, ShouldEqual, TokenMinus)
				So(tokens[3].Type, ShouldEqual, TokenLiteral)
				So(tokens[3].Value, ShouldEqual, "17")
				So(tokens[4].Type, ShouldEqual, TokenPlus)
				So(tokens[5].Type, ShouldEqual, TokenLiteral)
				So(tokens[5].Value, ShouldEqual, "2.5")
			})
		})

		Convey("Operator Tokenization", func() {
			Convey("should tokenize logical operators", func() {
				tokens := TokenizeExpression(`a || b && c`)
				So(len(tokens), ShouldEqual, 5)
				So(tokens[0].Value, ShouldEqual, "a")
				So(tokens[1].Type, ShouldEqual, TokenLogicalOr)
				So(tokens[2].Value, ShouldEqual, "b")
				So(tokens[3].Type, ShouldEqual, TokenLogicalAnd)
				So(tokens[4].Value, ShouldEqual, "c")
			})

			Convey("should tokenize parentheses", func() {
				tokens := TokenizeExpression(`(a || b) && c`)
				So(len(tokens), ShouldEqual, 7)
				So(tokens[0].Type, ShouldEqual, TokenOpenParen)
				So(tokens[1].Value, ShouldEqual, "a")
				So(tokens[2].Type, ShouldEqual, TokenLogicalOr)
				So(tokens[3].Value, ShouldEqual, "b")
				So(tokens[4].Type, ShouldEqual, TokenCloseParen)
				So(tokens[5].Type, ShouldEqual, TokenLogicalAnd)
				So(tokens[6].Value, ShouldEqual, "c")
			})

			Convey("should tokenize commas", func() {
				tokens := TokenizeExpression(`a, b, c`)
				So(len(tokens), ShouldEqual, 5)
				So(tokens[0].Value, ShouldEqual, "a")
				So(tokens[1].Type, ShouldEqual, TokenComma)
				So(tokens[2].Value, ShouldEqual, "b")
				So(tokens[3].Type, ShouldEqual, TokenComma)
				So(tokens[4].Value, ShouldEqual, "c")
			})

			Convey("should tokenize arithmetic operators", func() {
				tokens := TokenizeExpression(`2 + 3 * 4 - 5 / 2 % 3`)
				So(len(tokens), ShouldEqual, 11)
				So(tokens[1].Type, ShouldEqual, TokenPlus)
				So(tokens[3].Type, ShouldEqual, TokenMultiply)
				So(tokens[5].Type, ShouldEqual, TokenMinus)
				So(tokens[7].Type, ShouldEqual, TokenDivide)
				So(tokens[9].Type, ShouldEqual, TokenModulo)
			})

			Convey("should tokenize comparison operators", func() {
				tokens := TokenizeExpression(`a == b != c < d > e`)
				So(len(tokens), ShouldEqual, 9)
				So(tokens[1].Type, ShouldEqual, TokenEquals)
				So(tokens[3].Type, ShouldEqual, TokenNotEquals)
				So(tokens[5].Type, ShouldEqual, TokenLessThan)
				So(tokens[7].Type, ShouldEqual, TokenGreaterThan)
			})
		})

		Convey("Complex Expressions", func() {
			Convey("should tokenize vault with logical or and grab", func() {
				tokens := TokenizeExpression(`"secret:pass" || grab defaults.password`)
				So(len(tokens), ShouldEqual, 4)
				So(tokens[0].Type, ShouldEqual, TokenLiteral)
				So(tokens[0].Value, ShouldEqual, `"secret:pass"`)
				So(tokens[1].Type, ShouldEqual, TokenLogicalOr)
				So(tokens[2].Type, ShouldEqual, TokenOperator)
				So(tokens[2].Value, ShouldEqual, "grab")
				So(tokens[3].Type, ShouldEqual, TokenReference)
				So(tokens[3].Value, ShouldEqual, "defaults.password")
			})

			Convey("should tokenize nested expression with parentheses", func() {
				tokens := TokenizeExpression(`"path" || (grab x || concat y z)`)
				So(len(tokens), ShouldEqual, 10)
				So(tokens[0].Type, ShouldEqual, TokenLiteral)
				So(tokens[1].Type, ShouldEqual, TokenLogicalOr)
				So(tokens[2].Type, ShouldEqual, TokenOpenParen)
				So(tokens[3].Type, ShouldEqual, TokenOperator)
				So(tokens[3].Value, ShouldEqual, "grab")
				So(tokens[4].Type, ShouldEqual, TokenReference)
				So(tokens[5].Type, ShouldEqual, TokenLogicalOr)
				So(tokens[6].Type, ShouldEqual, TokenOperator)
				So(tokens[6].Value, ShouldEqual, "concat")
				So(tokens[7].Type, ShouldEqual, TokenReference)
				So(tokens[8].Type, ShouldEqual, TokenReference)
				So(tokens[9].Type, ShouldEqual, TokenCloseParen)
			})
		})

		Convey("Escaped Characters", func() {
			Convey("should handle escaped quotes", func() {
				tokens := TokenizeExpression(`"hello \"world\""`)
				So(len(tokens), ShouldEqual, 1)
				So(tokens[0].Type, ShouldEqual, TokenLiteral)
				// The tokenizer processes escapes, so \" becomes "
				So(tokens[0].Value, ShouldEqual, `"hello "world""`)
			})

			Convey("should handle escaped special chars", func() {
				tokens := TokenizeExpression(`"line1\nline2\ttab"`)
				So(len(tokens), ShouldEqual, 1)
				So(tokens[0].Type, ShouldEqual, TokenLiteral)
				// The actual newline and tab are in the string
				So(tokens[0].Value, ShouldContainSubstring, "\n")
				So(tokens[0].Value, ShouldContainSubstring, "\t")
			})
		})

		Convey("Position Tracking", func() {
			Convey("should track token positions", func() {
				tokens := TokenizeExpression(`grab a || b`)
				So(tokens[0].Pos, ShouldEqual, 0)  // grab
				So(tokens[1].Pos, ShouldEqual, 5)  // a
				So(tokens[2].Pos, ShouldEqual, 7)  // ||
				So(tokens[3].Pos, ShouldEqual, 10) // b
			})

			Convey("should track line and column", func() {
				tokens := TokenizeExpression("a\nb")
				So(tokens[0].Line, ShouldEqual, 1)
				So(tokens[0].Col, ShouldEqual, 1)
				So(tokens[1].Line, ShouldEqual, 2)
				So(tokens[1].Col, ShouldEqual, 1)
			})
		})

		Convey("Precedence and Associativity", func() {
			Convey("should have correct precedence ordering", func() {
				So(GetTokenPrecedence(TokenLogicalOr), ShouldBeLessThan, GetTokenPrecedence(TokenLogicalAnd))
				So(GetTokenPrecedence(TokenEquals), ShouldBeLessThan, GetTokenPrecedence(TokenLessThan))
				So(GetTokenPrecedence(TokenPlus), ShouldBeLessThan, GetTokenPrecedence(TokenMultiply))
			})

			Convey("should have correct associativity", func() {
				So(GetTokenAssociativity(TokenLogicalOr), ShouldEqual, RightAssociative)
				So(GetTokenAssociativity(TokenLogicalAnd), ShouldEqual, LeftAssociative)
				So(GetTokenAssociativity(TokenPlus), ShouldEqual, LeftAssociative)
			})
		})
	})
}