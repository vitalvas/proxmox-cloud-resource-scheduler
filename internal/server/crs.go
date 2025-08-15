package server

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/logging"
	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/proxmox"
	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/tools"
)

const (
	crsMaxNodePriority = 1000
	crsMinNodePriority = 1
	crsSkipTag         = "crs-skip"
	crsCriticalTag     = "crs-critical"
	crsGroupPrefix     = "crs-"

	// HA states
	haStateError    = "error"
	haStateDisabled = "disabled"
	haStateStarted  = "started"
	haStateStopped  = "stopped"
	haStateIgnored  = "ignored"

	// VM statuses
	vmStatusRunning = "running"
	vmStatusStopped = "stopped"

	// VM template flag
	vmTemplateFlag = 1

	// HA resource configuration
	haResourceType    = "vm"
	haResourceComment = "crs-managed"

	// API rate limiting
	apiRateLimit = 500 * time.Millisecond

	// VM resource types
	vmResourceType = "qemu"

	// VM startup configuration
	vmStartupCriticalOrder = "order=1"
)

// rateLimitSleep applies API rate limiting unless disabled for testing
func (s *Server) rateLimitSleep() {
	if !s.disableRateLimit {
		time.Sleep(apiRateLimit)
	}
}

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

	if err := s.UpdateVMMeta(); err != nil {
		return fmt.Errorf("update VM metadata: %w", err)
	}

	if err := s.SetupVMHAResources(); err != nil {
		return fmt.Errorf("setup VM HA resources: %w", err)
	}

	return nil
}

func (s *Server) SetupVMPin() error {
	haGroups, err := s.proxmox.GetClusterHAGroups()
	if err != nil {
		return err
	}

	nodes, err := s.proxmox.GetNodes()
	if err != nil {
		return err
	}

	for _, node := range nodes {
		haGroupPin := tools.GetHAVMPinGroupName(node.Node)
		groupExists := false

		for _, group := range haGroups {
			if haGroupPin == group.Group {
				groupExists = true
				break
			}
		}

		if !groupExists {
			logging.Infof("creating ha group %s", haGroupPin)

			_, err := s.proxmox.CreateClusterHAGroup(proxmox.ClusterHAGroup{
				Group:      haGroupPin,
				Nodes:      fmt.Sprintf("%s:%d", node.Node, crsMaxNodePriority),
				NoFailback: 1,
				Restricted: 1,
			})
			if err != nil {
				return fmt.Errorf("failed to create ha group %s: %s", haGroupPin, err)
			}
		}
	}

	return nil
}

func (s *Server) SetupVMPrefer() error {
	hasSharedStorage, err := s.proxmox.HasSharedStorage()
	if err != nil {
		return err
	}

	if !hasSharedStorage {
		return nil
	}

	haGroups, err := s.proxmox.GetClusterHAGroups()
	if err != nil {
		return err
	}

	nodes, err := s.proxmox.GetNodes()
	if err != nil {
		return err
	}

	for _, node := range nodes {
		haGroupPrefer := tools.GetHAVMPreferGroupName(node.Node)
		groupExists := false

		for _, group := range haGroups {
			if haGroupPrefer == group.Group {
				groupExists = true
				break
			}
		}

		if !groupExists {
			logging.Infof("creating ha group %s", haGroupPrefer)

			var groupNodes []string

			for _, n := range nodes {
				if n.Node == node.Node {
					groupNodes = append(groupNodes,
						fmt.Sprintf("%s:%d", n.Node, crsMaxNodePriority),
					)
				} else {
					groupNodes = append(groupNodes,
						fmt.Sprintf("%s:%d", n.Node, crsMinNodePriority),
					)
				}
			}

			sort.Strings(groupNodes)

			if _, err := s.proxmox.CreateClusterHAGroup(proxmox.ClusterHAGroup{
				Group:      haGroupPrefer,
				Nodes:      strings.Join(groupNodes, ","),
				NoFailback: 1,
				Restricted: 1,
			}); err != nil {
				return fmt.Errorf("failed to create ha group %s: %s", haGroupPrefer, err)
			}
		}
	}

	return nil
}

func (s *Server) generateActualHAGroupNames() (map[string]bool, error) {
	actualGroups := make(map[string]bool)

	nodes, err := s.proxmox.GetNodes()
	if err != nil {
		return nil, err
	}

	// Always generate pin groups for all nodes
	for _, node := range nodes {
		pinGroup := tools.GetHAVMPinGroupName(node.Node)
		actualGroups[pinGroup] = true
	}

	// Generate prefer groups only if shared storage exists
	hasSharedStorage, err := s.proxmox.HasSharedStorage()
	if err != nil {
		return nil, err
	}

	if hasSharedStorage {
		for _, node := range nodes {
			preferGroup := tools.GetHAVMPreferGroupName(node.Node)
			actualGroups[preferGroup] = true
		}
	}

	return actualGroups, nil
}

func (s *Server) removeVMsFromHAGroup(groupName string) error {
	resources, err := s.proxmox.GetClusterHAResources()
	if err != nil {
		return err
	}

	for _, resource := range resources {
		if resource.Group == groupName {
			logging.Infof("removing HA resource %s from group %s", resource.SID, groupName)
			if err := s.proxmox.DeleteClusterHAResource(resource.SID); err != nil {
				return fmt.Errorf("failed to remove HA resource %s from group %s: %w", resource.SID, groupName, err)
			}

			// Sleep to avoid overwhelming the Proxmox API
			s.rateLimitSleep()
		}
	}

	return nil
}

func (s *Server) CleanupOrphanedHAGroups() error {
	actualGroups, err := s.generateActualHAGroupNames()
	if err != nil {
		return err
	}

	haGroups, err := s.proxmox.GetClusterHAGroups()
	if err != nil {
		return err
	}

	for _, group := range haGroups {
		// Check if group has CRS prefix and is not in actual groups
		if strings.HasPrefix(group.Group, crsGroupPrefix) && !actualGroups[group.Group] {
			logging.Infof("found orphaned HA group: %s", group.Group)

			// First, remove all VMs from the group
			if err := s.removeVMsFromHAGroup(group.Group); err != nil {
				return fmt.Errorf("failed to remove VMs from group %s: %w", group.Group, err)
			}

			// Then delete the group
			logging.Infof("deleting orphaned HA group: %s", group.Group)
			if err := s.proxmox.DeleteClusterHAGroup(group.Group); err != nil {
				return fmt.Errorf("failed to delete orphaned HA group %s: %w", group.Group, err)
			}
		}
	}

	return nil
}

func (s *Server) ensureCRSTagRegistered() error {
	logging.Debug("Getting cluster options to check registered tags")
	options, err := s.proxmox.GetClusterOptions()
	if err != nil {
		logging.Debugf("Failed to get cluster options (this may be expected on older Proxmox versions): %v", err)
		return fmt.Errorf("failed to get cluster options: %w", err)
	}

	logging.Debugf("Current registered tags: %v", options.RegisteredTags)

	// Define CRS tags that need to be registered
	crsTagsToRegister := []string{crsSkipTag, crsCriticalTag}

	// Check which CRS tags are missing
	var missingTags []string
	for _, crsTag := range crsTagsToRegister {
		found := false
		for _, existingTag := range options.RegisteredTags {
			if strings.TrimSpace(existingTag) == crsTag {
				found = true
				break
			}
		}
		if !found {
			missingTags = append(missingTags, crsTag)
		}
	}

	// If all tags are already registered, nothing to do
	if len(missingTags) == 0 {
		logging.Debug("All CRS tags are already registered")
		return nil
	}

	// Add missing CRS tags to registered tags array
	newRegisteredTags := make([]string, len(options.RegisteredTags))
	copy(newRegisteredTags, options.RegisteredTags)
	newRegisteredTags = append(newRegisteredTags, missingTags...)

	logging.Debugf("New registered tags array: %v", newRegisteredTags)

	// Update cluster options with new registered tags
	updateOptions := proxmox.ClusterOptions{
		RegisteredTags: newRegisteredTags,
	}

	logging.Debug("Updating cluster options with new registered tags")
	if err := s.proxmox.UpdateClusterOptions(updateOptions); err != nil {
		logging.Debugf("Failed to update cluster options (this may be expected on older Proxmox versions or limited permissions): %v", err)
		return fmt.Errorf("failed to update cluster options with CRS tag: %w", err)
	}

	logging.Infof("Registered CRS tags %v in cluster options", missingTags)
	return nil
}

func (s *Server) UpdateHAStatus() error {
	return s.UpdateHAStatusWithOptions(30, 10*time.Second)
}

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

func (s *Server) findHAResource(vmSID string, haResources []proxmox.ClusterHAResource) *proxmox.ClusterHAResource {
	for _, res := range haResources {
		if res.SID == vmSID {
			return &res
		}
	}
	return nil
}

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

func (s *Server) hasVMSkipTag(vmTags string) bool {
	if vmTags == "" {
		return false
	}

	tags := strings.Split(vmTags, ";")
	for _, tag := range tags {
		if strings.TrimSpace(tag) == crsSkipTag {
			return true
		}
	}
	return false
}

func (s *Server) hasVMCriticalTag(vmTags string) bool {
	if vmTags == "" {
		return false
	}

	tags := strings.Split(vmTags, ";")
	for _, tag := range tags {
		if strings.TrimSpace(tag) == crsCriticalTag {
			return true
		}
	}
	return false
}

func (s *Server) RemoveSkippedVMsFromCRSGroups() error {
	logging.Debug("Checking for VMs with crs-skip tag in CRS HA groups")

	resources, err := s.proxmox.GetClusterResources()
	if err != nil {
		return fmt.Errorf("failed to get cluster resources: %w", err)
	}

	haResources, err := s.proxmox.GetClusterHAResources()
	if err != nil {
		return fmt.Errorf("failed to get HA resources: %w", err)
	}

	for _, haResource := range haResources {
		if !strings.HasPrefix(haResource.Group, crsGroupPrefix) {
			continue
		}

		var vmTags string
		for _, resource := range resources {
			resourceSID := fmt.Sprintf("vm:%d", resource.VMID)
			if haResource.SID == resourceSID {
				vmTags = resource.Tags
				break
			}
		}

		if s.hasVMSkipTag(vmTags) {
			logging.Infof("Removing VM %s with crs-skip tag from CRS HA group %s", haResource.SID, haResource.Group)
			if err := s.proxmox.DeleteClusterHAResource(haResource.SID); err != nil {
				return fmt.Errorf("failed to remove VM %s from HA group %s: %w", haResource.SID, haResource.Group, err)
			}

			s.rateLimitSleep()
		}
	}

	return nil
}

func (s *Server) UpdateVMMeta() error {
	logging.Debug("Checking VMs for metadata updates")

	// Get cluster resources to find VMs that need metadata updates
	resources, err := s.proxmox.GetClusterResources()
	if err != nil {
		return fmt.Errorf("failed to get cluster resources: %w", err)
	}

	var criticalVMs []proxmox.ClusterResource

	for _, resource := range resources {
		// Only check VMs (type=qemu)
		if resource.Type == vmResourceType {
			// Skip VMs with crs-skip tag
			if s.hasVMSkipTag(resource.Tags) {
				logging.Debugf("Skipping VM %d (%s) with crs-skip tag for metadata operations", resource.VMID, resource.Name)
				continue
			}

			// Check if VM has critical tag
			if s.hasVMCriticalTag(resource.Tags) {
				criticalVMs = append(criticalVMs, resource)
				logging.Debugf("VM %d with critical tag detected: name=%s, node=%s, tags=%s",
					resource.VMID, resource.Name, resource.Node, resource.Tags)
			}
		}
	}

	if len(criticalVMs) == 0 {
		logging.Debug("No critical VMs found that need metadata updates")
		return nil
	}

	logging.Debugf("Found %d critical VMs that need metadata updates: %v", len(criticalVMs), s.extractVMIDs(criticalVMs))

	var updatedCount int

	for _, vm := range criticalVMs {
		if s.updateCriticalVMStartOrder(vm.Node, vm.VMID) {
			updatedCount++
		}
	}

	if updatedCount > 0 {
		logging.Infof("Updated startup order for %d critical VMs", updatedCount)
	} else {
		logging.Debug("No critical VMs required startup order updates")
	}
	return nil
}

func (s *Server) extractVMIDs(vms []proxmox.ClusterResource) []int {
	vmids := make([]int, len(vms))
	for i, vm := range vms {
		vmids[i] = vm.VMID
	}
	return vmids
}

func (s *Server) updateCriticalVMStartOrder(node string, vmid int) bool {
	logging.Debugf("Checking startup order for critical VM %d on node %s", vmid, node)

	// Get current VM configuration
	config, err := s.proxmox.GetVMConfig(node, vmid)
	if err != nil {
		logging.Errorf("Failed to get VM %d config on node %s: %v", vmid, node, err)
		return false
	}

	// Check if startup order is already set to order=1
	if config.Startup == vmStartupCriticalOrder {
		logging.Debugf("VM %d already has correct startup order: %s", vmid, config.Startup)
		return false
	}

	logging.Infof("Updating critical VM %d startup order from '%s' to '%s'", vmid, config.Startup, vmStartupCriticalOrder)

	// Update only the startup configuration
	updateConfig := proxmox.VMConfig{
		Startup: vmStartupCriticalOrder,
	}

	if err := s.proxmox.UpdateVMConfig(node, vmid, updateConfig); err != nil {
		logging.Errorf("Failed to update VM %d startup order on node %s: %v", vmid, node, err)
		return false
	}

	logging.Infof("Successfully updated startup order for critical VM %d", vmid)
	return true
}

func (s *Server) SetupVMHAResources() error {
	logging.Debug("Setting up HA resources for VMs")

	// Get existing HA resources to avoid duplicates
	resources, err := s.proxmox.GetClusterHAResources()
	if err != nil {
		return fmt.Errorf("failed to get cluster HA resources: %w", err)
	}

	// Get all nodes to process their VMs
	nodeList, err := s.proxmox.GetNodes()
	if err != nil {
		return fmt.Errorf("failed to get nodes: %w", err)
	}

	var createdCount int

	for _, node := range nodeList {
		haGroupPin := tools.GetHAVMPinGroupName(node.Node)

		vmList, err := s.proxmox.GetNodeVMs(node.Node)
		if err != nil {
			return fmt.Errorf("failed to get VMs for node %s: %w", node.Node, err)
		}

		for _, vm := range vmList {
			// Skip templates
			if vm.Template == vmTemplateFlag {
				logging.Debugf("Skipping template VM %d (%s) on node %s", vm.VMID, vm.Name, node.Node)
				continue
			}

			// Skip VMs with crs-skip tag
			if s.hasVMSkipTag(vm.Tags) {
				logging.Debugf("Skipping VM %d (%s) with crs-skip tag on node %s", vm.VMID, vm.Name, node.Node)
				continue
			}

			sid := fmt.Sprintf("%s:%d", haResourceType, vm.VMID)

			// Check if VM already has HA resource
			haveResource := false
			for _, resource := range resources {
				if resource.SID == sid {
					haveResource = true
					break
				}
			}

			if haveResource {
				logging.Debugf("VM %d (%s) already has HA resource, skipping", vm.VMID, vm.Name)
				continue
			}

			// Determine initial HA state based on VM status
			var haState string
			switch vm.Status {
			case vmStatusRunning:
				haState = haStateStarted
			case vmStatusStopped:
				haState = haStateStopped
			default:
				haState = haStateIgnored
			}

			// Create HA resource
			data := proxmox.ClusterHAResource{
				SID:         sid,
				Type:        haResourceType,
				Comment:     haResourceComment,
				MaxRelocate: proxmox.HAMaxRelocate,
				MaxRestart:  proxmox.HAMaxRestart,
				Group:       haGroupPin,
				State:       haState,
			}

			if _, err := s.proxmox.CreateClusterHAResource(data); err != nil {
				return fmt.Errorf("failed to create HA resource for %s (%s): %w", sid, vm.Name, err)
			}

			logging.Infof("Created HA resource for VM %d (%s) on node %s with state %s", vm.VMID, vm.Name, node.Node, haState)
			createdCount++

			// Sleep to avoid overwhelming the Proxmox API
			s.rateLimitSleep()
		}
	}

	if createdCount > 0 {
		logging.Infof("Created %d new HA resources for VMs", createdCount)
	} else {
		logging.Debug("No new HA resources needed for VMs")
	}

	return nil
}
