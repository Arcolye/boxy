package manager

import "runtime"

func Detect() PackageManager {
	switch runtime.GOOS {
	case "darwin":
		mgr := &BrewManager{}
		if mgr.IsAvailable() {
			return mgr
		}
	case "linux":
		mgr := &AptManager{}
		if mgr.IsAvailable() {
			return mgr
		}
	}
	return nil
}
