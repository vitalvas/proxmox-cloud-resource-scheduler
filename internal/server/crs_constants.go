package server

import "time"

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
