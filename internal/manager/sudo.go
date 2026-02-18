package manager

import "os/exec"

// SudoCached returns true if sudo credentials are already cached
// (i.e., sudo can run without a password prompt).
func SudoCached() bool {
	err := exec.Command("sudo", "-n", "true").Run()
	return err == nil
}
