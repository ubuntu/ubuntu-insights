// Package testutils provides utility functions specific to the Ubuntu Insights server module.
// It should not be used outside of a testing context.
package testutils

import "testing"

func init() {
	if !testing.Testing() {
		panic("testutils package should only be used in a testing context")
	}
}
