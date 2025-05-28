package operators

import (
	"fmt"
	"testing"

	"github.com/starkandwayne/goutils/tree"
	. "github.com/smartystreets/goconvey/convey"
)

func TestVaultArgProcessor(t *testing.T) {
	Convey("VaultArgProcessor", t, func() {

		Convey("newVaultArgProcessor", func() {
			Convey("handles simple arguments without LogicalOr", func() {
				args := []*Expr{
					{Type: Literal, Literal: "secret/path"},
					{Type: Literal, Literal: ":key"},
				}

				processor := newVaultArgProcessor(args)
				So(processor.hasDefault, ShouldBeFalse)
				So(processor.defaultExpr, ShouldBeNil)
				So(len(processor.args), ShouldEqual, 2)
			})

			Convey("extracts LogicalOr from last position", func() {
				args := []*Expr{
					{Type: Literal, Literal: "secret/"},
					{Type: Reference, Reference: &tree.Cursor{}},
					{Type: LogicalOr,
						Left:  &Expr{Type: Literal, Literal: ":key"},
						Right: &Expr{Type: Literal, Literal: "default"},
					},
				}

				processor := newVaultArgProcessor(args)
				So(processor.hasDefault, ShouldBeTrue)
				So(processor.defaultExpr, ShouldNotBeNil)
				So(processor.defaultExpr.Literal, ShouldEqual, "default")
				So(processor.defaultIndex, ShouldEqual, 2)
				So(processor.args[2].Literal, ShouldEqual, ":key")
			})

			Convey("extracts LogicalOr from middle position", func() {
				args := []*Expr{
					{Type: Literal, Literal: "secret/"},
					{Type: LogicalOr,
						Left:  &Expr{Type: Literal, Literal: "prod"},
						Right: &Expr{Type: Literal, Literal: "dev"},
					},
					{Type: Literal, Literal: ":key"},
				}

				processor := newVaultArgProcessor(args)
				So(processor.hasDefault, ShouldBeTrue)
				So(processor.defaultIndex, ShouldEqual, 1)
				So(processor.args[1].Literal, ShouldEqual, "prod")
			})
		})

		Convey("resolveToString", func() {
			processor := &vaultArgProcessor{}
			yamlTree := map[interface{}]interface{}{
				"env":  "production",
				"port": 5432,
				"config": map[interface{}]interface{}{
					"nested": "value",
				},
				"list": []interface{}{"a", "b"},
			}
			ev := &Evaluator{Tree: yamlTree}

			Convey("handles literals", func() {
				expr := &Expr{Type: Literal, Literal: "test"}
				result, err := processor.resolveToString(ev, expr)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "test")
			})

			Convey("handles nil literal", func() {
				expr := &Expr{Type: Literal, Literal: nil}
				_, err := processor.resolveToString(ev, expr)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "cannot use nil")
			})

			Convey("handles reference to string", func() {
				cursor, _ := tree.ParseCursor("env")
				resolved := &Expr{Type: Reference, Reference: cursor}
				result, err := processor.resolveToString(ev, resolved)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "production")
			})

			Convey("handles reference to number", func() {
				cursor, _ := tree.ParseCursor("port")
				resolved := &Expr{Type: Reference, Reference: cursor}
				result, err := processor.resolveToString(ev, resolved)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "5432")
			})

			Convey("rejects reference to map", func() {
				cursor, _ := tree.ParseCursor("config")
				resolved := &Expr{Type: Reference, Reference: cursor}
				_, err := processor.resolveToString(ev, resolved)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "is a map")
			})

			Convey("rejects reference to list", func() {
				cursor, _ := tree.ParseCursor("list")
				resolved := &Expr{Type: Reference, Reference: cursor}
				_, err := processor.resolveToString(ev, resolved)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "is a list")
			})
		})

		Convey("buildVaultPath", func() {
			yamlTree := map[interface{}]interface{}{
				"env": "prod",
				"app": "myapp",
			}
			ev := &Evaluator{Tree: yamlTree}

			Convey("concatenates simple literals", func() {
				processor := &vaultArgProcessor{
					args: []*Expr{
						{Type: Literal, Literal: "secret/"},
						{Type: Literal, Literal: "path"},
						{Type: Literal, Literal: ":key"},
					},
				}

				path, err := processor.buildVaultPath(ev)
				So(err, ShouldBeNil)
				So(path, ShouldEqual, "secret/path:key")
			})

			Convey("resolves and concatenates mixed types", func() {
				envCursor, _ := tree.ParseCursor("env")
				appCursor, _ := tree.ParseCursor("app")

				processor := &vaultArgProcessor{
					args: []*Expr{
						{Type: Literal, Literal: "secret/"},
						{Type: Reference, Reference: envCursor},
						{Type: Literal, Literal: "/"},
						{Type: Reference, Reference: appCursor},
						{Type: Literal, Literal: ":password"},
					},
				}

				path, err := processor.buildVaultPath(ev)
				So(err, ShouldBeNil)
				So(path, ShouldEqual, "secret/prod/myapp:password")
			})
		})

		Convey("isVaultNotFound", func() {
			Convey("identifies not found errors", func() {
				So(isVaultNotFound(fmt.Errorf("secret not found")), ShouldBeTrue)
				So(isVaultNotFound(fmt.Errorf("404 Not Found")), ShouldBeTrue)
				So(isVaultNotFound(fmt.Errorf("secret secret/path:key not found")), ShouldBeTrue)
			})

			Convey("rejects other errors", func() {
				So(isVaultNotFound(fmt.Errorf("permission denied")), ShouldBeFalse)
				So(isVaultNotFound(fmt.Errorf("network timeout")), ShouldBeFalse)
				So(isVaultNotFound(nil), ShouldBeFalse)
			})
		})
	})
}
