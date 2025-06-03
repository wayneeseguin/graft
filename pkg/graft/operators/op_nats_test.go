package operators

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/wayneeseguin/graft/pkg/graft"
)

func startTestNATSServer() (*server.Server, string) {
	opts := &server.Options{
		Port:      -1, // Random available port
		JetStream: true,
	}
	
	ns, err := server.NewServer(opts)
	if err != nil {
		panic(err)
	}
	
	ns.Start()
	
	// Wait for server to be ready
	if !ns.ReadyForConnections(5 * time.Second) {
		panic("NATS server failed to start")
	}
	
	return ns, ns.ClientURL()
}

func TestNatsOperator(t *testing.T) {
	Convey("NATS Operator", t, func() {
		Convey("parseNatsPath", func() {
			testCases := []struct {
				path      string
				expectErr bool
				storeType string
				storePath string
			}{
				{"kv:mystore/mykey", false, "kv", "mystore/mykey"},
				{"obj:mybucket/myfile.yaml", false, "obj", "mybucket/myfile.yaml"},
				{"invalid", true, "", ""},
				{"unknown:store/key", true, "", ""},
				{"", true, "", ""},
				{"kv:", true, "", ""},
				{"obj:", true, "", ""},
			}
			
			for _, tc := range testCases {
				storeType, storePath, err := parseNatsPath(tc.path)
				if tc.expectErr {
					So(err, ShouldNotBeNil)
				} else {
					So(err, ShouldBeNil)
					So(storeType, ShouldEqual, tc.storeType)
					So(storePath, ShouldEqual, tc.storePath)
				}
			}
		})
		
		Convey("With test NATS server", func() {
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
			
			// Create test evaluator
			ev := &graft.Evaluator{
				Tree: map[interface{}]interface{}{},
			}
			
			// Create NATS operator
			op := NatsOperator{}
			
			Convey("KV store operations", func() {
				// Create a KV store
				kv, err := js.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{
					Bucket: "teststore",
				})
				So(err, ShouldBeNil)
				
				// Put test data
				_, err = kv.PutString(context.Background(), "simple", "hello world")
				So(err, ShouldBeNil)
				
				_, err = kv.Put(context.Background(), "yaml", []byte(`foo: bar
nested:
  key: value`))
				So(err, ShouldBeNil)
				
				Convey("Should fetch simple string", func() {
					ClearNatsCache()
					
					args := []*graft.Expr{
						{Type: graft.Literal, Literal: "kv:teststore/simple"},
						{Type: graft.Literal, Literal: url},
					}
					
					resp, err := op.Run(ev, args)
					So(err, ShouldBeNil)
					So(resp, ShouldNotBeNil)
					So(resp.Type, ShouldEqual, graft.Replace)
					So(resp.Value, ShouldEqual, "hello world")
				})
				
				Convey("Should fetch and parse YAML", func() {
					ClearNatsCache()
					
					args := []*graft.Expr{
						{Type: graft.Literal, Literal: "kv:teststore/yaml"},
						{Type: graft.Literal, Literal: url},
					}
					
					resp, err := op.Run(ev, args)
					So(err, ShouldBeNil)
					So(resp, ShouldNotBeNil)
					
					yamlData, ok := resp.Value.(map[interface{}]interface{})
					So(ok, ShouldBeTrue)
					So(yamlData["foo"], ShouldEqual, "bar")
					
					nested, ok := yamlData["nested"].(map[interface{}]interface{})
					So(ok, ShouldBeTrue)
					So(nested["key"], ShouldEqual, "value")
				})
				
				Convey("Should handle missing key", func() {
					ClearNatsCache()
					
					args := []*graft.Expr{
						{Type: graft.Literal, Literal: "kv:teststore/missing"},
						{Type: graft.Literal, Literal: url},
					}
					
					_, err := op.Run(ev, args)
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "failed to get key")
				})
			})
			
			Convey("Object store operations", func() {
				// Create an Object store
				obj, err := js.CreateObjectStore(context.Background(), jetstream.ObjectStoreConfig{
					Bucket: "testbucket",
				})
				So(err, ShouldBeNil)
				
				// Put test objects
				info := jetstream.ObjectMeta{
					Name: "config.yaml",
					Headers: nats.Header{
						"Content-Type": []string{"text/yaml"},
					},
				}
				_, err = obj.Put(context.Background(), info, strings.NewReader(`app:
  name: myapp
  version: 1.0.0`))
				So(err, ShouldBeNil)
				
				info = jetstream.ObjectMeta{
					Name: "readme.txt",
					Headers: nats.Header{
						"Content-Type": []string{"text/plain"},
					},
				}
				_, err = obj.Put(context.Background(), info, strings.NewReader("This is a readme"))
				So(err, ShouldBeNil)
				
				info = jetstream.ObjectMeta{
					Name: "binary.dat",
					Headers: nats.Header{
						"Content-Type": []string{"application/octet-stream"},
					},
				}
				_, err = obj.Put(context.Background(), info, bytes.NewReader([]byte{0x00, 0x01, 0x02, 0x03}))
				So(err, ShouldBeNil)
				
				Convey("Should fetch YAML object", func() {
					ClearNatsCache()
					
					args := []*graft.Expr{
						{Type: graft.Literal, Literal: "obj:testbucket/config.yaml"},
						{Type: graft.Literal, Literal: url},
					}
					
					resp, err := op.Run(ev, args)
					So(err, ShouldBeNil)
					
					yamlData, ok := resp.Value.(map[interface{}]interface{})
					So(ok, ShouldBeTrue)
					
					app, ok := yamlData["app"].(map[interface{}]interface{})
					So(ok, ShouldBeTrue)
					So(app["name"], ShouldEqual, "myapp")
					So(app["version"], ShouldEqual, "1.0.0")
				})
				
				Convey("Should fetch text object", func() {
					ClearNatsCache()
					
					args := []*graft.Expr{
						{Type: graft.Literal, Literal: "obj:testbucket/readme.txt"},
						{Type: graft.Literal, Literal: url},
					}
					
					resp, err := op.Run(ev, args)
					So(err, ShouldBeNil)
					So(resp.Value, ShouldEqual, "This is a readme")
				})
				
				Convey("Should base64 encode binary object", func() {
					ClearNatsCache()
					
					args := []*graft.Expr{
						{Type: graft.Literal, Literal: "obj:testbucket/binary.dat"},
						{Type: graft.Literal, Literal: url},
					}
					
					resp, err := op.Run(ev, args)
					So(err, ShouldBeNil)
					So(resp.Value, ShouldEqual, "AAECAw==") // base64 of {0x00, 0x01, 0x02, 0x03}
				})
			})
			
			Convey("Configuration options", func() {
				// Create a KV store for config test
				kv, err := js.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{
					Bucket: "configtest",
				})
				So(err, ShouldBeNil)
				
				_, err = kv.PutString(context.Background(), "testkey", "testvalue")
				So(err, ShouldBeNil)
				
				Convey("Should accept config map", func() {
					ClearNatsCache()
					
					configMap := map[interface{}]interface{}{
						"url":     url,
						"timeout": "10s",
						"retries": 5,
					}
					
					args := []*graft.Expr{
						{Type: graft.Literal, Literal: "kv:configtest/testkey"},
						{Type: graft.Literal, Literal: configMap},
					}
					
					resp, err := op.Run(ev, args)
					So(err, ShouldBeNil)
					So(resp.Value, ShouldEqual, "testvalue")
				})
			})
			
			Convey("Caching behavior", func() {
				// Create a KV store
				kv, err := js.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{
					Bucket: "cachetest",
				})
				So(err, ShouldBeNil)
				
				_, err = kv.PutString(context.Background(), "cachekey", "initial value")
				So(err, ShouldBeNil)
				
				Convey("Should use cache on second request", func() {
					ClearNatsCache()
					
					args := []*graft.Expr{
						{Type: graft.Literal, Literal: "kv:cachetest/cachekey"},
						{Type: graft.Literal, Literal: url},
					}
					
					// First request
					resp, err := op.Run(ev, args)
					So(err, ShouldBeNil)
					So(resp.Value, ShouldEqual, "initial value")
					
					// Update value in NATS
					_, err = kv.PutString(context.Background(), "cachekey", "updated value")
					So(err, ShouldBeNil)
					
					// Second request should use cache
					resp, err = op.Run(ev, args)
					So(err, ShouldBeNil)
					So(resp.Value, ShouldEqual, "initial value") // Still cached
				})
				
				Convey("Should respect custom cache TTL", func() {
					ClearNatsCache()
					
					// Use short TTL for testing
					configMap := map[interface{}]interface{}{
						"url":       url,
						"cache_ttl": "100ms",
					}
					
					args := []*graft.Expr{
						{Type: graft.Literal, Literal: "kv:cachetest/cachekey"},
						{Type: graft.Literal, Literal: configMap},
					}
					
					// First request
					resp, err := op.Run(ev, args)
					So(err, ShouldBeNil)
					So(resp.Value, ShouldEqual, "initial value")
					
					// Update value in NATS
					_, err = kv.PutString(context.Background(), "cachekey", "updated value")
					So(err, ShouldBeNil)
					
					// Wait for cache to expire
					time.Sleep(150 * time.Millisecond)
					
					// Third request should get updated value
					resp, err = op.Run(ev, args)
					So(err, ShouldBeNil)
					So(resp.Value, ShouldEqual, "updated value") // Cache expired, fresh value
				})
			})
			
			Convey("Error handling", func() {
				Convey("Should fail on missing KV store", func() {
					ClearNatsCache()
					
					args := []*graft.Expr{
						{Type: graft.Literal, Literal: "kv:nonexistent/key"},
						{Type: graft.Literal, Literal: url},
					}
					
					_, err := op.Run(ev, args)
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "failed to get key")
				})
				
				Convey("Should fail on missing object", func() {
					// Create bucket first
					_, err := js.CreateObjectStore(context.Background(), jetstream.ObjectStoreConfig{
						Bucket: "errortest",
					})
					So(err, ShouldBeNil)
					
					ClearNatsCache()
					
					args := []*graft.Expr{
						{Type: graft.Literal, Literal: "obj:errortest/missing.yaml"},
						{Type: graft.Literal, Literal: url},
					}
					
					_, err = op.Run(ev, args)
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "failed to get object")
				})
			})
			
			Convey("Audit logging", func() {
				// Test that audit logging can be enabled
				configMap := map[interface{}]interface{}{
					"url":           url,
					"audit_logging": true,
				}
				
				args := []*graft.Expr{
					{Type: graft.Literal, Literal: "kv:configtest/testkey"},
					{Type: graft.Literal, Literal: configMap},
				}
				
				resp, err := op.Run(ev, args)
				So(err, ShouldBeNil)
				So(resp, ShouldNotBeNil)
				So(resp.Value, ShouldEqual, "testvalue")
				
				// Test audit logging disabled (default)
				configMapNoAudit := map[interface{}]interface{}{
					"url":           url,
					"audit_logging": false,
				}
				
				argsNoAudit := []*graft.Expr{
					{Type: graft.Literal, Literal: "kv:configtest/testkey"},
					{Type: graft.Literal, Literal: configMapNoAudit},
				}
				
				resp, err = op.Run(ev, argsNoAudit)
				So(err, ShouldBeNil)
				So(resp, ShouldNotBeNil)
				So(resp.Value, ShouldEqual, "testvalue")
			})
			
			Convey("Streaming configuration", func() {
				// Test that streaming threshold can be configured
				configMap := map[interface{}]interface{}{
					"url":                url,
					"streaming_threshold": 5242880, // 5MB threshold
				}
				
				args := []*graft.Expr{
					{Type: graft.Literal, Literal: "kv:configtest/testkey"},
					{Type: graft.Literal, Literal: configMap},
				}
				
				resp, err := op.Run(ev, args)
				So(err, ShouldBeNil)
				So(resp, ShouldNotBeNil)
				So(resp.Value, ShouldEqual, "testvalue")
			})
			
			Convey("Metrics and observability", func() {
				// Create a KV store for testing metrics
				kv, err := js.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{
					Bucket: "metricstest",
				})
				So(err, ShouldBeNil)
				
				_, err = kv.PutString(context.Background(), "testkey", "testvalue")
				So(err, ShouldBeNil)
				
				// Clear cache and metrics for clean test
				ClearNatsCache()
				
				args := []*graft.Expr{
					{Type: graft.Literal, Literal: "kv:metricstest/testkey"},
					{Type: graft.Literal, Literal: url},
				}
				
				// First request (cache miss)
				_, err = op.Run(ev, args)
				So(err, ShouldBeNil)
				
				// Second request (cache hit)
				_, err = op.Run(ev, args)
				So(err, ShouldBeNil)
				
				// Get metrics
				metrics := GetNatsMetrics()
				So(metrics, ShouldNotBeNil)
				
				// Check KV operation metrics exist and are reasonable
				kvMetrics, ok := metrics["kv"].(map[string]interface{})
				So(ok, ShouldBeTrue)
				So(kvMetrics["total_operations"], ShouldBeGreaterThan, int64(0))
				So(kvMetrics["cache_hits"], ShouldBeGreaterThanOrEqualTo, int64(0))
				So(kvMetrics["total_errors"], ShouldBeGreaterThanOrEqualTo, int64(0))
				So(kvMetrics["avg_duration_ms"], ShouldBeGreaterThanOrEqualTo, 0.0)
				
				// Check general metrics
				So(metrics["operator_uptime"], ShouldNotBeNil)
				So(metrics["cache_size"], ShouldBeGreaterThan, 0)
			})
		})
		
		Convey("SkipNats flag", func() {
			SkipNats = true
			defer func() { SkipNats = false }()
			
			ev := &graft.Evaluator{
				Tree: map[interface{}]interface{}{},
			}
			
			op := NatsOperator{}
			args := []*graft.Expr{
				{Type: graft.Literal, Literal: "kv:any/key"},
			}
			
			resp, err := op.Run(ev, args)
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, "REDACTED")
		})
	})
}
