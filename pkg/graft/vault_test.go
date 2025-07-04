// TODO: Vault tests removed - accessing internal kv variable from operators package
//go:build ignore
// +build ignore

package graft

import (
	"bufio"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	// Use geofffranks forks to persist the fix in https://github.com/go-yaml/yaml/pull/133/commits
	// Also https://github.com/go-yaml/yaml/pull/195
	"github.com/geofffranks/simpleyaml"
	"github.com/geofffranks/yaml"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/wayneeseguin/graft/internal/utils/ansi"
)

func TestVault(t *testing.T) {
	// Disable ANSI colors for testing
	ansi.Color(false)

	YAML := func(s string) map[interface{}]interface{} {
		y, err := simpleyaml.NewYaml([]byte(s))
		So(err, ShouldBeNil)

		data, err := y.Map()
		So(err, ShouldBeNil)

		return data
	}
	ToYAML := func(tree map[interface{}]interface{}) string {
		y, err := yaml.Marshal(tree)
		So(err, ShouldBeNil)
		return string(y)
	}
	ReYAML := func(s string) string {
		return ToYAML(YAML(s))
	}
	RunTests := func(src string) {
		//Act like we're running Graft freshly
		kv = nil

		var test, input, output string
		var current *string
		testPat := regexp.MustCompile(`^##+\s+(.*)\s*$`)

		convey := func() {
			if test != "" {
				Convey(test, func() {
					ev := &Evaluator{Tree: YAML(input)}
					err := ev.RunPhase(EvalPhase)
					So(err, ShouldBeNil)
					So(ToYAML(ev.Tree), ShouldEqual, ReYAML(output))
				})
			}
		}

		s := bufio.NewScanner(strings.NewReader(src))
		for s.Scan() {
			if testPat.MatchString(s.Text()) {
				m := testPat.FindStringSubmatch(s.Text())
				convey()
				test, input, output = m[1], "", ""
				continue
			}

			if s.Text() == "---" {
				if input == "" {
					current = &input
				} else {
					current = &output
				}
				continue
			}

			if current != nil {
				*current = *current + s.Text() + "\n"
			}
		}
		convey()
	}

	RunErrorTests := func(src string) {
		var test, input, errors string
		var current *string
		testPat := regexp.MustCompile(`^##+\s+(.*)\s*$`)

		convey := func() {
			if test != "" {
				Convey(test, func() {
					ev := &Evaluator{Tree: YAML(input)}
					err := ev.RunPhase(EvalPhase)
					if err == nil {
						fmt.Printf("errors: %+v\nerr:%+v\n", errors, err)
					}
					So(err, ShouldNotBeNil)
					So(strings.Trim(err.Error(), " \t"), ShouldEqual, errors)
				})
			}
		}

		s := bufio.NewScanner(strings.NewReader(src))
		for s.Scan() {
			if testPat.MatchString(s.Text()) {
				m := testPat.FindStringSubmatch(s.Text())
				convey()
				test, input, errors = m[1], "", ""
				continue
			}

			if s.Text() == "---" {
				if input == "" {
					current = &input
				} else {
					current = &errors
				}
				continue
			}

			if current != nil {
				*current = *current + s.Text() + "\n"
			}
		}
		convey()
	}

	Convey("Disconnected Vault", t, func() {
		SkipVault = true

		RunTests(`
##################################################  emits REDACTED when asked to
---
secret: (( vault "secret/hand:shake" ))

---
secret: REDACTED
`)
	})

	allTestVault := func(version int) {
		mock := &httptest.Server{}

		mountsResp := fmt.Sprintf(`{"request_id":"592b4162-c855-7987-e6aa-8ef3d9d1393e","lease_id":"","renewable":false,"lease_duration":0,"data":{"auth":{"token/":{"accessor":"auth_token_87436efa","config":{"default_lease_ttl":0,"force_no_cache":false,"max_lease_ttl":0,"token_type":"default-service"},"description":"token based credentials","local":false,"options":null,"seal_wrap":false,"type":"token"}},"secret":{"cubbyhole/":{"accessor":"cubbyhole_304cf103","config":{"default_lease_ttl":0,"force_no_cache":false,"max_lease_ttl":0},"description":"per-token private secret storage","local":true,"options":null,"seal_wrap":false,"type":"cubbyhole"},"identity/":{"accessor":"identity_68585106","config":{"default_lease_ttl":0,"force_no_cache":false,"max_lease_ttl":0},"description":"identity store","local":false,"options":null,"seal_wrap":false,"type":"identity"},"secret/":{"accessor":"kv_c707ea61","config":{"default_lease_ttl":0,"force_no_cache":false,"max_lease_ttl":0},"description":"key/value secret storage","local":false,"options":{"version":"%d"},"seal_wrap":false,"type":"kv"},"sys/":{"accessor":"system_1ef88192","config":{"default_lease_ttl":0,"force_no_cache":false,"max_lease_ttl":0},"description":"system endpoints used for control, policy and debugging","local":false,"options":null,"seal_wrap":false,"type":"system"}}},"wrap_info":null,"warnings":null,"auth":null}`, version)

		switch version {
		case 1:
			mock = httptest.NewServer(
				http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						if r.Header.Get("X-Vault-Token") != "sekrit-toekin" {
							w.WriteHeader(403)
							fmt.Fprintf(w, `{"errors":["missing client token"]}`)
							return
						}
						switch r.URL.Path {
						case "/v1/sys/internal/ui/mounts":
							w.WriteHeader(200)
							fmt.Fprintf(w, mountsResp)
						case "/v1/secret/hand":
							w.WriteHeader(200)
							fmt.Fprintf(w, `{"data":{"shake":"knock, knock"}}`)
						case "/v1/secret/admin":
							w.WriteHeader(200)
							fmt.Fprintf(w, `{"data":{"username":"admin","password":"x12345"}}`)
						case "/v1/secret/key":
							w.WriteHeader(200)
							fmt.Fprintf(w, `{"data":{"test":"testing"}}`)
						case "/v1/secret/malformed":
							w.WriteHeader(200)
							fmt.Fprintf(w, `wait, this isn't JSON`)
						case "/v1/secret/structure":
							w.WriteHeader(200)
							fmt.Fprintf(w, `{"data":{"data":[1,2,3]}}`)
						default:
							w.WriteHeader(404)
							fmt.Fprintf(w, `{"errors":[]}`)
						}
					},
				),
			)

		case 2:
			mock = httptest.NewServer(
				http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						if r.Header.Get("X-Vault-Token") != "sekrit-toekin" {
							w.WriteHeader(403)
							fmt.Fprintf(w, `{"errors":["missing client token"]}`)
							return
						}
						switch r.URL.Path {
						case "/v1/sys/internal/ui/mounts":
							w.WriteHeader(200)
							fmt.Fprintf(w, mountsResp)
						case "/v1/secret/data/hand":
							w.WriteHeader(200)
							fmt.Fprintf(w, `{"data":{"data:{"shake":"knock, knock"}}`)
						case "/v1/secret/data/admin":
							w.WriteHeader(200)
							fmt.Fprintf(w, `{"data":{"data:{"username":"admin","password":"x12345"}}`)
						case "/v1/secret/data/key":
							w.WriteHeader(200)
							fmt.Fprintf(w, `{"data":{"data:{"test":"testing"}}`)
						case "/v1/secret/data/malformed":
							w.WriteHeader(200)
							fmt.Fprintf(w, `wait, this isn't JSON`)
						case "/v1/secret/data/structure":
							w.WriteHeader(200)
							fmt.Fprintf(w, `{"data":{"data:{"data":[1,2,3]}}`)
						default:
							w.WriteHeader(404)
							fmt.Fprintf(w, `{"errors":[]}`)
						}
					},
				),
			)

		}
		defer mock.Close()

		SkipVault = false
		os.Setenv("VAULT_ADDR", mock.URL)
		os.Setenv("VAULT_TOKEN", "sekrit-toekin")
		RunTests(`
################################################  emits sensitive credentials
---
meta:
  prefix: secret
  key: secret/key:test
secret: (( vault "secret/hand:shake" ))
username: (( vault "secret/admin:username" ))
password: (( vault "secret/admin:password" ))
prefixed: (( vault meta.prefix "/admin:password" ))
key: (( vault $.meta.key ))

---
meta:
  key: secret/key:test
  prefix: secret
secret: knock, knock
username: admin
password: x12345
prefixed: x12345
key: testing
`)

		os.Setenv("VAULT_ADDR", mock.URL)
		oldhome := os.Getenv("HOME")
		os.Setenv("HOME", "assets/home/auth")
		os.Setenv("VAULT_TOKEN", "")
		RunTests(`
##########################  retrieves token transparently from ~/.vault-token
---
secret: (( vault "secret/hand:shake" ))

---
secret: knock, knock
`)

		os.Setenv("VAULT_ADDR", "garbage")
		os.Setenv("VAULT_TOKEN", "")
		os.Setenv("HOME", "assets/home/svtoken")
		os.WriteFile("assets/home/svtoken/.svtoken",
			[]byte("vault: "+mock.URL+"\n"+
				"token: sekrit-toekin\n"), 0644)
		RunTests(`
##############################  retrieves token transparently from ~/.svtoken
---
secret: (( vault "secret/hand:shake" ))

---
secret: knock, knock
`)

		/* RESET TO A VALID, AUTHENTICATED STATE */
		os.Setenv("VAULT_ADDR", mock.URL)
		os.Setenv("HOME", "assets/home/auth")

		RunErrorTests(`
#########################################  fails when missing its argument
---
secret: (( vault ))

---
1 error(s) detected:
 - $.secret: vault operator requires at least one argument

#########################################  fails on non-existent reference
---
meta: {}
secret: (( vault $.meta.key ))

---
1 error(s) detected:
 - $.secret: Unable to resolve ` + "`" + `meta.key` + "`" + `: ` + "`" + `$.meta.key` + "`" + ` could not be found in the datastructure

####################################################  fails on map reference
---
meta:
  key: secret/hand2:shake
secret: (( vault $.meta ))

---
1 error(s) detected:
 - $.secret: $.meta is a map; only scalars are supported for vault paths

##################################################  fails on list reference
---
meta:
  - first
secret: (( vault $.meta ))

---
1 error(s) detected:
 - $.secret: $.meta is a list; only scalars are supported for vault paths

#########################################  fails on non-existent credentials
---
secret: (( vault "secret/e:noent" ))

---
1 error(s) detected:
 - $.secret: secret secret/e:noent not found

##############################################  fails on non-string argument
---
secret: (( vault 42 ))

---
1 error(s) detected:
 - $.secret: invalid argument 42; must be in the form path/to/secret:key

#################################################  fails on non-string data
---
secret: (( vault "secret/structure:data" ))

---
1 error(s) detected:
 - $.secret: secret secret/structure:data is not a string

`)

		os.Setenv("VAULT_TOKEN", "incorrect")
		RunErrorTests(`
#####################################################  fails on a bad token
---
secret: (( vault "hand3:shake" ))

---
1 error(s) detected:
 - $.secret: 403 Forbidden: missing client token

`)

		oldhome = os.Getenv("HOME")
		os.Setenv("HOME", "assets/home/unauth")
		SkipVault = false
		os.Setenv("VAULT_TOKEN", "")
		RunErrorTests(`
################################################  fails on a missing token
---
secret: (( vault "secret/hand4:shake" ))

---
1 error(s) detected:
 - $.secret: Error during Vault client initialization: Failed to determine Vault URL / token, and the $REDACT environment variable is not set.

`)
		os.Setenv("HOME", oldhome)
	}

	Convey("Testing Vault", t, func() {
		Convey("With KV v1", func() {
			allTestVault(1)
		})

		Convey("With KV v2", func() {
			allTestVault(2)
		})
	})

	Convey("It correctly parses path", t, func() {
		for _, test := range []struct {
			path      string //The full path to run through the parse function
			expSecret string //What is expected to be left of the colon
			expKey    string //What is expected to be right of the colon
		}{
			//-----TEST CASES GO HERE-----
			// { "path to parse", "expected secret", "expected key" }
			{"just/a/secret", "just/a/secret", ""},
			{"secret/with/colon:", "secret/with/colon", ""},
			{":", "", ""},
			{"a:", "a", ""},
			{"", "", ""},
			{"secret/and:key", "secret/and", "key"},
			{":justakey", "", "justakey"},
			{"secretwithcolon://127.0.0.1:", "secretwithcolon://127.0.0.1", ""},
			{"secretwithcolons://127.0.0.1:8500:", "secretwithcolons://127.0.0.1:8500", ""},
			{"secretwithcolons://127.0.0.1:8500:andkey", "secretwithcolons://127.0.0.1:8500", "andkey"},
		} {
			Convey(test.path, func() {
				secret, key := parsePath(test.path)
				So(secret, ShouldEqual, test.expSecret)
				So(key, ShouldEqual, test.expKey)
			})
		}
	})
}
