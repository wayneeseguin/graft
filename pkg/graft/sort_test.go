package graft_test

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/goutils/tree"
	"github.com/wayneeseguin/graft/pkg/graft"
	"github.com/wayneeseguin/graft/pkg/graft/operators"
)

func TestSort(t *testing.T) {
	// Disable ANSI colors for testing
	ansi.Color(false)
	
	Convey("that the sort operator returns the current value during evaluation", t, func() {
		op := &operators.SortOperator{}
		testData := map[interface{}]interface{}{"foobar": []interface{}{3, 1, 2}}
		ev := &graft.Evaluator{
			Here: &tree.Cursor{Nodes: []string{"foobar"}},
			Tree: testData,
		}

		resp, err := op.Run(ev, []*graft.Expr{})
		So(err, ShouldBeNil)
		So(resp, ShouldNotBeNil)
		So(resp.Type, ShouldEqual, graft.Replace)
		So(resp.Value, ShouldResemble, []interface{}{3, 1, 2}) // Value should be unchanged during evaluation
	})

	Convey("that sorting an empty list returns an empty list", t, func() {
		list := []interface{}{}
		err := graft.SortList("some.path", list, "")
		So(err, ShouldBeNil)
		So(list, ShouldResemble, []interface{}{})
	})

	Convey("that sorting of integers works", t, func() {
		list := []interface{}{2, 1}
		err := graft.SortList("some.path", list, "")
		So(err, ShouldBeNil)
		So(list, ShouldResemble, []interface{}{1, 2})
	})

	Convey("that sorting of floats works", t, func() {
		list := []interface{}{2.0, 1.0}
		err := graft.SortList("some.path", list, "")
		So(err, ShouldBeNil)
		So(list, ShouldResemble, []interface{}{1.0, 2.0})
	})

	Convey("that sorting of strings works", t, func() {
		list := []interface{}{"graft", "spiff"}
		err := graft.SortList("some.path", list, "")
		So(err, ShouldBeNil)
		So(list, ShouldResemble, []interface{}{"graft", "spiff"})
	})

	Convey("that sorting of named-entry lists works", t, func() {
		list := []interface{}{
			map[interface{}]interface{}{"name": "B"},
			map[interface{}]interface{}{"name": "A"},
		}
		err := graft.SortList("some.path", list, "")
		So(err, ShouldBeNil)
		So(list, ShouldResemble, []interface{}{
			map[interface{}]interface{}{"name": "A"},
			map[interface{}]interface{}{"name": "B"},
		})
	})

	Convey("that sorting of lists of lists fails", t, func() {
		list := []interface{}{
			[]interface{}{"B", "A"},
			[]interface{}{"A", "B"},
		}

		err := graft.SortList("some.path", list, "")
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldResemble, "$.some.path is a list with list entries (not a list with maps, strings or numbers)")
	})

	Convey("that sorting of a list of inhomogeneous types fails", t, func() {
		list := []interface{}{42, 42.0, "42"}
		err := graft.SortList("some.path", list, "")
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldResemble, "$.some.path is a list with different types (not a list with homogeneous entry types)")
	})

	Convey("that sorting of a list with nil values fails (by definition considered to be inhomogeneous types)", t, func() {
		list := []interface{}{"A", "B", "C", nil}
		err := graft.SortList("some.path", list, "")
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldResemble, "$.some.path is a list with different types (not a list with homogeneous entry types)")
	})

	Convey("that sorting of a named-entry list with inconsistent identifier fails", t, func() {
		list := []interface{}{
			map[interface{}]interface{}{"foo": "one"},
			map[interface{}]interface{}{"key": "two"},
		}
		err := graft.SortList("some.path", list, "foo")
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldResemble, "$.some.path is a list with map entries, where some do not contain foo (not a list with map entries each containing foo)")
	})
}
