package moduleeeprom

import (
	"errors"
	"fmt"
	"reflect"
	"sync"

	safchain "github.com/safchain/ethtool"
)

type ioctlClient interface {
	ModuleEeprom(interfaceName string) ([]byte, error)
	Close()
}

type netlinkModuleReader interface {
	ModuleEEPROM(interfaceName string) ([]byte, error)
	Close() error
}

// Reader reads module EEPROM data for network interfaces.
//
// Reader is safe for concurrent use. It serializes access to the underlying
// ethtool clients because the ioctl client keeps process-level file descriptor
// state and module EEPROM reads can require multi-page netlink fallback.
type Reader struct {
	mu      sync.Mutex
	ioctl   ioctlClient
	netlink netlinkModuleReader
}

// New opens an ethtool-backed Reader.
//
// The returned reader always attempts ioctl access first. When netlink support
// is available, CMIS modules are reread through netlink so upper pages and banked
// lane diagnostics can be collected.
func New() (*Reader, error) {
	ioctl, err := safchain.NewEthtool()
	if err != nil {
		return nil, err
	}

	netlink, _ := newNetlinkClient()
	return newReader(ioctl, netlink), nil
}

func newReader(ioctl ioctlClient, netlink netlinkModuleReader) *Reader {
	if isNilNetlinkReader(netlink) {
		netlink = nil
	}
	return &Reader{ioctl: ioctl, netlink: netlink}
}

func isNilNetlinkReader(reader netlinkModuleReader) bool {
	if reader == nil {
		return true
	}

	value := reflect.ValueOf(reader)
	return value.Kind() == reflect.Pointer && value.IsNil()
}

// Close releases resources held by the Reader.
func (r *Reader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var err error
	if r.netlink != nil {
		err = r.netlink.Close()
	}
	if r.ioctl != nil {
		r.ioctl.Close()
	}
	return err
}

// ModuleEEPROM reads raw module EEPROM bytes for interfaceName.
//
// For SFF-8472 and SFF-8636 modules, the returned data is usually the ioctl
// result. For CMIS modules, the reader prefers netlink data when available so
// callers can decode upper-page and banked diagnostics.
func (r *Reader) ModuleEEPROM(interfaceName string) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := r.ioctl.ModuleEeprom(interfaceName)
	if err == nil {
		if r.netlink != nil && isCMIS(data) {
			netlinkData, netlinkErr := r.netlink.ModuleEEPROM(interfaceName)
			if netlinkErr == nil {
				return netlinkData, nil
			}
		}
		return data, nil
	}
	if r.netlink == nil {
		return nil, err
	}

	netlinkData, netlinkErr := r.netlink.ModuleEEPROM(interfaceName)
	if netlinkErr != nil {
		return nil, errors.Join(
			fmt.Errorf("ioctl module eeprom: %w", err),
			fmt.Errorf("netlink module eeprom: %w", netlinkErr),
		)
	}
	return netlinkData, nil
}

func isCMIS(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	switch data[0] {
	case 0x18, 0x19, 0x1e, 0x1f, 0x20:
		return true
	default:
		return false
	}
}
