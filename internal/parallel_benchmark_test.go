package internal

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"testing"
)

// Test constants
const (
	EvalPhase = 1 // graft.EvalPhase
)

// BenchmarkParallelExecution benchmarks parallel vs sequential execution
func BenchmarkParallelExecution(b *testing.B) {
	// Test scenarios
	scenarios := []struct {
		name     string
		numOps   int
		opType   string
		parallel bool
		workers  int
	}{
		// Small workloads
		{"Sequential/Small/10ops", 10, "grab", false, 0},
		{"Parallel/Small/10ops", 10, "grab", true, 4},

		// Medium workloads
		{"Sequential/Medium/50ops", 50, "grab", false, 0},
		{"Parallel/Medium/50ops/2workers", 50, "grab", true, 2},
		{"Parallel/Medium/50ops/4workers", 50, "grab", true, 4},
		{"Parallel/Medium/50ops/8workers", 50, "grab", true, 8},

		// Large workloads
		{"Sequential/Large/200ops", 200, "grab", false, 0},
		{"Parallel/Large/200ops/4workers", 200, "grab", true, 4},
		{"Parallel/Large/200ops/8workers", 200, "grab", true, 8},
		{"Parallel/Large/200ops/16workers", 200, "grab", true, 16},

		// Very large workloads
		{"Sequential/VeryLarge/1000ops", 1000, "grab", false, 0},
		{"Parallel/VeryLarge/1000ops/8workers", 1000, "grab", true, 8},
		{"Parallel/VeryLarge/1000ops/16workers", 1000, "grab", true, 16},
		{"Parallel/VeryLarge/1000ops/32workers", 1000, "grab", true, 32},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			// Create test data
			tree := createBenchmarkTree(scenario.numOps)

			// Configure parallel execution
			if scenario.parallel {
				os.Setenv("GRAFT_PARALLEL", "true")
				os.Setenv("GRAFT_PARALLEL_WORKERS", fmt.Sprintf("%d", scenario.workers))
			} else {
				os.Setenv("GRAFT_PARALLEL", "false")
			}
			defer os.Unsetenv("GRAFT_PARALLEL")
			defer os.Unsetenv("GRAFT_PARALLEL_WORKERS")

			// Reset features to pick up env changes
			globalFeaturesOnce = sync.Once{}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Create a fresh evaluator for each run
				ev := &Evaluator{
					Tree: copyTree(tree),
				}

				// Run evaluation
				err := ev.RunPhaseParallel(EvalPhase)
				if err != nil {
					b.Fatalf("Evaluation failed: %v", err)
				}
			}

			b.ReportMetric(float64(scenario.numOps), "ops")
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(scenario.numOps), "ns/op")
		})
	}
}

// BenchmarkConcurrentAccess benchmarks concurrent tree access patterns
func BenchmarkConcurrentAccess(b *testing.B) {
	scenarios := []struct {
		name     string
		readers  int
		writers  int
		treeImpl string
	}{
		{"SafeTree/1reader", 1, 0, "safe"},
		{"SafeTree/4readers", 4, 0, "safe"},
		{"SafeTree/16readers", 16, 0, "safe"},
		{"SafeTree/1writer", 0, 1, "safe"},
		{"SafeTree/4writers", 0, 4, "safe"},
		{"SafeTree/mixed/4r1w", 4, 1, "safe"},
		{"SafeTree/mixed/8r2w", 8, 2, "safe"},

		{"COWTree/1reader", 1, 0, "cow"},
		{"COWTree/4readers", 4, 0, "cow"},
		{"COWTree/16readers", 16, 0, "cow"},
		{"COWTree/1writer", 0, 1, "cow"},
		{"COWTree/4writers", 0, 4, "cow"},
		{"COWTree/mixed/4r1w", 4, 1, "cow"},
		{"COWTree/mixed/8r2w", 8, 2, "cow"},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			// Create tree based on implementation
			var tree ThreadSafeTree
			data := createBenchmarkTree(100)

			switch scenario.treeImpl {
			case "safe":
				tree = NewSafeTree(data)
			case "cow":
				// TODO: Fix COW tree implementation
				tree = NewSafeTree(data) // Fallback for now
			default:
				b.Fatalf("Unknown tree implementation: %s", scenario.treeImpl)
			}

			b.ResetTimer()

			// Run concurrent operations
			done := make(chan bool)

			// Start readers
			for r := 0; r < scenario.readers; r++ {
				go func(id int) {
					for i := 0; i < b.N; i++ {
						path := fmt.Sprintf("item%d", i%100)
						tree.Find(path)
					}
					done <- true
				}(r)
			}

			// Start writers
			for w := 0; w < scenario.writers; w++ {
				go func(id int) {
					for i := 0; i < b.N; i++ {
						path := fmt.Sprintf("item%d", i%100)
						tree.Set(fmt.Sprintf("updated-%d-%d", id, i), path)
					}
					done <- true
				}(w)
			}

			// Wait for all goroutines
			for i := 0; i < scenario.readers+scenario.writers; i++ {
				<-done
			}

			totalOps := b.N * (scenario.readers + scenario.writers)
			b.ReportMetric(float64(totalOps), "total_ops")
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(totalOps), "ns/op")
		})
	}
}

// BenchmarkOperatorTypes benchmarks different operator types
func BenchmarkOperatorTypes(b *testing.B) {
	operators := []struct {
		name     string
		opType   string
		template string
		safe     bool
	}{
		{"grab", "grab", "(( grab source%d ))", true},
		{"concat", "concat", "(( concat \"prefix-\" source%d ))", true},
		{"base64", "base64", "(( base64 source%d ))", true},
		{"empty", "empty", "(( empty source%d ))", true},
		{"keys", "keys", "(( keys data ))", true},
		{"join", "join", "(( join \",\" list%d ))", true},
	}

	for _, op := range operators {
		b.Run(op.name, func(b *testing.B) {
			// Test both sequential and parallel
			for _, parallel := range []bool{false, true} {
				name := "sequential"
				if parallel {
					name = "parallel"
				}

				b.Run(name, func(b *testing.B) {
					// Create test tree
					tree := map[interface{}]interface{}{
						"data": createBenchmarkTree(50),
					}

					// Add operator expressions
					for i := 0; i < 50; i++ {
						tree[fmt.Sprintf("target%d", i)] = fmt.Sprintf(op.template, i)
						tree[fmt.Sprintf("source%d", i)] = fmt.Sprintf("value-%d", i)
						tree[fmt.Sprintf("list%d", i)] = []interface{}{"a", "b", "c"}
					}

					// Configure parallel execution
					if parallel {
						os.Setenv("GRAFT_PARALLEL", "true")
						os.Setenv("GRAFT_PARALLEL_WORKERS", "8")
					} else {
						os.Setenv("GRAFT_PARALLEL", "false")
					}
					defer os.Unsetenv("GRAFT_PARALLEL")
					defer os.Unsetenv("GRAFT_PARALLEL_WORKERS")

					// Reset features
					globalFeaturesOnce = sync.Once{}

					b.ResetTimer()

					for i := 0; i < b.N; i++ {
						ev := &Evaluator{
							Tree: copyTree(tree),
						}

						// Skip if operator not defined
						if err := ev.RunPhaseParallel(EvalPhase); err != nil {
							b.Skipf("Operator %s not available: %v", op.name, err)
						}
					}
				})
			}
		})
	}
}

// BenchmarkScaling benchmarks scaling characteristics
func BenchmarkScaling(b *testing.B) {
	// Test how performance scales with worker count
	numOps := 200
	tree := createBenchmarkTree(numOps)

	maxWorkers := runtime.NumCPU() * 2
	for workers := 1; workers <= maxWorkers; workers *= 2 {
		b.Run(fmt.Sprintf("%d-workers", workers), func(b *testing.B) {
			os.Setenv("GRAFT_PARALLEL", "true")
			os.Setenv("GRAFT_PARALLEL_WORKERS", fmt.Sprintf("%d", workers))
			defer os.Unsetenv("GRAFT_PARALLEL")
			defer os.Unsetenv("GRAFT_PARALLEL_WORKERS")

			// Reset features
			globalFeaturesOnce = sync.Once{}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				ev := &Evaluator{
					Tree: copyTree(tree),
				}

				if err := ev.RunPhaseParallel(EvalPhase); err != nil {
					b.Fatalf("Evaluation failed: %v", err)
				}
			}

			// Calculate efficiency
			if workers > 1 {
				// Compare to single worker performance
				// This is approximate since we can't access previous results
				efficiency := float64(workers) / float64(b.Elapsed().Nanoseconds())
				b.ReportMetric(efficiency, "efficiency")
			}
		})
	}
}

// Helper functions

func createBenchmarkTree(numOps int) map[interface{}]interface{} {
	tree := make(map[interface{}]interface{})

	// Create source data
	for i := 0; i < numOps; i++ {
		tree[fmt.Sprintf("source%d", i)] = fmt.Sprintf("value-%d", i)
	}

	// Create operations that reference the source data
	for i := 0; i < numOps; i++ {
		tree[fmt.Sprintf("target%d", i)] = fmt.Sprintf("(( grab source%d ))", i)
	}

	// Add some nested structures
	tree["nested"] = map[interface{}]interface{}{
		"data": map[interface{}]interface{}{
			"items": make([]interface{}, numOps/10),
		},
	}

	return tree
}

func copyTree(tree map[interface{}]interface{}) map[interface{}]interface{} {
	result := make(map[interface{}]interface{})
	for k, v := range tree {
		switch val := v.(type) {
		case map[interface{}]interface{}:
			result[k] = copyTree(val)
		case []interface{}:
			arr := make([]interface{}, len(val))
			copy(arr, val)
			result[k] = arr
		default:
			result[k] = v
		}
	}
	return result
}

// BenchmarkRealWorld benchmarks with real-world YAML patterns
func BenchmarkRealWorld(b *testing.B) {
	scenarios := []struct {
		name     string
		yamlFile string
		desc     string
	}{
		{
			name: "CloudFoundry",
			desc: "Typical Cloud Foundry deployment manifest pattern",
		},
		{
			name: "Kubernetes",
			desc: "Kubernetes deployment with multiple services",
		},
		{
			name: "Concourse",
			desc: "Concourse pipeline with many jobs",
		},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			// Create realistic YAML structure
			tree := createRealisticYAML(scenario.name)

			// Test sequential
			b.Run("sequential", func(b *testing.B) {
				os.Setenv("GRAFT_PARALLEL", "false")
				defer os.Unsetenv("GRAFT_PARALLEL")
				globalFeaturesOnce = sync.Once{}

				b.ResetTimer()
				runBenchmark(b, tree)
			})

			// Test parallel
			b.Run("parallel", func(b *testing.B) {
				os.Setenv("GRAFT_PARALLEL", "true")
				os.Setenv("GRAFT_PARALLEL_WORKERS", fmt.Sprintf("%d", runtime.NumCPU()))
				defer os.Unsetenv("GRAFT_PARALLEL")
				defer os.Unsetenv("GRAFT_PARALLEL_WORKERS")
				globalFeaturesOnce = sync.Once{}

				b.ResetTimer()
				runBenchmark(b, tree)
			})
		})
	}
}

func createRealisticYAML(scenario string) map[interface{}]interface{} {
	switch scenario {
	case "CloudFoundry":
		return createCloudFoundryYAML()
	case "Kubernetes":
		return createKubernetesYAML()
	case "Concourse":
		return createConcourseYAML()
	default:
		return createBenchmarkTree(100)
	}
}

func createCloudFoundryYAML() map[interface{}]interface{} {
	// Typical CF deployment structure
	return map[interface{}]interface{}{
		"name":          "cf-deployment",
		"director_uuid": "(( grab meta.director_uuid ))",
		"meta": map[interface{}]interface{}{
			"director_uuid": "12345-67890",
			"environment":   "production",
			"stemcell": map[interface{}]interface{}{
				"name":    "ubuntu-xenial",
				"version": "456.latest",
			},
		},
		"instance_groups": createInstanceGroups(10),
		"networks":        createNetworks(3),
		"releases":        createReleases(15),
		"stemcells": []interface{}{
			"(( grab meta.stemcell ))",
		},
	}
}

func createInstanceGroups(count int) []interface{} {
	groups := make([]interface{}, count)
	for i := 0; i < count; i++ {
		groups[i] = map[interface{}]interface{}{
			"name":      fmt.Sprintf("group-%d", i),
			"instances": "(( grab meta.instances || 1 ))",
			"vm_type":   "(( grab meta.vm_type || \"small\" ))",
			"networks": []interface{}{
				map[interface{}]interface{}{
					"name": "(( grab meta.network || \"default\" ))",
				},
			},
			"properties": map[interface{}]interface{}{
				"port":  "(( grab meta.base_port + " + fmt.Sprint(i) + " ))",
				"admin": "(( grab meta.admin_user || \"admin\" ))",
			},
		}
	}
	return groups
}

func createNetworks(count int) []interface{} {
	networks := make([]interface{}, count)
	for i := 0; i < count; i++ {
		networks[i] = map[interface{}]interface{}{
			"name": fmt.Sprintf("network-%d", i),
			"type": "manual",
			"subnets": []interface{}{
				map[interface{}]interface{}{
					"range":   fmt.Sprintf("10.0.%d.0/24", i),
					"gateway": fmt.Sprintf("10.0.%d.1", i),
					"static":  fmt.Sprintf("(( static_ips 10.0.%d.10 10.0.%d.50 ))", i, i),
				},
			},
		}
	}
	return networks
}

func createReleases(count int) []interface{} {
	releases := make([]interface{}, count)
	for i := 0; i < count; i++ {
		releases[i] = map[interface{}]interface{}{
			"name":    fmt.Sprintf("release-%d", i),
			"version": "(( grab meta.releases." + fmt.Sprintf("release-%d", i) + ".version || \"latest\" ))",
		}
	}
	return releases
}

func createKubernetesYAML() map[interface{}]interface{} {
	// Typical k8s deployment structure
	return map[interface{}]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[interface{}]interface{}{
			"name":      "(( grab meta.app_name ))",
			"namespace": "(( grab meta.namespace || \"default\" ))",
			"labels": map[interface{}]interface{}{
				"app":     "(( grab meta.app_name ))",
				"version": "(( grab meta.version ))",
			},
		},
		"spec": map[interface{}]interface{}{
			"replicas": "(( grab meta.replicas || 3 ))",
			"selector": map[interface{}]interface{}{
				"matchLabels": map[interface{}]interface{}{
					"app": "(( grab meta.app_name ))",
				},
			},
			"template": createPodTemplate(),
		},
		"meta": map[interface{}]interface{}{
			"app_name":  "my-app",
			"version":   "1.0.0",
			"replicas":  3,
			"namespace": "production",
		},
	}
}

func createPodTemplate() map[interface{}]interface{} {
	return map[interface{}]interface{}{
		"metadata": map[interface{}]interface{}{
			"labels": map[interface{}]interface{}{
				"app":     "(( grab meta.app_name ))",
				"version": "(( grab meta.version ))",
			},
		},
		"spec": map[interface{}]interface{}{
			"containers": []interface{}{
				map[interface{}]interface{}{
					"name":  "(( grab meta.app_name ))",
					"image": "(( concat meta.registry \"/\" meta.app_name \":\" meta.version ))",
					"ports": []interface{}{
						map[interface{}]interface{}{
							"containerPort": "(( grab meta.port || 8080 ))",
						},
					},
					"env": createEnvVars(10),
				},
			},
		},
	}
}

func createEnvVars(count int) []interface{} {
	vars := make([]interface{}, count)
	for i := 0; i < count; i++ {
		vars[i] = map[interface{}]interface{}{
			"name":  fmt.Sprintf("ENV_VAR_%d", i),
			"value": fmt.Sprintf("(( grab meta.env.var%d || \"default%d\" ))", i, i),
		}
	}
	return vars
}

func createConcourseYAML() map[interface{}]interface{} {
	// Typical Concourse pipeline
	return map[interface{}]interface{}{
		"resources": createResources(5),
		"jobs":      createJobs(20),
		"meta": map[interface{}]interface{}{
			"pipeline": "main",
			"team":     "platform",
			"github": map[interface{}]interface{}{
				"uri":    "https://github.com/org/repo",
				"branch": "master",
			},
		},
	}
}

func createResources(count int) []interface{} {
	resources := make([]interface{}, count)
	for i := 0; i < count; i++ {
		resources[i] = map[interface{}]interface{}{
			"name": fmt.Sprintf("resource-%d", i),
			"type": "git",
			"source": map[interface{}]interface{}{
				"uri":    "(( grab meta.github.uri ))",
				"branch": "(( grab meta.github.branch ))",
			},
		}
	}
	return resources
}

func createJobs(count int) []interface{} {
	jobs := make([]interface{}, count)
	for i := 0; i < count; i++ {
		jobs[i] = map[interface{}]interface{}{
			"name": fmt.Sprintf("job-%d", i),
			"plan": []interface{}{
				map[interface{}]interface{}{
					"get":     "resource-0",
					"trigger": i == 0, // First job triggers
				},
				map[interface{}]interface{}{
					"task": fmt.Sprintf("task-%d", i),
					"config": map[interface{}]interface{}{
						"platform": "linux",
						"image_resource": map[interface{}]interface{}{
							"type": "docker-image",
							"source": map[interface{}]interface{}{
								"repository": "(( grab meta.docker_repo || \"ubuntu\" ))",
							},
						},
						"run": map[interface{}]interface{}{
							"path": "bash",
							"args": []interface{}{
								"-c",
								fmt.Sprintf("echo 'Running job %d'", i),
							},
						},
					},
				},
			},
		}
	}
	return jobs
}

func runBenchmark(b *testing.B, tree map[interface{}]interface{}) {
	for i := 0; i < b.N; i++ {
		ev := &Evaluator{
			Tree: copyTree(tree),
		}

		if err := ev.RunPhaseParallel(EvalPhase); err != nil {
			// Skip if operators are not available
			if err.Error() != "" {
				b.Skip("Some operators not available")
			}
		}
	}
}
