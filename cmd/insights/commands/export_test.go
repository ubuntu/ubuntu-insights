package commands

type (
	NewUploader = newUploader
)

// SetArgs sets the arguments for the command.
func (a *App) SetArgs(args []string) {
	a.cmd.SetArgs(args)
}

// WithNewUploader sets the new uploader function for the app.
func WithNewUploader(nu newUploader) Options {
	return func(o *options) {
		o.newUploader = nu
	}
}
