package proxy

import (
	"errors"
	"net/http"
	"strings"

	"github.com/asenalabs/asena/internal/config"
	"go.uber.org/zap"
)

type matchFunc func(string, *http.Request) bool

var matcherRegistry = map[string]matchFunc{
	"Host": hostMatcher,
}

func (pm *Manager) MatchRouter(r *http.Request) (string, bool, error) {
	value := pm.RouterHolder.Load()
	routers, ok := value.(map[string]*config.RoutersCfg)
	if !ok || routers == nil {
		return "", false, errors.New("no routers configured")
	}

	for _, router := range routers {
		if router.Rule == nil {
			continue
		}
		rule := strings.TrimSpace(*router.Rule)

		ok, err := evaluateConditions(rule, r)
		if err != nil {
			pm.logg.Warn("Invalid rule", zap.String("rule", rule), zap.Error(err))
			continue
		}
		if ok {
			if router.Service != nil {
				return *router.Service, true, nil
			}
			pm.logg.Warn("Router matched but service is nil", zap.String("rule", rule))
			return "", false, nil
		}
	}
	return "", false, nil
}

func evaluateConditions(rule string, r *http.Request) (bool, error) {
	rule = strings.TrimSpace(rule)

	open := strings.Index(rule, "(")
	if open == -1 {
		return false, errors.New("invalid rule: " + rule)
	}
	name := rule[:open]

	matcher, exists := matcherRegistry[name]
	if !exists {
		return false, errors.New("unsupported matcher: " + name)
	}

	return matcher(rule, r), nil
}

func hostMatcher(cond string, r *http.Request) bool {
	expected := extractArgument(cond)
	if expected == "" {
		return false
	}

	host := r.Host
	if strings.Contains(host, ":") {
		host = strings.Split(host, ":")[0]
	}
	return strings.EqualFold(host, expected)
}

func extractArgument(s string) string {
	start := strings.Index(s, "(`")
	end := strings.Index(s, "`)")
	if start == -1 || end == -1 || end <= start+2 {
		return ""
	}
	return s[start+2 : end]
}
