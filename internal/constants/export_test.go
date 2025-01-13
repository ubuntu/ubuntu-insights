package constants

type Option = option

func WithBaseDir(baseDir func() (string, error)) func(o *options) {
	return func(o *options) {
		o.baseDir = baseDir
	}
}
