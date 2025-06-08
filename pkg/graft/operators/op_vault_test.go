package operators

import (
	"testing"

	"github.com/wayneeseguin/graft/internal/utils/tree"
	. "github.com/smartystreets/goconvey/convey"
)

func TestEnhancedVaultOperator(t *testing.T) {
	Convey("Enhanced Vault Operator", t, func() {

		Convey("Multiple paths with semicolon syntax", func() {
			yamlTree := map[interface{}]interface{}{
				"env": "production",
			}
			ev := &Evaluator{Tree: yamlTree}

			Convey("splitVaultPaths", func() {
				processor := &vaultArgProcessor{}

				Convey("returns single path when no semicolon", func() {
					paths := processor.splitVaultPaths("secret/path:key")
					So(len(paths), ShouldEqual, 1)
					So(paths[0], ShouldEqual, "secret/path:key")
				})

				Convey("splits multiple paths by semicolon", func() {
					paths := processor.splitVaultPaths("secret/prod:key; secret/dev:key; secret/default:key")
					So(len(paths), ShouldEqual, 3)
					So(paths[0], ShouldEqual, "secret/prod:key")
					So(paths[1], ShouldEqual, "secret/dev:key")
					So(paths[2], ShouldEqual, "secret/default:key")
				})

				Convey("trims whitespace around paths", func() {
					paths := processor.splitVaultPaths("  secret/a:key  ;  secret/b:key  ")
					So(len(paths), ShouldEqual, 2)
					So(paths[0], ShouldEqual, "secret/a:key")
					So(paths[1], ShouldEqual, "secret/b:key")
				})

				Convey("ignores empty segments", func() {
					paths := processor.splitVaultPaths("secret/a:key;;secret/b:key;")
					So(len(paths), ShouldEqual, 2)
					So(paths[0], ShouldEqual, "secret/a:key")
					So(paths[1], ShouldEqual, "secret/b:key")
				})
			})

			Convey("buildVaultPaths with semicolon syntax", func() {
				Convey("builds single path from concatenation", func() {
					envCursor, _ := tree.ParseCursor("env")
					processor := &vaultArgProcessor{
						args: []*Expr{
							{Type: Literal, Literal: "secret/"},
							{Type: Reference, Reference: envCursor},
							{Type: Literal, Literal: "/db:password"},
						},
					}

					paths, err := processor.buildVaultPaths(ev)
					So(err, ShouldBeNil)
					So(len(paths), ShouldEqual, 1)
					So(paths[0], ShouldEqual, "secret/production/db:password")
				})

				Convey("builds multiple paths from semicolon-separated string", func() {
					processor := &vaultArgProcessor{
						args: []*Expr{
							{Type: Literal, Literal: "secret/prod:key; secret/dev:key"},
						},
					}

					paths, err := processor.buildVaultPaths(ev)
					So(err, ShouldBeNil)
					So(len(paths), ShouldEqual, 2)
					So(paths[0], ShouldEqual, "secret/prod:key")
					So(paths[1], ShouldEqual, "secret/dev:key")
				})

				Convey("builds multiple paths with concatenation and semicolons", func() {
					envCursor, _ := tree.ParseCursor("env")
					processor := &vaultArgProcessor{
						args: []*Expr{
							{Type: Literal, Literal: "secret/"},
							{Type: Reference, Reference: envCursor},
							{Type: Literal, Literal: ":pass; secret/fallback:pass"},
						},
					}

					paths, err := processor.buildVaultPaths(ev)
					So(err, ShouldBeNil)
					So(len(paths), ShouldEqual, 2)
					So(paths[0], ShouldEqual, "secret/production:pass")
					So(paths[1], ShouldEqual, "secret/fallback:pass")
				})
			})
		})

		Convey("Multiple arguments mode (vault-try style)", func() {
			yamlTree := map[interface{}]interface{}{}
			ev := &Evaluator{Tree: yamlTree}

			Convey("detectMultiplePathArgs", func() {
				Convey("detects multiple vault path arguments", func() {
					processor := &vaultArgProcessor{
						args: []*Expr{
							{Type: Literal, Literal: "secret/prod:password"},
							{Type: Literal, Literal: "secret/dev:password"},
							{Type: Literal, Literal: "default-password"},
						},
					}

					So(processor.detectMultiplePathArgs(ev), ShouldBeTrue)
				})

				Convey("returns false with LogicalOr present", func() {
					processor := &vaultArgProcessor{
						args: []*Expr{
							{Type: Literal, Literal: "secret/prod:password"},
							{Type: Literal, Literal: "secret/dev:password"},
						},
						hasDefault: true,
					}

					So(processor.detectMultiplePathArgs(ev), ShouldBeFalse)
				})

				Convey("returns false with single argument", func() {
					processor := &vaultArgProcessor{
						args: []*Expr{
							{Type: Literal, Literal: "secret/prod:password"},
						},
					}

					So(processor.detectMultiplePathArgs(ev), ShouldBeFalse)
				})
			})

			Convey("buildVaultPaths with multiple arguments", func() {
				Convey("treats each argument as separate path", func() {
					processor := &vaultArgProcessor{
						args: []*Expr{
							{Type: Literal, Literal: "secret/prod:password"},
							{Type: Literal, Literal: "secret/dev:password"},
							{Type: Literal, Literal: "secret/staging:password"},
						},
					}

					paths, err := processor.buildVaultPaths(ev)
					So(err, ShouldBeNil)
					So(len(paths), ShouldEqual, 3)
					So(paths[0], ShouldEqual, "secret/prod:password")
					So(paths[1], ShouldEqual, "secret/dev:password")
					So(paths[2], ShouldEqual, "secret/staging:password")
				})

				Convey("identifies non-path last argument as default", func() {
					processor := &vaultArgProcessor{
						args: []*Expr{
							{Type: Literal, Literal: "secret/prod:password"},
							{Type: Literal, Literal: "secret/dev:password"},
							{Type: Literal, Literal: "default-password"},
						},
					}

					paths, err := processor.buildVaultPaths(ev)
					So(err, ShouldBeNil)
					So(len(paths), ShouldEqual, 2)
					So(paths[0], ShouldEqual, "secret/prod:password")
					So(paths[1], ShouldEqual, "secret/dev:password")
					So(processor.hasDefault, ShouldBeTrue)
					So(processor.defaultExpr.Literal, ShouldEqual, "default-password")
				})

				Convey("handles LogicalOr with multiple args", func() {
					processor := &vaultArgProcessor{
						args: []*Expr{
							{Type: Literal, Literal: "secret/prod:password"},
							{Type: Literal, Literal: "secret/dev:password"},
						},
						hasDefault:  true,
						defaultExpr: &Expr{Type: Literal, Literal: "logical-or-default"},
					}

					paths, err := processor.buildVaultPaths(ev)
					So(err, ShouldBeNil)
					So(len(paths), ShouldEqual, 1) // Falls back to concatenation mode
					So(paths[0], ShouldEqual, "secret/prod:passwordsecret/dev:password")
				})
			})
		})

		Convey("isVaultPathString", func() {
			ev := &Evaluator{Tree: map[interface{}]interface{}{}}

			Convey("identifies vault path strings", func() {
				expr := &Expr{Type: Literal, Literal: "secret/path:key"}
				So(isVaultPathString(ev, expr), ShouldBeTrue)
			})

			Convey("rejects non-path strings", func() {
				expr := &Expr{Type: Literal, Literal: "default-value"}
				So(isVaultPathString(ev, expr), ShouldBeFalse)
			})

			Convey("rejects non-literals", func() {
				expr := &Expr{Type: Reference, Reference: &tree.Cursor{}}
				So(isVaultPathString(ev, expr), ShouldBeFalse)
			})

			Convey("rejects non-string literals", func() {
				expr := &Expr{Type: Literal, Literal: 42}
				So(isVaultPathString(ev, expr), ShouldBeFalse)
			})
		})

		Convey("Combined syntax scenarios", func() {
			yamlTree := map[interface{}]interface{}{
				"env":  "prod",
				"team": "platform",
			}
			ev := &Evaluator{Tree: yamlTree}

			Convey("semicolon paths with LogicalOr default", func() {
				processor := &vaultArgProcessor{
					args: []*Expr{
						{Type: Literal, Literal: "secret/prod:key; secret/dev:key"},
					},
					hasDefault:  true,
					defaultExpr: &Expr{Type: Literal, Literal: "fallback"},
				}

				paths, err := processor.buildVaultPaths(ev)
				So(err, ShouldBeNil)
				So(len(paths), ShouldEqual, 2)
				So(processor.hasDefault, ShouldBeTrue)
			})

			Convey("concatenation with semicolons and LogicalOr", func() {
				envCursor, _ := tree.ParseCursor("env")
				teamCursor, _ := tree.ParseCursor("team")

				processor := &vaultArgProcessor{
					args: []*Expr{
						{Type: Literal, Literal: "secret/"},
						{Type: Reference, Reference: envCursor},
						{Type: Literal, Literal: "/"},
						{Type: Reference, Reference: teamCursor},
						{Type: Literal, Literal: ":key; secret/shared:key"},
					},
					hasDefault:  true,
					defaultExpr: &Expr{Type: Literal, Literal: "default"},
				}

				paths, err := processor.buildVaultPaths(ev)
				So(err, ShouldBeNil)
				So(len(paths), ShouldEqual, 2)
				So(paths[0], ShouldEqual, "secret/prod/platform:key")
				So(paths[1], ShouldEqual, "secret/shared:key")
				So(processor.hasDefault, ShouldBeTrue)
			})
		})
	})
}