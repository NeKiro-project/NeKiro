//go:build !windows

package file

func leafStateAfterAccessFailure(_, _ string) leafFailureState {
	return leafFailureUnknown
}
