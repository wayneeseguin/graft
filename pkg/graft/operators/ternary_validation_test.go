package operators

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/wayneeseguin/graft/pkg/graft"
)

func TestTernaryValidation(t *testing.T) {
	Convey("Ternary operator validation", t, func() {
		Convey("should return error for missing colon", func() {
			// This test ensures that the ternary parsing correctly validates
			// that both ? and : are present in a ternary expression

			// Test case: (( true ? "yes" )) - missing the : and false clause
			op, err := graft.ParseOpcallCompat(graft.EvalPhase, `(( true ? "yes" ))`)

			// Should return an error, not nil
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "expected ':' for ternary operator")
			So(op, ShouldBeNil)
		})

		Convey("should return error for missing question mark", func() {
			// Test case: (( true : "no" )) - missing the ? (not a valid ternary)
			op, _ := graft.ParseOpcallCompat(graft.EvalPhase, `(( true : "no" ))`)

			// This should fail to parse as it's not a valid operator syntax
			So(op, ShouldBeNil)
			// Note: This may return nil error as it's just not recognized as any operator
		})

		Convey("should parse valid ternary expressions correctly", func() {
			// Test case: (( true ? "yes" : "no" )) - valid ternary
			op, err := graft.ParseOpcallCompat(graft.EvalPhase, `(( true ? "yes" : "no" ))`)

			// Should parse successfully
			So(err, ShouldBeNil)
			So(op, ShouldNotBeNil)
			So(len(op.Args()), ShouldEqual, 3) // condition, true_value, false_value
		})
	})
}
