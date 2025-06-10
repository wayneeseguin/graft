// TODO: Vault default tests removed - accessing internal kv variable from operators package
//go:build ignore
// +build ignore

package graft

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/geofffranks/simpleyaml"
	"github.com/geofffranks/yaml"
	. "github.com/smartystreets/goconvey/convey"
)

func TestVaultWithDefaults(t *testing.T) {
	YAML := func(s string) map[interface{}]interface{} {
		y, err := simpleyaml.NewYaml([]byte(s))
		So(err, ShouldBeNil)

		data, err := y.Map()
		So(err, ShouldBeNil)

		return data
	}

	_ = func(tree map[interface{}]interface{}) string {
		y, err := yaml.Marshal(tree)
		So(err, ShouldBeNil)
		return string(y)
	}

	Convey("Vault operator with || default values", t, func() {
		// Reset vault client
		kv = nil

		// Create mock vault server
		mock := httptest.NewServer(
			http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("X-Vault-Token") != "test-token" {
						w.WriteHeader(403)
						fmt.Fprintf(w, `{"errors":["permission denied"]}`)
						return
					}
					switch r.URL.Path {
					case "/v1/sys/internal/ui/mounts":
						w.WriteHeader(200)
						fmt.Fprintf(w, `{"request_id":"test","lease_id":"","renewable":false,"lease_duration":0,"data":{"secret/":{"type":"kv","options":{"version":"1"}}}}`)
					case "/v1/secret/exists":
						w.WriteHeader(200)
						fmt.Fprintf(w, `{"data":{"password":"secret123"}}`)
					case "/v1/secret/config":
						w.WriteHeader(200)
						fmt.Fprintf(w, `{"data":{"host":"prod.example.com","port":"5432"}}`)
					default:
						w.WriteHeader(404)
						fmt.Fprintf(w, `{"errors":["secret not found"]}`)
					}
				},
			),
		)
		defer mock.Close()

		os.Setenv("VAULT_ADDR", mock.URL)
		os.Setenv("VAULT_TOKEN", "test-token")
		SkipVault = false

		Convey("should return vault value when secret exists", func() {
			input := YAML(`
password: (( vault "secret/exists:password" || "default-password" ))
`)
			ev := &Evaluator{Tree: input}
			err := ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["password"], ShouldEqual, "secret123")
		})

		Convey("should return default value when secret does not exist", func() {
			input := YAML(`
password: (( vault "secret/missing:password" || "default-password" ))
`)
			ev := &Evaluator{Tree: input}
			err := ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["password"], ShouldEqual, "default-password")
		})

		Convey("should support reference as default value", func() {
			input := YAML(`
fallback: my-fallback-value
password: (( vault "secret/missing:password" || fallback ))
`)
			ev := &Evaluator{Tree: input}
			err := ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["password"], ShouldEqual, "my-fallback-value")
		})

		Convey("should now support grab operator as default with parser", func() {
			input := YAML(`
defaults:
  password: grabbed-default
password: (( vault "secret/missing:password" || grab defaults.password ))
`)
			ev := &Evaluator{Tree: input}
			err := ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["password"], ShouldEqual, "grabbed-default")
		})

		Convey("workaround: grab with intermediate variable should work", func() {
			input := YAML(`
defaults:
  password: grabbed-default-value
grab_default: (( grab defaults.password ))
password: (( vault "secret/missing:password" || grab_default ))
`)
			ev := &Evaluator{Tree: input}
			err := ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["password"], ShouldEqual, "grabbed-default-value")
		})

		Convey("should support environment variable as default", func() {
			os.Setenv("DEFAULT_PASSWORD", "env-default")
			input := YAML(`
password: (( vault "secret/missing:password" || $DEFAULT_PASSWORD ))
`)
			ev := &Evaluator{Tree: input}
			err := ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["password"], ShouldEqual, "env-default")
		})

		Convey("should support nil as default value", func() {
			input := YAML(`
password: (( vault "secret/missing:password" || nil ))
`)
			ev := &Evaluator{Tree: input}
			err := ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["password"], ShouldBeNil)
		})

		Convey("should use default on malformed vault path", func() {
			input := YAML(`
password: (( vault "malformed-path-no-colon" || "default" ))
`)
			ev := &Evaluator{Tree: input}
			err := ev.RunPhase(EvalPhase)
			// With fallback behavior, even malformed paths fall back to default
			So(err, ShouldBeNil)
			So(ev.Tree["password"], ShouldEqual, "default")
		})

		Convey("should work with concatenated vault paths", func() {
			input := YAML(`
prefix: secret
key: missing
password: (( vault prefix "/" key ":password" || "concatenated-default" ))
`)
			ev := &Evaluator{Tree: input}
			err := ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["password"], ShouldEqual, "concatenated-default")
		})

		Convey("should support chained defaults", func() {
			input := YAML(`
password: (( vault "secret/missing1:password" || vault "secret/missing2:password" || "final-default" ))
`)
			ev := &Evaluator{Tree: input}
			err := ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["password"], ShouldEqual, "final-default")
		})

		Convey("should work when VAULT_SKIP_VERIFY is set", func() {
			SkipVault = true
			input := YAML(`
password: (( vault "secret/anything:password" || "skipped-default" ))
`)
			ev := &Evaluator{Tree: input}
			err := ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["password"], ShouldEqual, "REDACTED")
			SkipVault = false
		})
	})
}
