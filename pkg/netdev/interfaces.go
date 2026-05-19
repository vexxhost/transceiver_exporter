package netdev

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"syscall"
)

const sysClassNet = "/sys/class/net"

// CandidateNames returns physical non-loopback network devices that are worth
// trying for transceiver EEPROM reads.
func CandidateNames() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	return CandidateNamesFromInterfaces(sysClassNet, interfaces), nil
}

// CandidateNamesFromInterfaces filters a supplied interface list using sysfs.
func CandidateNamesFromInterfaces(sysClassNet string, interfaces []net.Interface) []string {
	names := make([]string, 0, len(interfaces))
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if !hasDevice(sysClassNet, iface.Name) {
			continue
		}
		names = append(names, iface.Name)
	}
	return names
}

// IsUnsupportedModuleError reports kernel errors used when a netdev does not
// implement ethtool module EEPROM access.
func IsUnsupportedModuleError(err error) bool {
	return errors.Is(err, syscall.EOPNOTSUPP) ||
		errors.Is(err, syscall.ENODEV) ||
		errors.Is(err, syscall.ENOTTY)
}

func hasDevice(sysClassNet, name string) bool {
	_, err := os.Stat(filepath.Join(sysClassNet, name, "device"))
	return err == nil
}
