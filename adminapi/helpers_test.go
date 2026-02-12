package adminapi

import "sync"

// Because getConfig in config.go calls sync.OnceValues, the new values set to
// SERVERADMIN_BASE_URL between test runs is never changed, as getConfig returns
// cached values.
// We use resetConfig() to reinitialize things, forcing getConfig() to return the
// values from the new env variables.
func resetConfig() {
	getConfig = sync.OnceValues(loadConfig)
}
