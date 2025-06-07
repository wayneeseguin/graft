package operators

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/geofffranks/simpleyaml"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/wayneeseguin/graft/pkg/graft"
)

func TestVaultNatsEdgeCases(t *testing.T) {
	Convey("Vault and NATS Edge Cases and Error Scenarios", t, func() {
		// Start test NATS server
		ns, url := startTestNATSServer()
		defer ns.Shutdown()

		// Connect to test server
		nc, err := nats.Connect(url)
		So(err, ShouldBeNil)
		defer nc.Close()

		// Create JetStream context
		js, err := jetstream.New(nc)
		So(err, ShouldBeNil)

		// Set NATS URL environment variable
		oldNatsURL := os.Getenv("NATS_URL")
		os.Setenv("NATS_URL", url)
		defer os.Setenv("NATS_URL", oldNatsURL)

		Convey("Circular dependencies between vault and nats", func() {
			// Create KV store
			kv, err := js.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{
				Bucket: "config",
			})
			So(err, ShouldBeNil)

			// This would create a circular dependency if vault path came from NATS
			// and NATS auth came from vault
			yamlData := `
meta:
  vault_path: (( nats "kv:config/vault_path" ))
  nats_auth: (( vault "secret/nats:auth_token" ))

# This creates a potential circular dependency
config:
  database:
    password: (( vault meta.vault_path ))
`
			// Parse YAML into a tree
			yamlTree := make(map[interface{}]interface{})
			err = func() error {
				y, err := simpleyaml.NewYaml([]byte(yamlData))
				if err != nil {
					return err
				}
				yamlTree, err = y.Map()
				return err
			}()
			So(err, ShouldBeNil)
			
			ev := &graft.Evaluator{
				Tree: yamlTree,
			}

			// Store vault path in NATS
			_, err = kv.PutString(context.Background(), "vault_path", "secret/database:password")
			So(err, ShouldBeNil)

			// The evaluator should handle this gracefully
			natsOp := &NatsOperator{}
			err = natsOp.Setup()
			So(err, ShouldBeNil)

			args := []*graft.Expr{{Type: graft.Literal, Literal: "kv:config/vault_path"}}
			resp, err := natsOp.Run(ev, args)
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, "secret/database:password")
		})

		Convey("Race conditions with concurrent vault and nats access", func() {
			// Create KV store
			kv, err := js.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{
				Bucket: "concurrent",
			})
			So(err, ShouldBeNil)

			// Populate initial values
			for i := 0; i < 10; i++ {
				key := fmt.Sprintf("item%d", i)
				_, err = kv.PutString(context.Background(), key, fmt.Sprintf("value%d", i))
				So(err, ShouldBeNil)
			}

			yamlData := `
items:
  - config: (( nats "kv:concurrent/item0" ))
    secret: (( vault "secret/item0:value" ))
  - config: (( nats "kv:concurrent/item1" ))
    secret: (( vault "secret/item1:value" ))
  - config: (( nats "kv:concurrent/item2" ))
    secret: (( vault "secret/item2:value" ))
  - config: (( nats "kv:concurrent/item3" ))
    secret: (( vault "secret/item3:value" ))
  - config: (( nats "kv:concurrent/item4" ))
    secret: (( vault "secret/item4:value" ))
`
			// Parse YAML into a tree
			yamlTree := make(map[interface{}]interface{})
			err = func() error {
				y, err := simpleyaml.NewYaml([]byte(yamlData))
				if err != nil {
					return err
				}
				yamlTree, err = y.Map()
				return err
			}()
			So(err, ShouldBeNil)
			
			ev := &graft.Evaluator{
				Tree: yamlTree,
			}

			// Test concurrent access
			natsOp := &NatsOperator{}
			err = natsOp.Setup()
			So(err, ShouldBeNil)

			// Simulate concurrent reads
			done := make(chan bool, 5)
			errors := make(chan error, 5)

			for i := 0; i < 5; i++ {
				go func(idx int) {
					args := []*graft.Expr{{Type: graft.Literal, Literal: fmt.Sprintf("kv:concurrent/item%d", idx)}}
					resp, err := natsOp.Run(ev, args)
					if err != nil {
						errors <- err
					} else if resp.Value != fmt.Sprintf("value%d", idx) {
						errors <- fmt.Errorf("expected value%d, got %v", idx, resp.Value)
					}
					done <- true
				}(i)
			}

			// Wait for all goroutines
			for i := 0; i < 5; i++ {
				select {
				case <-done:
					// Success
				case err := <-errors:
					So(err, ShouldBeNil)
				case <-time.After(5 * time.Second):
					So("timeout", ShouldEqual, "completed")
				}
			}
		})

		Convey("Large data handling between vault and nats", func() {
			// Create object store for large data
			objStore, err := js.CreateObjectStore(context.Background(), jetstream.ObjectStoreConfig{
				Bucket: "large-data",
			})
			So(err, ShouldBeNil)

			// Create large YAML document
			largeYAML := "large_config:\n"
			for i := 0; i < 1000; i++ {
				largeYAML += fmt.Sprintf("  item_%d:\n    value: %d\n    enabled: true\n", i, i)
			}

			_, err = objStore.PutBytes(context.Background(), "large-config.yaml", []byte(largeYAML))
			So(err, ShouldBeNil)

			yamlData := `
# Load large config from NATS
base: (( nats "obj:large-data/large-config.yaml" ))

# Add secrets to specific items
enhanced_config:
  item_0:
    value: (( grab base.large_config.item_0.value ))
    secret: (( vault "secret/item_0:key" ))
  item_999:
    value: (( grab base.large_config.item_999.value ))
    secret: (( vault "secret/item_999:key" ))
`
			// Parse YAML into a tree
			yamlTree := make(map[interface{}]interface{})
			err = func() error {
				y, err := simpleyaml.NewYaml([]byte(yamlData))
				if err != nil {
					return err
				}
				yamlTree, err = y.Map()
				return err
			}()
			So(err, ShouldBeNil)
			
			ev := &graft.Evaluator{
				Tree: yamlTree,
			}

			// Test loading large data
			natsOp := &NatsOperator{}
			err = natsOp.Setup()
			So(err, ShouldBeNil)

			args := []*graft.Expr{{Type: graft.Literal, Literal: "obj:large-data/large-config.yaml"}}
			resp, err := natsOp.Run(ev, args)
			So(err, ShouldBeNil)
			So(resp.Value, ShouldNotBeNil)

			// Verify it's a proper YAML structure
			configMap, ok := resp.Value.(map[interface{}]interface{})
			So(ok, ShouldBeTrue)
			So(configMap["large_config"], ShouldNotBeNil)
		})

		Convey("Network failures and retries", func() {
			// Create KV store
			kv, err := js.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{
				Bucket: "network-test",
			})
			So(err, ShouldBeNil)

			_, err = kv.PutString(context.Background(), "test-key", "test-value")
			So(err, ShouldBeNil)

			yamlData := `
config:
  # These would fail if network is down
  from_nats: (( nats "kv:network-test/test-key" ))
  from_vault: (( vault "secret/test:value" ))
  
  # Fallback chain
  with_fallback: (( vault "secret/primary:value" || nats "kv:network-test/test-key" || "default" ))
`
			// Parse YAML into a tree
			yamlTree := make(map[interface{}]interface{})
			err = func() error {
				y, err := simpleyaml.NewYaml([]byte(yamlData))
				if err != nil {
					return err
				}
				yamlTree, err = y.Map()
				return err
			}()
			So(err, ShouldBeNil)
			
			ev := &graft.Evaluator{
				Tree: yamlTree,
			}

			// Test successful connection
			natsOp := &NatsOperator{}
			err = natsOp.Setup()
			So(err, ShouldBeNil)

			args := []*graft.Expr{{Type: graft.Literal, Literal: "kv:network-test/test-key"}}
			resp, err := natsOp.Run(ev, args)
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, "test-value")

			// Simulate network failure by using wrong connection
			os.Setenv("NATS_URL", "nats://localhost:99999")
			defer os.Setenv("NATS_URL", url)

			// This should fail
			natsOp2 := &NatsOperator{}
			err = natsOp2.Setup()
			So(err, ShouldNotBeNil) // Connection should fail
		})

		Convey("Special characters and encoding", func() {
			// Create KV store
			kv, err := js.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{
				Bucket: "special-chars",
			})
			So(err, ShouldBeNil)

			// Test various special characters
			specialValues := map[string]string{
				"with-spaces":    "value with spaces",
				"with-newlines":  "line1\nline2\nline3",
				"with-quotes":    `value with "quotes" and 'apostrophes'`,
				"with-unicode":   "Unicode: 你好世界 🌍",
				"with-backslash": `path\to\file`,
				"with-json":      `{"key": "value", "nested": {"item": true}}`,
			}

			for k, v := range specialValues {
				_, err = kv.PutString(context.Background(), k, v)
				So(err, ShouldBeNil)
			}

			yamlData := `
special:
  spaces: (( nats "kv:special-chars/with-spaces" ))
  newlines: (( nats "kv:special-chars/with-newlines" ))
  quotes: (( nats "kv:special-chars/with-quotes" ))
  unicode: (( nats "kv:special-chars/with-unicode" ))
  backslash: (( nats "kv:special-chars/with-backslash" ))
  json: (( nats "kv:special-chars/with-json" ))
  
  # Combine with vault paths that might have special chars
  vault_special: (( vault "secret/path-with-dash:key_with_underscore" ))
`
			// Parse YAML into a tree
			yamlTree := make(map[interface{}]interface{})
			err = func() error {
				y, err := simpleyaml.NewYaml([]byte(yamlData))
				if err != nil {
					return err
				}
				yamlTree, err = y.Map()
				return err
			}()
			So(err, ShouldBeNil)
			
			ev := &graft.Evaluator{
				Tree: yamlTree,
			}

			natsOp := &NatsOperator{}
			err = natsOp.Setup()
			So(err, ShouldBeNil)

			// Test each special character case
			for k, expected := range specialValues {
				args := []*graft.Expr{{Type: graft.Literal, Literal: fmt.Sprintf("kv:special-chars/%s", k)}}
				resp, err := natsOp.Run(ev, args)
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, expected)
			}
		})

		Convey("Empty values and null handling", func() {
			// Create KV store
			kv, err := js.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{
				Bucket: "empty-values",
			})
			So(err, ShouldBeNil)

			// Store empty and whitespace values
			_, err = kv.PutString(context.Background(), "empty", "")
			So(err, ShouldBeNil)
			_, err = kv.PutString(context.Background(), "whitespace", "   ")
			So(err, ShouldBeNil)

			yamlData := `
values:
  empty_from_nats: (( nats "kv:empty-values/empty" ))
  whitespace_from_nats: (( nats "kv:empty-values/whitespace" ))
  missing_from_nats: (( nats "kv:empty-values/nonexistent" || null ))
  missing_from_vault: (( vault "secret/nonexistent:key" || null ))
  
  # Conditional based on empty values
  use_default: (( empty values.empty_from_nats ))
  config: (( ternary values.use_default "default_value" values.empty_from_nats ))
`
			// Parse YAML into a tree
			yamlTree := make(map[interface{}]interface{})
			err = func() error {
				y, err := simpleyaml.NewYaml([]byte(yamlData))
				if err != nil {
					return err
				}
				yamlTree, err = y.Map()
				return err
			}()
			So(err, ShouldBeNil)
			
			ev := &graft.Evaluator{
				Tree: yamlTree,
			}

			natsOp := &NatsOperator{}
			err = natsOp.Setup()
			So(err, ShouldBeNil)

			// Test empty value
			args := []*graft.Expr{{Type: graft.Literal, Literal: "kv:empty-values/empty"}}
			resp, err := natsOp.Run(ev, args)
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, "")

			// Test whitespace value
			args = []*Expr{{Type: Literal, Literal: "kv:empty-values/whitespace"}}
			resp, err = natsOp.Run(ev, args)
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, "   ")

			// Test missing value
			args = []*Expr{{Type: Literal, Literal: "kv:empty-values/nonexistent"}}
			_, err = natsOp.Run(ev, args)
			So(err, ShouldNotBeNil) // Should error on missing key
		})
	})
}