package operators

import (
	"testing"

	"github.com/geofffranks/simpleyaml"
	. "github.com/smartystreets/goconvey/convey"
)

// testArithmetic is a helper function to test arithmetic operations
func testArithmetic(input string, key string, expected interface{}) {
	y, err := simpleyaml.NewYaml([]byte(input))
	So(err, ShouldBeNil)
	
	data, err := y.Map()
	So(err, ShouldBeNil)
	
	ev := &Evaluator{Tree: data}
	err = ev.RunPhase(EvalPhase)
	So(err, ShouldBeNil)
	So(ev.Tree[key], ShouldEqual, expected)
}

// testArithmeticError is a helper function to test arithmetic errors
func testArithmeticError(input string, expectedError string) {
	y, err := simpleyaml.NewYaml([]byte(input))
	So(err, ShouldBeNil)
	
	data, err := y.Map()
	So(err, ShouldBeNil)
	
	ev := &Evaluator{Tree: data}
	err = ev.RunPhase(EvalPhase)
	So(err, ShouldNotBeNil)
	So(err.Error(), ShouldContainSubstring, expectedError)
}

func TestArithmeticOperators(t *testing.T) {
	Convey("Arithmetic Operators", t, func() {
		// Enable  parser for arithmetic operators
		// Arithmetic operators require  parser, which should be enabled by default

		Convey("Addition Operator (+)", func() {
			Convey("should add two integers", func() {
				input := `
a: 5
b: 3
result: (( a + b ))
`
				testArithmetic(input, "result", int64(8))
			})

			Convey("should add two floats", func() {
				input := `
a: 5.5
b: 3.2
result: (( a + b ))
`
				testArithmetic(input, "result", 8.7)
			})

			Convey("should add int and float", func() {
				input := `
a: 5
b: 3.2
result: (( a + b ))
`
				testArithmetic(input, "result", 8.2)
			})

			Convey("should concatenate strings", func() {
				input := `
a: "hello"
b: "world"
result: (( a + b ))
`
				testArithmetic(input, "result", "helloworld")
			})

			Convey("should concatenate string and number", func() {
				input := `
a: "value: "
b: 42
result: (( a + b ))
`
				testArithmetic(input, "result", "value: 42")
			})

			Convey("should handle nil as zero", func() {
				input := `
a: ~
b: 5
result: (( a + b ))
`
				testArithmetic(input, "result", int64(5))
			})

			Convey("should work with nested operators", func() {
				input := `
base: 10
multiplier: 2
addend: 5
result: (( (grab base) + (grab addend) ))
complex: (( (base * multiplier) + addend ))
`
				var data map[interface{}]interface{}
				y, err := simpleyaml.NewYaml([]byte(input))
				So(err, ShouldBeNil)
				data, err = y.Map()
				So(err, ShouldBeNil)

				ev := &Evaluator{Tree: data}
				err = ev.RunPhase(EvalPhase)
				So(err, ShouldBeNil)
				So(ev.Tree["result"], ShouldEqual, int64(15))
				So(ev.Tree["complex"], ShouldEqual, int64(25))
			})
		})

		Convey("Subtraction Operator (-)", func() {
			Convey("should subtract two integers", func() {
				input := `
a: 10
b: 3
result: (( a - b ))
`
				testArithmetic(input, "result", int64(7))
			})

			Convey("should subtract two floats", func() {
				input := `
a: 10.5
b: 3.2
result: (( a - b ))
`
				testArithmetic(input, "result", 7.3)
			})

			Convey("should handle negative results", func() {
				input := `
a: 3
b: 10
result: (( a - b ))
`
				testArithmetic(input, "result", int64(-7))
			})

			Convey("should error on string operands", func() {
				input := `
a: "hello"
b: 5
result: (( a - b ))
`
				testArithmeticError(input, "cannot use string")
			})
		})

		Convey("Multiplication Operator (*)", func() {
			Convey("should multiply two integers", func() {
				input := `
a: 5
b: 3
result: (( a * b ))
`
				testArithmetic(input, "result", int64(15))
			})

			Convey("should multiply two floats", func() {
				input := `
a: 5.5
b: 2.0
result: (( a * b ))
`
				testArithmetic(input, "result", 11.0)
			})

			Convey("should repeat string by integer", func() {
				input := `
text: "ha"
count: 3
result: (( text * count ))
`
				testArithmetic(input, "result", "hahaha")
			})

			Convey("should repeat string by integer (reversed)", func() {
				input := `
count: 3
text: "ha"
result: (( count * text ))
`
				testArithmetic(input, "result", "hahaha")
			})

			Convey("should handle zero multiplication", func() {
				input := `
a: 100
b: 0
result: (( a * b ))
`
				testArithmetic(input, "result", int64(0))
			})

			Convey("should error on large string repetition", func() {
				input := `
text: "x"
count: 10001
result: (( text * count ))
`
				testArithmeticError(input, "string repetition count too large")
			})
		})

		Convey("Division Operator (/)", func() {
			Convey("should divide two integers returning float", func() {
				input := `
a: 10
b: 3
result: (( a / b ))
`
				testArithmetic(input, "result", 10.0/3.0)
			})

			Convey("should divide two floats", func() {
				input := `
a: 10.5
b: 2.5
result: (( a / b ))
`
				testArithmetic(input, "result", 4.2)
			})

			Convey("should handle exact division", func() {
				input := `
a: 10
b: 2
result: (( a / b ))
`
				testArithmetic(input, "result", 5.0)
			})

			Convey("should error on division by zero", func() {
				input := `
a: 10
b: 0
result: (( a / b ))
`
				testArithmeticError(input, "division by zero")
			})
		})

		Convey("Modulo Operator (%)", func() {
			Convey("should calculate modulo of two integers", func() {
				input := `
a: 10
b: 3
result: (( a % b ))
`
				testArithmetic(input, "result", int64(1))
			})

			Convey("should handle negative operands", func() {
				input := `
a: -10
b: 3
result: (( a % b ))
`
				testArithmetic(input, "result", int64(-1))
			})

			Convey("should error on float operands", func() {
				input := `
a: 10.5
b: 3
result: (( a % b ))
`
				testArithmeticError(input, "not an integer")
			})

			Convey("should error on modulo by zero", func() {
				input := `
a: 10
b: 0
result: (( a % b ))
`
				testArithmeticError(input, "modulo by zero")
			})
		})

		Convey("Operator Precedence", func() {
			Convey("should respect multiplication before addition", func() {
				input := `
result: (( 2 + 3 * 4 ))
`
				testArithmetic(input, "result", int64(14)) // 2 + 12, not 5 * 4
			})

			Convey("should respect division before subtraction", func() {
				input := `
result: (( 10 - 6 / 2 ))
`
				testArithmetic(input, "result", 7.0) // 10 - 3, not 4 / 2
			})

			Convey("should handle multiple operators", func() {
				input := `
result: (( 2 * 3 + 4 * 5 ))
`
				testArithmetic(input, "result", int64(26)) // 6 + 20
			})

			Convey("should respect parentheses", func() {
				input := `
result1: (( (2 + 3) * 4 ))
result2: (( 2 + (3 * 4) ))
`
				var data map[interface{}]interface{}
				y, err := simpleyaml.NewYaml([]byte(input))
				So(err, ShouldBeNil)
				data, err = y.Map()
				So(err, ShouldBeNil)

				ev := &Evaluator{Tree: data}
				err = ev.RunPhase(EvalPhase)
				So(err, ShouldBeNil)
				So(ev.Tree["result1"], ShouldEqual, int64(20)) // 5 * 4
				So(ev.Tree["result2"], ShouldEqual, int64(14)) // 2 + 12
			})

			Convey("should handle complex expressions", func() {
				input := `
a: 10
b: 2
c: 3
d: 4
result: (( a / b + c * d ))
`
				testArithmetic(input, "result", 17.0) // 5.0 + 12
			})
		})

		Convey("Mixed Operations", func() {
			Convey("should handle arithmetic with references", func() {
				input := `
meta:
  base: 100
  factor: 2
  offset: 50
scaled: (( meta.base * meta.factor ))
adjusted: (( scaled + meta.offset ))
final: (( adjusted / 10 ))
`
				var data map[interface{}]interface{}
				y, err := simpleyaml.NewYaml([]byte(input))
				So(err, ShouldBeNil)
				data, err = y.Map()
				So(err, ShouldBeNil)

				ev := &Evaluator{Tree: data}
				err = ev.RunPhase(EvalPhase)
				So(err, ShouldBeNil)
				So(ev.Tree["scaled"], ShouldEqual, int64(200))
				So(ev.Tree["adjusted"], ShouldEqual, int64(250))
				So(ev.Tree["final"], ShouldEqual, 25.0)
			})

			Convey("should work with other operators", func() {
				input := `
list:
  - 10
  - 20
  - 30
first: (( grab list.0 ))
second: (( grab list.1 ))
sum: (( first + second ))
product: (( first * second ))
average: (( sum / 2 ))
`
				var data map[interface{}]interface{}
				y, err := simpleyaml.NewYaml([]byte(input))
				So(err, ShouldBeNil)
				data, err = y.Map()
				So(err, ShouldBeNil)

				ev := &Evaluator{Tree: data}
				err = ev.RunPhase(EvalPhase)
				So(err, ShouldBeNil)
				So(ev.Tree["sum"], ShouldEqual, int64(30))
				So(ev.Tree["product"], ShouldEqual, int64(200))
				So(ev.Tree["average"], ShouldEqual, 15.0)
			})
		})
	})
}

