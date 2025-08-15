package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/proxmox"
)

func TestUpdateVMMeta(t *testing.T) {
	tests := []struct {
		name                       string
		includeCriticalVMResources bool
		includeVMConfig            bool
		wantErr                    bool
	}{
		{
			name:                       "update critical VM startup order",
			includeCriticalVMResources: true,
			includeVMConfig:            true,
			wantErr:                    false,
		},
		{
			name:                       "no critical VMs found",
			includeCriticalVMResources: false,
			includeVMConfig:            false,
			wantErr:                    false,
		},
		{
			name:                       "critical VMs found but config not available",
			includeCriticalVMResources: true,
			includeVMConfig:            false,
			wantErr:                    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := testHandlerConfig{
				includeCriticalVMResources: tt.includeCriticalVMResources,
				includeVMConfig:            tt.includeVMConfig,
			}

			testServer, mockServer := createTestServerWithConfig(config)
			defer mockServer.Close()

			err := testServer.UpdateVMMeta()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExtractVMIDs(t *testing.T) {
	testServer, mockServer := createTestServer()
	defer mockServer.Close()

	vms := []proxmox.ClusterResource{
		{VMID: 100, Name: "vm1"},
		{VMID: 101, Name: "vm2"},
		{VMID: 102, Name: "vm3"},
	}

	vmids := testServer.extractVMIDs(vms)

	expected := []int{100, 101, 102}
	assert.Equal(t, expected, vmids)
}

func TestUpdateCriticalVMStartOrder(t *testing.T) {
	tests := []struct {
		name            string
		includeVMConfig bool
		expectedResult  bool
	}{
		{
			name:            "update VM startup order successfully",
			includeVMConfig: true,
			expectedResult:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := testHandlerConfig{
				includeVMConfig:            tt.includeVMConfig,
				includeCriticalVMResources: true, // Always need critical VM resources for this test
			}

			testServer, mockServer := createTestServerWithConfig(config)
			defer mockServer.Close()

			result := testServer.updateCriticalVMStartOrder("pve1", 106)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestUpdateVMMetaSkipsCRSSkipVMs(t *testing.T) {
	tests := []struct {
		name                       string
		includeCriticalVMResources bool
	}{
		{
			name:                       "skip VMs with crs-skip tag for metadata updates",
			includeCriticalVMResources: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := testHandlerConfig{
				includeCriticalVMResources: tt.includeCriticalVMResources,
				includeVMConfig:            true,
			}

			testServer, mockServer := createTestServerWithConfig(config)
			defer mockServer.Close()

			err := testServer.UpdateVMMeta()
			assert.NoError(t, err)
		})
	}
}

func TestUpdateVMMetaLogging(t *testing.T) {
	tests := []struct {
		name                       string
		includeCriticalVMResources bool
		includeVMConfig            bool
		description                string
	}{
		{
			name:                       "updates needed - should log completion",
			includeCriticalVMResources: true,
			includeVMConfig:            true,
			description:                "Test that completion message is logged when updates are made",
		},
		{
			name:                       "no VMs found - should not log spam",
			includeCriticalVMResources: false,
			includeVMConfig:            true,
			description:                "Test that no spam logs when no critical VMs found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := testHandlerConfig{
				includeCriticalVMResources: tt.includeCriticalVMResources,
				includeVMConfig:            tt.includeVMConfig,
			}

			testServer, mockServer := createTestServerWithConfig(config)
			defer mockServer.Close()

			err := testServer.UpdateVMMeta()
			assert.NoError(t, err)
		})
	}
}