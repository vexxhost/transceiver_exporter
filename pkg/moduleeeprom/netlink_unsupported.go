//go:build !linux

package moduleeeprom

import "errors"

type netlinkClient struct{}

func newNetlinkClient() (*netlinkClient, error) {
	return nil, errors.New("moduleeeprom: ethtool netlink module eeprom is only supported on linux")
}

func (c *netlinkClient) Close() error {
	return nil
}

func (c *netlinkClient) ModuleEEPROM(string) ([]byte, error) {
	return nil, errors.New("moduleeeprom: ethtool netlink module eeprom is only supported on linux")
}
