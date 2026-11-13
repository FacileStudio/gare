package systemd

import (
	"context"
	"fmt"
	"time"
)

// ServiceProperties represents runtime systemd status properties for an application.
type ServiceProperties struct {
	ActiveState          string
	SubState             string
	Result               string
	MainPID              int
	ActiveEnterTimestamp string
	MemoryCurrent        uint64
}

// activeStateSettleTime is how long a service systemd already reports active is given to finish
// starting, so a caller that acts on the result does not race its first request against a port that
// is not listening yet.
const activeStateSettleTime = 50 * time.Millisecond

// Start starts the specified user service.
func Start(ctx context.Context, name string) error {
	return runSystemctl(ctx, "start", name+".service")
}

// WaitForState polls the service until it reaches the expected active state or context expires.
func WaitForState(ctx context.Context, name, target string) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		props, err := GetServiceProperties(ctx, name)
		matched, stateErr := checkStateMatch(props, err, target, name)
		if stateErr != nil {
			return stateErr
		}
		if matched {
			settleActiveState(target)
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for %s to reach %s state: %w", name, target, ctx.Err())
		case <-ticker.C:
		}
	}
}

// settleActiveState waits out the settle time for the active state, the one target a caller waits
// for a service to reach rather than to leave.
func settleActiveState(target string) {
	if target == "active" {
		time.Sleep(activeStateSettleTime)
	}
}

// checkStateMatch reports whether the queried properties match the target state, treating a query
// that failed as not matched yet so the caller keeps polling.
func checkStateMatch(props *ServiceProperties, err error, target, name string) (bool, error) {
	if err != nil || props == nil {
		return false, nil
	}
	if props.ActiveState == target {
		return true, nil
	}
	if target == "active" && props.ActiveState == "failed" {
		return false, fmt.Errorf("service %s entered failed state", name)
	}
	return false, nil
}

// WaitForStop polls the service until it is no longer running and returns its final properties.
func WaitForStop(ctx context.Context, name string) (*ServiceProperties, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		props, err := GetServiceProperties(ctx, name)
		if err == nil && props != nil && !isRunningState(props.ActiveState) {
			return props, nil
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for %s to stop: %w", name, ctx.Err())
		case <-ticker.C:
		}
	}
}

func isRunningState(state string) bool {
	switch state {
	case "active", "activating", "deactivating", "reloading":
		return true
	}
	return false
}

// GetServiceProperties queries systemctl show for unit properties.
func GetServiceProperties(ctx context.Context, name string) (*ServiceProperties, error) {
	props := []string{
		"ActiveState",
		"SubState",
		"Result",
		"MainPID",
		"ActiveEnterTimestamp",
		"MemoryCurrent",
	}
	args := []string{"show", name + ".service"}
	for _, p := range props {
		args = append(args, "-p", p)
	}
	cmd := systemctlCmd(ctx, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to query service properties: %w", err)
	}
	return parseServiceProperties(string(output)), nil
}
