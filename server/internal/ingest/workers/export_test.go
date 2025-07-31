package workers

type (
	DConfigManager = dConfigManager
	DProcessor     = dProcessor
)

// WorkerNames returns the app names of active workers.
func (m *Pool) WorkerNames() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	names := make([]string, 0, len(m.workers))
	for name := range m.workers {
		names = append(names, name)
	}
	return names
}
