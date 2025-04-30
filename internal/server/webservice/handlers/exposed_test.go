package handlers

import "github.com/ubuntu/ubuntu-insights/internal/server/shared/config"

func (u *Upload) Config() config.Provider {
	return u.config
}
