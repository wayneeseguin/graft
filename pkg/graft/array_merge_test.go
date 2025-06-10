package graft

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestArrayMergeStrategies(t *testing.T) {
	Convey("Array Merge Strategies", t, func() {
		engine, err := NewEngine()
		So(err, ShouldBeNil)
		ctx := context.Background()

		Convey("Smart merge with identifiable elements", func() {
			base := []byte(`
users:
  - name: alice
    role: admin
  - name: bob
    role: user
`)
			overlay := []byte(`
users:
  - name: alice
    role: superadmin
  - name: charlie
    role: guest
`)
			baseDoc, _ := engine.ParseYAML(base)
			overlayDoc, _ := engine.ParseYAML(overlay)

			result, err := engine.Merge(ctx, baseDoc, overlayDoc).Execute()
			So(err, ShouldBeNil)

			users, _ := result.GetSlice("users")
			So(len(users), ShouldEqual, 3) // alice (updated), bob, charlie

			// Verify alice was updated
			alice := users[0].(map[interface{}]interface{})
			So(alice["name"], ShouldEqual, "alice")
			So(alice["role"], ShouldEqual, "superadmin")

			// Verify bob still exists
			bob := users[1].(map[interface{}]interface{})
			So(bob["name"], ShouldEqual, "bob")
			So(bob["role"], ShouldEqual, "user")

			// Verify charlie was added
			charlie := users[2].(map[interface{}]interface{})
			So(charlie["name"], ShouldEqual, "charlie")
			So(charlie["role"], ShouldEqual, "guest")
		})

		Convey("Replace strategy with WithArrayMergeStrategy", func() {
			base := []byte(`
tags:
  - production
  - stable
`)
			overlay := []byte(`
tags:
  - development
  - unstable
`)
			baseDoc, _ := engine.ParseYAML(base)
			overlayDoc, _ := engine.ParseYAML(overlay)

			result, err := engine.Merge(ctx, baseDoc, overlayDoc).
				WithArrayMergeStrategy(ReplaceArrays).
				Execute()
			So(err, ShouldBeNil)

			tags, _ := result.GetSlice("tags")
			So(len(tags), ShouldEqual, 2)
			So(tags[0], ShouldEqual, "development")
			So(tags[1], ShouldEqual, "unstable")
		})

		Convey("Append strategy with WithArrayMergeStrategy", func() {
			base := []byte(`
logs:
  - "Starting application"
  - "Loading configuration"
`)
			overlay := []byte(`
logs:
  - "Configuration loaded"
  - "Server started"
`)
			baseDoc, _ := engine.ParseYAML(base)
			overlayDoc, _ := engine.ParseYAML(overlay)

			result, err := engine.Merge(ctx, baseDoc, overlayDoc).
				WithArrayMergeStrategy(AppendArrays).
				Execute()
			So(err, ShouldBeNil)

			logs, _ := result.GetSlice("logs")
			So(len(logs), ShouldEqual, 4)
			So(logs[0], ShouldEqual, "Starting application")
			So(logs[1], ShouldEqual, "Loading configuration")
			So(logs[2], ShouldEqual, "Configuration loaded")
			So(logs[3], ShouldEqual, "Server started")
		})

		Convey("Prepend strategy with WithArrayMergeStrategy", func() {
			base := []byte(`
steps:
  - "Build"
  - "Test"
`)
			overlay := []byte(`
steps:
  - "Setup"
  - "Install"
`)
			baseDoc, _ := engine.ParseYAML(base)
			overlayDoc, _ := engine.ParseYAML(overlay)

			result, err := engine.Merge(ctx, baseDoc, overlayDoc).
				WithArrayMergeStrategy(PrependArrays).
				Execute()
			So(err, ShouldBeNil)

			steps, _ := result.GetSlice("steps")
			So(len(steps), ShouldEqual, 4)
			So(steps[0], ShouldEqual, "Setup")
			So(steps[1], ShouldEqual, "Install")
			So(steps[2], ShouldEqual, "Build")
			So(steps[3], ShouldEqual, "Test")
		})

		Convey("Array operators override merge strategy", func() {
			base := []byte(`
items:
  - one
  - two
`)
			overlay := []byte(`
items:
  - (( append ))
  - three
  - four
`)
			baseDoc, _ := engine.ParseYAML(base)
			overlayDoc, _ := engine.ParseYAML(overlay)

			// Even with ReplaceArrays strategy, the (( append )) operator takes precedence
			result, err := engine.Merge(ctx, baseDoc, overlayDoc).
				WithArrayMergeStrategy(ReplaceArrays).
				Execute()
			So(err, ShouldBeNil)

			items, _ := result.GetSlice("items")
			So(len(items), ShouldEqual, 4)
			So(items[0], ShouldEqual, "one")
			So(items[1], ShouldEqual, "two")
			So(items[2], ShouldEqual, "three")
			So(items[3], ShouldEqual, "four")
		})

		Convey("Mixed arrays default to replace", func() {
			base := []byte(`
mixed:
  - simple
  - name: complex
    value: 42
`)
			overlay := []byte(`
mixed:
  - name: new
    value: 100
  - another
`)
			baseDoc, _ := engine.ParseYAML(base)
			overlayDoc, _ := engine.ParseYAML(overlay)

			result, err := engine.Merge(ctx, baseDoc, overlayDoc).Execute()
			So(err, ShouldBeNil)

			mixed, _ := result.GetSlice("mixed")
			So(len(mixed), ShouldEqual, 2) // Replaced, not merged

			first := mixed[0].(map[interface{}]interface{})
			So(first["name"], ShouldEqual, "new")
			So(first["value"], ShouldEqual, 100)
			So(mixed[1], ShouldEqual, "another")
		})
	})
}
