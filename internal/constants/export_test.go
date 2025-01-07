package constants

type Option = option

func WithBaseDir(baseDir func() (string, error)) option {
	return func(o *options) {
		o.baseDir = baseDir
	}
}
