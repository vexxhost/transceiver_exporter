//go:build linux

package moduleeeprom

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mdlayher/genetlink"
	"github.com/mdlayher/netlink"
)

func TestParseModuleEEPROMData(t *testing.T) {
	want := []byte{0x1e, 0x52, 0x00, 0x07}

	ae := netlink.NewAttributeEncoder()
	ae.Bytes(ethtoolModuleEEPROMData, want)
	data, err := ae.Encode()
	if err != nil {
		t.Fatalf("encode fixture netlink attributes: %v", err)
	}

	got, err := parseModuleEEPROMData([]genetlink.Message{{Data: data}})
	if err != nil {
		t.Fatalf("parse module EEPROM data: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("module EEPROM data = %x, want %x", got, want)
	}

	got[0] = 0xff
	if want[0] != 0x1e {
		t.Fatalf("parser returned aliased data, source = %x", want)
	}
}

func TestParseModuleEEPROMDataRejectsMissingData(t *testing.T) {
	_, err := parseModuleEEPROMData([]genetlink.Message{{Data: nil}})
	if err == nil {
		t.Fatal("parse module EEPROM data succeeded, want error")
	}
	if !strings.Contains(err.Error(), "missing data") {
		t.Fatalf("error = %v, want missing data", err)
	}
}

func TestAppendPageExtendsAndCopiesAtOffset(t *testing.T) {
	data := appendPage([]byte{0xaa}, 3, []byte{0x01, 0x02})
	want := []byte{0xaa, 0x00, 0x00, 0x01, 0x02}

	if !bytes.Equal(data, want) {
		t.Fatalf("data = %x, want %x", data, want)
	}
}

func TestCMISNumBanks(t *testing.T) {
	tests := []struct {
		name     string
		mask     byte
		expected int
	}{
		{name: "default", mask: 0x00, expected: 1},
		{name: "two banks", mask: 0x01, expected: 2},
		{name: "four banks", mask: 0x02, expected: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page01 := make([]byte, modulePageSize)
			page01[cmisPagesAdvertOffset-modulePageSize] = tt.mask

			if got := cmisNumBanks(page01); got != tt.expected {
				t.Fatalf("cmis banks = %d, want %d", got, tt.expected)
			}
		})
	}
}
