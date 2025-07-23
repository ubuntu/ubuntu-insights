package handlers

// ConfigProvider is an interface that defines the configuration access methods used by the handlers.
type ConfigProvider interface {
	Allows(string) bool // Allows checks if a given item is allowed based on the present configuration state.
}
