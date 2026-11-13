package deploys

import (
	"net/http"
	"strconv"

	"github.com/FacileStudio/gare/apps/api/internal/deploys"
)

// lastEventID is the number of the last line a reconnecting EventSource received, 0 for a first
// connection or a header that is not a line number.
func lastEventID(r *http.Request) int {
	n, err := strconv.Atoi(r.Header.Get("Last-Event-ID"))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// ids is the set of the deploy ids in list.
func ids(list []deploys.Deploy) map[string]bool {
	set := make(map[string]bool, len(list))
	for _, d := range list {
		set[d.ID] = true
	}
	return set
}
