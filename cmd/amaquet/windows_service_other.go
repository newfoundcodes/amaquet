//go:build !windows

package main

func runAsWindowsService(run func(<-chan struct{})) bool {
	return false
}
