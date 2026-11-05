package mcpmgr

import "errors"

var (
	ErrNotConfigured = errors.New("mcpmgr: not configured")
	ErrNotRunning    = errors.New("mcpmgr: server not running")
)
