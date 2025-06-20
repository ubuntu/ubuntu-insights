package consent

// GetAllSourceConsentStates gets the consent states for all sources.
// It does not get the global consent state.
// If continueOnErr is true, it will continue to the next source if an error occurs.
func (cm Manager) GetAllSourceConsentStates(continueOnErr bool) (map[string]bool, error) {
	p, err := cm.getFiles()
	if err != nil {
		return nil, err
	}

	consentStates := make(map[string]bool)
	for source, path := range p {
		consent, err := readFile(cm.log, path)
		if err != nil && !continueOnErr {
			return nil, err
		}
		if err != nil {
			continue
		}

		consentStates[source] = consent.ConsentState
	}

	return consentStates, nil
}
