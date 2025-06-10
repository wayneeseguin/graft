package graft

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCherryPickWithComplexDependencies(t *testing.T) {
	Convey("Cherry-pick with complex dependency chains", t, func() {
		Convey("Should evaluate transitive dependencies", func() {
			// Create a document with complex dependency chains
			doc := NewDocument(map[interface{}]interface{}{
				"base": map[interface{}]interface{}{
					"url": "(( concat config.protocol \"://example.com\" ))",
				},
				"config": map[interface{}]interface{}{
					"protocol": "(( grab env.protocol || \"https\" ))",
					"timeout":  30,
				},
				"env": map[interface{}]interface{}{
					"protocol": "https",
					"debug":    false,
				},
				"api": map[interface{}]interface{}{
					"endpoint": "(( concat base.url \"/api/v1\" ))",
					"auth_url": "(( concat base.url \"/auth\" ))",
				},
				"unused": map[interface{}]interface{}{
					"error": "(( grab nonexistent.value ))", // Should not be evaluated
				},
			})

			engine, err := NewEngine()
			So(err, ShouldBeNil)

			// Cherry-pick only api section - should pull in base, config, and env
			result, err := engine.Merge(context.Background(), doc).
				WithCherryPick("api").
				Execute()

			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)

			// Check that operators were evaluated correctly
			data := result.RawData().(map[interface{}]interface{})
			api := data["api"].(map[interface{}]interface{})
			So(api["endpoint"], ShouldEqual, "https://example.com/api/v1")
			So(api["auth_url"], ShouldEqual, "https://example.com/auth")

			// Unused section should not be present
			So(data["unused"], ShouldBeNil)
		})

		Convey("Should handle circular dependencies gracefully", func() {
			// Create a document with potential circular refs
			doc := NewDocument(map[interface{}]interface{}{
				"a": map[interface{}]interface{}{
					"value": "(( grab b.value ))",
				},
				"b": map[interface{}]interface{}{
					"value": "(( grab c.value ))",
				},
				"c": map[interface{}]interface{}{
					"value": "final",
					"ref":   "(( grab a.value ))", // Creates a cycle
				},
			})

			engine, err := NewEngine()
			So(err, ShouldBeNil)

			// Cherry-pick a - should handle the cycle properly
			result, err := engine.Merge(context.Background(), doc).
				WithCherryPick("a").
				Execute()

			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)

			// Check result
			data := result.RawData().(map[interface{}]interface{})
			a := data["a"].(map[interface{}]interface{})
			So(a["value"], ShouldEqual, "final")
		})

		Convey("Should handle operators within arrays", func() {
			doc := NewDocument(map[interface{}]interface{}{
				"databases": []interface{}{
					map[interface{}]interface{}{
						"name": "primary",
						"host": "(( grab hosts.primary ))",
						"port": "(( grab defaults.db_port ))",
					},
					map[interface{}]interface{}{
						"name": "secondary",
						"host": "(( grab hosts.secondary ))",
						"port": "(( grab defaults.db_port ))",
					},
				},
				"hosts": map[interface{}]interface{}{
					"primary":   "db1.example.com",
					"secondary": "db2.example.com",
				},
				"defaults": map[interface{}]interface{}{
					"db_port":    5432,
					"cache_port": 6379,
				},
				"unused": map[interface{}]interface{}{
					"data": "(( grab missing ))",
				},
			})

			engine, err := NewEngine()
			So(err, ShouldBeNil)

			// Cherry-pick databases - should pull in hosts and defaults.db_port
			result, err := engine.Merge(context.Background(), doc).
				WithCherryPick("databases").
				Execute()

			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)

			// Check that array operators were evaluated
			data := result.RawData().(map[interface{}]interface{})
			databases := data["databases"].([]interface{})
			So(len(databases), ShouldEqual, 2)

			primary := databases[0].(map[interface{}]interface{})
			So(primary["host"], ShouldEqual, "db1.example.com")
			So(primary["port"], ShouldEqual, 5432)

			secondary := databases[1].(map[interface{}]interface{})
			So(secondary["host"], ShouldEqual, "db2.example.com")
			So(secondary["port"], ShouldEqual, 5432)

			// Unused should not be present
			So(data["unused"], ShouldBeNil)
		})

		Convey("Should handle deeply nested operator dependencies", func() {
			doc := NewDocument(map[interface{}]interface{}{
				"level1": map[interface{}]interface{}{
					"level2": map[interface{}]interface{}{
						"level3": map[interface{}]interface{}{
							"value": "(( grab level4.level5.level6.final ))",
						},
					},
				},
				"level4": map[interface{}]interface{}{
					"level5": map[interface{}]interface{}{
						"level6": map[interface{}]interface{}{
							"final": "(( concat prefix.value suffix.value ))",
						},
					},
				},
				"prefix": map[interface{}]interface{}{
					"value": "start-",
				},
				"suffix": map[interface{}]interface{}{
					"value": "-end",
				},
				"unrelated": map[interface{}]interface{}{
					"error": "(( grab does.not.exist ))",
				},
			})

			engine, err := NewEngine()
			So(err, ShouldBeNil)

			// Cherry-pick level1 - should pull in entire dependency chain
			result, err := engine.Merge(context.Background(), doc).
				WithCherryPick("level1").
				Execute()

			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)

			// Check deep evaluation
			data := result.RawData().(map[interface{}]interface{})
			level1 := data["level1"].(map[interface{}]interface{})
			level2 := level1["level2"].(map[interface{}]interface{})
			level3 := level2["level3"].(map[interface{}]interface{})
			So(level3["value"], ShouldEqual, "start--end")

			// Unrelated should not be present
			So(data["unrelated"], ShouldBeNil)
		})

		Convey("Should handle conditional operators", func() {
			doc := NewDocument(map[interface{}]interface{}{
				"environment": "production",
				"is_prod":     true,
				"features": map[interface{}]interface{}{
					"ssl":      "(( grab is_prod ))",
					"debug":    false,
					"replicas": "(( is_prod ? 3 : 1 ))",
				},
				"config": map[interface{}]interface{}{
					"url": "(( features.ssl ? \"https://api.example.com\" : \"http://api.example.com\" ))",
				},
				"unused": map[interface{}]interface{}{
					"bad": "(( grab nowhere ))",
				},
			})

			engine, err := NewEngine()
			So(err, ShouldBeNil)

			// Cherry-pick config - should evaluate conditional chain
			result, err := engine.Merge(context.Background(), doc).
				WithCherryPick("config").
				Execute()

			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)

			// Check conditional evaluation
			data := result.RawData().(map[interface{}]interface{})
			config := data["config"].(map[interface{}]interface{})
			So(config["url"], ShouldEqual, "https://api.example.com")

			// Unused should not be present
			So(data["unused"], ShouldBeNil)
		})

		Convey("Should handle operators with array paths", func() {
			doc := NewDocument(map[interface{}]interface{}{
				"services": []interface{}{
					map[interface{}]interface{}{
						"name": "web",
						"instances": []interface{}{
							map[interface{}]interface{}{
								"id":   "web-1",
								"port": "(( grab defaults.web_port ))",
							},
							map[interface{}]interface{}{
								"id":   "web-2",
								"port": "(( grab services.0.instances.0.port ))", // Reference to sibling
							},
						},
					},
				},
				"defaults": map[interface{}]interface{}{
					"web_port": 8080,
					"api_port": 9090,
				},
				"monitoring": map[interface{}]interface{}{
					"targets": []interface{}{
						"(( grab services.0.instances.0.id ))",
						"(( grab services.0.instances.1.id ))",
					},
				},
				"unused": map[interface{}]interface{}{
					"error": "(( grab fail ))",
				},
			})

			engine, err := NewEngine()
			So(err, ShouldBeNil)

			// Cherry-pick monitoring - should resolve array path dependencies
			result, err := engine.Merge(context.Background(), doc).
				WithCherryPick("monitoring").
				Execute()

			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)

			// Check array path resolution
			data := result.RawData().(map[interface{}]interface{})
			monitoring := data["monitoring"].(map[interface{}]interface{})
			targets := monitoring["targets"].([]interface{})
			So(len(targets), ShouldEqual, 2)
			So(targets[0], ShouldEqual, "web-1")
			So(targets[1], ShouldEqual, "web-2")

			// Unused should not be present
			So(data["unused"], ShouldBeNil)
		})
	})
}

func TestMultipleCherryPickPaths(t *testing.T) {
	Convey("Cherry-pick with multiple paths", t, func() {
		Convey("Should evaluate operators under multiple paths", func() {
			doc := NewDocument(map[interface{}]interface{}{
				"database": map[interface{}]interface{}{
					"host": "(( grab defaults.db_host ))",
					"port": "(( grab defaults.db_port ))",
					"name": "myapp",
				},
				"cache": map[interface{}]interface{}{
					"host": "(( grab defaults.cache_host ))",
					"port": "(( grab defaults.cache_port ))",
					"ttl":  3600,
				},
				"defaults": map[interface{}]interface{}{
					"db_host":    "localhost",
					"db_port":    5432,
					"cache_host": "redis.local",
					"cache_port": 6379,
					"unused":     "value",
				},
				"monitoring": map[interface{}]interface{}{
					"enabled": true,
					"url":     "(( grab invalid.path ))", // Should not be evaluated
				},
			})

			engine, err := NewEngine()
			So(err, ShouldBeNil)

			// Cherry-pick multiple paths
			result, err := engine.Merge(context.Background(), doc).
				WithCherryPick("database", "cache").
				Execute()

			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)

			// Check that both paths are included and evaluated
			data := result.RawData().(map[interface{}]interface{})

			database := data["database"].(map[interface{}]interface{})
			So(database["host"], ShouldEqual, "localhost")
			So(database["port"], ShouldEqual, 5432)
			So(database["name"], ShouldEqual, "myapp")

			cache := data["cache"].(map[interface{}]interface{})
			So(cache["host"], ShouldEqual, "redis.local")
			So(cache["port"], ShouldEqual, 6379)
			So(cache["ttl"], ShouldEqual, 3600)

			// Monitoring should not be included
			So(data["monitoring"], ShouldBeNil)
			// Note: defaults might be included due to dependencies, but unused field doesn't matter
		})

		Convey("Should handle overlapping dependencies", func() {
			doc := NewDocument(map[interface{}]interface{}{
				"app1": map[interface{}]interface{}{
					"url":     "(( concat shared.protocol \"://\" shared.domain \"/app1\" ))",
					"timeout": "(( grab shared.timeout ))",
				},
				"app2": map[interface{}]interface{}{
					"url":     "(( concat shared.protocol \"://\" shared.domain \"/app2\" ))",
					"retries": "(( grab shared.retries ))",
				},
				"shared": map[interface{}]interface{}{
					"protocol": "https",
					"domain":   "example.com",
					"timeout":  30,
					"retries":  3,
				},
				"other": map[interface{}]interface{}{
					"data": "(( grab missing ))",
				},
			})

			engine, err := NewEngine()
			So(err, ShouldBeNil)

			// Cherry-pick both apps - should share the same dependencies
			result, err := engine.Merge(context.Background(), doc).
				WithCherryPick("app1", "app2").
				Execute()

			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)

			data := result.RawData().(map[interface{}]interface{})

			app1 := data["app1"].(map[interface{}]interface{})
			So(app1["url"], ShouldEqual, "https://example.com/app1")
			So(app1["timeout"], ShouldEqual, 30)

			app2 := data["app2"].(map[interface{}]interface{})
			So(app2["url"], ShouldEqual, "https://example.com/app2")
			So(app2["retries"], ShouldEqual, 3)

			// Other should not be included
			So(data["other"], ShouldBeNil)
		})

		Convey("Should handle nested paths with multiple picks", func() {
			doc := NewDocument(map[interface{}]interface{}{
				"services": map[interface{}]interface{}{
					"web": map[interface{}]interface{}{
						"port":     8080,
						"replicas": "(( grab config.web_replicas ))",
					},
					"api": map[interface{}]interface{}{
						"port":     9090,
						"replicas": "(( grab config.api_replicas ))",
					},
					"worker": map[interface{}]interface{}{
						"replicas": "(( grab config.worker_replicas ))",
					},
				},
				"config": map[interface{}]interface{}{
					"web_replicas":    3,
					"api_replicas":    2,
					"worker_replicas": 5,
				},
				"monitoring": map[interface{}]interface{}{
					"dashboards": map[interface{}]interface{}{
						"web": "(( grab services.web.port ))",
						"api": "(( grab services.api.port ))",
					},
				},
				"unused": map[interface{}]interface{}{
					"error": "(( grab fail ))",
				},
			})

			engine, err := NewEngine()
			So(err, ShouldBeNil)

			// Cherry-pick nested paths
			result, err := engine.Merge(context.Background(), doc).
				WithCherryPick("services.web", "services.api", "monitoring.dashboards").
				Execute()

			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)

			data := result.RawData().(map[interface{}]interface{})

			// Check services structure
			services := data["services"].(map[interface{}]interface{})
			web := services["web"].(map[interface{}]interface{})
			So(web["port"], ShouldEqual, 8080)
			So(web["replicas"], ShouldEqual, 3)

			api := services["api"].(map[interface{}]interface{})
			So(api["port"], ShouldEqual, 9090)
			So(api["replicas"], ShouldEqual, 2)

			// Worker should not be included
			So(services["worker"], ShouldBeNil)

			// Check monitoring structure
			monitoring := data["monitoring"].(map[interface{}]interface{})
			dashboards := monitoring["dashboards"].(map[interface{}]interface{})
			So(dashboards["web"], ShouldEqual, 8080)
			So(dashboards["api"], ShouldEqual, 9090)

			// Unused should not be included
			So(data["unused"], ShouldBeNil)
		})

		Convey("Should handle array paths in multiple picks", func() {
			doc := NewDocument(map[interface{}]interface{}{
				"servers": map[interface{}]interface{}{
					"web": map[interface{}]interface{}{
						"host": "(( concat hosts.prefix \"-web.example.com\" ))",
						"port": 8080,
					},
					"api": map[interface{}]interface{}{
						"host": "(( concat hosts.prefix \"-api.example.com\" ))",
						"port": 9090,
					},
					"db": map[interface{}]interface{}{
						"host": "(( concat hosts.prefix \"-db.example.com\" ))",
						"port": 5432,
					},
				},
				"hosts": map[interface{}]interface{}{
					"prefix": "prod",
				},
				"monitoring": map[interface{}]interface{}{
					"targets": []interface{}{
						"(( grab servers.web.host ))",
						"(( grab servers.api.host ))",
					},
				},
				"unused": map[interface{}]interface{}{
					"data": "(( grab missing ))",
				},
			})

			engine, err := NewEngine()
			So(err, ShouldBeNil)

			// Cherry-pick specific servers and monitoring
			result, err := engine.Merge(context.Background(), doc).
				WithCherryPick("servers.web", "servers.api", "monitoring").
				Execute()

			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)

			data := result.RawData().(map[interface{}]interface{})

			// Check servers
			servers := data["servers"].(map[interface{}]interface{})

			web := servers["web"].(map[interface{}]interface{})
			So(web["host"], ShouldEqual, "prod-web.example.com")
			So(web["port"], ShouldEqual, 8080)

			api := servers["api"].(map[interface{}]interface{})
			So(api["host"], ShouldEqual, "prod-api.example.com")
			So(api["port"], ShouldEqual, 9090)

			// db should not be included
			So(servers["db"], ShouldBeNil)

			// Check monitoring
			monitoring := data["monitoring"].(map[interface{}]interface{})
			targets := monitoring["targets"].([]interface{})
			So(len(targets), ShouldEqual, 2)
			So(targets[0], ShouldEqual, "prod-web.example.com")
			So(targets[1], ShouldEqual, "prod-api.example.com")

			// Unused should not be included
			So(data["unused"], ShouldBeNil)
		})

		Convey("Should handle empty cherry-pick list", func() {
			doc := NewDocument(map[interface{}]interface{}{
				"data": map[interface{}]interface{}{
					"value": "(( grab source.value ))",
				},
				"source": map[interface{}]interface{}{
					"value": 42,
				},
			})

			engine, err := NewEngine()
			So(err, ShouldBeNil)

			// Empty cherry-pick should include everything
			result, err := engine.Merge(context.Background(), doc).
				WithCherryPick().
				Execute()

			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)

			// Everything should be evaluated as normal
			data := result.RawData().(map[interface{}]interface{})
			dataMap := data["data"].(map[interface{}]interface{})
			So(dataMap["value"], ShouldEqual, 42)
		})
	})
}

func TestCherryPickWithPruneOperator(t *testing.T) {
	Convey("Cherry-pick interaction with prune operator", t, func() {
		Convey("Should handle prune operator within cherry-picked paths", func() {
			doc := NewDocument(map[interface{}]interface{}{
				"database": map[interface{}]interface{}{
					"host":     "localhost",
					"port":     5432,
					"password": "(( prune ))",
					"username": "admin",
				},
				"cache": map[interface{}]interface{}{
					"host": "redis.local",
					"ttl":  "(( prune ))",
				},
				"unused": map[interface{}]interface{}{
					"secret": "(( prune ))",
					"data":   "value",
				},
			})

			engine, err := NewEngine()
			So(err, ShouldBeNil)

			// Cherry-pick database and cache
			result, err := engine.Merge(context.Background(), doc).
				WithCherryPick("database", "cache").
				Execute()

			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)

			// Check that prune was applied within cherry-picked paths
			data := result.RawData().(map[interface{}]interface{})

			database := data["database"].(map[interface{}]interface{})
			So(database["host"], ShouldEqual, "localhost")
			So(database["port"], ShouldEqual, 5432)
			So(database["username"], ShouldEqual, "admin")
			So(database["password"], ShouldBeNil) // Should be pruned

			cache := data["cache"].(map[interface{}]interface{})
			So(cache["host"], ShouldEqual, "redis.local")
			So(cache["ttl"], ShouldBeNil) // Should be pruned

			// Unused should not be included
			So(data["unused"], ShouldBeNil)
		})

		Convey("Should handle prune references to cherry-picked paths", func() {
			doc := NewDocument(map[interface{}]interface{}{
				"config": map[interface{}]interface{}{
					"api_key": "(( grab secrets.api_key ))",
					"url":     "https://api.example.com",
				},
				"secrets": map[interface{}]interface{}{
					"api_key": "secret123",
					"unused":  "(( prune ))",
				},
				"metadata": map[interface{}]interface{}{
					"version":    "1.0",
					"deprecated": "(( prune ))",
				},
			})

			engine, err := NewEngine()
			So(err, ShouldBeNil)

			// Cherry-pick config only - should pull in secrets.api_key
			result, err := engine.Merge(context.Background(), doc).
				WithCherryPick("config").
				Execute()

			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)

			data := result.RawData().(map[interface{}]interface{})

			// Config should be evaluated
			config := data["config"].(map[interface{}]interface{})
			So(config["api_key"], ShouldEqual, "secret123")
			So(config["url"], ShouldEqual, "https://api.example.com")

			// Metadata should not be included
			So(data["metadata"], ShouldBeNil)

			// Note: secrets might be partially included due to dependency,
			// but we don't enforce that unused fields are pruned in dependencies
		})

		Convey("Should apply both cherry-pick and explicit prune", func() {
			doc := NewDocument(map[interface{}]interface{}{
				"services": map[interface{}]interface{}{
					"web": map[interface{}]interface{}{
						"port":  8080,
						"debug": true,
					},
					"api": map[interface{}]interface{}{
						"port":  9090,
						"debug": false,
					},
				},
				"monitoring": map[interface{}]interface{}{
					"enabled": true,
					"endpoints": []interface{}{
						"(( grab services.web.port ))",
						"(( grab services.api.port ))",
					},
				},
			})

			engine, err := NewEngine()
			So(err, ShouldBeNil)

			// Cherry-pick services and monitoring, then prune debug fields
			result, err := engine.Merge(context.Background(), doc).
				WithCherryPick("services", "monitoring").
				WithPrune("services.web.debug", "services.api.debug").
				Execute()

			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)

			data := result.RawData().(map[interface{}]interface{})

			// Check services
			services := data["services"].(map[interface{}]interface{})
			web := services["web"].(map[interface{}]interface{})
			So(web["port"], ShouldEqual, 8080)
			So(web["debug"], ShouldBeNil) // Explicitly pruned

			api := services["api"].(map[interface{}]interface{})
			So(api["port"], ShouldEqual, 9090)
			So(api["debug"], ShouldBeNil) // Explicitly pruned

			// Check monitoring
			monitoring := data["monitoring"].(map[interface{}]interface{})
			So(monitoring["enabled"], ShouldEqual, true)
			endpoints := monitoring["endpoints"].([]interface{})
			So(endpoints[0], ShouldEqual, 8080)
			So(endpoints[1], ShouldEqual, 9090)
		})

		Convey("Should handle prune operator in arrays", func() {
			doc := NewDocument(map[interface{}]interface{}{
				"environments": []interface{}{
					map[interface{}]interface{}{
						"name":    "dev",
						"secrets": "(( prune ))",
						"url":     "http://dev.example.com",
					},
					map[interface{}]interface{}{
						"name":    "prod",
						"secrets": "(( prune ))",
						"url":     "https://prod.example.com",
					},
				},
				"deployment": map[interface{}]interface{}{
					"target": "(( grab environments.1.name ))",
				},
			})

			engine, err := NewEngine()
			So(err, ShouldBeNil)

			// Cherry-pick everything (to test prune within arrays)
			result, err := engine.Merge(context.Background(), doc).
				Execute()

			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)

			data := result.RawData().(map[interface{}]interface{})

			// Check environments
			environments := data["environments"].([]interface{})
			So(len(environments), ShouldEqual, 2)

			dev := environments[0].(map[interface{}]interface{})
			So(dev["name"], ShouldEqual, "dev")
			So(dev["url"], ShouldEqual, "http://dev.example.com")
			So(dev["secrets"], ShouldBeNil) // Should be pruned

			prod := environments[1].(map[interface{}]interface{})
			So(prod["name"], ShouldEqual, "prod")
			So(prod["url"], ShouldEqual, "https://prod.example.com")
			So(prod["secrets"], ShouldBeNil) // Should be pruned

			// Check deployment
			deployment := data["deployment"].(map[interface{}]interface{})
			So(deployment["target"], ShouldEqual, "prod")
		})

		Convey("Should handle conditional values with cherry-pick", func() {
			doc := NewDocument(map[interface{}]interface{}{
				"feature_flags": map[interface{}]interface{}{
					"new_ui":     true,
					"debug_mode": false,
				},
				"config": map[interface{}]interface{}{
					"ui_version": "(( feature_flags.new_ui ? \"v2\" : \"v1\" ))",
					"log_level":  "(( feature_flags.debug_mode ? \"debug\" : \"info\" ))",
					"api_url":    "https://api.example.com",
				},
				"admin": map[interface{}]interface{}{
					"debug_panel": "(( prune ))",
					"user":        "admin",
				},
				"unused": map[interface{}]interface{}{
					"data": "(( grab missing ))",
				},
			})

			engine, err := NewEngine()
			So(err, ShouldBeNil)

			// Cherry-pick config only
			result, err := engine.Merge(context.Background(), doc).
				WithCherryPick("config").
				Execute()

			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)

			data := result.RawData().(map[interface{}]interface{})

			// Check config
			config := data["config"].(map[interface{}]interface{})
			So(config["ui_version"], ShouldEqual, "v2")
			So(config["log_level"], ShouldEqual, "info")
			So(config["api_url"], ShouldEqual, "https://api.example.com")

			// Admin and unused should not be included
			So(data["admin"], ShouldBeNil)
			So(data["unused"], ShouldBeNil)
		})
	})
}

func TestCherryPickWithDeferOperator(t *testing.T) {
	Convey("Cherry-pick with defer operator", t, func() {
		Convey("Should handle defer operator within cherry-picked paths", func() {
			doc := NewDocument(map[interface{}]interface{}{
				"templates": map[interface{}]interface{}{
					"web_url": "(( defer concat \"https://\" domain.name \"/\" app.path ))",
					"api_url": "(( defer concat \"https://api.\" domain.name ))",
					"db_url":  "(( defer grab database.url || \"postgres://localhost\" ))",
				},
				"app": map[interface{}]interface{}{
					"path":    "myapp",
					"version": "1.0",
				},
				"domain": map[interface{}]interface{}{
					"name": "example.com",
				},
				"unused": map[interface{}]interface{}{
					"error": "(( grab missing ))",
				},
			})

			engine, err := NewEngine()
			So(err, ShouldBeNil)

			// Cherry-pick templates only
			result, err := engine.Merge(context.Background(), doc).
				WithCherryPick("templates").
				Execute()

			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)

			data := result.RawData().(map[interface{}]interface{})

			// Check that defer expressions are preserved
			templates := data["templates"].(map[interface{}]interface{})
			So(templates["web_url"], ShouldEqual, "(( concat \"https://\" domain.name \"/\" app.path ))")
			So(templates["api_url"], ShouldEqual, "(( concat \"https://api.\" domain.name ))")
			So(templates["db_url"], ShouldEqual, "(( grab database.url || \"postgres://localhost\" ))")

			// Unused should not be included
			So(data["unused"], ShouldBeNil)
		})

		Convey("Should handle defer with references outside cherry-pick scope", func() {
			doc := NewDocument(map[interface{}]interface{}{
				"config": map[interface{}]interface{}{
					"url_template": "(( defer concat protocol \"://\" server.host \":\" server.port ))",
					"timeout":      30,
				},
				"server": map[interface{}]interface{}{
					"host": "localhost",
					"port": 8080,
				},
				"protocol": "https",
				"other": map[interface{}]interface{}{
					"data": "(( grab fail ))",
				},
			})

			engine, err := NewEngine()
			So(err, ShouldBeNil)

			// Cherry-pick config only - defer should work even though it references external values
			result, err := engine.Merge(context.Background(), doc).
				WithCherryPick("config").
				Execute()

			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)

			data := result.RawData().(map[interface{}]interface{})

			// Check config
			config := data["config"].(map[interface{}]interface{})
			So(config["url_template"], ShouldEqual, "(( concat protocol \"://\" server.host \":\" server.port ))")
			So(config["timeout"], ShouldEqual, 30)

			// Other sections should not be included
			So(data["server"], ShouldBeNil)
			So(data["protocol"], ShouldBeNil)
			So(data["other"], ShouldBeNil)
		})

		Convey("Should handle nested defer expressions", func() {
			doc := NewDocument(map[interface{}]interface{}{
				"generators": map[interface{}]interface{}{
					"urls": map[interface{}]interface{}{
						"base": "(( defer concat scheme \"://\" host ))",
						"full": "(( defer concat generators.urls.base \"/\" path ))",
					},
				},
				"scheme": "https",
				"host":   "api.example.com",
				"path":   "v1/users",
				"unused": map[interface{}]interface{}{
					"fail": "(( grab missing.value ))",
				},
			})

			engine, err := NewEngine()
			So(err, ShouldBeNil)

			// Cherry-pick generators only
			result, err := engine.Merge(context.Background(), doc).
				WithCherryPick("generators").
				Execute()

			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)

			data := result.RawData().(map[interface{}]interface{})

			// Check generators structure
			generators := data["generators"].(map[interface{}]interface{})
			urls := generators["urls"].(map[interface{}]interface{})
			So(urls["base"], ShouldEqual, "(( concat scheme \"://\" host ))")
			So(urls["full"], ShouldEqual, "(( concat generators.urls.base \"/\" path ))")

			// Unused should not be included
			So(data["unused"], ShouldBeNil)
		})
	})
}
