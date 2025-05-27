package operators


func TestNegationOperator(t *testing.T) {
	Convey("Negation Operator (!)", t, func() {
		ev := &Evaluator{}
		op := NegationOperator{}
		
		Convey("negates boolean values", func() {
			resp, err := op.Run(ev, []*Expr{
				&Expr{Type: Literal, Literal: true},
			})
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, false)
			
			resp, err = op.Run(ev, []*Expr{
				&Expr{Type: Literal, Literal: false},
			})
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, true)
		})
		
		Convey("negates truthy/falsy values", func() {
			// Non-empty string is truthy, so !truthy = false
			resp, err := op.Run(ev, []*Expr{
				&Expr{Type: Literal, Literal: "hello"},
			})
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, false)
			
			// Empty string is falsy, so !falsy = true
			resp, err = op.Run(ev, []*Expr{
				&Expr{Type: Literal, Literal: ""},
			})
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, true)
			
			// Zero is falsy
			resp, err = op.Run(ev, []*Expr{
				&Expr{Type: Literal, Literal: int64(0)},
			})
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, true)
			
			// Non-zero is truthy
			resp, err = op.Run(ev, []*Expr{
				&Expr{Type: Literal, Literal: int64(42)},
			})
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, false)
		})
		
		Convey("requires exactly 1 argument", func() {
			_, err := op.Run(ev, []*Expr{})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "1 argument")
			
			_, err = op.Run(ev, []*Expr{
				&Expr{Type: Literal, Literal: true},
				&Expr{Type: Literal, Literal: false},
			})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "1 argument")
		})
	})
}
