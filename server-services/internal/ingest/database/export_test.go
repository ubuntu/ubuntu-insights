package database

import "context"

type DBPool = dbPool

// WithMewPool is an option to override the default newPool function.
func WithNewPool(newPool func(ctx context.Context, dsn string) (DBPool, error)) Options {
	return func(opts *options) {
		opts.newPool = newPool
	}
}
