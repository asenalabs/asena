package proxy

import (
	"sort"
	"strings"

	"github.com/asenalabs/asena/internal/config"
	"github.com/asenalabs/asena/internal/rule"
	"go.uber.org/zap"
)

// Route is one router, already fully read and ready to use: the rule text has already been 
// turned into a rule. Node tree, and its specificity score has already been worked out. 
// MatchRouter's job on each request is now simple: walk the list, call Tree.Match, stop at
// the first hit.
type Route struct {
	Name        string
	Rule        string
	Tree        rule.Node
	Service     string
	Specificity int
}

// compileRoutes turns the raw router config into a list of Route, sorted from most specific 
// to least specific.
//
// We sort once here, when the config reloads, instead of on every request. A config reload 
// happens rarely, a match happens many times per second. Any work we can do once instead of 
// every time is worth doing once.
//
// If a router has no rule, no service, or a rule that fails to read, we log a warning and 
// skip just that router. One broken router should not stop the rest of the config from loading.
func compileRoutes(routers map[string]*config.RoutersCfg, logg *zap.Logger) []Route {
	routes := make([]Route, 0, len(routers))

	for name, r := range routers {
		if r.Rule == nil {
			logg.Warn("Skipping router: rule is nil", zap.String("router", name))
			continue
		}
		if r.Service == nil {
			logg.Warn("Skipping router: service is nil", zap.String("router", name))
			continue
		}

		ruleStr := strings.TrimSpace(*r.Rule)
		tree, err := rule.ParseRule(ruleStr)
		if err != nil {
			logg.Warn("Skipping router: invalid rule",
				zap.String("router", name), zap.String("rule", ruleStr), zap.Error(err))
			continue
		}

		spec := tree.Specificity()
		routes = append(routes, Route{
			Name:        name,
			Rule:        ruleStr,
			Tree:        tree,
			Service:     *r.Service,
			Specificity: spec,
		})
		logg.Info("Router compiled",
			zap.String("router", name), zap.String("rule", ruleStr), zap.Int("specificity", spec))
	}

	sort.SliceStable(routes, func(i, j int) bool {
		if routes[i].Specificity != routes[j].Specificity {
			return routes[i].Specificity > routes[j].Specificity
		}
		// If two routes score the same, sort by name. Go's map order is random, so without this,
		// the winner between two equal routes could change on every reload for no real reason.
		return routes[i].Name < routes[j].Name
	})

	return routes
}
