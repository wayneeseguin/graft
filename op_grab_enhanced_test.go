package graft

import (
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/yaml.v3"
)

func TestEnhancedGrabOperator(t *testing.T) {
	Convey("Enhanced Grab Operator", t, func() {
		// Enable enhanced parser for these tests
		oldUseEnhanced := UseEnhancedParser
		UseEnhancedParser = true
		EnableEnhancedGrab()
		defer func() {
			UseEnhancedParser = oldUseEnhanced
			if !oldUseEnhanced {
				RegisterOp("grab", GrabOperator{})
			}
		}()

		Convey("should support nested concat expressions", func() {
			input := `
meta:
  env: production
  region: us-east-1
config:
  production:
    host: prod.example.com
  staging:
    host: stage.example.com
result: (( grab (concat "config." meta.env ".host") ))
`
			var data map[interface{}]interface{}
			err := yaml.Unmarshal([]byte(input), &data)
			So(err, ShouldBeNil)

			ev := &Evaluator{Tree: data}
			err = ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["result"], ShouldEqual, "prod.example.com")
		})

		Convey("should support nested grab expressions", func() {
			input := `
meta:
  key: "data.services"
data:
  services:
    - web
    - api
    - db
result: (( grab (grab meta.key) ))
`
			var data map[interface{}]interface{}
			err := yaml.Unmarshal([]byte(input), &data)
			So(err, ShouldBeNil)

			ev := &Evaluator{Tree: data}
			err = ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			
			result := ev.Tree["result"].([]interface{})
			So(len(result), ShouldEqual, 3)
			So(result[0], ShouldEqual, "web")
			So(result[1], ShouldEqual, "api")
			So(result[2], ShouldEqual, "db")
		})

		Convey("should support multiple nested expressions", func() {
			input := `
meta:
  prefix: "config"
  suffix: "host"
  env: prod
config:
  prod:
    host: production.example.com
  dev:
    host: development.example.com
result: (( grab (concat meta.prefix "." meta.env "." meta.suffix) ))
`
			var data map[interface{}]interface{}
			err := yaml.Unmarshal([]byte(input), &data)
			So(err, ShouldBeNil)

			ev := &Evaluator{Tree: data}
			err = ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["result"], ShouldEqual, "production.example.com")
		})

		Convey("should support environment variables in nested expressions", func() {
			os.Setenv("GRAFT_ENV", "staging")
			defer os.Unsetenv("GRAFT_ENV")

			input := `
config:
  production:
    host: prod.example.com
  staging:
    host: stage.example.com
result: (( grab (concat "config." $GRAFT_ENV ".host") ))
`
			var data map[interface{}]interface{}
			err := yaml.Unmarshal([]byte(input), &data)
			So(err, ShouldBeNil)

			ev := &Evaluator{Tree: data}
			err = ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["result"], ShouldEqual, "stage.example.com")
		})

		Convey("should handle invalid paths gracefully", func() {
			input := `
meta:
  key: "nonexistent"
result: (( grab (concat "config." meta.key) ))
`
			var data map[interface{}]interface{}
			err := yaml.Unmarshal([]byte(input), &data)
			So(err, ShouldBeNil)

			ev := &Evaluator{Tree: data}
			err = ev.RunPhase(EvalPhase)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Unable to resolve")
		})

		Convey("should still work with simple references", func() {
			input := `
data:
  value: 42
result: (( grab data.value ))
`
			var data map[interface{}]interface{}
			err := yaml.Unmarshal([]byte(input), &data)
			So(err, ShouldBeNil)

			ev := &Evaluator{Tree: data}
			err = ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["result"], ShouldEqual, 42)
		})

		Convey("should flatten lists from multiple nested grabs", func() {
			input := `
meta:
  keys:
    - "list1"
    - "list2"
data:
  list1:
    - a
    - b
  list2:
    - c
    - d
result: (( grab data.list1 data.list2 ))
`
			var data map[interface{}]interface{}
			err := yaml.Unmarshal([]byte(input), &data)
			So(err, ShouldBeNil)

			ev := &Evaluator{Tree: data}
			err = ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			
			result := ev.Tree["result"].([]interface{})
			So(len(result), ShouldEqual, 4)
			So(result[0], ShouldEqual, "a")
			So(result[1], ShouldEqual, "b")
			So(result[2], ShouldEqual, "c")
			So(result[3], ShouldEqual, "d")
		})
	})
}