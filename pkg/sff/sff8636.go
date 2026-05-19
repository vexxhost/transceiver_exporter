package sff

import (
	"encoding/binary"
	"fmt"

	"github.com/vexxhost/transceiver_exporter/pkg/transceiver"
)

const (
	sff8636StatusOffset       = 0x02
	sff8636StatusDataNotReady = 0x01

	sff8636LOSFlagsOffset   = 0x03
	sff8636FaultFlagsOffset = 0x04
	sff8636TempFlagsOffset  = 0x06
	sff8636VCCFlagsOffset   = 0x07

	sff8636RXPower12FlagsOffset = 0x09
	sff8636RXPower34FlagsOffset = 0x0a
	sff8636TXBias12FlagsOffset  = 0x0b
	sff8636TXBias34FlagsOffset  = 0x0c
	sff8636TXPower12FlagsOffset = 0x0d
	sff8636TXPower34FlagsOffset = 0x0e

	sff8636RXPowerOffset = 0x22
	sff8636TXBiasOffset  = 0x2a
	sff8636TXPowerOffset = 0x32

	sff8636PowerModeOffset       = 0x5d
	sff8636HighPowerEnable       = 0x04
	sff8636LowPowerMode          = 0x02
	sff8636PowerOverride         = 0x01
	sff8636ConnectorOffset       = 0x82
	sff8636BitRateOffset         = 0x8c
	sff8636SMLengthOffset        = 0x8e
	sff8636OM3LengthOffset       = 0x8f
	sff8636OM2LengthOffset       = 0x90
	sff8636OM1LengthOffset       = 0x91
	sff8636CableLengthOffset     = 0x92
	sff8636VendorNameStartOffset = 0x94
	sff8636VendorNameEndOffset   = 0xa4
	sff8636VendorOUIOffset       = 0xa5
	sff8636VendorPNStartOffset   = 0xa8
	sff8636VendorPNEndOffset     = 0xb8
	sff8636VendorRevStartOffset  = 0xb8
	sff8636VendorRevEndOffset    = 0xba
	sff8636WavelengthOffset      = 0xba
	sff8636WaveToleranceOffset   = 0xbc
	sff8636Option4Offset         = 0xc3
	sff8636VendorSNStartOffset   = 0xc4
	sff8636VendorSNEndOffset     = 0xd4
	sff8636DateStartOffset       = 0xd4
	sff8636DateEndOffset         = 0xda

	sff8636TemperatureThresholdOffset = 0x200
	sff8636VoltageThresholdOffset     = 0x210
	sff8636RXPowerThresholdOffset     = 0x230
	sff8636TXBiasThresholdOffset      = 0x238
	sff8636TXPowerThresholdOffset     = 0x240
)

type sff8636Decoder struct{}

func (d sff8636Decoder) decode(data []byte) (transceiver.Module, error) {
	if len(data) < 128 {
		return transceiver.Module{}, fmt.Errorf("sff8636 eeprom too short: %d bytes", len(data))
	}

	module := transceiver.Module{
		MemoryMap:  transceiver.MemoryMapSFF8636,
		FormFactor: transceiver.FormFactor(data[0]),
	}
	if len(data) > sff8636StatusOffset {
		module.Status = d.status(data)
	}
	if len(data) > sff8636ConnectorOffset {
		module.Connector = transceiver.Connector(data[sff8636ConnectorOffset])
	}
	if len(data) > sff8636BitRateOffset {
		module.BitRate = transceiver.BitRate{NominalMBd: int(data[sff8636BitRateOffset]) * 100}
	}
	if len(data) > sff8636CableLengthOffset {
		module.Lengths = d.lengths(data)
	}
	if len(data) >= sff8636VendorSNEndOffset {
		module.Vendor = transceiver.Vendor{
			Name:         cleanString(data[sff8636VendorNameStartOffset:sff8636VendorNameEndOffset]),
			OUI:          oui(data[sff8636VendorOUIOffset : sff8636VendorOUIOffset+3]),
			PartNumber:   cleanString(data[sff8636VendorPNStartOffset:sff8636VendorPNEndOffset]),
			Revision:     cleanString(data[sff8636VendorRevStartOffset:sff8636VendorRevEndOffset]),
			SerialNumber: cleanString(data[sff8636VendorSNStartOffset:sff8636VendorSNEndOffset]),
		}
	}
	if len(data) >= sff8636DateEndOffset {
		module.Vendor.DateCode = cleanString(data[sff8636DateStartOffset:sff8636DateEndOffset])
	}
	if media := d.media(data); media != nil {
		module.Media = media
	}
	if power := d.power(data); power != nil {
		module.Power = power
	}

	module.Diagnostics.TemperatureCelsius = transceiver.NewReading(temp(data[22:24]))
	module.Diagnostics.VoltageVolts = transceiver.NewReading(voltage(data[26:28]))

	if len(data) >= 58 {
		for i := 0; i < 4; i++ {
			module.Diagnostics.Lanes = append(module.Diagnostics.Lanes, transceiver.Lane{
				Index:             i + 1,
				RXPowerMilliWatts: transceiver.NewReading(power(data[34+i*2 : 36+i*2])),
				TXBiasMilliAmps:   transceiver.NewReading(bias(data[42+i*2 : 44+i*2])),
				TXPowerMilliWatts: transceiver.NewReading(power(data[50+i*2 : 52+i*2])),
			})
		}
	}
	module.Diagnostics.Supported = len(module.Diagnostics.Lanes) > 0 ||
		module.Diagnostics.TemperatureCelsius.Valid ||
		module.Diagnostics.VoltageVolts.Valid
	module.Diagnostics.Alarms = d.alarms(data, module.Diagnostics.Lanes)
	module.Diagnostics.Thresholds = d.thresholds(data, module.Diagnostics.Lanes)

	return module, nil
}

func (d sff8636Decoder) status(data []byte) *transceiver.ModuleStatus {
	status := &transceiver.ModuleStatus{State: "ready"}
	if data[sff8636StatusOffset]&sff8636StatusDataNotReady != 0 {
		status.State = "not_ready"
	}
	if len(data) > sff8636PowerModeOffset {
		status.LowPowerRequestSoftware = data[sff8636PowerModeOffset]&sff8636LowPowerMode != 0
	}
	return status
}

func (d sff8636Decoder) lengths(data []byte) []transceiver.Length {
	lengths := []transceiver.Length{}
	lengths = appendSFF8636Length(lengths, transceiver.LengthMediumSingleMode, int(data[sff8636SMLengthOffset])*1000)
	lengths = appendSFF8636Length(lengths, transceiver.LengthMediumOM3, int(data[sff8636OM3LengthOffset])*2)
	lengths = appendSFF8636Length(lengths, transceiver.LengthMediumOM2, int(data[sff8636OM2LengthOffset]))
	lengths = appendSFF8636Length(lengths, transceiver.LengthMediumOM1, int(data[sff8636OM1LengthOffset]))
	lengths = appendSFF8636Length(lengths, transceiver.LengthMediumCableAssembly, int(data[sff8636CableLengthOffset]))
	return lengths
}

func (d sff8636Decoder) media(data []byte) *transceiver.Media {
	if len(data) < sff8636WaveToleranceOffset+2 {
		return nil
	}

	media := &transceiver.Media{}
	if !allZero(data[sff8636WavelengthOffset : sff8636WavelengthOffset+2]) {
		media.WavelengthNanometers = float64(binary.BigEndian.Uint16(data[sff8636WavelengthOffset:sff8636WavelengthOffset+2])) * 0.05
	}
	if !allZero(data[sff8636WaveToleranceOffset : sff8636WaveToleranceOffset+2]) {
		media.WavelengthToleranceNanometers = float64(binary.BigEndian.Uint16(data[sff8636WaveToleranceOffset:sff8636WaveToleranceOffset+2])) * 0.005
	}
	if media.WavelengthNanometers == 0 && media.WavelengthToleranceNanometers == 0 {
		return nil
	}
	return media
}

func (d sff8636Decoder) power(data []byte) *transceiver.ModulePower {
	if len(data) <= sff8636ConnectorOffset {
		return nil
	}

	extID := data[sff8636ConnectorOffset-1]
	class, maxWatts := sff8636PowerClass(extID)
	if len(data) > sff8636PowerModeOffset && data[sff8636PowerModeOffset]&sff8636HighPowerEnable == 0 && class > 4 {
		class = 4
		maxWatts = 3.5
	}
	if class == 0 {
		return nil
	}
	return &transceiver.ModulePower{Class: class, MaxWatts: maxWatts}
}

func (d sff8636Decoder) alarms(data []byte, lanes []transceiver.Lane) []transceiver.Alarm {
	alarms := []transceiver.Alarm{
		flagAlarm("temperature_high", "alarm", byteAt(data, sff8636TempFlagsOffset), 0x80, 0),
		flagAlarm("temperature_low", "alarm", byteAt(data, sff8636TempFlagsOffset), 0x40, 0),
		flagAlarm("temperature_high", "warning", byteAt(data, sff8636TempFlagsOffset), 0x20, 0),
		flagAlarm("temperature_low", "warning", byteAt(data, sff8636TempFlagsOffset), 0x10, 0),
		flagAlarm("voltage_high", "alarm", byteAt(data, sff8636VCCFlagsOffset), 0x80, 0),
		flagAlarm("voltage_low", "alarm", byteAt(data, sff8636VCCFlagsOffset), 0x40, 0),
		flagAlarm("voltage_high", "warning", byteAt(data, sff8636VCCFlagsOffset), 0x20, 0),
		flagAlarm("voltage_low", "warning", byteAt(data, sff8636VCCFlagsOffset), 0x10, 0),
	}

	if block := thresholdBlock(data, sff8636TemperatureThresholdOffset); block != nil {
		alarms = append(alarms, thresholdAlarms(transceiver.NewReading(temp(data[22:24])), "temperature", temp, block)...)
	}
	if block := thresholdBlock(data, sff8636VoltageThresholdOffset); block != nil {
		alarms = append(alarms, thresholdAlarms(transceiver.NewReading(voltage(data[26:28])), "voltage", voltage, block)...)
	}

	for _, lane := range lanes {
		alarms = append(alarms, d.laneAlarms(data, lane)...)
	}

	return mergeAlarms(alarms)
}

func (d sff8636Decoder) laneAlarms(data []byte, lane transceiver.Lane) []transceiver.Alarm {
	alarms := []transceiver.Alarm{
		flagAlarm("tx_los", "fault", byteAt(data, sff8636LOSFlagsOffset), byte(1<<(lane.Index+3)), lane.Index),
		flagAlarm("rx_los", "fault", byteAt(data, sff8636LOSFlagsOffset), byte(1<<(lane.Index-1)), lane.Index),
		flagAlarm("tx_fault", "fault", byteAt(data, sff8636FaultFlagsOffset), byte(1<<(lane.Index-1)), lane.Index),
	}

	alarms = append(alarms, laneMonitorFlagAlarms("rx_power", data, sff8636RXPower12FlagsOffset, sff8636RXPower34FlagsOffset, lane.Index)...)
	alarms = append(alarms, laneMonitorFlagAlarms("tx_bias", data, sff8636TXBias12FlagsOffset, sff8636TXBias34FlagsOffset, lane.Index)...)
	alarms = append(alarms, laneMonitorFlagAlarms("tx_power", data, sff8636TXPower12FlagsOffset, sff8636TXPower34FlagsOffset, lane.Index)...)

	if data := thresholdBlock(data, sff8636RXPowerThresholdOffset); data != nil {
		alarms = append(alarms, thresholdAlarmsWithLane(lane.RXPowerMilliWatts, "rx_power", power, data, lane.Index)...)
	}
	if data := thresholdBlock(data, sff8636TXBiasThresholdOffset); data != nil {
		alarms = append(alarms, thresholdAlarmsWithLane(lane.TXBiasMilliAmps, "tx_bias", bias, data, lane.Index)...)
	}
	if data := thresholdBlock(data, sff8636TXPowerThresholdOffset); data != nil {
		alarms = append(alarms, thresholdAlarmsWithLane(lane.TXPowerMilliWatts, "tx_power", power, data, lane.Index)...)
	}

	return alarms
}

func (d sff8636Decoder) thresholds(data []byte, lanes []transceiver.Lane) []transceiver.Threshold {
	thresholdValues := []transceiver.Threshold{}
	if data := thresholdBlock(data, sff8636TemperatureThresholdOffset); data != nil {
		thresholdValues = append(thresholdValues, thresholds("temperature_celsius", temp, data)...)
	}
	if data := thresholdBlock(data, sff8636VoltageThresholdOffset); data != nil {
		thresholdValues = append(thresholdValues, thresholds("voltage_volts", voltage, data)...)
	}
	for _, lane := range lanes {
		if data := thresholdBlock(data, sff8636RXPowerThresholdOffset); data != nil && lane.RXPowerMilliWatts.Valid {
			thresholdValues = append(thresholdValues, thresholdsWithLane("rx_power_milliwatts", power, data, lane.Index)...)
		}
		if data := thresholdBlock(data, sff8636TXBiasThresholdOffset); data != nil && lane.TXBiasMilliAmps.Valid {
			thresholdValues = append(thresholdValues, thresholdsWithLane("tx_bias_milliamps", bias, data, lane.Index)...)
		}
		if data := thresholdBlock(data, sff8636TXPowerThresholdOffset); data != nil && lane.TXPowerMilliWatts.Valid {
			thresholdValues = append(thresholdValues, thresholdsWithLane("tx_power_milliwatts", power, data, lane.Index)...)
		}
	}
	return thresholdValues
}

func laneMonitorFlagAlarms(name string, data []byte, offset12, offset34, lane int) []transceiver.Alarm {
	offset := offset12
	if lane > 2 {
		offset = offset34
	}

	flags := byteAt(data, offset)
	shift := 4
	if lane%2 == 0 {
		shift = 0
	}

	return []transceiver.Alarm{
		flagAlarm(name+"_high", "alarm", flags, byte(0x08<<shift), lane),
		flagAlarm(name+"_low", "alarm", flags, byte(0x04<<shift), lane),
		flagAlarm(name+"_high", "warning", flags, byte(0x02<<shift), lane),
		flagAlarm(name+"_low", "warning", flags, byte(0x01<<shift), lane),
	}
}

func thresholdBlock(data []byte, offset int) []byte {
	if len(data) < offset+8 {
		return nil
	}
	block := data[offset : offset+8]
	if allZero(block) {
		return nil
	}
	return block
}

func byteAt(data []byte, offset int) byte {
	if len(data) <= offset {
		return 0
	}
	return data[offset]
}

func appendSFF8636Length(lengths []transceiver.Length, medium transceiver.LengthMedium, meters int) []transceiver.Length {
	if meters == 0 {
		return lengths
	}
	return append(lengths, transceiver.Length{Medium: medium, Meters: meters})
}

func sff8636PowerClass(extID byte) (int, float64) {
	switch extID & 0x03 {
	case 1:
		return 5, 4.0
	case 2:
		return 6, 4.5
	case 3:
		return 7, 5.0
	}

	switch (extID >> 6) & 0x03 {
	case 0:
		return 1, 1.5
	case 1:
		return 2, 2.0
	case 2:
		return 3, 2.5
	case 3:
		return 4, 3.5
	default:
		return 0, 0
	}
}
