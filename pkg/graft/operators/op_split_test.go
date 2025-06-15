package operators

import (
	"testing"

	"github.com/geofffranks/simpleyaml"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/wayneeseguin/graft/internal/utils/tree"
)

func TestSplitOperator(t *testing.T) {
	YAML := func(s string) map[interface{}]interface{} {
		y, err := simpleyaml.NewYaml([]byte(s))
		So(err, ShouldBeNil)

		data, err := y.Map()
		So(err, ShouldBeNil)

		return data
	}

	Convey("Split Operator", t, func() {
		ev := &Evaluator{Tree: YAML(`{}`)}
		op := SplitOperator{}

		Convey("should split strings with comma delimiter", func() {
			resp, err := op.Run(ev, []*Expr{
				{Type: Literal, Literal: ","},
				{Type: Literal, Literal: "sg-1,sg-2,sg-3"},
			})
			So(err, ShouldBeNil)
			So(resp.Type, ShouldEqual, Replace)
			So(resp.Value, ShouldResemble, []interface{}{"sg-1", "sg-2", "sg-3"})
		})

		Convey("should split strings with custom delimiter", func() {
			resp, err := op.Run(ev, []*Expr{
				{Type: Literal, Literal: ":"},
				{Type: Literal, Literal: "host:port:path"},
			})
			So(err, ShouldBeNil)
			So(resp.Type, ShouldEqual, Replace)
			So(resp.Value, ShouldResemble, []interface{}{"host", "port", "path"})
		})

		Convey("should split strings with space delimiter", func() {
			resp, err := op.Run(ev, []*Expr{
				{Type: Literal, Literal: " "},
				{Type: Literal, Literal: "hello world test"},
			})
			So(err, ShouldBeNil)
			So(resp.Type, ShouldEqual, Replace)
			So(resp.Value, ShouldResemble, []interface{}{"hello", "world", "test"})
		})

		Convey("should handle empty string delimiter (split into characters)", func() {
			resp, err := op.Run(ev, []*Expr{
				{Type: Literal, Literal: ""},
				{Type: Literal, Literal: "abc"},
			})
			So(err, ShouldBeNil)
			So(resp.Type, ShouldEqual, Replace)
			So(resp.Value, ShouldResemble, []interface{}{"a", "b", "c"})
		})

		Convey("should handle single element (no delimiter found)", func() {
			resp, err := op.Run(ev, []*Expr{
				{Type: Literal, Literal: ","},
				{Type: Literal, Literal: "sg-1"},
			})
			So(err, ShouldBeNil)
			So(resp.Type, ShouldEqual, Replace)
			So(resp.Value, ShouldResemble, []interface{}{"sg-1"})
		})

		Convey("should handle empty string input", func() {
			resp, err := op.Run(ev, []*Expr{
				{Type: Literal, Literal: ","},
				{Type: Literal, Literal: ""},
			})
			So(err, ShouldBeNil)
			So(resp.Type, ShouldEqual, Replace)
			So(resp.Value, ShouldResemble, []interface{}{""})
		})

		Convey("should handle consecutive delimiters", func() {
			resp, err := op.Run(ev, []*Expr{
				{Type: Literal, Literal: ","},
				{Type: Literal, Literal: "a,,b,c"},
			})
			So(err, ShouldBeNil)
			So(resp.Type, ShouldEqual, Replace)
			So(resp.Value, ShouldResemble, []interface{}{"a", "", "b", "c"})
		})

		Convey("should split references", func() {
			ev.Tree = YAML(`
meta:
  default_sgs_list: "sg-1,sg-2,sg-3"
`)
			cursor, err := tree.ParseCursor("meta.default_sgs_list")
			So(err, ShouldBeNil)

			resp, err := op.Run(ev, []*Expr{
				{Type: Literal, Literal: ","},
				{Type: Reference, Reference: cursor},
			})
			So(err, ShouldBeNil)
			So(resp.Type, ShouldEqual, Replace)
			So(resp.Value, ShouldResemble, []interface{}{"sg-1", "sg-2", "sg-3"})
		})

		Convey("should convert non-string values to strings before splitting", func() {
			resp, err := op.Run(ev, []*Expr{
				{Type: Literal, Literal: "."},
				{Type: Literal, Literal: 123.456},
			})
			So(err, ShouldBeNil)
			So(resp.Type, ShouldEqual, Replace)
			So(resp.Value, ShouldResemble, []interface{}{"123", "456"})
		})

		Convey("should handle multi-character delimiters", func() {
			resp, err := op.Run(ev, []*Expr{
				{Type: Literal, Literal: "::"},
				{Type: Literal, Literal: "a::b::c"},
			})
			So(err, ShouldBeNil)
			So(resp.Type, ShouldEqual, Replace)
			So(resp.Value, ShouldResemble, []interface{}{"a", "b", "c"})
		})

		Convey("error handling", func() {
			Convey("should error with no arguments", func() {
				_, err := op.Run(ev, []*Expr{})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "no arguments specified")
			})

			Convey("should error with only one argument", func() {
				_, err := op.Run(ev, []*Expr{
					{Type: Literal, Literal: ","},
				})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "too few arguments")
			})

			Convey("should error with too many arguments", func() {
				_, err := op.Run(ev, []*Expr{
					{Type: Literal, Literal: ","},
					{Type: Literal, Literal: "a,b"},
					{Type: Literal, Literal: "extra"},
				})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "too many arguments")
			})

			Convey("should error if separator is not literal", func() {
				cursor, _ := tree.ParseCursor("some.path")
				_, err := op.Run(ev, []*Expr{
					{Type: Reference, Reference: cursor},
					{Type: Literal, Literal: "a,b"},
				})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "only accepts literal argument for the separator")
			})

			Convey("should error if value to split is nil", func() {
				ev.Tree = YAML(`
meta:
  nonexistent: ~
`)
				cursor, err := tree.ParseCursor("meta.nonexistent")
				So(err, ShouldBeNil)

				_, err = op.Run(ev, []*Expr{
					{Type: Literal, Literal: ","},
					{Type: Reference, Reference: cursor},
				})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "cannot split nil value")
			})

			Convey("should error if reference cannot be resolved", func() {
				cursor, err := tree.ParseCursor("does.not.exist")
				So(err, ShouldBeNil)

				_, err = op.Run(ev, []*Expr{
					{Type: Literal, Literal: ","},
					{Type: Reference, Reference: cursor},
				})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "Unable to resolve")
			})
		})

		Convey("Dependencies", func() {
			Convey("should return auto dependencies", func() {
				cursor1, _ := tree.ParseCursor("auto.dep1")
				cursor2, _ := tree.ParseCursor("auto.dep2")
				auto := []*tree.Cursor{cursor1, cursor2}

				deps := op.Dependencies(ev, []*Expr{}, nil, auto)
				So(len(deps), ShouldEqual, 2)
				So(deps[0], ShouldEqual, cursor1)
				So(deps[1], ShouldEqual, cursor2)
			})

			Convey("should include reference dependencies", func() {
				cursor, _ := tree.ParseCursor("meta.some_list")
				args := []*Expr{
					{Type: Literal, Literal: ","},
					{Type: Reference, Reference: cursor},
				}

				deps := op.Dependencies(ev, args, nil, nil)
				So(len(deps), ShouldEqual, 1)
				So(deps[0], ShouldEqual, cursor)
			})
		})

		Convey("Regex delimiter support", func() {
			Convey("should split with simple regex patterns", func() {
				// Split on comma or space
				resp, err := op.Run(ev, []*Expr{
					{Type: Literal, Literal: "/,| "},
					{Type: Literal, Literal: "a,b c,d e"},
				})
				So(err, ShouldBeNil)
				So(resp.Type, ShouldEqual, Replace)
				So(resp.Value, ShouldResemble, []interface{}{"a", "b", "c", "d", "e"})
			})

			Convey("should split with comma and optional spaces", func() {
				// Split on comma with optional spaces
				resp, err := op.Run(ev, []*Expr{
					{Type: Literal, Literal: "/, *"},
					{Type: Literal, Literal: "a,b, c,  d,   e"},
				})
				So(err, ShouldBeNil)
				So(resp.Type, ShouldEqual, Replace)
				So(resp.Value, ShouldResemble, []interface{}{"a", "b", "c", "d", "e"})
			})

			Convey("should split with character classes", func() {
				// Split on any whitespace
				resp, err := op.Run(ev, []*Expr{
					{Type: Literal, Literal: "/\\s+"},
					{Type: Literal, Literal: "a b\tc\nd\r\ne"},
				})
				So(err, ShouldBeNil)
				So(resp.Type, ShouldEqual, Replace)
				So(resp.Value, ShouldResemble, []interface{}{"a", "b", "c", "d", "e"})
			})

			Convey("should split with alternation", func() {
				// Split on semicolon or pipe
				resp, err := op.Run(ev, []*Expr{
					{Type: Literal, Literal: "/;|\\|"},
					{Type: Literal, Literal: "a;b|c;d|e"},
				})
				So(err, ShouldBeNil)
				So(resp.Type, ShouldEqual, Replace)
				So(resp.Value, ShouldResemble, []interface{}{"a", "b", "c", "d", "e"})
			})

			Convey("should split with complex patterns", func() {
				// Split on various delimiters
				resp, err := op.Run(ev, []*Expr{
					{Type: Literal, Literal: "/[,;:|]+"},
					{Type: Literal, Literal: "a,b;c:d|e,,f::g"},
				})
				So(err, ShouldBeNil)
				So(resp.Type, ShouldEqual, Replace)
				So(resp.Value, ShouldResemble, []interface{}{"a", "b", "c", "d", "e", "f", "g"})
			})

			Convey("should handle word boundaries", func() {
				// Split on word boundaries
				resp, err := op.Run(ev, []*Expr{
					{Type: Literal, Literal: "/\\b"},
					{Type: Literal, Literal: "hello-world test"},
				})
				So(err, ShouldBeNil)
				So(resp.Type, ShouldEqual, Replace)
				// PCRE word boundaries create empty strings at start/end
				expected := []interface{}{"", "hello", "-", "world", " ", "test", ""}
				So(resp.Value, ShouldResemble, expected)
			})

			Convey("should handle numeric patterns", func() {
				// Split on sequences of digits
				resp, err := op.Run(ev, []*Expr{
					{Type: Literal, Literal: "/\\d+"},
					{Type: Literal, Literal: "abc123def456ghi"},
				})
				So(err, ShouldBeNil)
				So(resp.Type, ShouldEqual, Replace)
				So(resp.Value, ShouldResemble, []interface{}{"abc", "def", "ghi"})
			})

			Convey("should preserve empty strings from regex split", func() {
				// Split that creates empty strings
				resp, err := op.Run(ev, []*Expr{
					{Type: Literal, Literal: "/,+"},
					{Type: Literal, Literal: "a,,b,,,c"},
				})
				So(err, ShouldBeNil)
				So(resp.Type, ShouldEqual, Replace)
				So(resp.Value, ShouldResemble, []interface{}{"a", "b", "c"})
			})

			Convey("should handle regex with references", func() {
				ev.Tree = YAML(`
meta:
  mixed_delims: "a,b;c:d|e"
`)
				cursor, err := tree.ParseCursor("meta.mixed_delims")
				So(err, ShouldBeNil)

				resp, err := op.Run(ev, []*Expr{
					{Type: Literal, Literal: "/[,;:|]"},
					{Type: Reference, Reference: cursor},
				})
				So(err, ShouldBeNil)
				So(resp.Type, ShouldEqual, Replace)
				So(resp.Value, ShouldResemble, []interface{}{"a", "b", "c", "d", "e"})
			})

			Convey("error handling for regex", func() {
				Convey("should error on invalid regex", func() {
					_, err := op.Run(ev, []*Expr{
						{Type: Literal, Literal: "/["}, // Unclosed character class
						{Type: Literal, Literal: "test"},
					})
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "invalid PCRE regex pattern")
				})

				Convey("regex with special path pattern", func() {
					// Now /abc is treated as a regex pattern "abc"
					resp, err := op.Run(ev, []*Expr{
						{Type: Literal, Literal: "/abc"},
						{Type: Literal, Literal: "testabctest"},
					})
					So(err, ShouldBeNil)
					So(resp.Value, ShouldResemble, []interface{}{"test", "test"})
				})
			})

			Convey("backward compatibility", func() {
				Convey("literal separator not starting with /", func() {
					// Normal literal path separator
					resp, err := op.Run(ev, []*Expr{
						{Type: Literal, Literal: "path"},
						{Type: Literal, Literal: "/path/to/file"},
					})
					So(err, ShouldBeNil)
					So(resp.Type, ShouldEqual, Replace)
					So(resp.Value, ShouldResemble, []interface{}{"/", "/to/file"})
				})

				Convey("strings with regex metacharacters work as literals", func() {
					resp, err := op.Run(ev, []*Expr{
						{Type: Literal, Literal: ".*"},
						{Type: Literal, Literal: "file.*txt"},
					})
					So(err, ShouldBeNil)
					So(resp.Type, ShouldEqual, Replace)
					So(resp.Value, ShouldResemble, []interface{}{"file", "txt"})
				})
			})

			Convey("PCRE-specific features", func() {
				Convey("should support lookahead assertions", func() {
					// Split before uppercase letters (positive lookahead)
					resp, err := op.Run(ev, []*Expr{
						{Type: Literal, Literal: "/(?=[A-Z])"},
						{Type: Literal, Literal: "camelCaseString"},
					})
					So(err, ShouldBeNil)
					So(resp.Type, ShouldEqual, Replace)
					So(resp.Value, ShouldResemble, []interface{}{"camel", "Case", "String"})
				})

				Convey("should support lookbehind assertions", func() {
					// Split after sequence of digits (positive lookbehind)
					resp, err := op.Run(ev, []*Expr{
						{Type: Literal, Literal: "/(?<=\\d)(?=\\D)"},
						{Type: Literal, Literal: "abc123def456ghi"},
					})
					So(err, ShouldBeNil)
					So(resp.Type, ShouldEqual, Replace)
					So(resp.Value, ShouldResemble, []interface{}{"abc123", "def456", "ghi"})
				})

				Convey("should support negative lookahead", func() {
					// Split on comma not followed by space
					resp, err := op.Run(ev, []*Expr{
						{Type: Literal, Literal: "/,(?! )"},
						{Type: Literal, Literal: "a,b, c,d, e"},
					})
					So(err, ShouldBeNil)
					So(resp.Type, ShouldEqual, Replace)
					So(resp.Value, ShouldResemble, []interface{}{"a", "b, c", "d, e"})
				})

				Convey("should support non-capturing groups", func() {
					// Split on various delimiters using non-capturing group
					resp, err := op.Run(ev, []*Expr{
						{Type: Literal, Literal: "/(?:,|;|:)"},
						{Type: Literal, Literal: "a,b;c:d"},
					})
					So(err, ShouldBeNil)
					So(resp.Type, ShouldEqual, Replace)
					So(resp.Value, ShouldResemble, []interface{}{"a", "b", "c", "d"})
				})

				Convey("should support multiple quantifiers", func() {
					// Split on multiple spaces
					resp, err := op.Run(ev, []*Expr{
						{Type: Literal, Literal: "/ +"},
						{Type: Literal, Literal: "a  b   c    d"},
					})
					So(err, ShouldBeNil)
					So(resp.Type, ShouldEqual, Replace)
					So(resp.Value, ShouldResemble, []interface{}{"a", "b", "c", "d"})
				})

				Convey("should support atomic groups", func() {
					// Split using atomic group
					resp, err := op.Run(ev, []*Expr{
						{Type: Literal, Literal: "/(?>\\s+)"},
						{Type: Literal, Literal: "a b  c   d"},
					})
					So(err, ShouldBeNil)
					So(resp.Type, ShouldEqual, Replace)
					So(resp.Value, ShouldResemble, []interface{}{"a", "b", "c", "d"})
				})
			})
		})
	})

	Convey("Reversibility with Join Operator", t, func() {
		// Test split(join(...)) and join(split(...)) scenarios
		Convey("join(split(...)) should return original string", func() {
			ev := &Evaluator{Tree: YAML(`{}`)}
			splitOp := SplitOperator{}
			joinOp := JoinOperator{}

			testCases := []struct {
				delimiter string
				original  string
			}{
				{",", "sg-1,sg-2,sg-3"},
				{":", "host:port:path"},
				{" ", "hello world test"},
				{"::", "part1::part2::part3"},
				{"-", "one-two-three-four"},
				{"|", "a|b|c|d|e"},
				{".", "192.168.1.1"},
			}

			for _, tc := range testCases {
				// First split
				splitResp, err := splitOp.Run(ev, []*Expr{
					{Type: Literal, Literal: tc.delimiter},
					{Type: Literal, Literal: tc.original},
				})
				So(err, ShouldBeNil)
				So(splitResp.Type, ShouldEqual, Replace)

				// Then join the result back
				joinArgs := []*Expr{{Type: Literal, Literal: tc.delimiter}}
				// Convert the split result to expressions for join
				splitResult := splitResp.Value.([]interface{})
				joinArgs = append(joinArgs, &Expr{Type: Literal, Literal: splitResult})

				joinResp, err := joinOp.Run(ev, joinArgs)
				So(err, ShouldBeNil)
				So(joinResp.Type, ShouldEqual, Replace)
				So(joinResp.Value, ShouldEqual, tc.original)
			}
		})

		Convey("split(join(...)) should return original array", func() {
			ev := &Evaluator{Tree: YAML(`{}`)}
			splitOp := SplitOperator{}
			joinOp := JoinOperator{}

			testCases := []struct {
				delimiter string
				original  []interface{}
			}{
				{",", []interface{}{"sg-1", "sg-2", "sg-3"}},
				{":", []interface{}{"host", "port", "path"}},
				{" ", []interface{}{"hello", "world", "test"}},
				{"::", []interface{}{"part1", "part2", "part3"}},
				{"-", []interface{}{"one", "two", "three", "four"}},
			}

			for _, tc := range testCases {
				// First join
				joinArgs := []*Expr{{Type: Literal, Literal: tc.delimiter}}
				joinArgs = append(joinArgs, &Expr{Type: Literal, Literal: tc.original})

				joinResp, err := joinOp.Run(ev, joinArgs)
				So(err, ShouldBeNil)
				So(joinResp.Type, ShouldEqual, Replace)

				// Then split the result back
				splitResp, err := splitOp.Run(ev, []*Expr{
					{Type: Literal, Literal: tc.delimiter},
					{Type: Literal, Literal: joinResp.Value},
				})
				So(err, ShouldBeNil)
				So(splitResp.Type, ShouldEqual, Replace)
				So(splitResp.Value, ShouldResemble, tc.original)
			}
		})

		Convey("edge cases for reversibility", func() {
			ev := &Evaluator{Tree: YAML(`{}`)}
			splitOp := SplitOperator{}
			joinOp := JoinOperator{}

			Convey("empty strings in array", func() {
				original := []interface{}{"a", "", "b", "", "c"}
				delimiter := ","

				// Join then split
				joinArgs := []*Expr{
					{Type: Literal, Literal: delimiter},
					{Type: Literal, Literal: original},
				}
				joinResp, err := joinOp.Run(ev, joinArgs)
				So(err, ShouldBeNil)

				splitResp, err := splitOp.Run(ev, []*Expr{
					{Type: Literal, Literal: delimiter},
					{Type: Literal, Literal: joinResp.Value},
				})
				So(err, ShouldBeNil)
				So(splitResp.Value, ShouldResemble, original)
			})

			Convey("strings containing the delimiter", func() {
				// This test shows a limitation: if array elements contain the delimiter,
				// reversibility is not guaranteed
				delimiter := ","
				problematic := []interface{}{"a,b", "c", "d"}

				// Join will produce "a,b,c,d"
				joinArgs := []*Expr{
					{Type: Literal, Literal: delimiter},
					{Type: Literal, Literal: problematic},
				}
				joinResp, err := joinOp.Run(ev, joinArgs)
				So(err, ShouldBeNil)
				So(joinResp.Value, ShouldEqual, "a,b,c,d")

				// Split will produce ["a", "b", "c", "d"] - not the original!
				splitResp, err := splitOp.Run(ev, []*Expr{
					{Type: Literal, Literal: delimiter},
					{Type: Literal, Literal: joinResp.Value},
				})
				So(err, ShouldBeNil)
				// This demonstrates the limitation
				So(splitResp.Value, ShouldResemble, []interface{}{"a", "b", "c", "d"})
				So(splitResp.Value, ShouldNotResemble, problematic)
			})

			Convey("single element arrays", func() {
				original := []interface{}{"single-item"}
				delimiter := ","

				// Join then split
				joinArgs := []*Expr{
					{Type: Literal, Literal: delimiter},
					{Type: Literal, Literal: original},
				}
				joinResp, err := joinOp.Run(ev, joinArgs)
				So(err, ShouldBeNil)

				splitResp, err := splitOp.Run(ev, []*Expr{
					{Type: Literal, Literal: delimiter},
					{Type: Literal, Literal: joinResp.Value},
				})
				So(err, ShouldBeNil)
				So(splitResp.Value, ShouldResemble, original)
			})

			Convey("unicode and special characters", func() {
				testCases := []struct {
					delimiter string
					original  string
				}{
					{"→", "part1→part2→part3"},
					{"•", "bullet•point•list"},
					{"🔥", "fire🔥emoji🔥test"},
					{"\t", "tab\tseparated\tvalues"},
					{"\n", "line1\nline2\nline3"},
				}

				for _, tc := range testCases {
					// Split then join
					splitResp, err := splitOp.Run(ev, []*Expr{
						{Type: Literal, Literal: tc.delimiter},
						{Type: Literal, Literal: tc.original},
					})
					So(err, ShouldBeNil)

					joinArgs := []*Expr{{Type: Literal, Literal: tc.delimiter}}
					splitResult := splitResp.Value.([]interface{})
					joinArgs = append(joinArgs, &Expr{Type: Literal, Literal: splitResult})

					joinResp, err := joinOp.Run(ev, joinArgs)
					So(err, ShouldBeNil)
					So(joinResp.Value, ShouldEqual, tc.original)
				}
			})
		})

	})
}
