package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetNodeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple node name",
			input:    "pve1",
			expected: "pve1",
		},
		{
			name:     "uppercase node name",
			input:    "PVE1",
			expected: "pve1",
		},
		{
			name:     "FQDN node name",
			input:    "pve1.example.com",
			expected: "pve1.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetNodeName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetHAVMPinGroupName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple node name",
			input:    "pve1",
			expected: "crs-vm-pin-pve1",
		},
		{
			name:     "uppercase node name",
			input:    "PVE1",
			expected: "crs-vm-pin-pve1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetHAVMPinGroupName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetHAVMPreferGroupName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple node name",
			input:    "pve1",
			expected: "crs-vm-prefer-pve1",
		},
		{
			name:     "uppercase node name",
			input:    "PVE1",
			expected: "crs-vm-prefer-pve1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetHAVMPreferGroupName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
