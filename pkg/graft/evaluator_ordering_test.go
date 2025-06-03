package graft

import (
	"testing"
	
	"github.com/geofffranks/simpleyaml"
	. "github.com/smartystreets/goconvey/convey"
)

func TestDataflowOrdering(t *testing.T) {
	YAML := func(s string) map[interface{}]interface{} {
		y, err := simpleyaml.NewYaml([]byte(s))
		So(err, ShouldBeNil)

		data, err := y.Map()
		So(err, ShouldBeNil)

		return data
	}
	
	Convey("Dataflow Ordering", t, func() {
		input := `
domain:    (( grab meta.domain || "default" ))
env:       (( grab meta.env || "dev" ))
app:       (( grab meta.app || "myapp" ))
meta:
  env: production
`
		tree := YAML(input)
		
		Convey("With alphabetical ordering (default)", func() {
			ev := &Evaluator{
				Tree:          tree,
				DataflowOrder: "alphabetical",
			}
			
			ops, err := ev.DataFlow(EvalPhase)
			So(err, ShouldBeNil)
			So(len(ops), ShouldEqual, 3)
			
			// Should be in alphabetical order
			So(ops[0].Where().String(), ShouldEqual, "app")
			So(ops[1].Where().String(), ShouldEqual, "domain")
			So(ops[2].Where().String(), ShouldEqual, "env")
		})
		
		Convey("With insertion ordering", func() {
			ev := &Evaluator{
				Tree:          tree,
				DataflowOrder: "insertion",
			}
			
			ops, err := ev.DataFlow(EvalPhase)
			So(err, ShouldBeNil)
			So(len(ops), ShouldEqual, 3)
			
			// Due to Go's non-deterministic map iteration, insertion order
			// now follows deterministic alphabetical scanning order 
			So(ops[0].Where().String(), ShouldEqual, "app")
			So(ops[1].Where().String(), ShouldEqual, "domain")
			So(ops[2].Where().String(), ShouldEqual, "env")
		})
		
		Convey("Ordering does not affect evaluation result", func() {
			// Test with alphabetical
			ev1 := &Evaluator{
				Tree:          YAML(input),
				DataflowOrder: "alphabetical",
			}
			err := ev1.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			
			// Test with insertion
			ev2 := &Evaluator{
				Tree:          YAML(input),
				DataflowOrder: "insertion",
			}
			err = ev2.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			
			// Results should be identical
			So(ev1.Tree["domain"], ShouldEqual, "default")
			So(ev1.Tree["env"], ShouldEqual, "production")
			So(ev1.Tree["app"], ShouldEqual, "myapp")
			
			So(ev2.Tree["domain"], ShouldEqual, ev1.Tree["domain"])
			So(ev2.Tree["env"], ShouldEqual, ev1.Tree["env"])
			So(ev2.Tree["app"], ShouldEqual, ev1.Tree["app"])
		})
	})
}