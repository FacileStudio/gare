package systemd

import (
	"bufio"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ServiceProperties represents runtime systemd status properties for an application.
type ServiceProperties struct {
	ActiveState          string
	SubState             string
	MainPID              int
	ActiveEnterTimestamp string
	MemoryCurrent        uint64
	CPUUsageNSec         uint64
}

// Start starts the specified user service.
func Start(ctx context.Context, name string) error {
	cmd := systemctlCmd(ctx, "start", name+".service")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
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
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for %s to reach %s state: %w", name, target, ctx.Err())
		case <-ticker.C:
		}
	}
}

func checkStateMatch(props *ServiceProperties, err error, target, name string) (bool, error) {
	if err != nil || props == nil {
		return false, nil
	}
	if props.ActiveState == target {
		if target == "active" {
			time.Sleep(50 * time.Millisecond)
		}
		return true, nil
	}
	if target == "active" && props.ActiveState == "failed" {
		return false, fmt.Errorf("service %s entered failed state", name)
	}
	return false, nil
}

// GetServiceProperties queries systemctl show for unit properties.
func GetServiceProperties(ctx context.Context, name string) (*ServiceProperties, error) {
	props := []string{
		"ActiveState",
		"SubState",
		"MainPID",
		"ActiveEnterTimestamp",
		"MemoryCurrent",
		"CPUUsageNSec",
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

func parseServiceProperties(output string) *ServiceProperties {
	res := &ServiceProperties{}
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		assignServiceProperty(res, k, v)
	}
	return res
}

func assignServiceProperty(res *ServiceProperties, key, val string) {
	switch key {
	case "ActiveState":
		res.ActiveState = val
	case "SubState":
		res.SubState = val
	case "MainPID":
		res.MainPID, _ = strconv.Atoi(val)
	case "ActiveEnterTimestamp":
		res.ActiveEnterTimestamp = val
	case "MemoryCurrent":
		if val != "[not set]" {
			res.MemoryCurrent, _ = strconv.ParseUint(val, 10, 64)
		}
	case "CPUUsageNSec":
		if val != "[not set]" {
			res.CPUUsageNSec, _ = strconv.ParseUint(val, 10, 64)
		}
	}
}
