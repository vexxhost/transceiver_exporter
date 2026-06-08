//go:build linux

package moduleeeprom

import (
	"syscall"
	"testing"
)

func TestErrorsIndicateAbsentModule(t *testing.T) {
	if !errorsIndicateAbsentModule(syscall.EIO) {
		t.Fatal("errorsIndicateAbsentModule(EIO) = false, want true")
	}
	if errorsIndicateAbsentModule(syscall.EINVAL) {
		t.Fatal("errorsIndicateAbsentModule(EINVAL) = true, want false")
	}
}
