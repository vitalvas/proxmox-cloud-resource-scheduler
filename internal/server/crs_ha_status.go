package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/logging"
	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/proxmox"
)

// UpdateHAStatus updates HA status for VMs with issues using default options
func (s *Server) UpdateHAStatus() error {
	return s.UpdateHAStatusWithOptions(30, 10*time.Second)
}

// UpdateHAStatusWithOptions updates HA status for VMs with configurable retry options
func (s *Server) UpdateHAStatusWithOptions(maxAttempts int, waitInterval time.Duration) error {
	logging.Debug("Checking for VMs with HA issues and critical VMs in CRS-managed groups")

	// Get cluster resources to find VMs with hastate=error, hastate=disabled, or critical VMs not started
	resources, err := s.proxmox.GetClusterResources()
	if err != nil {
		return fmt.Errorf("failed to get cluster resources: %w", err)
	}

	var errorVMs []string
	var disabledVMs []string
	var criticalNotStartedVMs []string

	for _, resource := range resources {
		// Only check VMs (type=qemu)
		if resource.Type == vmResourceType {
			// Skip VMs with crs-skip tag
			if s.hasVMSkipTag(resource.Tags) {
				logging.Debugf("Skipping VM %d (%s) with crs-skip tag for HA status operations", resource.VMID, resource.Name)
				continue
			}

			vmSID := fmt.Sprintf("vm:%d", resource.VMID)

			// Check if VM has critical tag and is not in started state
			if s.hasVMCriticalTag(resource.Tags) && resource.HAState != haStateStarted {
				criticalNotStartedVMs = append(criticalNotStartedVMs, vmSID)
				logging.Debugf("VM %s with critical tag detected in non-started state: name=%s, node=%s, status=%s, hastate=%s",
					vmSID, resource.Name, resource.Node, resource.Status, resource.HAState)
			}

			// Check for error and disabled states (as before)
			switch resource.HAState {
			case haStateError:
				errorVMs = append(errorVMs, vmSID)
				logging.Debugf("VM %s detected with HA error state: name=%s, node=%s, status=%s, hastate=%s",
					vmSID, resource.Name, resource.Node, resource.Status, resource.HAState)
			case haStateDisabled:
				disabledVMs = append(disabledVMs, vmSID)
				logging.Debugf("VM %s detected with HA disabled state: name=%s, node=%s, status=%s, hastate=%s",
					vmSID, resource.Name, resource.Node, resource.Status, resource.HAState)
			}
		}
	}

	if len(errorVMs) == 0 && len(disabledVMs) == 0 && len(criticalNotStartedVMs) == 0 {
		logging.Debug("No VMs with HA issues or critical VMs needing attention found")
		return nil
	}

	if len(errorVMs) > 0 {
		logging.Infof("Found %d VMs with HA error state that need fixing: %v", len(errorVMs), errorVMs)
	}
	if len(disabledVMs) > 0 {
		logging.Infof("Found %d VMs with HA disabled state that need restarting: %v", len(disabledVMs), disabledVMs)
	}
	if len(criticalNotStartedVMs) > 0 {
		logging.Infof("Found %d critical VMs that need to be started: %v", len(criticalNotStartedVMs), criticalNotStartedVMs)
	}

	// Now get the actual HA resources to update them
	haResources, err := s.proxmox.GetClusterHAResources()
	if err != nil {
		return fmt.Errorf("failed to get HA resources: %w", err)
	}

	var crsVMsProcessed int

	// Process VMs with error state (disable -> wait -> restore)
	for _, vmSID := range errorVMs {
		haResource := s.findHAResource(vmSID, haResources)
		if haResource == nil {
			logging.Warnf("VM %s has HA error state but no corresponding HA resource found", vmSID)
			continue
		}

		// Only fix VMs that are in CRS-managed HA groups
		if !strings.HasPrefix(haResource.Group, crsGroupPrefix) {
			logging.Debugf("Skipping VM %s as it's not in a CRS-managed HA group (group: %s)", vmSID, haResource.Group)
			continue
		}

		crsVMsProcessed++
		logging.Infof("Fixing HA resource %s with error state by temporarily disabling (CRS group: %s)", vmSID, haResource.Group)

		if s.fixErrorStateVM(vmSID, haResource, maxAttempts, waitInterval) {
			logging.Infof("Successfully fixed HA error state for VM %s", vmSID)
		}
	}

	// Process VMs with disabled state (move directly to started)
	for _, vmSID := range disabledVMs {
		haResource := s.findHAResource(vmSID, haResources)
		if haResource == nil {
			logging.Warnf("VM %s has HA disabled state but no corresponding HA resource found", vmSID)
			continue
		}

		// Only fix VMs that are in CRS-managed HA groups
		if !strings.HasPrefix(haResource.Group, crsGroupPrefix) {
			logging.Debugf("Skipping VM %s as it's not in a CRS-managed HA group (group: %s)", vmSID, haResource.Group)
			continue
		}

		crsVMsProcessed++
		logging.Infof("Starting HA resource %s that is in disabled state (CRS group: %s)", vmSID, haResource.Group)

		if s.startDisabledVM(vmSID, haResource, maxAttempts, waitInterval) {
			logging.Infof("Successfully started disabled VM %s", vmSID)
		}
	}

	// Process critical VMs that need to be started
	for _, vmSID := range criticalNotStartedVMs {
		haResource := s.findHAResource(vmSID, haResources)
		if haResource == nil {
			logging.Warnf("Critical VM %s has non-started HA state but no corresponding HA resource found", vmSID)
			continue
		}

		// Only process VMs that are in CRS-managed HA groups
		if !strings.HasPrefix(haResource.Group, crsGroupPrefix) {
			logging.Debugf("Skipping critical VM %s as it's not in a CRS-managed HA group (group: %s)", vmSID, haResource.Group)
			continue
		}

		crsVMsProcessed++
		logging.Infof("Ensuring critical VM %s is in started state (CRS group: %s)", vmSID, haResource.Group)

		if s.ensureCriticalVMStarted(vmSID, haResource, maxAttempts, waitInterval) {
			logging.Infof("Successfully ensured critical VM %s is started", vmSID)
		}
	}

	totalVMs := len(errorVMs) + len(disabledVMs) + len(criticalNotStartedVMs)
	logging.Infof("Completed updating %d CRS-managed VMs with HA issues (found %d total VMs with problems)", crsVMsProcessed, totalVMs)
	return nil
}

// findHAResource finds an HA resource by VM SID
func (s *Server) findHAResource(vmSID string, haResources []proxmox.ClusterHAResource) *proxmox.ClusterHAResource {
	for _, res := range haResources {
		if res.SID == vmSID {
			return &res
		}
	}
	return nil
}

// fixErrorStateVM fixes VMs in error state by temporarily disabling them
func (s *Server) fixErrorStateVM(vmSID string, haResource *proxmox.ClusterHAResource, maxAttempts int, waitInterval time.Duration) bool {
	// Determine what the original state should be
	originalState := haStateStarted
	if haResource.RequestedState != "" && haResource.RequestedState != haStateError {
		originalState = haResource.RequestedState
	}

	// Set state to disabled
	disabledResource := *haResource
	disabledResource.State = haStateDisabled
	if err := s.proxmox.UpdateClusterHAResource(disabledResource); err != nil {
		logging.Errorf("Failed to set HA resource %s to disabled: %v", vmSID, err)
		return false
	}

	logging.Infof("Set HA resource %s to disabled, waiting for state to stabilize", vmSID)

	// Wait for the disabled state to be applied and error to clear
	if !s.waitForHAStateChangeWithInterval(vmSID, haStateError, haStateDisabled, maxAttempts, waitInterval) {
		logging.Errorf("HA resource %s did not transition from error state within timeout", vmSID)
		return false
	}

	// Return to original state with retry logic
	originalResource := *haResource
	originalResource.State = originalState

	// Try to restore the original state
	if err := s.proxmox.UpdateClusterHAResource(originalResource); err != nil {
		logging.Errorf("Failed to restore HA resource %s to %s state: %v", vmSID, originalState, err)
		return false
	}

	logging.Infof("Attempting to restore HA resource %s to %s state, waiting for confirmation", vmSID, originalState)

	// Wait for the restoration to take effect
	if s.waitForHAStateChangeWithInterval(vmSID, haStateDisabled, originalState, maxAttempts, waitInterval) {
		logging.Infof("Successfully restored HA resource %s to %s state", vmSID, originalState)
		return true
	}
	logging.Warnf("HA resource %s restoration to %s state could not be confirmed within timeout", vmSID, originalState)
	return false
}

// startDisabledVM starts VMs that are in disabled state
func (s *Server) startDisabledVM(vmSID string, haResource *proxmox.ClusterHAResource, maxAttempts int, waitInterval time.Duration) bool {
	// Set state to started
	startedResource := *haResource
	startedResource.State = haStateStarted
	if err := s.proxmox.UpdateClusterHAResource(startedResource); err != nil {
		logging.Errorf("Failed to set HA resource %s to started: %v", vmSID, err)
		return false
	}

	logging.Infof("Set HA resource %s to started, waiting for state to change", vmSID)

	// Wait for the started state to be applied
	if s.waitForHAStateChangeWithInterval(vmSID, haStateDisabled, haStateStarted, maxAttempts, waitInterval) {
		logging.Infof("Successfully started HA resource %s", vmSID)
		return true
	}
	logging.Warnf("HA resource %s did not transition to started state within timeout", vmSID)
	return false
}

// ensureCriticalVMStarted ensures critical VMs are always in started state
func (s *Server) ensureCriticalVMStarted(vmSID string, haResource *proxmox.ClusterHAResource, maxAttempts int, waitInterval time.Duration) bool {
	// Critical VMs must always be in started state
	startedResource := *haResource
	startedResource.State = haStateStarted
	if err := s.proxmox.UpdateClusterHAResource(startedResource); err != nil {
		logging.Errorf("Failed to set critical HA resource %s to started: %v", vmSID, err)
		return false
	}

	logging.Infof("Set critical HA resource %s to started, waiting for state to change", vmSID)

	// Wait for the started state to be applied
	if s.waitForHAStateChangeWithInterval(vmSID, haResource.State, haStateStarted, maxAttempts, waitInterval) {
		logging.Infof("Successfully started critical HA resource %s", vmSID)
		return true
	}
	logging.Warnf("Critical HA resource %s did not transition to started state within timeout", vmSID)
	return false
}

// waitForHAStateChangeWithInterval waits for HA resource state changes with polling
func (s *Server) waitForHAStateChangeWithInterval(vmSID, fromState, toState string, maxAttempts int, interval time.Duration) bool {
	logging.Debugf("Waiting for HA resource %s to change from %s to %s state (max %d attempts, %v intervals)", vmSID, fromState, toState, maxAttempts, interval)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		time.Sleep(interval)

		resources, err := s.proxmox.GetClusterResources()
		if err != nil {
			logging.Warnf("Attempt %d/%d: Failed to get cluster resources for %s: %v", attempt, maxAttempts, vmSID, err)
			continue
		}

		for _, resource := range resources {
			if resource.Type == vmResourceType {
				if vmidStr := fmt.Sprintf("vm:%d", resource.VMID); vmidStr == vmSID {
					currentState := resource.HAState
					logging.Debugf("Attempt %d/%d: HA resource %s current state: %s (target: %s)",
						attempt, maxAttempts, vmSID, currentState, toState)

					// Check if we've moved away from the error state (for disable operation)
					if fromState == haStateError && currentState != haStateError {
						logging.Debugf("HA resource %s successfully moved away from error state to: %s", vmSID, currentState)
						return true
					}

					// Check if we've reached the target state (for restore operation)
					if currentState == toState {
						logging.Debugf("HA resource %s successfully reached target state: %s", vmSID, toState)
						return true
					}

					// For restore operations, also accept "started" if that's a valid end state
					if toState == haStateStarted && currentState == haStateStarted {
						logging.Debugf("HA resource %s successfully reached started state", vmSID)
						return true
					}

					break
				}
			}
		}
	}

	totalTime := time.Duration(maxAttempts) * interval
	logging.Warnf("HA resource %s did not change from %s to %s state after %d attempts (%v total)",
		vmSID, fromState, toState, maxAttempts, totalTime)
	return false
}