package parser

import (
	"testing"
	
	. "github.com/smartystreets/goconvey/convey"
)

// Define phase constants locally to avoid circular import
const (
	testMergePhase = 0
	testEvalPhase  = 1
	testParamPhase = 2
)

func TestEnhancedParser(t *testing.T) {
	Convey("Enhanced Parser", t, func() {
		var registry *OperatorRegistry
		
		// Setup registry
		registry = NewOperatorRegistry()
		registry.Register(&OperatorInfo{
			Name:       "grab",
			Precedence: PrecedencePostfix,
			MinArgs:    1,
			MaxArgs:    1,
			Phase:      testEvalPhase,
		})
		registry.Register(&OperatorInfo{
			Name:       "vault",
			Precedence: PrecedencePostfix,
			MinArgs:    1,
			MaxArgs:    -1,
			Phase:      testEvalPhase,
		})
		registry.Register(&OperatorInfo{
			Name:       "concat",
			Precedence: PrecedencePostfix,
			MinArgs:    1,
			MaxArgs:    -1,
			Phase:      testMergePhase,
		})
		registry.Register(&OperatorInfo{
			Name:          "+",
			Precedence:    PrecedenceAddition,
			Associativity: AssociativityLeft,
			MinArgs:       2,
			MaxArgs:       2,
			Phase:         testEvalPhase,
		})
		registry.Register(&OperatorInfo{
			Name:          "*",
			Precedence:    PrecedenceMultiplication,
			Associativity: AssociativityLeft,
			MinArgs:       2,
			MaxArgs:       2,
			Phase:         testEvalPhase,
		})
		
		parseExpression := func(input string) (*Expr, error) {
			tokenizer := NewEnhancedTokenizer(input)
			tokens := tokenizer.Tokenize()
			
			parser := NewEnhancedParser(tokens, registry)
			return parser.Parse()
		}
		
		Convey("Basic Expressions", func() {
			Convey("should parse literals", func() {
				expr, err := parseExpression("42")
				So(err, ShouldBeNil)
				So(expr.Type, ShouldEqual, Literal)
				So(expr.Literal, ShouldEqual, int64(42))
				
				expr, err = parseExpression("\"hello world\"")
				So(err, ShouldBeNil)
				So(expr.Type, ShouldEqual, Literal)
				So(expr.Literal, ShouldEqual, "hello world")
				
				expr, err = parseExpression("true")
				So(err, ShouldBeNil)
				So(expr.Type, ShouldEqual, Literal)
				So(expr.Literal, ShouldEqual, true)
				
				expr, err = parseExpression("null")
				So(err, ShouldBeNil)
				So(expr.Type, ShouldEqual, Literal)
				So(expr.Literal, ShouldBeNil)
			})
			
			Convey("should parse references", func() {
				expr, err := parseExpression("foo.bar")
				So(err, ShouldBeNil)
				So(expr.Type, ShouldEqual, Reference)
				So(expr.Reference.String(), ShouldEqual, "foo.bar")
			})
			
			Convey("should parse environment variables", func() {
				expr, err := parseExpression("$HOME")
				So(err, ShouldBeNil)
				So(expr.Type, ShouldEqual, EnvVar)
				So(expr.Name, ShouldEqual, "HOME")
			})
		})
		
		Convey("Operator Calls", func() {
			Convey("should parse simple operator calls", func() {
				expr, err := parseExpression("grab foo.bar")
				So(err, ShouldBeNil)
				So(expr.Type, ShouldEqual, OperatorCall)
				So(expr.Op(), ShouldEqual, "grab")
				So(len(expr.Args()), ShouldEqual, 1)
				So(expr.Args()[0].Type, ShouldEqual, Reference)
				So(expr.Args()[0].Reference.String(), ShouldEqual, "foo.bar")
			})
			
			Convey("should parse operator calls with multiple arguments", func() {
				expr, err := parseExpression("concat \"hello\" \"world\"")
				So(err, ShouldBeNil)
				So(expr.Type, ShouldEqual, OperatorCall)
				So(expr.Op(), ShouldEqual, "concat")
				So(len(expr.Args()), ShouldEqual, 2)
				So(expr.Args()[0].Literal, ShouldEqual, "hello")
				So(expr.Args()[1].Literal, ShouldEqual, "world")
			})
			
			Convey("should parse operator calls with parentheses", func() {
				expr, err := parseExpression("vault(\"secret/data\")")
				So(err, ShouldBeNil)
				So(expr.Type, ShouldEqual, OperatorCall)
				So(expr.Op(), ShouldEqual, "vault")
				So(len(expr.Args()), ShouldEqual, 1)
				So(expr.Args()[0].Literal, ShouldEqual, "secret/data")
			})
		})
		
		Convey("Logical OR Expressions", func() {
			Convey("should parse simple logical OR", func() {
				expr, err := parseExpression("foo || \"default\"")
				So(err, ShouldBeNil)
				So(expr.Type, ShouldEqual, LogicalOr)
				So(expr.Left.Type, ShouldEqual, Reference)
				So(expr.Left.Reference.String(), ShouldEqual, "foo")
				So(expr.Right.Type, ShouldEqual, Literal)
				So(expr.Right.Literal, ShouldEqual, "default")
			})
			
			Convey("should parse logical OR with operators", func() {
				expr, err := parseExpression("grab foo.bar || \"default\"")
				So(err, ShouldBeNil)
				So(expr.Type, ShouldEqual, LogicalOr)
				So(expr.Left.Type, ShouldEqual, OperatorCall)
				So(expr.Left.Op(), ShouldEqual, "grab")
				So(expr.Right.Type, ShouldEqual, Literal)
				So(expr.Right.Literal, ShouldEqual, "default")
			})
			
			Convey("should parse chained logical OR", func() {
				expr, err := parseExpression("a || b || c")
				So(err, ShouldBeNil)
				So(expr.Type, ShouldEqual, LogicalOr)
				So(expr.Left.Type, ShouldEqual, Reference)
				So(expr.Left.Reference.String(), ShouldEqual, "a")
				So(expr.Right.Type, ShouldEqual, LogicalOr)
				So(expr.Right.Left.Type, ShouldEqual, Reference)
				So(expr.Right.Left.Reference.String(), ShouldEqual, "b")
				So(expr.Right.Right.Type, ShouldEqual, Reference)
				So(expr.Right.Right.Reference.String(), ShouldEqual, "c")
			})
		})
		
		Convey("Parenthesized Expressions", func() {
			Convey("should parse parenthesized expressions", func() {
				expr, err := parseExpression("(42)")
				So(err, ShouldBeNil)
				So(expr.Type, ShouldEqual, Literal)
				So(expr.Literal, ShouldEqual, int64(42))
			})
			
			Convey("should respect parentheses for precedence", func() {
				expr, err := parseExpression("(grab foo || grab bar) || \"default\"")
				So(err, ShouldBeNil)
				So(expr.Type, ShouldEqual, LogicalOr)
				So(expr.Left.Type, ShouldEqual, LogicalOr)
				So(expr.Left.Left.Type, ShouldEqual, OperatorCall)
				So(expr.Left.Left.Op(), ShouldEqual, "grab")
				So(expr.Right.Type, ShouldEqual, Literal)
			})
		})
		
		Convey("Complex Expressions", func() {
			Convey("should parse vault with logical OR", func() {
				expr, err := parseExpression("vault \"secret/data:password\" || \"default-pass\"")
				So(err, ShouldBeNil)
				So(expr.Type, ShouldEqual, LogicalOr)
				So(expr.Left.Type, ShouldEqual, OperatorCall)
				So(expr.Left.Op(), ShouldEqual, "vault")
				So(expr.Left.Args()[0].Literal, ShouldEqual, "secret/data:password")
				So(expr.Right.Literal, ShouldEqual, "default-pass")
			})
			
			Convey("should parse nested operator calls", func() {
				expr, err := parseExpression("concat (grab foo.prefix) \"-\" (grab foo.suffix)")
				So(err, ShouldBeNil)
				So(expr.Type, ShouldEqual, OperatorCall)
				So(expr.Op(), ShouldEqual, "concat")
				So(len(expr.Args()), ShouldEqual, 3)
				So(expr.Args()[0].Type, ShouldEqual, OperatorCall)
				So(expr.Args()[0].Op(), ShouldEqual, "grab")
				So(expr.Args()[1].Literal, ShouldEqual, "-")
				So(expr.Args()[2].Type, ShouldEqual, OperatorCall)
				So(expr.Args()[2].Op(), ShouldEqual, "grab")
			})
		})
		
		Convey("Arithmetic Expressions", func() {
			Convey("should parse addition", func() {
				expr, err := parseExpression("1 + 2")
				So(err, ShouldBeNil)
				So(expr.Type, ShouldEqual, OperatorCall)
				So(expr.Op(), ShouldEqual, "+")
				So(len(expr.Args()), ShouldEqual, 2)
				So(expr.Args()[0].Literal, ShouldEqual, int64(1))
				So(expr.Args()[1].Literal, ShouldEqual, int64(2))
			})
			
			Convey("should respect precedence", func() {
				expr, err := parseExpression("1 + 2 * 3")
				So(err, ShouldBeNil)
				So(expr.Type, ShouldEqual, OperatorCall)
				So(expr.Op(), ShouldEqual, "+")
				So(expr.Args()[0].Literal, ShouldEqual, int64(1))
				So(expr.Args()[1].Type, ShouldEqual, OperatorCall)
				So(expr.Args()[1].Op(), ShouldEqual, "*")
				So(expr.Args()[1].Args()[0].Literal, ShouldEqual, int64(2))
				So(expr.Args()[1].Args()[1].Literal, ShouldEqual, int64(3))
			})
			
			Convey("should handle parentheses", func() {
				expr, err := parseExpression("(1 + 2) * 3")
				So(err, ShouldBeNil)
				So(expr.Type, ShouldEqual, OperatorCall)
				So(expr.Op(), ShouldEqual, "*")
				So(expr.Args()[0].Type, ShouldEqual, OperatorCall)
				So(expr.Args()[0].Op(), ShouldEqual, "+")
				So(expr.Args()[1].Literal, ShouldEqual, int64(3))
			})
		})
		
		Convey("Error Handling", func() {
			Convey("should error on empty expression", func() {
				_, err := parseExpression("")
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "no tokens")
			})
			
			Convey("should error on unclosed parentheses", func() {
				_, err := parseExpression("(42")
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "expected ) to match opening parenthesis")
			})
			
			Convey("should error on unknown operator", func() {
				_, err := parseExpression("unknown_op foo")
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "unexpected token")
			})
			
			Convey("should parse operator without arguments as reference", func() {
				// When grab appears alone, it's treated as a reference
				expr, err := parseExpression("grab")
				So(err, ShouldBeNil)
				So(expr.Type, ShouldEqual, Reference)
				So(expr.Reference.String(), ShouldEqual, "grab")
			})
			
			Convey("should error on trailing tokens", func() {
				_, err := parseExpression("42 extra")
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "unexpected token")
			})
		})
	})
}