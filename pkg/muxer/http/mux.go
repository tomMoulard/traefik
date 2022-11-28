package http

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v2/pkg/rules"
	"github.com/vulcand/predicate"
)

// Muxer handles routing with rules.
type Muxer struct {
	routes []*route
	parser predicate.Parser
}

// NewMuxer returns a new muxer instance.
func NewMuxer() (*Muxer, error) {
	var matchers []string
	for matcher := range httpFuncs {
		matchers = append(matchers, matcher)
	}

	parser, err := rules.NewParser(matchers)
	if err != nil {
		return nil, err
	}

	return &Muxer{
		parser: parser,
	}, nil
}

//
// // Match returns the handler of the first route matching the request.
// func (m *Muxer) Match(req *http.Request) http.Handler {
// 	for _, route := range m.routes {
// 		if route.matchers.match(req) {
// 			return route.handler
// 		}
// 	}
//
// 	return nil
// }

// ServeHTTP .
func (m *Muxer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	var handler http.Handler
	for _, route := range m.routes {
		if route.matchers.match(req) {
			handler = route.handler
			break
		}

		if route.matchers.notMatchingReason == "Method" {
			handler = http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
				rw.WriteHeader(http.StatusMethodNotAllowed)
			})
		}
	}

	if handler != nil {
		handler.ServeHTTP(rw, req)
		return
	}

	http.NotFoundHandler().ServeHTTP(rw, req)
}

// AddRoute add a new route to the router.
func (m *Muxer) AddRoute(rule string, priority int, handler http.Handler) error {
	parse, err := m.parser.Parse(rule)
	if err != nil {
		return fmt.Errorf("error while parsing rule %s: %w", rule, err)
	}

	buildTree, ok := parse.(rules.TreeBuilder)
	if !ok {
		return fmt.Errorf("error while parsing rule %s", rule)
	}

	if priority == 0 {
		priority = len(rule)
	}

	var matchers matchersTree
	err = addRule(&matchers, buildTree())
	if err != nil {
		return fmt.Errorf("error while adding rule %s: %w", rule, err)
	}

	m.routes = append(m.routes, &route{
		handler:  handler,
		matchers: matchers,
		priority: priority,
	})

	sort.Sort(routes(m.routes))

	return nil
}

func addRule(tree *matchersTree, rule *rules.Tree) error {
	switch rule.Matcher {
	case "and", "or":
		tree.operator = rule.Matcher
		tree.left = &matchersTree{}
		err := addRule(tree.left, rule.RuleLeft)
		if err != nil {
			return err
		}

		tree.right = &matchersTree{}
		return addRule(tree.right, rule.RuleRight)
	default:
		err := rules.CheckRule(rule)
		if err != nil {
			return err
		}

		err = httpFuncs[rule.Matcher](tree, rule.Value...)
		if err != nil {
			return err
		}

		tree.matcherName = rule.Matcher

		if rule.Not {
			matcherFunc := tree.matcher
			tree.matcher = func(req *http.Request) bool {
				return !matcherFunc(req)
			}
		}
	}

	return nil
}

// ParseDomains extract domains from rule.
func ParseDomains(rule string) ([]string, error) {
	var matchers []string
	for matcher := range httpFuncs {
		matchers = append(matchers, matcher)
	}

	parser, err := rules.NewParser(matchers)
	if err != nil {
		return nil, err
	}

	parse, err := parser.Parse(rule)
	if err != nil {
		return nil, err
	}

	buildTree, ok := parse.(rules.TreeBuilder)
	if !ok {
		return nil, fmt.Errorf("error while parsing rule %s", rule)
	}

	return buildTree().ParseMatchers([]string{"Host"}), nil
}

// routes implements sort.Interface.
type routes []*route

// Len implements sort.Interface.
func (r routes) Len() int { return len(r) }

// Swap implements sort.Interface.
func (r routes) Swap(i, j int) { r[i], r[j] = r[j], r[i] }

// Less implements sort.Interface.
func (r routes) Less(i, j int) bool { return r[i].priority > r[j].priority }

// route holds the matchers to match HTTP route,
// and the handler that will serve the connection.
type route struct {
	// matchers tree structure reflecting the rule.
	matchers matchersTree
	// handler responsible for handling the route.
	handler http.Handler
	// priority is used to disambiguate between two (or more) rules that would
	// all match for a given request.
	// Computed from the matching rule length, if not user-set.
	priority int
}

// matcher is a matcher func used to match connection properties.
type matcher func(*http.Request) bool

// matchersTree represents the matchers tree structure.
type matchersTree struct {
	// If matcher is not nil, it means that this matcherTree is a leaf of the tree.
	// It is therefore mutually exclusive with left and right.
	matcher matcher
	// operator to combine the evaluation of left and right leaves.
	operator string
	// Mutually exclusive with matcher.
	left  *matchersTree
	right *matchersTree

	matcherName       string
	notMatchingReason string
}

func (m *matchersTree) match(req *http.Request) bool {
	if m == nil {
		// This should never happen as it should have been detected during parsing.
		log.Warn().Msg("Rule matcher is nil")
		return false
	}

	if m.matcher != nil {
		match := m.matcher(req)
		if !match {
			m.notMatchingReason = m.matcherName
		}
		return match
	}

	switch m.operator {
	case "or":
		leftMatch := m.left.match(req)
		rightMatch := m.right.match(req)

		if !leftMatch {
			m.notMatchingReason = m.left.matcherName
		} else if !rightMatch {
			m.notMatchingReason = m.right.matcherName
		}

		return leftMatch || rightMatch
	case "and":
		leftMatch := m.left.match(req)
		if !leftMatch {
			m.notMatchingReason = m.left.matcherName
			return false
		}

		rightMatch := m.right.match(req)
		if !rightMatch {
			m.notMatchingReason = m.right.matcherName
		}
		return rightMatch
	default:
		// This should never happen as it should have been detected during parsing.
		log.Warn().Str("operator", m.operator).Msg("Invalid rule operator")
		return false
	}
}
