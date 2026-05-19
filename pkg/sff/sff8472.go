package sff

import (
	"fmt"

	"github.com/vexxhost/transceiver_exporter/pkg/transceiver"
)

type sff8472Decoder struct{}

func (d sff8472Decoder) decode(data []byte) (transceiver.Module, error) {
	if len(data) < 96 {
		return transceiver.Module{}, fmt.Errorf("sff8472 eeprom too short: %d bytes", len(data))
	}

	module := transceiver.Module{
		MemoryMap:          transceiver.MemoryMapSFF8472,
		FormFactor:         transceiver.FormFactor(data[0]),
		ExtendedIdentifier: d.extendedIdentifier(data[1]),
		Connector:          transceiver.Connector(data[2]),
		Encoding:           transceiver.Encoding(data[11]),
		RateIdentifier:     transceiver.RateIdentifier(data[13]),
		Capabilities:       d.capabilities(data),
		CableCompliance: transceiver.CableCompliance{
			Kind:     d.cableKind(data),
			Standard: d.copperCompliance(data),
		},
		Vendor: transceiver.Vendor{
			Name:         cleanString(data[20:36]),
			OUI:          oui(data[37:40]),
			PartNumber:   cleanString(data[40:56]),
			Revision:     cleanString(data[56:60]),
			SerialNumber: cleanString(data[68:84]),
			DateCode:     cleanString(data[84:92]),
		},
		BitRate: d.bitRate(data),
		Lengths: d.lengths(data),
		Options: d.options(data[64:66]),
		Raw: transceiver.Raw{
			ExtendedIdentifier: data[1],
			TransceiverCodes:   d.transceiverCodes(data),
			OptionValues:       append([]byte(nil), data[64:66]...),
			CableCompliance:    data[60],
		},
	}

	if len(data) < 512 {
		return module, nil
	}

	module.Raw.DiagnosticsSupportReported = true
	module.Raw.DiagnosticsSupported = d.diagnosticsSupported(data)
	module.Diagnostics.Supported = module.Raw.DiagnosticsSupported
	if !module.Diagnostics.Supported {
		return module, nil
	}

	a2 := data[256:]
	module.Diagnostics.TemperatureCelsius = transceiver.NewReading(temp(a2[96:98]))
	module.Diagnostics.VoltageVolts = transceiver.NewReading(voltage(a2[98:100]))

	lane := transceiver.Lane{
		Index:             1,
		TXBiasMilliAmps:   transceiver.NewReading(bias(a2[100:102])),
		TXPowerMilliWatts: transceiver.NewReading(power(a2[102:104])),
		RXPowerMilliWatts: transceiver.NewReading(power(a2[104:106])),
	}
	module.Diagnostics.Lanes = []transceiver.Lane{lane}
	module.Diagnostics.Alarms = append(module.Diagnostics.Alarms, thresholdAlarms(module.Diagnostics.TemperatureCelsius, "temperature", temp, a2[0:8])...)
	module.Diagnostics.Alarms = append(module.Diagnostics.Alarms, thresholdAlarms(module.Diagnostics.VoltageVolts, "voltage", voltage, a2[8:16])...)
	module.Diagnostics.Alarms = append(module.Diagnostics.Alarms, thresholdAlarmsWithLane(lane.TXBiasMilliAmps, "tx_bias", bias, a2[16:24], lane.Index)...)
	module.Diagnostics.Alarms = append(module.Diagnostics.Alarms, thresholdAlarmsWithLane(lane.TXPowerMilliWatts, "tx_power", power, a2[24:32], lane.Index)...)
	module.Diagnostics.Alarms = append(module.Diagnostics.Alarms, thresholdAlarmsWithLane(lane.RXPowerMilliWatts, "rx_power", power, a2[32:40], lane.Index)...)
	module.Diagnostics.Thresholds = append(module.Diagnostics.Thresholds, thresholds("temperature_celsius", temp, a2[0:8])...)
	module.Diagnostics.Thresholds = append(module.Diagnostics.Thresholds, thresholds("voltage_volts", voltage, a2[8:16])...)
	module.Diagnostics.Thresholds = append(module.Diagnostics.Thresholds, thresholdsWithLane("tx_bias_milliamps", bias, a2[16:24], lane.Index)...)
	module.Diagnostics.Thresholds = append(module.Diagnostics.Thresholds, thresholdsWithLane("tx_power_milliwatts", power, a2[24:32], lane.Index)...)
	module.Diagnostics.Thresholds = append(module.Diagnostics.Thresholds, thresholdsWithLane("rx_power_milliwatts", power, a2[32:40], lane.Index)...)

	return module, nil
}

func (d sff8472Decoder) lengths(data []byte) []transceiver.Length {
	var lengths []transceiver.Length
	addLength := func(medium transceiver.LengthMedium, meters int) {
		if meters > 0 {
			lengths = append(lengths, transceiver.Length{Medium: medium, Meters: meters})
		}
	}

	addLength(transceiver.LengthMediumSingleMode, int(data[14])*1000)
	addLength(transceiver.LengthMediumSingleMode, int(data[15])*100)
	addLength(transceiver.LengthMediumMultimode50um, int(data[16])*10)
	addLength(transceiver.LengthMediumMultimode625um, int(data[17])*10)
	addLength(transceiver.LengthMediumCopper, int(data[18]))
	addLength(transceiver.LengthMediumOM3, int(data[19])*10)

	return lengths
}

func (d sff8472Decoder) bitRate(data []byte) transceiver.BitRate {
	switch data[12] {
	case 0x00:
		return transceiver.BitRate{}
	case 0xff:
		margin := int(data[67])
		return transceiver.BitRate{
			NominalMBd:       int(data[66]) * 250,
			MaxMarginPercent: margin,
			MinMarginPercent: margin,
		}
	default:
		return transceiver.BitRate{
			NominalMBd:       int(data[12]) * 100,
			MaxMarginPercent: int(data[66]),
			MinMarginPercent: int(data[67]),
		}
	}
}

func (d sff8472Decoder) transceiverCodes(data []byte) []byte {
	codes := append([]byte(nil), data[3:11]...)
	return append(codes, data[36])
}

func (d sff8472Decoder) diagnosticsSupported(data []byte) bool {
	return data[92]&0x40 != 0
}

func (d sff8472Decoder) extendedIdentifier(code byte) string {
	switch code {
	case 0x04:
		return "GBIC/SFP defined by 2-wire interface ID"
	default:
		return "unknown"
	}
}

func (d sff8472Decoder) capabilities(data []byte) []transceiver.Capability {
	var capabilities []transceiver.Capability

	if data[3]&0x01 != 0 {
		capabilities = append(capabilities, transceiver.CapabilityInfiniBand1XCopperPassive)
	}
	if data[3]&0x02 != 0 {
		capabilities = append(capabilities, transceiver.CapabilityInfiniBand1XCopperActive)
	}
	if data[6]&0x04 != 0 {
		capabilities = append(capabilities, transceiver.CapabilityEthernet1000BaseCX)
	}
	if data[7]&0x01 != 0 {
		capabilities = append(capabilities, transceiver.CapabilityFCShortDistance)
	}
	if data[7]&0x40 != 0 {
		capabilities = append(capabilities, transceiver.CapabilityFCElectricalInterEnclosure)
	}
	if data[8]&0x80 != 0 {
		capabilities = append(capabilities, transceiver.CapabilityFCElectricalIntraEnclosure)
	}
	if data[8]&0x04 != 0 {
		capabilities = append(capabilities, transceiver.CapabilityPassiveCable)
	}
	if data[8]&0x08 != 0 {
		capabilities = append(capabilities, transceiver.CapabilityActiveCable)
	}
	if data[9]&0x80 != 0 {
		capabilities = append(capabilities, transceiver.CapabilityFCTwinAxialPair)
	}
	if data[10]&0x80 != 0 {
		capabilities = append(capabilities, transceiver.CapabilityFC1200MBytes)
	}
	if data[10]&0x40 != 0 {
		capabilities = append(capabilities, transceiver.CapabilityFC800MBytes)
	}
	if data[10]&0x10 != 0 {
		capabilities = append(capabilities, transceiver.CapabilityFC400MBytes)
	}
	if data[10]&0x04 != 0 {
		capabilities = append(capabilities, transceiver.CapabilityFC200MBytes)
	}
	if data[10]&0x01 != 0 {
		capabilities = append(capabilities, transceiver.CapabilityFC100MBytes)
	}
	if data[36] == 0x0c {
		capabilities = append(capabilities, transceiver.CapabilityExtended25GBaseCRCAS)
	}

	return capabilities
}

func (d sff8472Decoder) cableKind(data []byte) transceiver.CableKind {
	for _, capability := range d.capabilities(data) {
		if capability == transceiver.CapabilityActiveCable {
			return transceiver.CableKindActive
		}
	}
	return transceiver.CableKindPassive
}

func (d sff8472Decoder) copperCompliance(data []byte) string {
	if d.cableKind(data) == transceiver.CableKindActive {
		switch data[60] {
		case 0x01:
			return "SFF-8431 appendix E"
		case 0x04:
			return "SFF-8431 limiting"
		case 0x00:
			return "unspecified"
		default:
			return "unknown"
		}
	}

	switch data[60] {
	case 0x00:
		return "unspecified"
	case 0x01:
		return "SFF-8431 appendix E"
	default:
		return "unknown"
	}
}

func (d sff8472Decoder) options(data []byte) []transceiver.Option {
	if len(data) < 2 {
		return nil
	}

	var options []transceiver.Option
	if data[1]&0x10 != 0 {
		options = append(options, transceiver.OptionRXLOS)
	}
	if data[1]&0x02 != 0 {
		options = append(options, transceiver.OptionTXDisable)
	}
	return options
}
