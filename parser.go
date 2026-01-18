package main

import "fmt"

// Parser parses transaction files into a list of transactions
type Parser interface {
	Parse(path string) ([]Transaction, error)
}

// ParserFunc is a function that implements Parser
type ParserFunc func(path string) ([]Transaction, error)

func (f ParserFunc) Parse(path string) ([]Transaction, error) {
	return f(path)
}

// parsers is the registry of available parsers
var parsers = map[string]Parser{}

// RegisterParser registers a parser with the given name
func RegisterParser(name string, p Parser) {
	parsers[name] = p
}

// GetParser returns the parser for the given source type
func GetParser(source string) (Parser, error) {
	p, ok := parsers[source]
	if !ok {
		return nil, fmt.Errorf("unknown source type: %s (available: %v)", source, AvailableSources())
	}
	return p, nil
}

// AvailableSources returns a list of registered source types
func AvailableSources() []string {
	var sources []string
	for name := range parsers {
		sources = append(sources, name)
	}
	return sources
}

func init() {
	// Register built-in parsers
	RegisterParser("handelsbanken-xlsx", ParserFunc(ParseHandelsbankenXLSX))
}
