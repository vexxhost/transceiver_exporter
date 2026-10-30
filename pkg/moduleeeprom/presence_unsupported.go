//go:build !linux

package moduleeeprom

import "errors"

func newModulePresenceProbe() (modulePresenceProber, error) {
	return nil, errors.New("moduleeeprom: module presence probe is only supported on linux")
}
