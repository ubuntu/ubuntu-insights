package handlers

// ConfigProvider is an interface that defines the configuration access methods used by the handlers.
type ConfigProvider interface {
	IsAllowed(string) bool // IsAllowed checks if a given item is allowed based on the present configuration state.
}
