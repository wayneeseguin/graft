package graft

import (
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/yaml.v3"
	. "github.com/wayneeseguin/graft/log"
)

func TestEnhancedBase64Operators(t *testing.T) {
	Convey("Enhanced Base64 Operators", t, func() {
		// Enable debug temporarily
		oldDebug := DebugOn
		// DebugOn = true  // Disable for cleaner output
		
		// Enable enhanced parser for these tests
		oldUseEnhanced := UseEnhancedParser
		EnableEnhancedParser() // This will set UseEnhancedParser and register all enhanced operators
		defer func() {
			DebugOn = oldDebug
			if !oldUseEnhanced {
				DisableEnhancedParser()
			}
		}()

		Convey("Base64 Encoding", func() {
			Convey("should support nested concat expressions", func() {
				// Verify enhanced parser is enabled
				So(UseEnhancedParser, ShouldBeTrue)
				
				input := `
user: alice
pass: secret123
encoded: (( base64 (concat user ":" pass) ))
`
				var data map[interface{}]interface{}
				err := yaml.Unmarshal([]byte(input), &data)
				So(err, ShouldBeNil)

				ev := &Evaluator{Tree: data}
				err = ev.RunPhase(EvalPhase)
				So(err, ShouldBeNil)
				So(ev.Tree["encoded"], ShouldEqual, "YWxpY2U6c2VjcmV0MTIz") // base64("alice:secret123")
			})

			Convey("should support nested join expressions", func() {
				input := `
parts:
  - hello
  - world
encoded: (( base64 (join " " parts) ))
`
				var data map[interface{}]interface{}
				err := yaml.Unmarshal([]byte(input), &data)
				So(err, ShouldBeNil)

				ev := &Evaluator{Tree: data}
				err = ev.RunPhase(EvalPhase)
				So(err, ShouldBeNil)
				So(ev.Tree["encoded"], ShouldEqual, "aGVsbG8gd29ybGQ=") // base64("hello world")
			})

			Convey("should support environment variables", func() {
				os.Setenv("SECRET_VALUE", "my-secret")
				defer os.Unsetenv("SECRET_VALUE")

				input := `
encoded: (( base64 $SECRET_VALUE ))
`
				var data map[interface{}]interface{}
				err := yaml.Unmarshal([]byte(input), &data)
				So(err, ShouldBeNil)

				ev := &Evaluator{Tree: data}
				err = ev.RunPhase(EvalPhase)
				So(err, ShouldBeNil)
				So(ev.Tree["encoded"], ShouldEqual, "bXktc2VjcmV0") // base64("my-secret")
			})

			Convey("should support nested grab", func() {
				input := `
config:
  apikey: "super-secret-key"
encoded: (( base64 (grab config.apikey) ))
`
				var data map[interface{}]interface{}
				err := yaml.Unmarshal([]byte(input), &data)
				So(err, ShouldBeNil)

				ev := &Evaluator{Tree: data}
				err = ev.RunPhase(EvalPhase)
				So(err, ShouldBeNil)
				So(ev.Tree["encoded"], ShouldEqual, "c3VwZXItc2VjcmV0LWtleQ==") // base64("super-secret-key")
			})

			Convey("should handle non-string scalars", func() {
				input := `
number: 42
encoded: (( base64 number ))
`
				var data map[interface{}]interface{}
				err := yaml.Unmarshal([]byte(input), &data)
				So(err, ShouldBeNil)

				ev := &Evaluator{Tree: data}
				err = ev.RunPhase(EvalPhase)
				So(err, ShouldBeNil)
				So(ev.Tree["encoded"], ShouldEqual, "NDI=") // base64("42")
			})
		})

		Convey("Base64 Decoding", func() {
			Convey("should support nested expressions", func() {
				input := `
encoded: "aGVsbG8gd29ybGQ="
decoded: (( base64-decode (grab encoded) ))
`
				var data map[interface{}]interface{}
				err := yaml.Unmarshal([]byte(input), &data)
				So(err, ShouldBeNil)

				ev := &Evaluator{Tree: data}
				err = ev.RunPhase(EvalPhase)
				So(err, ShouldBeNil)
				So(ev.Tree["decoded"], ShouldEqual, "hello world")
			})

			Convey("should support environment variables", func() {
				os.Setenv("ENCODED_SECRET", "bXktc2VjcmV0")
				defer os.Unsetenv("ENCODED_SECRET")

				input := `
decoded: (( base64-decode $ENCODED_SECRET ))
`
				var data map[interface{}]interface{}
				err := yaml.Unmarshal([]byte(input), &data)
				So(err, ShouldBeNil)

				ev := &Evaluator{Tree: data}
				err = ev.RunPhase(EvalPhase)
				So(err, ShouldBeNil)
				So(ev.Tree["decoded"], ShouldEqual, "my-secret")
			})

			Convey("should error on invalid base64", func() {
				input := `
invalid: "not-valid-base64!"
decoded: (( base64-decode invalid ))
`
				var data map[interface{}]interface{}
				err := yaml.Unmarshal([]byte(input), &data)
				So(err, ShouldBeNil)

				ev := &Evaluator{Tree: data}
				err = ev.RunPhase(EvalPhase)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "base64 decoding failed")
			})

			Convey("should error on non-string input", func() {
				input := `
number: 123
decoded: (( base64-decode number ))
`
				var data map[interface{}]interface{}
				err := yaml.Unmarshal([]byte(input), &data)
				So(err, ShouldBeNil)

				ev := &Evaluator{Tree: data}
				err = ev.RunPhase(EvalPhase)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "which is not a string")
			})
		})

		Convey("Round-trip encoding/decoding", func() {
			input := `
original: "Hello, World! 123 @#$"
encoded: (( base64 original ))
decoded: (( base64-decode encoded ))
`
			var data map[interface{}]interface{}
			err := yaml.Unmarshal([]byte(input), &data)
			So(err, ShouldBeNil)

			ev := &Evaluator{Tree: data}
			err = ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["decoded"], ShouldEqual, ev.Tree["original"])
		})
	})
}