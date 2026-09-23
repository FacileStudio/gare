package systemd

import (
	"bufio"
	"strconv"
	"strings"
)

// parseServiceProperties reads the KEY=VALUE lines systemctl show prints into the properties gare
// reports, ignoring any line systemctl states in another shape.
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
	case "Result":
		res.Result = val
	case "MainPID":
		res.MainPID, _ = strconv.Atoi(val)
	case "ActiveEnterTimestamp":
		res.ActiveEnterTimestamp = val
	case "MemoryCurrent":
		if val != "[not set]" {
			res.MemoryCurrent, _ = strconv.ParseUint(val, 10, 64)
		}
	}
}
