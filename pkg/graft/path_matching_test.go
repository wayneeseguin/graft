package graft

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/wayneeseguin/graft/internal/utils/tree"
)

func TestIsUnderPath(t *testing.T) {
	Convey("isUnderPath() path matching", t, func() {
		Convey("Should match exact paths", func() {
			ev := &Evaluator{
				Tree: map[interface{}]interface{}{
					"params": map[interface{}]interface{}{
						"username": "admin",
					},
				},
			}

			So(ev.isUnderPath("params", "params"), ShouldBeTrue)
			So(ev.isUnderPath("params.username", "params"), ShouldBeTrue)
			So(ev.isUnderPath("params.username", "params.username"), ShouldBeTrue)
		})

		Convey("Should not match unrelated paths", func() {
			ev := &Evaluator{
				Tree: map[interface{}]interface{}{},
			}

			So(ev.isUnderPath("meta", "params"), ShouldBeFalse)
			So(ev.isUnderPath("params", "params.username"), ShouldBeFalse) // Shorter path
			So(ev.isUnderPath("other.field", "params"), ShouldBeFalse)
		})

		Convey("Should handle nested paths correctly", func() {
			ev := &Evaluator{
				Tree: map[interface{}]interface{}{
					"meta": map[interface{}]interface{}{
						"environment": map[interface{}]interface{}{
							"name": "prod",
						},
					},
				},
			}

			So(ev.isUnderPath("meta.environment.name", "meta"), ShouldBeTrue)
			So(ev.isUnderPath("meta.environment.name", "meta.environment"), ShouldBeTrue)
			So(ev.isUnderPath("meta.environment", "meta.environment.name"), ShouldBeFalse)
		})

		Convey("Should handle numeric array indices", func() {
			ev := &Evaluator{
				Tree: map[interface{}]interface{}{
					"instances": []interface{}{
						map[interface{}]interface{}{"name": "web", "port": 8080},
						map[interface{}]interface{}{"name": "api", "port": 9090},
					},
				},
			}

			So(ev.isUnderPath("instances.0", "instances"), ShouldBeTrue)
			So(ev.isUnderPath("instances.1", "instances"), ShouldBeTrue)
			So(ev.isUnderPath("instances.0.port", "instances.0"), ShouldBeTrue)
			So(ev.isUnderPath("instances.1.port", "instances.0"), ShouldBeFalse)
		})

		Convey("Should handle named array entries", func() {
			ev := &Evaluator{
				Tree: map[interface{}]interface{}{
					"jobs": []interface{}{
						map[interface{}]interface{}{"name": "web-server", "instances": 3},
						map[interface{}]interface{}{"name": "api-server", "instances": 2},
					},
				},
			}

			// Named entries should match their numeric indices
			So(ev.isUnderPath("jobs.0", "jobs.web-server"), ShouldBeTrue)
			So(ev.isUnderPath("jobs.web-server", "jobs.0"), ShouldBeTrue)
			So(ev.isUnderPath("jobs.1", "jobs.api-server"), ShouldBeTrue)
			So(ev.isUnderPath("jobs.0.instances", "jobs.web-server"), ShouldBeTrue)

			// Wrong matches should fail
			So(ev.isUnderPath("jobs.0", "jobs.api-server"), ShouldBeFalse)
			So(ev.isUnderPath("jobs.1", "jobs.web-server"), ShouldBeFalse)
		})

		Convey("Should handle invalid paths gracefully", func() {
			ev := &Evaluator{
				Tree: map[interface{}]interface{}{},
			}

			// Invalid path syntax should return false
			So(ev.isUnderPath("", "params"), ShouldBeFalse)
			So(ev.isUnderPath("params", ""), ShouldBeFalse)
			So(ev.isUnderPath("", ""), ShouldBeFalse)
		})

		Convey("Should handle mixed path types", func() {
			ev := &Evaluator{
				Tree: map[interface{}]interface{}{
					"networks": []interface{}{
						map[interface{}]interface{}{
							"name": "default",
							"subnets": []interface{}{
								map[interface{}]interface{}{"range": "10.0.0.0/24"},
								map[interface{}]interface{}{"range": "10.0.1.0/24"},
							},
						},
					},
				},
			}

			// Mix of named and numeric indices
			So(ev.isUnderPath("networks.0.subnets.0", "networks.default"), ShouldBeTrue)
			So(ev.isUnderPath("networks.default.subnets.1", "networks.0"), ShouldBeTrue)
			So(ev.isUnderPath("networks.0.subnets.1.range", "networks.default.subnets"), ShouldBeTrue)
		})

		Convey("Should handle edge cases", func() {
			ev := &Evaluator{
				Tree: map[interface{}]interface{}{
					"a": map[interface{}]interface{}{
						"b": map[interface{}]interface{}{
							"c": "value",
						},
					},
				},
			}

			// Single character paths
			So(ev.isUnderPath("a", "a"), ShouldBeTrue)
			So(ev.isUnderPath("a.b", "a"), ShouldBeTrue)
			So(ev.isUnderPath("a.b.c", "a"), ShouldBeTrue)

			// Path with numbers as keys (not indices)
			ev.Tree = map[interface{}]interface{}{
				"123": map[interface{}]interface{}{
					"456": "value",
				},
			}
			So(ev.isUnderPath("123.456", "123"), ShouldBeTrue)
		})

		Convey("Should handle deeply nested structures", func() {
			ev := &Evaluator{
				Tree: map[interface{}]interface{}{
					"level1": map[interface{}]interface{}{
						"level2": []interface{}{
							map[interface{}]interface{}{
								"name": "first",
								"level3": map[interface{}]interface{}{
									"level4": []interface{}{
										map[interface{}]interface{}{
											"id": "item1",
											"data": map[interface{}]interface{}{
												"value": 100,
											},
										},
									},
								},
							},
						},
					},
				},
			}

			// Deep path matching
			So(ev.isUnderPath("level1.level2.0.level3.level4.0.data.value", "level1"), ShouldBeTrue)
			So(ev.isUnderPath("level1.level2.first.level3.level4.item1.data", "level1.level2"), ShouldBeTrue)
			So(ev.isUnderPath("level1.level2.0.level3", "level1.level2.first"), ShouldBeTrue)

			// Mixed numeric and named indices
			So(ev.isUnderPath("level1.level2.first.level3.level4.0", "level1.level2.0.level3"), ShouldBeTrue)
		})

		Convey("Should handle complex real-world structures", func() {
			ev := &Evaluator{
				Tree: map[interface{}]interface{}{
					"instance_groups": []interface{}{
						map[interface{}]interface{}{
							"name": "web",
							"azs":  []interface{}{"z1", "z2"},
							"jobs": []interface{}{
								map[interface{}]interface{}{
									"name": "nginx",
									"properties": map[interface{}]interface{}{
										"port": 80,
										"ssl": map[interface{}]interface{}{
											"enabled": true,
											"cert":    "(( vault secret/certs:cert ))",
										},
									},
								},
							},
						},
						map[interface{}]interface{}{
							"name": "database",
							"azs":  []interface{}{"z1"},
							"jobs": []interface{}{
								map[interface{}]interface{}{
									"name": "postgres",
									"properties": map[interface{}]interface{}{
										"port": 5432,
									},
								},
							},
						},
					},
				},
			}

			// Cherry-pick specific instance group
			So(ev.isUnderPath("instance_groups.0.jobs.0.properties.ssl.cert", "instance_groups.web"), ShouldBeTrue)
			So(ev.isUnderPath("instance_groups.database.jobs.postgres.properties.port", "instance_groups.1"), ShouldBeTrue)

			// Cross-matching numeric and named
			So(ev.isUnderPath("instance_groups.web.jobs.nginx", "instance_groups.0.jobs.0"), ShouldBeTrue)

			// Should not match different instance groups
			So(ev.isUnderPath("instance_groups.0.jobs", "instance_groups.database"), ShouldBeFalse)
			So(ev.isUnderPath("instance_groups.database.jobs", "instance_groups.web"), ShouldBeFalse)
		})

		Convey("Should handle arrays within arrays", func() {
			ev := &Evaluator{
				Tree: map[interface{}]interface{}{
					"matrix": []interface{}{
						[]interface{}{
							map[interface{}]interface{}{"value": 1},
							map[interface{}]interface{}{"value": 2},
						},
						[]interface{}{
							map[interface{}]interface{}{"value": 3},
							map[interface{}]interface{}{"value": 4},
						},
					},
				},
			}

			// Note: This might not work as expected with current implementation
			// as tree.Cursor might not handle nested arrays well
			So(ev.isUnderPath("matrix.0", "matrix"), ShouldBeTrue)
			So(ev.isUnderPath("matrix.1", "matrix"), ShouldBeTrue)
		})
	})
}

func TestSegmentsMatchWithContext(t *testing.T) {
	Convey("segmentsMatchWithContext() segment matching", t, func() {
		Convey("Should match identical segments", func() {
			ev := &Evaluator{
				Tree: map[interface{}]interface{}{},
			}
			cursor := &tree.Cursor{}

			So(ev.segmentsMatchWithContext("params", "params", cursor), ShouldBeTrue)
			So(ev.segmentsMatchWithContext("0", "0", cursor), ShouldBeTrue)
			So(ev.segmentsMatchWithContext("web-server", "web-server", cursor), ShouldBeTrue)
		})

		Convey("Should match numeric indices correctly", func() {
			ev := &Evaluator{
				Tree: map[interface{}]interface{}{},
			}
			cursor := &tree.Cursor{}

			So(ev.segmentsMatchWithContext("0", "0", cursor), ShouldBeTrue)
			So(ev.segmentsMatchWithContext("1", "1", cursor), ShouldBeTrue)
			So(ev.segmentsMatchWithContext("0", "1", cursor), ShouldBeFalse)
			So(ev.segmentsMatchWithContext("10", "10", cursor), ShouldBeTrue)
		})

		Convey("Should match named entries with numeric indices", func() {
			ev := &Evaluator{
				Tree: map[interface{}]interface{}{
					"jobs": []interface{}{
						map[interface{}]interface{}{"name": "web"},
						map[interface{}]interface{}{"name": "api"},
					},
				},
			}

			// Create cursor pointing to jobs array
			cursor := &tree.Cursor{}
			cursor.Push("jobs")

			// Numeric index should match named entry
			So(ev.segmentsMatchWithContext("0", "web", cursor), ShouldBeTrue)
			So(ev.segmentsMatchWithContext("web", "0", cursor), ShouldBeTrue)
			So(ev.segmentsMatchWithContext("1", "api", cursor), ShouldBeTrue)
			So(ev.segmentsMatchWithContext("api", "1", cursor), ShouldBeTrue)

			// Wrong matches
			So(ev.segmentsMatchWithContext("0", "api", cursor), ShouldBeFalse)
			So(ev.segmentsMatchWithContext("1", "web", cursor), ShouldBeFalse)
		})

		Convey("Should handle arrays with id field", func() {
			ev := &Evaluator{
				Tree: map[interface{}]interface{}{
					"resources": []interface{}{
						map[interface{}]interface{}{"id": "db-1", "type": "database"},
						map[interface{}]interface{}{"id": "cache-1", "type": "redis"},
					},
				},
			}

			cursor := &tree.Cursor{}
			cursor.Push("resources")

			// Should match by id field
			So(ev.segmentsMatchWithContext("0", "db-1", cursor), ShouldBeTrue)
			So(ev.segmentsMatchWithContext("1", "cache-1", cursor), ShouldBeTrue)
		})

		Convey("Should handle non-array contexts", func() {
			ev := &Evaluator{
				Tree: map[interface{}]interface{}{
					"config": map[interface{}]interface{}{
						"port": 8080,
					},
				},
			}

			cursor := &tree.Cursor{}
			cursor.Push("config")

			// When not in array context, only exact matches work
			So(ev.segmentsMatchWithContext("port", "port", cursor), ShouldBeTrue)
			So(ev.segmentsMatchWithContext("0", "port", cursor), ShouldBeFalse)
			So(ev.segmentsMatchWithContext("port", "0", cursor), ShouldBeFalse)
		})

		Convey("Should handle empty cursor", func() {
			ev := &Evaluator{
				Tree: map[interface{}]interface{}{},
			}
			cursor := &tree.Cursor{}

			// With empty cursor, only exact matches work
			So(ev.segmentsMatchWithContext("test", "test", cursor), ShouldBeTrue)
			So(ev.segmentsMatchWithContext("0", "test", cursor), ShouldBeFalse)
		})

		Convey("Should handle missing name fields", func() {
			ev := &Evaluator{
				Tree: map[interface{}]interface{}{
					"items": []interface{}{
						map[interface{}]interface{}{"value": 100}, // No name field
						"simple-string", // Not a map
					},
				},
			}

			cursor := &tree.Cursor{}
			cursor.Push("items")

			// Without name fields, numeric indices don't match names
			So(ev.segmentsMatchWithContext("0", "something", cursor), ShouldBeFalse)
			So(ev.segmentsMatchWithContext("1", "simple-string", cursor), ShouldBeFalse)
		})
	})
}
