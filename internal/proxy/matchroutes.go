package proxy

import (
	"net/http"
)

// MatchRouter finds the service name for the first Route whose rule matches r.
//
// All the slow work - reading and sorting the rules - already happened once,
// in compileRoutes, when the config last reloaded. So this function just walks
// an already-sorted list and checks each one. The list is sorted from most
// specific to least specific, so the first match is always the right match.
func (pm *Manager) MatchRouter(r *http.Request) (string, bool, error) {
	value := pm.RouterHolder.Load()
	routes, ok := value.([]Route)
	if !ok {
		return "", false, nil
	}

	for _, route := range routes {
		if route.Tree.Match(r) {
			return route.Service, true, nil
		}
	}
	return "", false, nil
}
