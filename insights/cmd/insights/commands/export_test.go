package commands

type (
	NewUploader  = newUploader
	NewCollector = newCollector
)

// SetArgs sets the arguments for the command.
func (a *App) SetArgs(args []string) {
	a.cmd.SetArgs(args)
}

// WithNewUploader sets the new uploader function for the app.
func WithNewUploader(nu NewUploader) Options {
	return func(o *options) {
		o.newUploader = nu
	}
}

// WithNewCollector sets the new collector function for the app.
func WithNewCollector(nc NewCollector) Options {
	return func(o *options) {
		o.newCollector = nc
	}
}
