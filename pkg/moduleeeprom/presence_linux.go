//go:build linux

package moduleeeprom

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	siocEthtool         = 0x8946
	ethtoolGModuleInfo  = 0x00000042
	interfaceNameLength = 16
)

type moduleInfoProbe struct {
	fd int
}

type moduleInfoRequest struct {
	cmd        uint32
	moduleType uint32
	eepromLen  uint32
	reserved   [8]uint32
}

type ioctlRequest struct {
	name [interfaceNameLength]byte
	data uintptr
}

func newModulePresenceProbe() (modulePresenceProber, error) {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, 0)
	if err != nil {
		return nil, err
	}

	return &moduleInfoProbe{fd: fd}, nil
}

func (p *moduleInfoProbe) Close() error {
	return unix.Close(p.fd)
}

func (p *moduleInfoProbe) ModulePresent(interfaceName string) (bool, error) {
	req := moduleInfoRequest{cmd: ethtoolGModuleInfo}
	if err := p.ioctl(interfaceName, uintptr(unsafe.Pointer(&req))); err != nil {
		if errorsIndicateAbsentModule(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (p *moduleInfoProbe) ioctl(interfaceName string, data uintptr) error {
	var name [interfaceNameLength]byte
	copy(name[:], []byte(interfaceName))

	req := ioctlRequest{
		name: name,
		data: data,
	}

	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(p.fd), siocEthtool, uintptr(unsafe.Pointer(&req)))
	if errno != 0 {
		return errno
	}

	return nil
}

func errorsIndicateAbsentModule(err error) bool {
	// Drivers often surface an empty module cage as EIO on the ETHTOOL_GMODULEINFO
	// probe before any EEPROM bytes are read.
	return err == syscall.EIO
}
