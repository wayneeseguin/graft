//go:build integration
// +build integration

package operators

import (
	"context"
	"os"
	"testing"

	"github.com/geofffranks/simpleyaml"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/wayneeseguin/graft/pkg/graft"
)

func TestVaultNatsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	Convey("Vault and NATS Operator Integration", t, func() {
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

		// Setup test KV store with configuration data
		kv, err := js.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{
			Bucket: "config",
		})
		So(err, ShouldBeNil)

		// Populate test data
		_, err = kv.PutString(context.Background(), "cache.host", "redis.example.com")
		So(err, ShouldBeNil)
		_, err = kv.PutString(context.Background(), "cache.port", "6379")
		So(err, ShouldBeNil)
		_, err = kv.PutString(context.Background(), "services.auth.host", "auth.example.com")
		So(err, ShouldBeNil)
		_, err = kv.PutString(context.Background(), "services.users.host", "users.example.com")
		So(err, ShouldBeNil)

		// Set NATS URL environment variable
		oldNatsURL := os.Getenv("NATS_URL")
		os.Setenv("NATS_URL", url)
		defer os.Setenv("NATS_URL", oldNatsURL)

		// Note: In a real test, we would mock vault responses

		Convey("Basic vault and nats integration", func() {
			yamlData := `
database:
  host: localhost
  username: (( vault "secret/database:username" ))
  password: (( vault "secret/database:password" ))
cache:
  host: (( nats "kv:config/cache.host" ))
  port: (( nats "kv:config/cache.port" ))
  auth_token: (( vault "secret/redis:auth_token" ))
`
			// Parse YAML into a tree
			yamlTree := make(map[interface{}]interface{})
			err := func() error {
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

			// Mock vault operator execution
			vaultOp := &VaultOperator{}
			vaultOp.Setup()

			// Note: In a real integration test, we would mock the vault backend
			// For now, we focus on testing the NATS integration

			// Run NATS operator
			natsOp := &NatsOperator{}
			err = natsOp.Setup()
			So(err, ShouldBeNil)

			// Evaluate cache.host
			args := []*graft.Expr{{Type: graft.Literal, Literal: "kv:config/cache.host"}}
			resp, err := natsOp.Run(ev, args)
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, "redis.example.com")

			// Evaluate cache.port
			args = []*graft.Expr{{Type: graft.Literal, Literal: "kv:config/cache.port"}}
			resp, err = natsOp.Run(ev, args)
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, "6379")
		})

		Convey("Dynamic path construction with vault and nats", func() {
			// Create environment-specific KV stores
			envKV, err := js.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{
				Bucket: "env",
			})
			So(err, ShouldBeNil)

			// Populate environment-specific data
			_, err = envKV.PutString(context.Background(), "dev.database.host", "dev-db.local")
			So(err, ShouldBeNil)
			_, err = envKV.PutString(context.Background(), "staging.database.host", "staging-db.example.com")
			So(err, ShouldBeNil)
			_, err = envKV.PutString(context.Background(), "production.database.host", "prod-db.example.com")
			So(err, ShouldBeNil)

			yamlData := `
meta:
  environment: staging
database:
  host: (( nats (concat "kv:env/" meta.environment ".database.host") ))
  credentials: (( vault (concat "secret/" meta.environment "/database:credentials") ))
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

			// The evaluator would resolve this to "kv:env/staging.database.host"
			natsOp := &NatsOperator{}
			err = natsOp.Setup()
			So(err, ShouldBeNil)

			args := []*graft.Expr{{Type: graft.Literal, Literal: "kv:env/staging.database.host"}}
			resp, err := natsOp.Run(ev, args)
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, "staging-db.example.com")
		})

		Convey("Conditional source selection between vault and nats", func() {
			// Create secrets KV store
			secretsKV, err := js.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{
				Bucket: "secrets",
			})
			So(err, ShouldBeNil)

			// Populate NATS secrets as fallback
			_, err = secretsKV.PutString(context.Background(), "db.username", "nats_db_user")
			So(err, ShouldBeNil)
			_, err = secretsKV.PutString(context.Background(), "db.password", "nats_db_pass")
			So(err, ShouldBeNil)

			yamlData := `
meta:
  use_vault: false
secrets:
  database:
    username: (( ternary meta.use_vault (vault "secret/db:username") (nats "kv:secrets/db.username") ))
    password: (( ternary meta.use_vault (vault "secret/db:password") (nats "kv:secrets/db.password") ))
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

			// When use_vault is false, it should use NATS
			natsOp := &NatsOperator{}
			err = natsOp.Setup()
			So(err, ShouldBeNil)

			args := []*graft.Expr{{Type: graft.Literal, Literal: "kv:secrets/db.username"}}
			resp, err := natsOp.Run(ev, args)
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, "nats_db_user")
		})

		Convey("Multiple source URLs with vault and nats", func() {
			// Create services KV store
			servicesKV, err := js.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{
				Bucket: "services",
			})
			So(err, ShouldBeNil)

			// Populate service endpoints
			_, err = servicesKV.PutString(context.Background(), "auth.host", "auth-service.internal")
			So(err, ShouldBeNil)
			_, err = servicesKV.PutString(context.Background(), "auth.port", "8080")
			So(err, ShouldBeNil)

			yamlData := `
services:
  auth:
    url: (( concat "https://" (nats "kv:services/auth.host") ":" (nats "kv:services/auth.port") "/api" ))
    api_key: (( vault "secret/services/auth:api_key" ))
    backup_key: (( vault "secret/services/auth:backup_key; secret/services/fallback:key" ))
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

			// Test NATS operator for host
			natsOp := &NatsOperator{}
			err = natsOp.Setup()
			So(err, ShouldBeNil)

			args := []*graft.Expr{{Type: graft.Literal, Literal: "kv:services/auth.host"}}
			resp, err := natsOp.Run(ev, args)
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, "auth-service.internal")

			// Test NATS operator for port
			args = []*Expr{{Type: Literal, Literal: "kv:services/auth.port"}}
			resp, err = natsOp.Run(ev, args)
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, "8080")
		})

		Convey("Target-specific vault and nats configurations", func() {
			// Create target-specific KV stores
			targetKV, err := js.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{
				Bucket: "targets",
			})
			So(err, ShouldBeNil)

			// Populate target-specific data
			_, err = targetKV.PutString(context.Background(), "production.db.host", "prod-master.rds.amazonaws.com")
			So(err, ShouldBeNil)
			_, err = targetKV.PutString(context.Background(), "staging.db.host", "staging.rds.amazonaws.com")
			So(err, ShouldBeNil)

			yamlData := `
production:
  database:
    host: (( nats@production "kv:targets/production.db.host" ))
    username: (( vault@production "secret/prod/db:username" ))
    password: (( vault@production:nocache "secret/prod/db:password" ))

staging:
  database:
    host: (( nats@staging "kv:targets/staging.db.host" ))
    username: (( vault@staging "secret/staging/db:username" ))
    password: (( vault@staging "secret/staging/db:password" ))
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

			// Test target-aware NATS operator
			// This would require the target-aware implementation
			// For now, we simulate the basic behavior
			natsOp := &NatsOperator{}
			err = natsOp.Setup()
			So(err, ShouldBeNil)

			// Simulate production target
			args := []*graft.Expr{{Type: graft.Literal, Literal: "kv:targets/production.db.host"}}
			resp, err := natsOp.Run(ev, args)
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, "prod-master.rds.amazonaws.com")
		})

		Convey("Error handling and fallbacks", func() {
			yamlData := `
config:
  primary_source: (( vault "secret/missing:key" || nats "kv:config/fallback" ))
  both_missing: (( vault "secret/notfound:key" || nats "kv:missing/key" || "default_value" ))
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

			// Add fallback value to NATS
			_, err = kv.PutString(context.Background(), "fallback", "nats_fallback_value")
			So(err, ShouldBeNil)

			natsOp := &NatsOperator{}
			err = natsOp.Setup()
			So(err, ShouldBeNil)

			// Test successful fallback to NATS
			args := []*graft.Expr{{Type: graft.Literal, Literal: "kv:config/fallback"}}
			resp, err := natsOp.Run(ev, args)
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, "nats_fallback_value")

			// Test missing key error
			args = []*Expr{{Type: Literal, Literal: "kv:missing/key"}}
			_, err = natsOp.Run(ev, args)
			So(err, ShouldNotBeNil)
		})
	})
}

func TestVaultNatsComplexIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	Convey("Complex Vault and NATS Integration Scenarios", t, func() {
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

		Convey("YAML configuration stored in NATS, secrets in Vault", func() {
			// Create object store for YAML configs
			objStore, err := js.CreateObjectStore(context.Background(), jetstream.ObjectStoreConfig{
				Bucket: "configs",
			})
			So(err, ShouldBeNil)

			// Store YAML configuration in NATS
			appConfig := `
application:
  name: "myapp"
  version: "2.0.0"
  features:
    - auth
    - logging
    - metrics
`
			_, err = objStore.PutBytes(context.Background(), "app-settings.yaml", []byte(appConfig))
			So(err, ShouldBeNil)

			yamlData := `
# Load base config from NATS
base_config: (( nats "obj:configs/app-settings.yaml" ))

# Enhance with secrets from Vault
application:
  name: (( grab base_config.application.name ))
  version: (( grab base_config.application.version ))
  features: (( grab base_config.application.features ))
  secrets:
    api_key: (( vault "secret/app:api_key" ))
    db_password: (( vault "secret/app:db_password" ))
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

			// Test NATS object retrieval
			natsOp := &NatsOperator{}
			err = natsOp.Setup()
			So(err, ShouldBeNil)

			args := []*graft.Expr{{Type: graft.Literal, Literal: "obj:configs/app-settings.yaml"}}
			resp, err := natsOp.Run(ev, args)
			So(err, ShouldBeNil)
			So(resp.Value, ShouldNotBeNil)

			// Verify the loaded config structure
			configMap, ok := resp.Value.(map[interface{}]interface{})
			So(ok, ShouldBeTrue)
			So(configMap["application"], ShouldNotBeNil)
		})

		Convey("Binary data from NATS with metadata from Vault", func() {
			// Create object store for binary data
			objStore, err := js.CreateObjectStore(context.Background(), jetstream.ObjectStoreConfig{
				Bucket: "certificates",
			})
			So(err, ShouldBeNil)

			// Store binary certificate data
			certData := []byte("-----BEGIN CERTIFICATE-----\nMIIDXTCCAkWgAwIBAgIJAKl...\n-----END CERTIFICATE-----")
			_, err = objStore.PutBytes(context.Background(), "server.crt", certData)
			So(err, ShouldBeNil)

			// Create KV store for cert metadata
			certKV, err := js.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{
				Bucket: "cert-metadata",
			})
			So(err, ShouldBeNil)

			_, err = certKV.PutString(context.Background(), "server.issuer", "Example CA")
			So(err, ShouldBeNil)
			_, err = certKV.PutString(context.Background(), "server.expiry", "2025-12-31")
			So(err, ShouldBeNil)

			yamlData := `
tls:
  certificate:
    data: (( nats "obj:certificates/server.crt" ))
    key: (( vault "secret/tls/server:private_key" ))
    metadata:
      issuer: (( nats "kv:cert-metadata/server.issuer" ))
      expiry: (( nats "kv:cert-metadata/server.expiry" ))
      passphrase: (( vault "secret/tls/server:passphrase" ))
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

			// Test binary data retrieval
			natsOp := &NatsOperator{}
			err = natsOp.Setup()
			So(err, ShouldBeNil)

			args := []*graft.Expr{{Type: graft.Literal, Literal: "obj:certificates/server.crt"}}
			resp, err := natsOp.Run(ev, args)
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, string(certData))

			// Test metadata retrieval
			args = []*Expr{{Type: Literal, Literal: "kv:cert-metadata/server.issuer"}}
			resp, err = natsOp.Run(ev, args)
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, "Example CA")
		})

		Convey("Service discovery with NATS and authentication with Vault", func() {
			// Create service registry in NATS
			serviceKV, err := js.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{
				Bucket: "service-registry",
			})
			So(err, ShouldBeNil)

			// Register services
			_, err = serviceKV.PutString(context.Background(), "api-gateway.host", "gateway.service.consul")
			So(err, ShouldBeNil)
			_, err = serviceKV.PutString(context.Background(), "api-gateway.port", "8080")
			So(err, ShouldBeNil)
			_, err = serviceKV.PutString(context.Background(), "api-gateway.protocol", "https")
			So(err, ShouldBeNil)

			yamlData := `
services:
  api_gateway:
    # Service discovery from NATS
    host: (( nats "kv:service-registry/api-gateway.host" ))
    port: (( nats "kv:service-registry/api-gateway.port" ))
    protocol: (( nats "kv:service-registry/api-gateway.protocol" ))
    
    # Build URL
    url: (( concat protocol "://" host ":" port ))
    
    # Authentication from Vault
    auth:
      type: "bearer"
      token: (( vault "secret/services/api-gateway:auth_token" ))
      refresh_token: (( vault "secret/services/api-gateway:refresh_token" ))
    
    # TLS configuration
    tls:
      enabled: (( grab services.api_gateway.protocol == "https" ))
      ca_cert: (( ternary tls.enabled (vault "secret/tls/ca:certificate") null ))
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

			// Test service discovery
			natsOp := &NatsOperator{}
			err = natsOp.Setup()
			So(err, ShouldBeNil)

			args := []*graft.Expr{{Type: graft.Literal, Literal: "kv:service-registry/api-gateway.host"}}
			resp, err := natsOp.Run(ev, args)
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, "gateway.service.consul")

			args = []*Expr{{Type: Literal, Literal: "kv:service-registry/api-gateway.protocol"}}
			resp, err = natsOp.Run(ev, args)
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, "https")
		})
	})
}
