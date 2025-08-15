package server

import (
	"fmt"

	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/logging"
)

// SetupCRS orchestrates the complete CRS setup process
func (s *Server) SetupCRS() error {
	// Try to register CRS tag, but don't fail if it doesn't work
	if err := s.ensureCRSTagRegistered(); err != nil {
		logging.Warnf("Failed to register CRS tag (this may be expected): %v", err)
	}

	if err := s.SetupVMPin(); err != nil {
		return fmt.Errorf("setup VM pin: %w", err)
	}

	if err := s.SetupVMPrefer(); err != nil {
		return fmt.Errorf("setup VM prefer: %w", err)
	}

	if err := s.CleanupOrphanedHAGroups(); err != nil {
		return fmt.Errorf("cleanup orphaned HA groups: %w", err)
	}

	if err := s.RemoveSkippedVMsFromCRSGroups(); err != nil {
		return fmt.Errorf("remove skipped VMs from CRS groups: %w", err)
	}

	if err := s.UpdateHAStatus(); err != nil {
		return fmt.Errorf("update HA status: %w", err)
	}

	if err := s.HandleNodeMaintenance(); err != nil {
		return fmt.Errorf("handle node maintenance: %w", err)
	}

	if err := s.UpdateVMMeta(); err != nil {
		return fmt.Errorf("update VM metadata: %w", err)
	}

	if err := s.SetupVMHAResources(); err != nil {
		return fmt.Errorf("setup VM HA resources: %w", err)
	}

	return nil
}
