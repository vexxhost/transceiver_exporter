package sff

import (
	"errors"

	"github.com/vexxhost/transceiver_exporter/pkg/transceiver"
)

// ErrUnsupportedFormat is returned when EEPROM bytes do not match a supported
// transceiver memory map.
var ErrUnsupportedFormat = errors.New("sff: unsupported transceiver eeprom format")

// Decode decodes raw transceiver EEPROM bytes into a normalized module model.
//
// Decode supports SFF-8472 SFP/SFP+, SFF-8636 QSFP/QSFP+/QSFP28, and CMIS
// modules. The returned module is independent of any host interface name; use
// DecodeObservation when the interface label is part of the desired result.
func Decode(data []byte) (transceiver.Module, error) {
	switch detect(data) {
	case transceiver.MemoryMapSFF8472:
		return sff8472Decoder{}.decode(data)
	case transceiver.MemoryMapSFF8636:
		return sff8636Decoder{}.decode(data)
	case transceiver.MemoryMapCMIS:
		return cmisDecoder{}.decode(data)
	default:
		return transceiver.Module{}, ErrUnsupportedFormat
	}
}

// DecodeObservation decodes raw EEPROM bytes and attaches an interface name to
// the result.
func DecodeObservation(interfaceName string, data []byte) (transceiver.Observation, error) {
	module, err := Decode(data)
	if err != nil {
		return transceiver.Observation{}, err
	}
	return transceiver.Observation{Interface: interfaceName, Module: module}, nil
}

func detect(data []byte) transceiver.MemoryMap {
	if len(data) == 0 {
		return transceiver.MemoryMapUnknown
	}

	switch data[0] {
	case 0x03:
		return transceiver.MemoryMapSFF8472
	case 0x0c, 0x0d, 0x11:
		return transceiver.MemoryMapSFF8636
	case 0x18, 0x19, 0x1e, 0x1f, 0x20:
		return transceiver.MemoryMapCMIS
	default:
		return transceiver.MemoryMapUnknown
	}
}
