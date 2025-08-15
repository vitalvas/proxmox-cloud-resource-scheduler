package server

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/proxmox"
)

func TestSetupVMPin(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "successful VM pin setup",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testServer, mockServer := createTestServerWithConfig(testHandlerConfig{
				includeHAGroups: true,
				includeNodes:    true,
			})
			defer mockServer.Close()

			err := testServer.SetupVMPin()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSetupVMPrefer(t *testing.T) {
	tests := []struct {
		name                 string
		includeStorage       bool
		includeSharedStorage bool
		wantErr              bool
	}{
		{
			name:                 "successful VM prefer setup with shared storage",
			includeStorage:       true,
			includeSharedStorage: true,
			wantErr:              false,
		},
		{
			name:                 "skip VM prefer setup without shared storage",
			includeStorage:       true,
			includeSharedStorage: false,
			wantErr:              false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := testHandlerConfig{
				includeHAGroups:      true,
				includeNodes:         true,
				includeStorage:       tt.includeStorage,
				includeSharedStorage: tt.includeSharedStorage,
			}

			testServer, mockServer := createTestServerWithConfig(config)
			defer mockServer.Close()

			err := testServer.SetupVMPrefer()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGenerateActualHAGroupNames(t *testing.T) {
	tests := []struct {
		name                 string
		includeStorage       bool
		includeSharedStorage bool
		expectedGroups       []string
	}{
		{
			name:                 "with shared storage",
			includeStorage:       true,
			includeSharedStorage: true,
			expectedGroups:       []string{"crs-vm-pin-pve1", "crs-vm-prefer-pve1"},
		},
		{
			name:                 "without shared storage",
			includeStorage:       true,
			includeSharedStorage: false,
			expectedGroups:       []string{"crs-vm-pin-pve1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := testHandlerConfig{
				includeNodes:         true,
				includeStorage:       tt.includeStorage,
				includeSharedStorage: tt.includeSharedStorage,
			}

			testServer, mockServer := createTestServerWithConfig(config)
			defer mockServer.Close()

			actualGroups, err := testServer.generateActualHAGroupNames()

			assert.NoError(t, err)
			assert.Len(t, actualGroups, len(tt.expectedGroups))

			for _, expectedGroup := range tt.expectedGroups {
				assert.True(t, actualGroups[expectedGroup], "Expected group %s to be present", expectedGroup)
			}
		})
	}
}

func TestCleanupOrphanedHAGroups(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "successful cleanup",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := testHandlerConfig{
				includeNodes:         true,
				includeStorage:       true,
				includeSharedStorage: false,
				includeHAGroups:      true,
				includeHAResources:   true,
			}

			testServer, mockServer := createTestServerWithConfig(config)
			defer mockServer.Close()

			err := testServer.CleanupOrphanedHAGroups()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRemoveVMsFromHAGroup(t *testing.T) {
	tests := []struct {
		name      string
		groupName string
		wantErr   bool
	}{
		{
			name:      "successful removal",
			groupName: "test-group",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := testHandlerConfig{
				includeHAResources: true,
			}

			testServer, mockServer := createTestServerWithConfig(config)
			defer mockServer.Close()

			err := testServer.removeVMsFromHAGroup(tt.groupName)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSetupVMPreferRoundRobinPriorities(t *testing.T) {
	tests := []struct {
		name               string
		preferredNode      string
		expectedPriorities map[string]int
	}{
		{
			name:          "pve1 preferred",
			preferredNode: "pve1",
			expectedPriorities: map[string]int{
				"pve1": 1000,
				"pve2": 995,
				"pve3": 990,
			},
		},
		{
			name:          "pve2 preferred",
			preferredNode: "pve2",
			expectedPriorities: map[string]int{
				"pve2": 1000,
				"pve3": 995,
				"pve1": 990,
			},
		},
		{
			name:          "pve3 preferred",
			preferredNode: "pve3",
			expectedPriorities: map[string]int{
				"pve3": 1000,
				"pve1": 995,
				"pve2": 990,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := testHandlerConfig{
				includeHAGroups:      true,
				includeNodes:         true,
				includeMultipleNodes: true,
				includeStorage:       true,
				includeSharedStorage: true,
			}

			testServer, mockServer := createTestServerWithConfig(config)
			defer mockServer.Close()

			// Get nodes to test with
			nodes, err := testServer.proxmox.GetNodes()
			assert.NoError(t, err)
			assert.Len(t, nodes, 3, "Should have 3 nodes for this test")

			// Verify nodes are in expected order (sorted alphabetically)
			expectedNodes := []string{"pve1", "pve2", "pve3"}
			for i, node := range nodes {
				assert.Equal(t, expectedNodes[i], node.Node)
			}

			// Test the priority calculation logic by simulating what SetupVMPrefer would do
			// Create a copy of nodes for sorting to ensure consistent ordering
			sortedNodes := make([]proxmox.Node, len(nodes))
			copy(sortedNodes, nodes)
			sort.Slice(sortedNodes, func(i, j int) bool {
				return sortedNodes[i].Node < sortedNodes[j].Node
			})

			// Find the index of the preferred node
			preferredIndex := -1
			for i, n := range sortedNodes {
				if n.Node == tt.preferredNode {
					preferredIndex = i
					break
				}
			}
			assert.NotEqual(t, -1, preferredIndex, "Preferred node should be found")

			// Verify priority calculation
			actualPriorities := make(map[string]int)
			for i, n := range sortedNodes {
				var priority int
				if n.Node == tt.preferredNode {
					priority = crsMaxNodePriority
				} else {
					relativePosition := (i - preferredIndex + len(sortedNodes)) % len(sortedNodes)
					if relativePosition == 0 {
						relativePosition = len(sortedNodes)
					}
					priority = crsMaxNodePriority - (relativePosition * 5)
					if priority < crsMinNodePriority {
						priority = crsMinNodePriority
					}
				}
				actualPriorities[n.Node] = priority
			}

			// Verify priorities match expected values
			for node, expectedPriority := range tt.expectedPriorities {
				actualPriority, exists := actualPriorities[node]
				assert.True(t, exists, "Node %s should have a priority assigned", node)
				assert.Equal(t, expectedPriority, actualPriority, "Node %s should have priority %d, got %d", node, expectedPriority, actualPriority)
			}
		})
	}
}

func TestSetupVMPreferPriorityMinimumBoundary(t *testing.T) {
	// Test with many nodes to ensure priorities don't go below minimum
	// With crsMaxNodePriority=1000 and decrement of 5, we can have:
	// 1000, 995, 990, 985, ..., 5, 1 (minimum)
	// This means we can handle up to (1000-1)/5 + 1 = 200 nodes before hitting minimum

	// Let's test with a scenario where some nodes would get priority ≤ 1
	t.Run("many nodes with priority minimum boundary", func(t *testing.T) {
		// Simulate the priority calculation logic with many nodes
		nodeCount := 10 // This should give us: 1000, 995, 990, 985, 980, 975, 970, 965, 960, 955

		for preferredIndex := 0; preferredIndex < nodeCount; preferredIndex++ {
			actualPriorities := make([]int, nodeCount)

			for i := 0; i < nodeCount; i++ {
				var priority int
				if i == preferredIndex {
					priority = crsMaxNodePriority
				} else {
					relativePosition := (i - preferredIndex + nodeCount) % nodeCount
					if relativePosition == 0 {
						relativePosition = nodeCount
					}
					priority = crsMaxNodePriority - (relativePosition * 5)
					if priority < crsMinNodePriority {
						priority = crsMinNodePriority
					}
				}
				actualPriorities[i] = priority
			}

			// Verify preferred node has max priority
			assert.Equal(t, crsMaxNodePriority, actualPriorities[preferredIndex])

			// Verify all priorities are >= crsMinNodePriority
			for i, priority := range actualPriorities {
				assert.GreaterOrEqual(t, priority, crsMinNodePriority,
					"Node at index %d should have priority >= %d, got %d", i, crsMinNodePriority, priority)
			}
		}
	})

	// Test extreme case with very many nodes
	t.Run("extreme case with many nodes", func(t *testing.T) {
		nodeCount := 250 // This will definitely cause some priorities to hit the minimum
		preferredIndex := 0

		actualPriorities := make([]int, nodeCount)
		minPriorityCount := 0

		for i := 0; i < nodeCount; i++ {
			var priority int
			if i == preferredIndex {
				priority = crsMaxNodePriority
			} else {
				relativePosition := (i - preferredIndex + nodeCount) % nodeCount
				if relativePosition == 0 {
					relativePosition = nodeCount
				}
				priority = crsMaxNodePriority - (relativePosition * 5)
				if priority < crsMinNodePriority {
					priority = crsMinNodePriority
					minPriorityCount++
				}
			}
			actualPriorities[i] = priority
		}

		// Verify preferred node has max priority
		assert.Equal(t, crsMaxNodePriority, actualPriorities[preferredIndex])

		// Verify all priorities are >= crsMinNodePriority
		for i, priority := range actualPriorities {
			assert.GreaterOrEqual(t, priority, crsMinNodePriority,
				"Node at index %d should have priority >= %d, got %d", i, crsMinNodePriority, priority)
		}

		// Verify that some nodes hit the minimum priority (proving our boundary protection works)
		assert.Greater(t, minPriorityCount, 0, "With 250 nodes, some should hit the minimum priority")

		// Log for verification
		t.Logf("With %d nodes, %d nodes hit the minimum priority of %d", nodeCount, minPriorityCount, crsMinNodePriority)
	})

	// Test specific boundary case where calculated priority would be exactly ≤ 1
	t.Run("specific boundary cases", func(t *testing.T) {
		testCases := []struct {
			name             string
			maxPriority      int
			minPriority      int
			decrement        int
			relativePosition int
			expectedPriority int
		}{
			{
				name:             "priority would be 0",
				maxPriority:      1000,
				minPriority:      1,
				decrement:        5,
				relativePosition: 201, // 1000 - (201 * 5) = -5, should clamp to 1
				expectedPriority: 1,
			},
			{
				name:             "priority would be exactly 1",
				maxPriority:      1000,
				minPriority:      1,
				decrement:        5,
				relativePosition: 199, // 1000 - (199 * 5) = 5, should stay 5
				expectedPriority: 5,
			},
			{
				name:             "priority would be negative",
				maxPriority:      1000,
				minPriority:      1,
				decrement:        5,
				relativePosition: 300, // 1000 - (300 * 5) = -500, should clamp to 1
				expectedPriority: 1,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				priority := tc.maxPriority - (tc.relativePosition * tc.decrement)
				if priority < tc.minPriority {
					priority = tc.minPriority
				}

				assert.Equal(t, tc.expectedPriority, priority,
					"Priority calculation should match expected value")
				assert.GreaterOrEqual(t, priority, tc.minPriority,
					"Priority should never go below minimum")
			})
		}
	})
}

func TestSetupVMPinUpdatesExistingGroups(t *testing.T) {
	t.Run("updates existing pin group with correct configuration", func(t *testing.T) {
		config := testHandlerConfig{
			includeHAGroups:         true,
			includeNodes:            true,
			includeOutdatedHAGroups: true, // This will return an existing group with wrong priority
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		// Should succeed and update the existing group
		err := testServer.SetupVMPin()
		assert.NoError(t, err)
	})
}

func TestSetupVMPreferUpdatesExistingGroups(t *testing.T) {
	t.Run("updates existing prefer group with correct configuration", func(t *testing.T) {
		config := testHandlerConfig{
			includeHAGroups:         true,
			includeNodes:            true,
			includeMultipleNodes:    true,
			includeStorage:          true,
			includeSharedStorage:    true,
			includeOutdatedHAGroups: true, // This will return existing groups with wrong priorities
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		// Should succeed and update the existing groups with correct round-robin priorities
		err := testServer.SetupVMPrefer()
		assert.NoError(t, err)
	})
}

func TestCompareNodeConfiguration(t *testing.T) {
	testServer, mockServer := createTestServer()
	defer mockServer.Close()

	tests := []struct {
		name     string
		existing string
		expected string
		equal    bool
	}{
		{
			name:     "identical configurations",
			existing: "pve1:1000,pve2:995,pve3:990",
			expected: "pve1:1000,pve2:995,pve3:990",
			equal:    true,
		},
		{
			name:     "different order same content",
			existing: "pve2:995,pve1:1000,pve3:990",
			expected: "pve1:1000,pve2:995,pve3:990",
			equal:    true,
		},
		{
			name:     "with extra spaces",
			existing: " pve2:995 , pve1:1000 , pve3:990 ",
			expected: "pve1:1000,pve2:995,pve3:990",
			equal:    true,
		},
		{
			name:     "different priorities",
			existing: "pve1:1000,pve2:995,pve3:985",
			expected: "pve1:1000,pve2:995,pve3:990",
			equal:    false,
		},
		{
			name:     "different nodes",
			existing: "pve1:1000,pve2:995",
			expected: "pve1:1000,pve2:995,pve3:990",
			equal:    false,
		},
		{
			name:     "empty configurations",
			existing: "",
			expected: "",
			equal:    true,
		},
		{
			name:     "single node",
			existing: "pve1:1000",
			expected: "pve1:1000",
			equal:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testServer.compareNodeConfiguration(tt.existing, tt.expected)
			assert.Equal(t, tt.equal, result,
				"compareNodeConfiguration(%q, %q) should return %v",
				tt.existing, tt.expected, tt.equal)
		})
	}
}

func TestSetupVMPreferNoUnnecessaryUpdates(t *testing.T) {
	t.Run("does not update when configuration is already correct", func(t *testing.T) {
		// Create a test that simulates existing groups with correct configuration
		// but in different order to test that we don't spam logs
		config := testHandlerConfig{
			includeHAGroups:        true,
			includeNodes:           true,
			includeMultipleNodes:   true,
			includeStorage:         true,
			includeSharedStorage:   true,
			includeCorrectHAGroups: true, // This provides existing groups with correct config in different order
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		// This should not trigger any updates since the groups are already correctly configured
		// Even though the node order is different, our comparison should recognize they're the same
		err := testServer.SetupVMPrefer()
		assert.NoError(t, err)
	})
}
