package netdev

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"slices"
	"syscall"
	"testing"
)

func TestCandidateNamesFromInterfacesFiltersVirtualDevices(t *testing.T) {
	sysClassNet := t.TempDir()
	mkdir(t, sysClassNet, "lo")
	mkdir(t, sysClassNet, "eno1", "device")
	mkdir(t, sysClassNet, "bond0")
	mkdir(t, sysClassNet, "lxc1234")
	mkdir(t, sysClassNet, "eno2", "device")

	got := CandidateNamesFromInterfaces(sysClassNet, []net.Interface{
		{Name: "lo", Flags: net.FlagLoopback},
		{Name: "eno1"},
		{Name: "bond0"},
		{Name: "lxc1234"},
		{Name: "eno2"},
	})
	want := []string{"eno1", "eno2"}

	if !slices.Equal(got, want) {
		t.Fatalf("candidate names = %v, want %v", got, want)
	}
}

func TestIsUnsupportedModuleError(t *testing.T) {
	for _, err := range []error{
		syscall.EOPNOTSUPP,
		syscall.ENODEV,
		syscall.ENOTTY,
		fmt.Errorf("wrapped: %w", syscall.EOPNOTSUPP),
	} {
		if !IsUnsupportedModuleError(err) {
			t.Fatalf("IsUnsupportedModuleError(%v) = false, want true", err)
		}
	}

	if IsUnsupportedModuleError(os.ErrPermission) {
		t.Fatal("IsUnsupportedModuleError(permission denied) = true, want false")
	}
	if IsUnsupportedModuleError(syscall.EINVAL) {
		t.Fatal("IsUnsupportedModuleError(invalid argument) = true, want false")
	}
}

func mkdir(t *testing.T, root string, parts ...string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Join(append([]string{root}, parts...)...), 0o755); err != nil {
		t.Fatalf("mkdir %v: %v", parts, err)
	}
}
