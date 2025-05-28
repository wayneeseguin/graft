package operators

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/geofffranks/yaml"
	. "github.com/smartystreets/goconvey/convey"
)

func TestEnhancedOperatorsComprehensive(t *testing.T) {
	Convey("Enhanced Operators Comprehensive Test", t, func() {
		// Enable enhanced parser for these tests
		oldUseEnhanced := UseEnhancedParser
		EnableEnhancedParser()
		defer func() {
			if !oldUseEnhanced {
				DisableEnhancedParser()
			}
		}()

		Convey("file operator with nested concat", func() {
			// Create a temp file
			tmpDir, err := ioutil.TempDir("", "graft-test")
			So(err, ShouldBeNil)
			defer os.RemoveAll(tmpDir)

			err = ioutil.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("Hello from file!"), 0644)
			So(err, ShouldBeNil)

			input := fmt.Sprintf(`
dir: "%s"
filename: "test.txt"
content: (( file (concat dir "/" filename) ))
`, tmpDir)
			var data map[interface{}]interface{}
			err = yaml.Unmarshal([]byte(input), &data)
			So(err, ShouldBeNil)

			ev := &Evaluator{Tree: data}
			err = ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["content"], ShouldEqual, "Hello from file!")
		})

		Convey("keys operator with nested grab", func() {
			input := `
data:
  config:
    host: localhost
    port: 8080
    timeout: 30
result: (( keys (grab data.config) ))
`
			var data map[interface{}]interface{}
			err := yaml.Unmarshal([]byte(input), &data)
			So(err, ShouldBeNil)

			ev := &Evaluator{Tree: data}
			err = ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)

			keys := ev.Tree["result"].([]interface{})
			So(len(keys), ShouldEqual, 3)
			// Keys should be sorted
			So(keys[0], ShouldEqual, "host")
			So(keys[1], ShouldEqual, "port")
			So(keys[2], ShouldEqual, "timeout")
		})

		Convey("stringify operator with nested expression", func() {
			input := `
data:
  users:
    - name: alice
      role: admin
    - name: bob
      role: user
result: (( stringify (grab data.users) ))
`
			var data map[interface{}]interface{}
			err := yaml.Unmarshal([]byte(input), &data)
			So(err, ShouldBeNil)

			ev := &Evaluator{Tree: data}
			err = ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)

			result := ev.Tree["result"].(string)
			So(result, ShouldContainSubstring, "- name: alice")
			So(result, ShouldContainSubstring, "role: admin")
			So(result, ShouldContainSubstring, "- name: bob")
			So(result, ShouldContainSubstring, "role: user")
		})

		Convey("complex nested expressions", func() {
			// Create temp files
			tmpDir, err := ioutil.TempDir("", "graft-test")
			So(err, ShouldBeNil)
			defer os.RemoveAll(tmpDir)

			err = ioutil.WriteFile(filepath.Join(tmpDir, "users.yml"), []byte("alice,bob,charlie"), 0644)
			So(err, ShouldBeNil)

			input := fmt.Sprintf(`
config:
  dir: "%s"
  file: "users.yml"
users_raw: (( file (concat config.dir "/" config.file) ))
users_encoded: (( base64 users_raw ))
`, tmpDir)
			var data map[interface{}]interface{}
			err = yaml.Unmarshal([]byte(input), &data)
			So(err, ShouldBeNil)

			ev := &Evaluator{Tree: data}
			err = ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)

			So(ev.Tree["users_raw"], ShouldEqual, "alice,bob,charlie")
			So(ev.Tree["users_encoded"], ShouldEqual, "YWxpY2UsYm9iLGNoYXJsaWU=") // base64("alice,bob,charlie")
		})

		Convey("environment variables in nested expressions", func() {
			os.Setenv("TEST_CONFIG_KEY", "database")
			defer os.Unsetenv("TEST_CONFIG_KEY")

			input := `
configs:
  database:
    host: db.example.com
    port: 5432
  cache:
    host: cache.example.com
    port: 6379
selected: (( grab (concat "configs." $TEST_CONFIG_KEY) ))
keys_list: (( keys selected ))
host_info: (( stringify keys_list ))
`
			var data map[interface{}]interface{}
			err := yaml.Unmarshal([]byte(input), &data)
			So(err, ShouldBeNil)

			ev := &Evaluator{Tree: data}
			err = ev.RunPhase(EvalPhase)
			So(err, ShouldBeNil)

			// Handle both map types
			var host, port interface{}
			switch s := ev.Tree["selected"].(type) {
			case map[interface{}]interface{}:
				host = s["host"]
				port = s["port"]
			case map[string]interface{}:
				host = s["host"]
				port = s["port"]
			default:
				t.Fatalf("unexpected type for selected: %T", s)
			}
			So(host, ShouldEqual, "db.example.com")
			So(port, ShouldEqual, 5432)

			hostInfo := ev.Tree["host_info"].(string)
			So(hostInfo, ShouldContainSubstring, "host")
			So(hostInfo, ShouldContainSubstring, "port")
		})
	})
}
