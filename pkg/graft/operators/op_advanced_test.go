package operators

import (
	"testing"

	"github.com/geofffranks/simpleyaml"
	. "github.com/smartystreets/goconvey/convey"
)

func TestAdvancedOperatorIntegration(t *testing.T) {
	YAML := func(s string) map[interface{}]interface{} {
		y, err := simpleyaml.NewYaml([]byte(s))
		So(err, ShouldBeNil)

		data, err := y.Map()
		So(err, ShouldBeNil)

		return data
	}
	Convey("Advanced operator integration", t, func() {
		ev := &Evaluator{Tree: YAML(`
users:
  - name: alice
    age: 30
    role: admin
    active: true
  - name: bob
    age: 25
    role: user
    active: false
  - name: charlie
    age: 35
    role: admin
    active: true
config:
  debug: false
  environment: production
  max_users: 100
  min_age: 18
`)}

		Convey("complex expressions with multiple operators", func() {
			// Mix of comparison, boolean, and ternary
			result, err := parseAndEvaluateExpression(ev, `(( config.environment == "production" && !config.debug ? "optimized" : "debug mode" ))`)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, "optimized")

			// Nested comparisons
			result, err = parseAndEvaluateExpression(ev, `(( config.max_users > 50 && config.min_age >= 18 ))`)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, true)
		})

		Convey("parentheses override precedence correctly", func() {
			// Without parentheses: false || true && false = false || false = false
			// With parentheses: (false || true) && false = true && false = false
			result, err := parseAndEvaluateExpression(ev, `(( (config.debug || config.environment == "production") && config.max_users < 50 ))`)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, false) // (false || true) && false = true && false = false

			// Complex precedence
			result, err = parseAndEvaluateExpression(ev, `(( 10 + 5 * 2 > 15 ? "yes" : "no" ))`)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, "yes") // 10 + (5 * 2) = 20 > 15

			result, err = parseAndEvaluateExpression(ev, `(( (10 + 5) * 2 > 25 ? "yes" : "no" ))`)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, "yes") // (10 + 5) * 2 = 30 > 25
		})

		Convey("chained comparisons work correctly", func() {
			// Age range check
			result, err := parseAndEvaluateExpression(ev, `(( users.0.age >= 25 && users.0.age <= 35 ))`)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, true)
		})

		Convey("mixed type comparisons", func() {
			// String and number comparison (should convert to string)
			result, err := parseAndEvaluateExpression(ev, `(( "100" > "50" ))`)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, false) // Lexicographic: "100" < "50"

			// Boolean in ternary
			result, err = parseAndEvaluateExpression(ev, `(( users.0.active ? users.0.name : "inactive" ))`)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, "alice")
		})

		Convey("error handling", func() {
			// Invalid comparison
			_, err := parseAndEvaluateExpression(ev, `(( users < config ))`)
			So(err, ShouldNotBeNil)

			// Missing ternary colon
			_, err = parseAndEvaluateExpression(ev, `(( true ? "yes" ))`)
			So(err, ShouldNotBeNil)
		})
	})
}

func TestOperatorPrecedenceTable(t *testing.T) {
	// Enable debug logging temporarily
	//log.DebugOn = true
	//defer func() { log.DebugOn = false }()
	
	YAML := func(s string) map[interface{}]interface{} {
		y, err := simpleyaml.NewYaml([]byte(s))
		So(err, ShouldBeNil)

		data, err := y.Map()
		So(err, ShouldBeNil)

		return data
	}
	Convey("Operator precedence", t, func() {
		ev := &Evaluator{Tree: YAML(`{}`)}

		testCases := []struct {
			expr     string
			expected interface{}
			desc     string
		}{
			// Arithmetic precedence
			{`(( 2 + 3 * 4 ))`, float64(14), "multiplication before addition"},
			{`(( 10 - 6 / 2 ))`, float64(7), "division before subtraction"},
			{`(( 10 % 3 + 1 ))`, float64(2), "modulo same precedence as multiplication"},

			// Comparison precedence
			{`(( 2 + 3 > 4 ))`, true, "arithmetic before comparison (greater than)"},
			{`(( 5 * 2 <= 10 ))`, true, "arithmetic before comparison (less equal)"},

			// Boolean precedence (|| is fallback in Graft, not boolean OR)
			{`(( true || false && false ))`, true, "&& before ||"},
			{`(( false && true || true ))`, false, "&& before || (with false)"}, // fallback returns left value

			// Comparison and boolean
			{`(( 5 > 3 && 2 < 4 ))`, true, "comparison before boolean"},
			{`(( 1 == 1 || 2 != 2 ))`, true, "equality before boolean (with OR)"}, // 1==1 is true, fallback returns it

			// Ternary lowest precedence
			{`(( true || false ? "yes" : "no" ))`, "yes", "boolean before ternary"},
			{`(( 1 + 1 == 2 ? 10 * 2 : 5 ))`, float64(20), "all operators before ternary"},

			// Parentheses override
			{`(( (2 + 3) * 4 ))`, float64(20), "parentheses override precedence (left)"},
			{`(( 2 * (3 + 4) ))`, float64(14), "parentheses override precedence (right)"},
			{`(( (true || false) && false ))`, false, "parentheses change boolean evaluation"},
		}

		for _, tc := range testCases {
			Convey(tc.desc, func() {
				result, err := parseAndEvaluateExpression(ev, tc.expr)
				So(err, ShouldBeNil)
				if f, ok := tc.expected.(float64); ok {
					// Handle both float64 and int64 results
					switch v := result.(type) {
					case float64:
						So(v, ShouldAlmostEqual, f, 0.001)
					case int64:
						So(float64(v), ShouldAlmostEqual, f, 0.001)
					default:
						So(result, ShouldEqual, tc.expected)
					}
				} else {
					So(result, ShouldEqual, tc.expected)
				}
			})
		}
	})
}

func TestEdgeCases(t *testing.T) {
	YAML := func(s string) map[interface{}]interface{} {
		y, err := simpleyaml.NewYaml([]byte(s))
		So(err, ShouldBeNil)

		data, err := y.Map()
		So(err, ShouldBeNil)

		return data
	}
	Convey("Edge cases", t, func() {
		ev := &Evaluator{Tree: YAML(`
nil_value: ~
empty_string: ""
zero: 0
false_value: false
empty_array: []
empty_map: {}
`)}

		Convey("nil handling", func() {
			result, err := parseAndEvaluateExpression(ev, `(( nil_value == nil ))`)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, true)

			result, err = parseAndEvaluateExpression(ev, `(( nil_value != "something" ))`)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, true)

			// nil is falsy
			result, err = parseAndEvaluateExpression(ev, `(( nil_value ? "yes" : "no" ))`)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, "no")
		})

		Convey("empty value handling", func() {
			// Empty string with fallback
			result, err := parseAndEvaluateExpression(ev, `(( empty_string || "default" ))`)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, "") // Fallback operator returns the value, not boolean

			// Zero is falsy
			result, err = parseAndEvaluateExpression(ev, `(( zero && true ))`)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, false)

			// Empty collections are falsy
			result, err = parseAndEvaluateExpression(ev, `(( empty_array ? "has items" : "empty" ))`)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, "empty")
		})

		Convey("type coercion in comparisons", func() {
			result, err := parseAndEvaluateExpression(ev, `(( 5 == "5" ))`)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, false) // No automatic coercion

			result, err = parseAndEvaluateExpression(ev, `(( zero == false ))`)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, false) // Different types
		})
	})
}
