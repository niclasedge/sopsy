//go:build !darwin && !windows

package web

var browserCommand = []string{"xdg-open"}
