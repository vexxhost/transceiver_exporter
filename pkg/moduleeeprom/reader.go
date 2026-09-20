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
	ModuleIdentifier(interfaceName string) (uint8, error)
	ModuleEEPROM(interfaceName string) ([]byte, error)
	Close() error
}

type modulePresenceProber interface {
	ModulePresent(interfaceName string) (bool, error)
	Close() error
}

// ErrModuleAbsent reports that the interface currently has no plugged
// transceiver module to read.
var ErrModuleAbsent = errors.New("moduleeeprom: module absent")

// Reader reads module EEPROM data for network interfaces.
//
// Reader is safe for concurrent use. It serializes access to the underlying
// ethtool clients because the ioctl client keeps process-level file descriptor
// state and module EEPROM reads can require multi-page netlink fallback.
type Reader struct {
	mu       sync.Mutex
	ioctl    ioctlClient
	netlink  netlinkModuleReader
	presence modulePresenceProber
}

// New opens an ethtool-backed Reader.
//
// When netlink support is available, the reader probes the module identifier
// through netlink first so CMIS modules can avoid drivers whose legacy ioctl
// EEPROM path does not recognize newer module IDs. SFF-8472 and SFF-8636
// modules continue to use the legacy ioctl path by default. CMIS modules are
// read through netlink so upper pages and banked lane diagnostics can be
// collected.
func New() (*Reader, error) {
	ioctl, err := safchain.NewEthtool()
	if err != nil {
		return nil, err
	}

	netlink, _ := newNetlinkClient()
	presence, _ := newModulePresenceProbe()
	return newReaderWithPresence(ioctl, netlink, presence), nil
}

func newReader(ioctl ioctlClient, netlink netlinkModuleReader) *Reader {
	return newReaderWithPresence(ioctl, netlink, nil)
}

func newReaderWithPresence(ioctl ioctlClient, netlink netlinkModuleReader, presence modulePresenceProber) *Reader {
	if isNilNetlinkReader(netlink) {
		netlink = nil
	}
	return &Reader{ioctl: ioctl, netlink: netlink, presence: presence}
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
	if r.presence != nil {
		closeErr := r.presence.Close()
		if err == nil {
			err = closeErr
		}
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

	if r.presence != nil {
		present, err := r.presence.ModulePresent(interfaceName)
		if err == nil && !present {
			return nil, ErrModuleAbsent
		}
	}

	var (
		netlinkData  []byte
		netlinkErr   error
		triedNetlink bool
	)

	if r.netlink != nil {
		moduleID, err := r.netlink.ModuleIdentifier(interfaceName)
		if err == nil && isCMISIdentifier(moduleID) {
			triedNetlink = true
			netlinkData, netlinkErr = r.netlink.ModuleEEPROM(interfaceName)
			if netlinkErr == nil {
				return netlinkData, nil
			}
		}
	}

	data, err := r.ioctl.ModuleEeprom(interfaceName)
	if err == nil {
		if r.netlink != nil && !triedNetlink && isCMIS(data) {
			netlinkData, netlinkErr = r.netlink.ModuleEEPROM(interfaceName)
			if netlinkErr == nil {
				return netlinkData, nil
			}
		}
		return data, nil
	}
	if r.netlink == nil {
		return nil, err
	}

	if !triedNetlink {
		netlinkData, netlinkErr = r.netlink.ModuleEEPROM(interfaceName)
		if netlinkErr == nil {
			return netlinkData, nil
		}
	}

	return nil, errors.Join(
		fmt.Errorf("ioctl module eeprom: %w", err),
		fmt.Errorf("netlink module eeprom: %w", netlinkErr),
	)
}

func isCMIS(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	return isCMISIdentifier(data[0])
}

func isCMISIdentifier(moduleID uint8) bool {
	switch moduleID {
	case 0x18, 0x19, 0x1e, 0x1f, 0x20:
		return true
	default:
		return false
	}
}
