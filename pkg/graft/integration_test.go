package graft

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestIntegration_BasicMergeAndEvaluate(t *testing.T) {
	Convey("Given base and override configurations", t, func() {
		engine, err := NewEngine()
		So(err, ShouldBeNil)

		baseConfig := []byte(`
meta:
  app_name: "myapp"
  version: "1.0.0"
  environment: "dev"

database:
  host: "localhost"
  port: 5432
  name: "myapp_dev"

features:
  auth: true
  logging: false
`)

		overrideConfig := []byte(`
database:
  host: "prod.example.com"
  ssl: true

features:
  logging: true
  metrics: true

deployment:
  replicas: 1
  image: '(( concat meta.app_name ":" meta.version ))'
`)

		Convey("When merging and evaluating", func() {
			baseDoc, err := engine.ParseYAML(baseConfig)
			So(err, ShouldBeNil)

			overrideDoc, err := engine.ParseYAML(overrideConfig)
			So(err, ShouldBeNil)

			ctx := context.Background()
			merged, err := engine.Merge(ctx, baseDoc, overrideDoc).Execute()
			So(err, ShouldBeNil)

			result, err := engine.Evaluate(ctx, merged)
			So(err, ShouldBeNil)

			Convey("Then the configuration should be properly merged and evaluated", func() {
				// Check merged values
				host, err := result.GetString("database.host")
				So(err, ShouldBeNil)
				So(host, ShouldEqual, "prod.example.com")

				ssl, err := result.GetBool("database.ssl")
				So(err, ShouldBeNil)
				So(ssl, ShouldEqual, true)

				port, err := result.GetInt("database.port")
				So(err, ShouldBeNil)
				So(port, ShouldEqual, 5432)

				// Check evaluated operators
				image, err := result.GetString("deployment.image")
				So(err, ShouldBeNil)
				So(image, ShouldEqual, "myapp:1.0.0")

				replicas, err := result.GetInt("deployment.replicas")
				So(err, ShouldBeNil)
				So(replicas, ShouldEqual, 1) // dev environment
			})
		})
	})
}

func TestIntegration_ComplexMergeWithArrays(t *testing.T) {
	Convey("Given configurations with array merging", t, func() {
		engine, err := NewEngine()
		So(err, ShouldBeNil)

		baseConfig := []byte(`
services:
  - name: "api"
    port: 8080
  - name: "db"
    port: 5432

middleware:
  - auth
  - logging
`)

		overrideConfig := []byte(`
services:
  - (( replace ))
  - name: "cache"
    port: 6379

middleware:
  - name: cors
    enabled: true
  - metrics
  - tracing
`)

		Convey("When merging with explicit replace operator", func() {
			baseDoc, err := engine.ParseYAML(baseConfig)
			So(err, ShouldBeNil)

			overrideDoc, err := engine.ParseYAML(overrideConfig)
			So(err, ShouldBeNil)

			ctx := context.Background()
			result, err := engine.Merge(ctx, baseDoc, overrideDoc).Execute()
			So(err, ShouldBeNil)

			Convey("Then services array should be replaced", func() {
				services, err := result.GetSlice("services")
				So(err, ShouldBeNil)
				So(len(services), ShouldEqual, 1)
				
				service := services[0].(map[interface{}]interface{})
				So(service["name"], ShouldEqual, "cache")
				So(service["port"], ShouldEqual, 6379)
			})
			
			Convey("And middleware array should be replaced due to mixed types", func() {
				// When arrays contain mixed types (strings and objects), graft replaces
				// instead of merging because it can't match simple strings to objects
				middleware, err := result.GetSlice("middleware")
				So(err, ShouldBeNil)
				So(len(middleware), ShouldEqual, 3) // cors, metrics, tracing
				
				// Verify the content
				corsMiddleware := middleware[0].(map[interface{}]interface{})
				So(corsMiddleware["name"], ShouldEqual, "cors")
				So(corsMiddleware["enabled"], ShouldEqual, true)
				So(middleware[1], ShouldEqual, "metrics")
				So(middleware[2], ShouldEqual, "tracing")
			})
		})
		
		Convey("When merging with ReplaceArrays strategy", func() {
			baseDoc, err := engine.ParseYAML(baseConfig)
			So(err, ShouldBeNil)

			overrideDoc, err := engine.ParseYAML(overrideConfig)
			So(err, ShouldBeNil)

			ctx := context.Background()
			result, err := engine.Merge(ctx, baseDoc, overrideDoc).
				WithArrayMergeStrategy(ReplaceArrays).
				Execute()
			So(err, ShouldBeNil)

			Convey("Then both arrays should be replaced", func() {
				services, err := result.GetSlice("services")
				So(err, ShouldBeNil)
				So(len(services), ShouldEqual, 1)

				middleware, err := result.GetSlice("middleware")
				So(err, ShouldBeNil)
				So(len(middleware), ShouldEqual, 3)
			})
		})
	})
}

func TestIntegration_PruningWorkflow(t *testing.T) {
	Convey("Given a configuration with sensitive data", t, func() {
		engine, err := NewEngine()
		So(err, ShouldBeNil)

		config := []byte(`
app:
  name: "myapp"
  version: "1.0.0"

database:
  host: "localhost"
  port: 5432
  password: "secret123"

secrets:
  api_key: "super-secret-key"
  db_password: "another-secret"

public_config:
  timeout: 30
  retries: 3
`)

		Convey("When processing with pruning", func() {
			doc, err := engine.ParseYAML(config)
			So(err, ShouldBeNil)

			ctx := context.Background()
			result, err := engine.Merge(ctx, doc).
				WithPrune("database.password").
				WithPrune("secrets").
				Execute()
			So(err, ShouldBeNil)

			Convey("Then sensitive data should be removed", func() {
				// Public data should remain
				name, err := result.GetString("app.name")
				So(err, ShouldBeNil)
				So(name, ShouldEqual, "myapp")

				timeout, err := result.GetInt("public_config.timeout")
				So(err, ShouldBeNil)
				So(timeout, ShouldEqual, 30)

				// Sensitive data should be removed
				_, err = result.Get("database.password")
				So(err, ShouldNotBeNil)

				_, err = result.Get("secrets")
				So(err, ShouldNotBeNil)

				// Other database config should remain
				host, err := result.GetString("database.host")
				So(err, ShouldBeNil)
				So(host, ShouldEqual, "localhost")
			})
		})
	})
}

func TestIntegration_ConditionalConfiguration(t *testing.T) {
	Convey("Given environment-specific configuration", t, func() {
		engine, err := NewEngine()
		So(err, ShouldBeNil)

		config := []byte(`
meta:
  environment: "production"

app:
  name: "myapp"
  debug: false
  log_level: "error"

database:
  pool_size: 20
  ssl: true

features:
  metrics: true
  profiling: false
`)

		Convey("When evaluating for production environment", func() {
			doc, err := engine.ParseYAML(config)
			So(err, ShouldBeNil)

			ctx := context.Background()
			result, err := engine.Evaluate(ctx, doc)
			So(err, ShouldBeNil)

			Convey("Then production settings should be applied", func() {
				debug, err := result.GetBool("app.debug")
				So(err, ShouldBeNil)
				So(debug, ShouldEqual, false)

				logLevel, err := result.GetString("app.log_level")
				So(err, ShouldBeNil)
				So(logLevel, ShouldEqual, "error")

				poolSize, err := result.GetInt("database.pool_size")
				So(err, ShouldBeNil)
				So(poolSize, ShouldEqual, 20)

				ssl, err := result.GetBool("database.ssl")
				So(err, ShouldBeNil)
				So(ssl, ShouldEqual, true)

				metrics, err := result.GetBool("features.metrics")
				So(err, ShouldBeNil)
				So(metrics, ShouldEqual, true)

				profiling, err := result.GetBool("features.profiling")
				So(err, ShouldBeNil)
				So(profiling, ShouldEqual, false)
			})
		})
	})
}

func TestIntegration_ErrorHandling(t *testing.T) {
	Convey("Given various error scenarios", t, func() {
		engine, err := NewEngine()
		So(err, ShouldBeNil)

		Convey("When document has syntax errors", func() {
			invalidYAML := []byte(`
name: test
  invalid: indentation
`)
			doc, err := engine.ParseYAML(invalidYAML)

			Convey("Then it should return a parse error", func() {
				So(err, ShouldNotBeNil)
				So(doc, ShouldBeNil)

				graftErr, ok := err.(*GraftError)
				So(ok, ShouldBeTrue)
				So(graftErr.Type, ShouldEqual, ParseError)
			})
		})

		Convey("When evaluation has unknown operators", func() {
			invalidOperator := []byte(`
result: (( unknown_operator "test" ))
`)
			doc, err := engine.ParseYAML(invalidOperator)
			So(err, ShouldBeNil)

			ctx := context.Background()
			result, err := engine.Evaluate(ctx, doc)

			Convey("Then it should succeed but leave operator unevaluated", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)

				val, err := result.GetString("result")
				So(err, ShouldBeNil)
				So(val, ShouldEqual, "(( unknown_operator \"test\" ))")
			})
		})

		Convey("When evaluation has reference errors", func() {
			invalidReference := []byte(`
result: (( grab nonexistent.path ))
`)
			doc, err := engine.ParseYAML(invalidReference)
			So(err, ShouldBeNil)

			ctx := context.Background()
			result, err := engine.Evaluate(ctx, doc)

			Convey("Then it should return a reference error", func() {
				So(err, ShouldNotBeNil)
				So(result, ShouldBeNil)

				// The evaluator returns MultiError, not GraftError
				multiErr, ok := err.(MultiError)
				So(ok, ShouldBeTrue)
				So(len(multiErr.Errors), ShouldBeGreaterThan, 0)
				So(multiErr.Error(), ShouldContainSubstring, "could not be found")
			})
		})
	})
}

func TestIntegration_MultiDocumentProcessing(t *testing.T) {
	Convey("Given multiple documents to process", t, func() {
		engine, err := NewEngine()
		So(err, ShouldBeNil)

		configs := [][]byte{
			[]byte(`
meta:
  app: "myapp"
  team: "platform"
`),
			[]byte(`
database:
  host: "localhost"
  name: (( concat meta.app "_db" ))
`),
			[]byte(`
services:
  api:
    name: (( grab meta.app ))
    team: (( grab meta.team ))
    database: (( grab database.name ))
`),
		}

		Convey("When processing multiple documents in sequence", func() {
			var result Document
			ctx := context.Background()

			for i, configData := range configs {
				doc, err := engine.ParseYAML(configData)
				So(err, ShouldBeNil)

				if i == 0 {
					result = doc
				} else {
					result, err = engine.Merge(ctx, result, doc).Execute()
					So(err, ShouldBeNil)
				}
			}

			finalResult, err := engine.Evaluate(ctx, result)
			So(err, ShouldBeNil)

			Convey("Then all documents should be properly merged and evaluated", func() {
				appName, err := finalResult.GetString("services.api.name")
				So(err, ShouldBeNil)
				So(appName, ShouldEqual, "myapp")

				team, err := finalResult.GetString("services.api.team")
				So(err, ShouldBeNil)
				So(team, ShouldEqual, "platform")

				dbName, err := finalResult.GetString("services.api.database")
				So(err, ShouldBeNil)
				So(dbName, ShouldEqual, "myapp_db")
			})
		})
	})
}