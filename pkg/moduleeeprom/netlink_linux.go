//go:build linux

package moduleeeprom

import (
	"fmt"

	"github.com/mdlayher/genetlink"
	"github.com/mdlayher/netlink"
	"golang.org/x/sys/unix"
)

const (
	ethtoolModuleEEPROMHeader     = 1
	ethtoolModuleEEPROMOffset     = 2
	ethtoolModuleEEPROMLength     = 3
	ethtoolModuleEEPROMPage       = 4
	ethtoolModuleEEPROMBank       = 5
	ethtoolModuleEEPROMI2CAddress = 6
	ethtoolModuleEEPROMData       = 7

	modulePageSize     = 128
	moduleI2CAddressA0 = 0x50

	cmisMemoryModelOffset = 0x02
	cmisMemoryModelFlat   = 0x80
	cmisPagesAdvertOffset = 0x8e
	cmisBanksMask         = 0x03

	cmisPage00UpperOffset = 128
	cmisPage01Offset      = 256
	cmisPage02Offset      = 384
	cmisPage11Bank0Offset = 512
)

type netlinkClient struct {
	conn   *genetlink.Conn
	family uint16
}

func newNetlinkClient() (*netlinkClient, error) {
	conn, err := genetlink.Dial(&netlink.Config{Strict: true})
	if err != nil {
		return nil, err
	}

	family, err := conn.GetFamily(unix.ETHTOOL_GENL_NAME)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	return &netlinkClient{
		conn:   conn,
		family: family.ID,
	}, nil
}

func (c *netlinkClient) Close() error {
	return c.conn.Close()
}

func (c *netlinkClient) ModuleIdentifier(interfaceName string) (uint8, error) {
	data, err := c.read(interfaceName, 0, 0x00, 0, 1)
	if err != nil {
		return 0, err
	}
	if len(data) == 0 {
		return 0, fmt.Errorf("ethtool netlink module eeprom reply missing identifier")
	}
	return data[0], nil
}

func (c *netlinkClient) ModuleEEPROM(interfaceName string) ([]byte, error) {
	lower, err := c.read(interfaceName, 0, 0x00, 0, modulePageSize)
	if err != nil {
		return nil, err
	}

	data := make([]byte, cmisPage00UpperOffset+modulePageSize)
	copy(data, lower)

	page00, err := c.read(interfaceName, 0, 0x00, modulePageSize, modulePageSize)
	if err != nil {
		return data, nil
	}
	copy(data[cmisPage00UpperOffset:], page00)

	if len(lower) > cmisMemoryModelOffset && lower[cmisMemoryModelOffset]&cmisMemoryModelFlat != 0 {
		return data, nil
	}

	page01, err := c.read(interfaceName, 0, 0x01, modulePageSize, modulePageSize)
	if err != nil {
		return data, nil
	}
	data = appendPage(data, cmisPage01Offset, page01)

	page02, err := c.read(interfaceName, 0, 0x02, modulePageSize, modulePageSize)
	if err == nil {
		data = appendPage(data, cmisPage02Offset, page02)
	}

	for bank := 0; bank < cmisNumBanks(page01); bank++ {
		page11, err := c.read(interfaceName, uint8(bank), 0x11, modulePageSize, modulePageSize)
		if err != nil {
			continue
		}
		data = appendPage(data, cmisPage11Bank0Offset+bank*modulePageSize, page11)
	}

	return data, nil
}

func (c *netlinkClient) read(interfaceName string, bank, page uint8, offset, length uint32) ([]byte, error) {
	ae := netlink.NewAttributeEncoder()
	ae.Nested(ethtoolModuleEEPROMHeader, func(nae *netlink.AttributeEncoder) error {
		nae.String(unix.ETHTOOL_A_HEADER_DEV_NAME, interfaceName)
		return nil
	})
	ae.Uint32(ethtoolModuleEEPROMOffset, offset)
	ae.Uint32(ethtoolModuleEEPROMLength, length)
	ae.Uint8(ethtoolModuleEEPROMPage, page)
	ae.Uint8(ethtoolModuleEEPROMBank, bank)
	ae.Uint8(ethtoolModuleEEPROMI2CAddress, moduleI2CAddressA0)

	data, err := ae.Encode()
	if err != nil {
		return nil, err
	}

	msgs, err := c.conn.Execute(
		genetlink.Message{
			Header: genetlink.Header{
				Command: unix.ETHTOOL_MSG_MODULE_EEPROM_GET,
				Version: unix.ETHTOOL_GENL_VERSION,
			},
			Data: data,
		},
		c.family,
		netlink.Request,
	)
	if err != nil {
		return nil, err
	}

	return parseModuleEEPROMData(msgs)
}

func appendPage(data []byte, offset int, page []byte) []byte {
	end := offset + len(page)
	if len(data) < end {
		data = append(data, make([]byte, end-len(data))...)
	}
	copy(data[offset:end], page)
	return data
}

func cmisNumBanks(page01 []byte) int {
	if len(page01) <= cmisPagesAdvertOffset-modulePageSize {
		return 1
	}

	switch page01[cmisPagesAdvertOffset-modulePageSize] & cmisBanksMask {
	case 0x01:
		return 2
	case 0x02:
		return 4
	default:
		return 1
	}
}

func parseModuleEEPROMData(msgs []genetlink.Message) ([]byte, error) {
	for _, msg := range msgs {
		ad, err := netlink.NewAttributeDecoder(msg.Data)
		if err != nil {
			return nil, err
		}

		for ad.Next() {
			if ad.Type() == ethtoolModuleEEPROMData {
				return append([]byte(nil), ad.Bytes()...), nil
			}
		}
		if err := ad.Err(); err != nil {
			return nil, err
		}
	}

	return nil, fmt.Errorf("ethtool netlink module eeprom reply missing data")
}
