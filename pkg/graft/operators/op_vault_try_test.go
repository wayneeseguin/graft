package operators

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/geofffranks/simpleyaml"
	. "github.com/smartystreets/goconvey/convey"
)

func TestVaultTryOperator(t *testing.T) {
	YAML := func(s string) map[interface{}]interface{} {
		y, err := simpleyaml.NewYaml([]byte(s))
		So(err, ShouldBeNil)

		data, err := y.Map()
		So(err, ShouldBeNil)

		return data
	}

	Convey("vault-try operator", t, func() {
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
						fmt.Fprintf(w, `{"request_id":"test","lease_id":"","renewable":false,"lease_duration":0,"data":{"auth":{"token/":{"type":"token"}},"secret":{"secret/":{"type":"kv","options":{"version":"1"}}}}}`)
					case "/v1/secret/prod":
						w.WriteHeader(200)
						fmt.Fprintf(w, `{"data":{"password":"prod-secret-password"}}`)
					case "/v1/secret/staging":
						w.WriteHeader(200)
						fmt.Fprintf(w, `{"data":{"password":"staging-secret-password"}}`)
					case "/v1/secret/common":
						w.WriteHeader(200)
						fmt.Fprintf(w, `{"data":{"username":"admin","password":"common-password","port":"5432"}}`)
					default:
						w.WriteHeader(404)
						fmt.Fprintf(w, `{"errors":[]}`)
					}
				},
			),
		)
		defer mock.Close()

		os.Setenv("VAULT_ADDR", mock.URL)
		os.Setenv("VAULT_TOKEN", "test-token")
		SkipVault = false

		Convey("should fail with too few arguments", func() {
			input := YAML(`password: (( vault-try "secret/prod:password" ))`)
			ev := &Evaluator{Tree: input}
			err := ev.RunPhase(EvalPhase)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "vault-try operator requires at least 2 arguments")
		})

		Convey("should return first successful path", func() {
			input := YAML(`password: (( vault-try "secret/prod:password" "secret/dev:password" "default-password" ))`)
			ev := &Evaluator{Tree: input}
			err := ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["password"], ShouldEqual, "prod-secret-password")
		})

		Convey("should try second path if first fails", func() {
			input := YAML(`password: (( vault-try "secret/missing:password" "secret/staging:password" "default-password" ))`)
			ev := &Evaluator{Tree: input}
			err := ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["password"], ShouldEqual, "staging-secret-password")
		})

		Convey("should use default if all paths fail", func() {
			input := YAML(`password: (( vault-try "secret/missing1:password" "secret/missing2:password" "default-password" ))`)
			ev := &Evaluator{Tree: input}
			err := ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["password"], ShouldEqual, "default-password")
		})

		Convey("should support references for paths", func() {
			input := YAML(`
prodPath: secret/prod:password
password: (( vault-try prodPath "secret/dev:password" "default-password" ))`)
			ev := &Evaluator{Tree: input}
			err := ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["password"], ShouldEqual, "prod-secret-password")
		})

		Convey("should support references as default", func() {
			input := YAML(`
default: my-default-password
password: (( vault-try "secret/missing:password" default ))`)
			ev := &Evaluator{Tree: input}
			err := ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["password"], ShouldEqual, "my-default-password")
		})

		Convey("should support environment variable as default", func() {
			os.Setenv("DEFAULT_PASSWORD", "env-default-password")
			input := YAML(`password: (( vault-try "secret/missing:password" $DEFAULT_PASSWORD ))`)
			ev := &Evaluator{Tree: input}
			err := ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["password"], ShouldEqual, "env-default-password")
		})

		Convey("should support nil as default", func() {
			input := YAML(`password: (( vault-try "secret/missing:password" nil ))`)
			ev := &Evaluator{Tree: input}
			err := ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["password"], ShouldBeNil)
		})

		Convey("should handle multiple missing paths gracefully", func() {
			input := YAML(`password: (( vault-try "secret/a:password" "secret/b:password" "secret/c:password" "fallback-password" ))`)
			ev := &Evaluator{Tree: input}
			err := ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["password"], ShouldEqual, "fallback-password")
		})

		Convey("should use default when vault paths are malformed", func() {
			input := YAML(`password: (( vault-try "malformed-no-colon" "also-malformed" "default" ))`)
			ev := &Evaluator{Tree: input}
			err := ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["password"], ShouldEqual, "default")
		})

		Convey("should handle grab operator as default", func() {
			input := YAML(`
defaults:
  password: grabbed-default
password: (( vault-try "secret/missing:password" grab defaults.password ))`)
			ev := &Evaluator{Tree: input}
			err := ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["password"], ShouldEqual, "grabbed-default")
		})

		Convey("should work with SkipVault enabled", func() {
			SkipVault = true
			input := YAML(`password: (( vault-try "secret/any:password" "never-tried" ))`)
			ev := &Evaluator{Tree: input}
			err := ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["password"], ShouldEqual, "REDACTED")
			SkipVault = false
		})

		Convey("should handle different key names in paths", func() {
			input := YAML(`
username: (( vault-try "secret/missing:username" "secret/common:username" "default-user" ))
password: (( vault-try "secret/missing:password" "secret/common:password" "default-pass" ))
port: (( vault-try "secret/missing:port" "secret/common:port" "5432" ))`)
			ev := &Evaluator{Tree: input}
			err := ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["username"], ShouldEqual, "admin")
			So(ev.Tree["password"], ShouldEqual, "common-password")
			So(ev.Tree["port"], ShouldEqual, "5432")
		})

		Convey("should fall back to default when path is not a string", func() {
			input := YAML(`
config:
  nested: map
password: (( vault-try config "default-when-not-string" ))`)
			ev := &Evaluator{Tree: input}
			err := ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["password"], ShouldEqual, "default-when-not-string")
		})
	})
}
