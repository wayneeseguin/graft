package parser

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestQuotedStringsWithSpaces(t *testing.T) {
	Convey("Parser should handle quoted strings with spaces correctly", t, func() {
		registry := NewOperatorRegistry()

		// Register common operators for testing
		registry.Register(&OperatorInfo{
			Name:    "concat",
			MinArgs: 1,
			MaxArgs: -1,
		})
		registry.Register(&OperatorInfo{
			Name:    "grab",
			MinArgs: 1,
			MaxArgs: 1, // grab typically takes only one argument - the path
		})

		Convey("Tokenizer should correctly tokenize quoted strings with spaces", func() {
			input := `"hello world"`
			tokens := TokenizeExpression(input)

			So(tokens, ShouldHaveLength, 1)
			So(tokens[0].Type, ShouldEqual, TokenLiteral)
			So(tokens[0].Value, ShouldEqual, `"hello world"`)
		})

		Convey("Parser should preserve spaces in quoted string literals", func() {
			input := `"hello world"`
			expr, err := ParseExpression(input, registry)

			So(err, ShouldBeNil)
			So(expr, ShouldNotBeNil)
			So(expr.Type, ShouldEqual, Literal)
			So(expr.Literal, ShouldEqual, "hello world") // Without quotes
		})

		Convey("Parser should handle quoted strings with multiple spaces", func() {
			input := `"hello    world"`
			expr, err := ParseExpression(input, registry)

			So(err, ShouldBeNil)
			So(expr.Type, ShouldEqual, Literal)
			So(expr.Literal, ShouldEqual, "hello    world")
		})

		Convey("Parser should handle quoted strings as operator arguments", func() {
			input := `concat "hello world" "foo bar"`
			expr, err := ParseExpression(input, registry)

			So(err, ShouldBeNil)
			So(expr.Type, ShouldEqual, OperatorCall)
			So(expr.Name, ShouldEqual, "concat")
			So(expr.Args(), ShouldHaveLength, 2)
			So(expr.Args()[0].Type, ShouldEqual, Literal)
			So(expr.Args()[0].Literal, ShouldEqual, "hello world")
			So(expr.Args()[1].Type, ShouldEqual, Literal)
			So(expr.Args()[1].Literal, ShouldEqual, "foo bar")
		})

		Convey("Parser should handle empty quoted strings", func() {
			input := `""`
			expr, err := ParseExpression(input, registry)

			So(err, ShouldBeNil)
			So(expr.Type, ShouldEqual, Literal)
			So(expr.Literal, ShouldEqual, "")
		})

		Convey("Parser should handle quoted strings with special characters", func() {
			input := `"hello\nworld\ttab"`
			expr, err := ParseExpression(input, registry)

			So(err, ShouldBeNil)
			So(expr.Type, ShouldEqual, Literal)
			So(expr.Literal, ShouldEqual, "hello\nworld\ttab")
		})

		Convey("Parser should handle quoted strings with escaped quotes", func() {
			input := `"hello \"world\""`
			expr, err := ParseExpression(input, registry)

			So(err, ShouldBeNil)
			So(expr.Type, ShouldEqual, Literal)
			So(expr.Literal, ShouldEqual, `hello "world"`)
		})

		Convey("Regression test: quoted strings with leading/trailing spaces", func() {
			input := `" hello world "`
			expr, err := ParseExpression(input, registry)

			So(err, ShouldBeNil)
			So(expr.Type, ShouldEqual, Literal)
			So(expr.Literal, ShouldEqual, " hello world ")
		})

		Convey("Regression test: quoted strings in complex expressions", func() {
			input := `concat "hello " grab foo " world"`
			expr, err := ParseExpression(input, registry)

			So(err, ShouldBeNil)
			So(expr.Type, ShouldEqual, OperatorCall)
			So(expr.Name, ShouldEqual, "concat")
			So(expr.Args(), ShouldHaveLength, 3)
			So(expr.Args()[0].Type, ShouldEqual, Literal)
			So(expr.Args()[0].Literal, ShouldEqual, "hello ")
			So(expr.Args()[1].Type, ShouldEqual, OperatorCall)
			So(expr.Args()[1].Name, ShouldEqual, "grab")
			So(expr.Args()[2].Type, ShouldEqual, Literal)
			So(expr.Args()[2].Literal, ShouldEqual, " world")
		})

		Convey("Tokenizer should handle multiple quoted strings correctly", func() {
			input := `"first string" "second string"`
			tokens := TokenizeExpression(input)

			So(tokens, ShouldHaveLength, 2)
			So(tokens[0].Type, ShouldEqual, TokenLiteral)
			So(tokens[0].Value, ShouldEqual, `"first string"`)
			So(tokens[1].Type, ShouldEqual, TokenLiteral)
			So(tokens[1].Value, ShouldEqual, `"second string"`)
		})
	})
}
