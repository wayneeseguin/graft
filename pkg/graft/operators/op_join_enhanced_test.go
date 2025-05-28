package operators

import (
	"os"
	"testing"

	"github.com/geofffranks/yaml"
	. "github.com/smartystreets/goconvey/convey"
)

func TestEnhancedJoinOperator(t *testing.T) {
	Convey("Enhanced Join Operator", t, func() {
		// Enable enhanced parser for these tests
		oldUseEnhanced := UseEnhancedParser
		UseEnhancedParser = true
		EnableEnhancedJoin()
		defer func() {
			UseEnhancedParser = oldUseEnhanced
			if !oldUseEnhanced {
				RegisterOp("join", JoinOperator{})
			}
		}()

		Convey("should support nested expressions for separator", func() {
			input := `
meta:
  separator: " - "
items:
  - apple
  - banana
  - cherry
result: (( join (grab meta.separator) items ))
`
			var data map[interface{}]interface{}
			err := yaml.Unmarshal([]byte(input), &data)
			So(err, ShouldBeNil)

			ev := &Evaluator{Tree: data}
			err = ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["result"], ShouldEqual, "apple - banana - cherry")
		})

		Convey("should support nested concat for separator", func() {
			input := `
meta:
  prefix: "["
  suffix: "]"
items:
  - one
  - two
  - three
result: (( join (concat meta.prefix "," meta.suffix) items ))
`
			var data map[interface{}]interface{}
			err := yaml.Unmarshal([]byte(input), &data)
			So(err, ShouldBeNil)

			ev := &Evaluator{Tree: data}
			err = ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["result"], ShouldEqual, "one[,]two[,]three")
		})

		Convey("should support nested grab for list arguments", func() {
			input := `
data:
  users:
    - alice
    - bob
  roles:
    - admin
    - user
result: (( join ", " (grab data.users) (grab data.roles) ))
`
			var data map[interface{}]interface{}
			err := yaml.Unmarshal([]byte(input), &data)
			So(err, ShouldBeNil)

			ev := &Evaluator{Tree: data}
			err = ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["result"], ShouldEqual, "alice, bob, admin, user")
		})

		Convey("should support environment variables", func() {
			os.Setenv("JOIN_SEP", " | ")
			defer os.Unsetenv("JOIN_SEP")

			input := `
items:
  - first
  - second
  - third
result: (( join $JOIN_SEP items ))
`
			var data map[interface{}]interface{}
			err := yaml.Unmarshal([]byte(input), &data)
			So(err, ShouldBeNil)

			ev := &Evaluator{Tree: data}
			err = ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["result"], ShouldEqual, "first | second | third")
		})

		Convey("should support mixed nested expressions and literals", func() {
			input := `
meta:
  key: "services"
data:
  services:
    - web
    - api
result: (( join "-" data.services "extra" ))
`
			var data map[interface{}]interface{}
			err := yaml.Unmarshal([]byte(input), &data)
			So(err, ShouldBeNil)

			ev := &Evaluator{Tree: data}
			err = ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["result"], ShouldEqual, "web-api-extra")
		})

		Convey("should handle nil values gracefully", func() {
			input := `
items:
  - one
  - two
result: (( join "," items "three" ))
`
			var data map[interface{}]interface{}
			err := yaml.Unmarshal([]byte(input), &data)
			So(err, ShouldBeNil)

			ev := &Evaluator{Tree: data}
			err = ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["result"], ShouldEqual, "one,two,three")
		})

		Convey("should still work with simple references", func() {
			input := `
sep: ":"
items:
  - a
  - b
  - c
result: (( join sep items ))
`
			var data map[interface{}]interface{}
			err := yaml.Unmarshal([]byte(input), &data)
			So(err, ShouldBeNil)

			ev := &Evaluator{Tree: data}
			err = ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["result"], ShouldEqual, "a:b:c")
		})

		Convey("should error on invalid list entries", func() {
			input := `
items:
  - string
  - {key: value}
result: (( join "," items ))
`
			var data map[interface{}]interface{}
			err := yaml.Unmarshal([]byte(input), &data)
			So(err, ShouldBeNil)

			// Check the data structure - just verify it's a map
			items := data["items"].([]interface{})
			So(len(items), ShouldEqual, 2)
			// The second item should be a map (either map[string]interface{} or map[interface{}]interface{})
			switch items[1].(type) {
			case map[interface{}]interface{}, map[string]interface{}:
				// Good, it's a map
			default:
				t.Errorf("Expected map, got %T", items[1])
			}

			ev := &Evaluator{Tree: data}
			err = ev.RunPhase(EvalPhase)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "entry #1 in list is not compatible")
		})
	})
}
