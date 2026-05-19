package sff

import (
	"encoding/binary"
	"fmt"

	"github.com/vexxhost/transceiver_exporter/pkg/transceiver"
)

type cmisDecoder struct{}

const (
	cmisPageSize          = 128
	cmisPage00UpperOffset = 128
	cmisPage01Offset      = 256
	cmisPage02Offset      = 384
	cmisPage11Bank0Offset = 512

	cmisRevisionComplianceOffset = 0x01
	cmisModuleStateOffset        = 0x03
	cmisModuleStateMask          = 0x0e
	cmisModuleStateShift         = 1
	cmisModuleControlOffset      = 0x1a
	cmisLowPowerAllowRequestHW   = 0x40
	cmisLowPowerRequestSW        = 0x10
	cmisModuleFaultOffset        = 0x29
	cmisMediaTypeOffset          = 0x55

	cmisAppDescriptorStart    = 0x56
	cmisAppDescriptorSize     = 4
	cmisAppDescriptorCount    = 8
	cmisAppDescriptorEnd      = 0xff
	cmisAppMediaLaneCountMask = 0x0f

	cmisVendorNameStart = 0x81
	cmisVendorNameEnd   = 0x91
	cmisVendorOUIStart  = 0x91
	cmisVendorOUIEnd    = 0x94
	cmisVendorPNStart   = 0x94
	cmisVendorPNEnd     = 0xa4
	cmisVendorRevStart  = 0xa4
	cmisVendorRevEnd    = 0xa6
	cmisVendorSNStart   = 0xa6
	cmisVendorSNEnd     = 0xb6
	cmisDateStart       = 0xb6
	cmisDateEnd         = 0xbe

	cmisPowerClassOffset       = 0xc8 - cmisPageSize
	cmisMaxPowerOffset         = 0xc9 - cmisPageSize
	cmisCableAssemblyLenOffset = 0xca - cmisPageSize
	cmisConnectorOffset        = 0xcb - cmisPageSize
	cmisMediaTechOffset        = 0xd4 - cmisPageSize

	cmisSMFLengthOffset           = 0x84 - cmisPageSize
	cmisOM5LengthOffset           = 0x85 - cmisPageSize
	cmisOM4LengthOffset           = 0x86 - cmisPageSize
	cmisOM3LengthOffset           = 0x87 - cmisPageSize
	cmisOM2LengthOffset           = 0x88 - cmisPageSize
	cmisWavelengthOffset          = 0x8a - cmisPageSize
	cmisWavelengthToleranceOffset = 0x8c - cmisPageSize
	cmisDiagTypeOffset            = 0x97 - cmisPageSize
	cmisRxPowerAvgMask            = 0x10

	cmisDiagFlagsTXOffset       = 0x9d - cmisPageSize
	cmisDiagFlagTXAdaptiveEQ    = 0x08
	cmisDiagFlagTXLOL           = 0x04
	cmisDiagFlagTXLOS           = 0x02
	cmisDiagFlagTXFault         = 0x01
	cmisDiagFlagsRXOffset       = 0x9e - cmisPageSize
	cmisDiagFlagRXLOL           = 0x04
	cmisDiagFlagRXLOS           = 0x02
	cmisDiagAdvertOffset        = 0xa0 - cmisPageSize
	cmisTxBiasSupported         = 0x01
	cmisTxPowerSupported        = 0x02
	cmisRxPowerSupported        = 0x04
	cmisTxBiasScaleMask         = 0x18
	cmisTxBiasScale2            = 0x08
	cmisTxBiasScale4            = 0x10
	cmisSignalIntegrityTXOffset = 0xa1 - cmisPageSize
	cmisSignalIntegrityRXOffset = 0xa2 - cmisPageSize
	cmisCDRSupported            = 0x01
	cmisCDRBypassSupported      = 0x02

	cmisTempFlagsOffset = 0x09
	cmisTempHighAlarm   = 0x01
	cmisTempLowAlarm    = 0x02
	cmisTempHighWarning = 0x04
	cmisTempLowWarning  = 0x08
	cmisVCCHighAlarm    = 0x10
	cmisVCCLowAlarm     = 0x20
	cmisVCCHighWarning  = 0x40
	cmisVCCLowWarning   = 0x80

	cmisTempThresholdOffset    = 0x80 - cmisPageSize
	cmisVCCThresholdOffset     = 0x88 - cmisPageSize
	cmisTxPowerThresholdOffset = 0xb0 - cmisPageSize
	cmisTxBiasThresholdOffset  = 0xb8 - cmisPageSize
	cmisRxPowerThresholdOffset = 0xc0 - cmisPageSize

	cmisDataPathStateOffset      = 0x80 - cmisPageSize
	cmisTXFaultOffset            = 0x87 - cmisPageSize
	cmisTXLOSOffset              = 0x88 - cmisPageSize
	cmisTXLOLOffset              = 0x89 - cmisPageSize
	cmisTXEQFaultOffset          = 0x8a - cmisPageSize
	cmisTXPowerHighAlarmOffset   = 0x8b - cmisPageSize
	cmisTXPowerLowAlarmOffset    = 0x8c - cmisPageSize
	cmisTXPowerHighWarningOffset = 0x8d - cmisPageSize
	cmisTXPowerLowWarningOffset  = 0x8e - cmisPageSize
	cmisTXBiasHighAlarmOffset    = 0x8f - cmisPageSize
	cmisTXBiasLowAlarmOffset     = 0x90 - cmisPageSize
	cmisTXBiasHighWarningOffset  = 0x91 - cmisPageSize
	cmisTXBiasLowWarningOffset   = 0x92 - cmisPageSize
	cmisRXLOSOffset              = 0x93 - cmisPageSize
	cmisRXLOLOffset              = 0x94 - cmisPageSize
	cmisRXPowerHighAlarmOffset   = 0x95 - cmisPageSize
	cmisRXPowerLowAlarmOffset    = 0x96 - cmisPageSize
	cmisRXPowerHighWarningOffset = 0x97 - cmisPageSize
	cmisRXPowerLowWarningOffset  = 0x98 - cmisPageSize

	cmisTxPowerOffset = 154 - cmisPageSize
	cmisTxBiasOffset  = 170 - cmisPageSize
	cmisRxPowerOffset = 186 - cmisPageSize
)

func (d cmisDecoder) decode(data []byte) (transceiver.Module, error) {
	if len(data) < 18 {
		return transceiver.Module{}, fmt.Errorf("cmis eeprom too short: %d bytes", len(data))
	}

	module := transceiver.Module{
		MemoryMap:          transceiver.MemoryMapCMIS,
		FormFactor:         transceiver.FormFactor(data[0]),
		RevisionCompliance: cmisRevisionCompliance(data[cmisRevisionComplianceOffset]),
		Status:             d.status(data),
		Diagnostics: transceiver.Diagnostics{
			Supported: true,
		},
	}

	if page00, ok := cmisPage(data, cmisPage00UpperOffset); ok && !allZero(page00) {
		module.Connector = transceiver.Connector(page00[cmisConnectorOffset])
		module.Power = d.power(page00)
		module.Media = d.media(data)
		module.Lengths = d.lengths(data)
		module.Vendor = transceiver.Vendor{
			Name:         cleanString(data[cmisVendorNameStart:cmisVendorNameEnd]),
			OUI:          oui(data[cmisVendorOUIStart:cmisVendorOUIEnd]),
			PartNumber:   cleanString(data[cmisVendorPNStart:cmisVendorPNEnd]),
			Revision:     cleanString(data[cmisVendorRevStart:cmisVendorRevEnd]),
			SerialNumber: cleanString(data[cmisVendorSNStart:cmisVendorSNEnd]),
		}
		if len(data) >= cmisDateEnd {
			module.Vendor.DateCode = cleanString(data[cmisDateStart:cmisDateEnd])
		}
	}

	module.Diagnostics.TemperatureCelsius = transceiver.NewReading(temp(data[14:16]))
	module.Diagnostics.VoltageVolts = transceiver.NewReading(voltage(data[16:18]))
	module.Diagnostics.Lanes = d.lanes(data)
	module.Diagnostics.Alarms = d.alarms(data, module.Diagnostics.Lanes)
	module.Diagnostics.Thresholds = d.thresholds(data, module.Diagnostics.Lanes)

	return module, nil
}

func (d cmisDecoder) lanes(data []byte) []transceiver.Lane {
	page01, ok := cmisPage(data, cmisPage01Offset)
	if !ok {
		return nil
	}

	advert := page01[cmisDiagAdvertOffset]
	laneLimit := cmisAdvertisedMediaLaneCount(data)
	var lanes []transceiver.Lane
	for bank := 0; ; bank++ {
		page11, ok := cmisPage(data, cmisPage11Bank0Offset+bank*cmisPageSize)
		if !ok || allZero(page11) {
			break
		}

		for i := 0; i < 8; i++ {
			index := bank*8 + i + 1
			if laneLimit > 0 && index > laneLimit {
				continue
			}
			if laneLimit == 0 && !d.laneHasMonitorData(page11, i, advert) {
				continue
			}

			lane := transceiver.Lane{
				Index:         index,
				DataPathState: cmisDataPathState(page11, i),
			}
			hasReading := false

			if advert&cmisTxBiasSupported != 0 {
				lane.TXBiasMilliAmps = transceiver.NewReading(d.bias(page11[cmisTxBiasOffset+i*2:cmisTxBiasOffset+i*2+2], advert))
				hasReading = true
			}
			if advert&cmisTxPowerSupported != 0 {
				lane.TXPowerMilliWatts = transceiver.NewReading(power(page11[cmisTxPowerOffset+i*2 : cmisTxPowerOffset+i*2+2]))
				hasReading = true
			}
			if advert&cmisRxPowerSupported != 0 {
				lane.RXPowerMilliWatts = transceiver.NewReading(power(page11[cmisRxPowerOffset+i*2 : cmisRxPowerOffset+i*2+2]))
				hasReading = true
			}
			if hasReading {
				lanes = append(lanes, lane)
			}
		}
	}

	return lanes
}

func (d cmisDecoder) alarms(data []byte, lanes []transceiver.Lane) []transceiver.Alarm {
	alarms := d.moduleFlagAlarms(data)

	if page02, ok := cmisPage(data, cmisPage02Offset); ok && !allZero(page02) {
		alarms = append(alarms, thresholdAlarms(transceiver.NewReading(temp(data[14:16])), "temperature", temp, page02[cmisTempThresholdOffset:cmisTempThresholdOffset+8])...)
		alarms = append(alarms, thresholdAlarms(transceiver.NewReading(voltage(data[16:18])), "voltage", voltage, page02[cmisVCCThresholdOffset:cmisVCCThresholdOffset+8])...)

		page01, page01OK := cmisPage(data, cmisPage01Offset)
		if page01OK {
			advert := page01[cmisDiagAdvertOffset]
			biasScale := func(raw []byte) float64 {
				return d.bias(raw, advert)
			}
			for _, lane := range lanes {
				alarms = append(alarms, thresholdAlarmsWithLane(lane.TXBiasMilliAmps, "tx_bias", biasScale, page02[cmisTxBiasThresholdOffset:cmisTxBiasThresholdOffset+8], lane.Index)...)
				alarms = append(alarms, thresholdAlarmsWithLane(lane.TXPowerMilliWatts, "tx_power", power, page02[cmisTxPowerThresholdOffset:cmisTxPowerThresholdOffset+8], lane.Index)...)
				alarms = append(alarms, thresholdAlarmsWithLane(lane.RXPowerMilliWatts, "rx_power", power, page02[cmisRxPowerThresholdOffset:cmisRxPowerThresholdOffset+8], lane.Index)...)
			}
		}
	}

	alarms = append(alarms, d.laneFlagAlarms(data, lanes)...)
	return mergeAlarms(alarms)
}

func (d cmisDecoder) thresholds(data []byte, lanes []transceiver.Lane) []transceiver.Threshold {
	page02, ok := cmisPage(data, cmisPage02Offset)
	if !ok || allZero(page02) {
		return nil
	}

	thresholdValues := thresholds("temperature_celsius", temp, page02[cmisTempThresholdOffset:cmisTempThresholdOffset+8])
	thresholdValues = append(thresholdValues, thresholds("voltage_volts", voltage, page02[cmisVCCThresholdOffset:cmisVCCThresholdOffset+8])...)

	page01, page01OK := cmisPage(data, cmisPage01Offset)
	if !page01OK {
		return thresholdValues
	}

	advert := page01[cmisDiagAdvertOffset]
	biasScale := func(raw []byte) float64 {
		return d.bias(raw, advert)
	}
	for _, lane := range lanes {
		if lane.TXBiasMilliAmps.Valid {
			thresholdValues = append(
				thresholdValues,
				thresholdsWithLane("tx_bias_milliamps", biasScale, page02[cmisTxBiasThresholdOffset:cmisTxBiasThresholdOffset+8], lane.Index)...,
			)
		}
		if lane.TXPowerMilliWatts.Valid {
			thresholdValues = append(
				thresholdValues,
				thresholdsWithLane("tx_power_milliwatts", power, page02[cmisTxPowerThresholdOffset:cmisTxPowerThresholdOffset+8], lane.Index)...,
			)
		}
		if lane.RXPowerMilliWatts.Valid {
			thresholdValues = append(
				thresholdValues,
				thresholdsWithLane("rx_power_milliwatts", power, page02[cmisRxPowerThresholdOffset:cmisRxPowerThresholdOffset+8], lane.Index)...,
			)
		}
	}

	return thresholdValues
}

func (d cmisDecoder) moduleFlagAlarms(data []byte) []transceiver.Alarm {
	if len(data) <= cmisTempFlagsOffset {
		return nil
	}

	flags := data[cmisTempFlagsOffset]
	return []transceiver.Alarm{
		flagAlarm("temperature_high", "alarm", flags, cmisTempHighAlarm, 0),
		flagAlarm("temperature_low", "alarm", flags, cmisTempLowAlarm, 0),
		flagAlarm("temperature_high", "warning", flags, cmisTempHighWarning, 0),
		flagAlarm("temperature_low", "warning", flags, cmisTempLowWarning, 0),
		flagAlarm("voltage_high", "alarm", flags, cmisVCCHighAlarm, 0),
		flagAlarm("voltage_low", "alarm", flags, cmisVCCLowAlarm, 0),
		flagAlarm("voltage_high", "warning", flags, cmisVCCHighWarning, 0),
		flagAlarm("voltage_low", "warning", flags, cmisVCCLowWarning, 0),
	}
}

func (d cmisDecoder) laneFlagAlarms(data []byte, lanes []transceiver.Lane) []transceiver.Alarm {
	page01, ok := cmisPage(data, cmisPage01Offset)
	if !ok {
		return nil
	}

	laneSet := map[int]struct{}{}
	for _, lane := range lanes {
		laneSet[lane.Index] = struct{}{}
	}

	defs := []cmisLaneFlagDef{
		{name: "tx_bias_high", severity: "alarm", offset: cmisTXBiasHighAlarmOffset, advertOffset: cmisDiagAdvertOffset, advertMask: cmisTxBiasSupported},
		{name: "tx_bias_low", severity: "alarm", offset: cmisTXBiasLowAlarmOffset, advertOffset: cmisDiagAdvertOffset, advertMask: cmisTxBiasSupported},
		{name: "tx_bias_high", severity: "warning", offset: cmisTXBiasHighWarningOffset, advertOffset: cmisDiagAdvertOffset, advertMask: cmisTxBiasSupported},
		{name: "tx_bias_low", severity: "warning", offset: cmisTXBiasLowWarningOffset, advertOffset: cmisDiagAdvertOffset, advertMask: cmisTxBiasSupported},
		{name: "tx_power_high", severity: "alarm", offset: cmisTXPowerHighAlarmOffset, advertOffset: cmisDiagAdvertOffset, advertMask: cmisTxPowerSupported},
		{name: "tx_power_low", severity: "alarm", offset: cmisTXPowerLowAlarmOffset, advertOffset: cmisDiagAdvertOffset, advertMask: cmisTxPowerSupported},
		{name: "tx_power_high", severity: "warning", offset: cmisTXPowerHighWarningOffset, advertOffset: cmisDiagAdvertOffset, advertMask: cmisTxPowerSupported},
		{name: "tx_power_low", severity: "warning", offset: cmisTXPowerLowWarningOffset, advertOffset: cmisDiagAdvertOffset, advertMask: cmisTxPowerSupported},
		{name: "rx_power_high", severity: "alarm", offset: cmisRXPowerHighAlarmOffset, advertOffset: cmisDiagAdvertOffset, advertMask: cmisRxPowerSupported},
		{name: "rx_power_low", severity: "alarm", offset: cmisRXPowerLowAlarmOffset, advertOffset: cmisDiagAdvertOffset, advertMask: cmisRxPowerSupported},
		{name: "rx_power_high", severity: "warning", offset: cmisRXPowerHighWarningOffset, advertOffset: cmisDiagAdvertOffset, advertMask: cmisRxPowerSupported},
		{name: "rx_power_low", severity: "warning", offset: cmisRXPowerLowWarningOffset, advertOffset: cmisDiagAdvertOffset, advertMask: cmisRxPowerSupported},
		{name: "tx_fault", severity: "fault", offset: cmisTXFaultOffset, advertOffset: cmisDiagFlagsTXOffset, advertMask: cmisDiagFlagTXFault},
		{name: "tx_los", severity: "fault", offset: cmisTXLOSOffset, advertOffset: cmisDiagFlagsTXOffset, advertMask: cmisDiagFlagTXLOS},
		{name: "tx_lol", severity: "fault", offset: cmisTXLOLOffset, advertOffset: cmisDiagFlagsTXOffset, advertMask: cmisDiagFlagTXLOL},
		{name: "tx_adaptive_eq_fault", severity: "fault", offset: cmisTXEQFaultOffset, advertOffset: cmisDiagFlagsTXOffset, advertMask: cmisDiagFlagTXAdaptiveEQ},
		{name: "rx_los", severity: "fault", offset: cmisRXLOSOffset, advertOffset: cmisDiagFlagsRXOffset, advertMask: cmisDiagFlagRXLOS},
		{name: "rx_lol", severity: "fault", offset: cmisRXLOLOffset, advertOffset: cmisDiagFlagsRXOffset, advertMask: cmisDiagFlagRXLOL},
	}

	var alarms []transceiver.Alarm
	for bank := 0; ; bank++ {
		page11, ok := cmisPage(data, cmisPage11Bank0Offset+bank*cmisPageSize)
		if !ok || allZero(page11) {
			break
		}
		for i := 0; i < 8; i++ {
			lane := bank*8 + i + 1
			if _, ok := laneSet[lane]; !ok {
				continue
			}
			for _, def := range defs {
				if page01[def.advertOffset]&def.advertMask == 0 {
					continue
				}
				alarms = append(alarms, transceiver.Alarm{
					Name:     def.name,
					Severity: def.severity,
					Active:   page11[def.offset]&(1<<uint(i)) != 0,
					Lane:     lane,
				})
			}
		}
	}

	return alarms
}

func (d cmisDecoder) bias(data []byte, advert byte) float64 {
	scale := 1.0
	switch advert & cmisTxBiasScaleMask {
	case cmisTxBiasScale2:
		scale = 2
	case cmisTxBiasScale4:
		scale = 4
	}
	return bias(data) * scale
}

func (d cmisDecoder) laneHasMonitorData(page11 []byte, lane int, advert byte) bool {
	for _, field := range []struct {
		offset int
		mask   byte
	}{
		{cmisTxBiasOffset, cmisTxBiasSupported},
		{cmisTxPowerOffset, cmisTxPowerSupported},
		{cmisRxPowerOffset, cmisRxPowerSupported},
	} {
		if advert&field.mask == 0 {
			continue
		}
		raw := page11[field.offset+lane*2 : field.offset+lane*2+2]
		if !allZero(raw) && !allEqual(raw, 0xff) {
			return true
		}
	}
	return false
}

func (d cmisDecoder) status(data []byte) *transceiver.ModuleStatus {
	status := &transceiver.ModuleStatus{
		State: cmisModuleState(data[cmisModuleStateOffset]),
	}
	if len(data) > cmisModuleControlOffset {
		status.LowPowerAllowRequestHardware = data[cmisModuleControlOffset]&cmisLowPowerAllowRequestHW != 0
		status.LowPowerRequestSoftware = data[cmisModuleControlOffset]&cmisLowPowerRequestSW != 0
	}
	if len(data) > cmisModuleFaultOffset {
		if fault := cmisModuleFault(data[cmisModuleFaultOffset]); fault != "" && (status.State == "fault" || fault != "none") {
			status.Fault = fault
		}
	}
	return status
}

func (d cmisDecoder) power(page00 []byte) *transceiver.ModulePower {
	return &transceiver.ModulePower{
		Class:    int((page00[cmisPowerClassOffset]>>5)&0x07) + 1,
		MaxWatts: float64(page00[cmisMaxPowerOffset]) * 0.25,
	}
}

func (d cmisDecoder) media(data []byte) *transceiver.Media {
	page00, page00OK := cmisPage(data, cmisPage00UpperOffset)
	page01, page01OK := cmisPage(data, cmisPage01Offset)
	if !page00OK && !page01OK {
		return nil
	}

	media := &transceiver.Media{
		Type: cmisMediaType(data[cmisMediaTypeOffset]),
	}
	if page00OK {
		media.InterfaceTechnology = cmisMediaTechnology(page00[cmisMediaTechOffset])
	}
	if page01OK {
		media.TXCDR = page01[cmisSignalIntegrityTXOffset]&cmisCDRSupported != 0
		media.RXCDR = page01[cmisSignalIntegrityRXOffset]&cmisCDRSupported != 0
		media.TXCDRBypassControlSupported = page01[cmisSignalIntegrityTXOffset]&cmisCDRBypassSupported != 0
		media.RXCDRBypassControlSupported = page01[cmisSignalIntegrityRXOffset]&cmisCDRBypassSupported != 0
		if !allZero(page01[cmisWavelengthOffset : cmisWavelengthOffset+2]) {
			media.WavelengthNanometers = float64(binary.BigEndian.Uint16(page01[cmisWavelengthOffset:cmisWavelengthOffset+2])) * 0.05
		}
		if !allZero(page01[cmisWavelengthToleranceOffset : cmisWavelengthToleranceOffset+2]) {
			media.WavelengthToleranceNanometers = float64(binary.BigEndian.Uint16(page01[cmisWavelengthToleranceOffset:cmisWavelengthToleranceOffset+2])) * 0.005
		}
	}
	return media
}

func (d cmisDecoder) lengths(data []byte) []transceiver.Length {
	var lengths []transceiver.Length
	if page00, ok := cmisPage(data, cmisPage00UpperOffset); ok {
		lengths = appendCMISLength(lengths, transceiver.LengthMediumCableAssembly, cmisCableAssemblyLengthMeters(page00[cmisCableAssemblyLenOffset]))
	}
	if page01, ok := cmisPage(data, cmisPage01Offset); ok {
		lengths = appendCMISLength(lengths, transceiver.LengthMediumSingleMode, cmisSMFLengthMeters(page01[cmisSMFLengthOffset]))
		lengths = appendCMISLength(lengths, transceiver.LengthMediumOM5, int(page01[cmisOM5LengthOffset])*2)
		lengths = appendCMISLength(lengths, transceiver.LengthMediumOM4, int(page01[cmisOM4LengthOffset])*2)
		lengths = appendCMISLength(lengths, transceiver.LengthMediumOM3, int(page01[cmisOM3LengthOffset])*2)
		lengths = appendCMISLength(lengths, transceiver.LengthMediumOM2, int(page01[cmisOM2LengthOffset]))
	}
	return lengths
}

func cmisPage(data []byte, offset int) ([]byte, bool) {
	if len(data) < offset+cmisPageSize {
		return nil, false
	}
	return data[offset : offset+cmisPageSize], true
}

type cmisLaneFlagDef struct {
	name         string
	severity     string
	offset       int
	advertOffset int
	advertMask   byte
}

func cmisRevisionCompliance(value byte) string {
	return fmt.Sprintf("%d.%d", value>>4, value&0x0f)
}

func cmisModuleState(value byte) string {
	switch (value & cmisModuleStateMask) >> cmisModuleStateShift {
	case 1:
		return "low_power"
	case 2:
		return "powering_up"
	case 3:
		return "ready"
	case 4:
		return "powering_down"
	case 5:
		return "fault"
	default:
		return "unknown"
	}
}

func cmisModuleFault(value byte) string {
	switch value {
	case 0:
		return "none"
	case 1:
		return "tec_runaway"
	case 2:
		return "data_memory_corrupted"
	case 3:
		return "program_memory_corrupted"
	default:
		return "unknown"
	}
}

func cmisDataPathState(page11 []byte, lane int) string {
	value := page11[cmisDataPathStateOffset+lane/2]
	if lane%2 == 0 {
		value &= 0x0f
	} else {
		value >>= 4
	}

	switch value {
	case 1:
		return "deactivated"
	case 2:
		return "initializing"
	case 3:
		return "deinitializing"
	case 4:
		return "activated"
	case 5:
		return "tx_turning_on"
	case 6:
		return "tx_turning_off"
	case 7:
		return "initialized"
	default:
		return ""
	}
}

func cmisAdvertisedMediaLaneCount(data []byte) int {
	maxLanes := 0
	for i := 0; i < cmisAppDescriptorCount; i++ {
		offset := cmisAppDescriptorStart + i*cmisAppDescriptorSize
		if len(data) <= offset+2 {
			return maxLanes
		}
		if data[offset] == cmisAppDescriptorEnd {
			break
		}
		lanes := int(data[offset+2] & cmisAppMediaLaneCountMask)
		if lanes > maxLanes && lanes <= 8 {
			maxLanes = lanes
		}
	}
	return maxLanes
}

func cmisMediaType(value byte) string {
	switch value {
	case 0x01:
		return "mmf"
	case 0x02:
		return "smf"
	case 0x03:
		return "passive_copper"
	case 0x04:
		return "active_cable"
	case 0x05:
		return "base_t"
	default:
		return "unknown"
	}
}

func cmisMediaTechnology(value byte) string {
	switch value {
	case 0x00:
		return "850_nm_vcsel"
	case 0x01:
		return "1310_nm_vcsel"
	case 0x02:
		return "1550_nm_vcsel"
	case 0x03:
		return "1310_nm_fp"
	case 0x04:
		return "1310_nm_dfb"
	case 0x05:
		return "1550_nm_dfb"
	case 0x06:
		return "1310_nm_eml"
	case 0x07:
		return "1550_nm_eml"
	case 0x08:
		return "other"
	case 0x09:
		return "1490_nm_dfb"
	case 0x0a:
		return "copper_unequalized"
	case 0x0b:
		return "copper_passive_equalized"
	case 0x0c:
		return "copper_near_and_far_end_limiting_active_equalizers"
	case 0x0d:
		return "copper_far_end_limiting_active_equalizers"
	case 0x0e:
		return "copper_near_end_limiting_active_equalizers"
	case 0x0f:
		return "copper_linear_active_equalizers"
	default:
		return "unknown"
	}
}

func appendCMISLength(lengths []transceiver.Length, medium transceiver.LengthMedium, meters int) []transceiver.Length {
	if meters <= 0 {
		return lengths
	}
	return append(lengths, transceiver.Length{Medium: medium, Meters: meters})
}

func cmisCableAssemblyLengthMeters(value byte) int {
	if value == 0xff {
		return 6300
	}
	base := int(value & 0x3f)
	switch value & 0xc0 {
	case 0x00:
		return (base + 5) / 10
	case 0x40:
		return base
	case 0x80:
		return base * 10
	case 0xc0:
		return base * 100
	default:
		return 0
	}
}

func cmisSMFLengthMeters(value byte) int {
	base := int(value & 0x3f)
	switch value & 0xc0 {
	case 0x00:
		return base * 100
	case 0x40:
		return base * 1000
	default:
		return 0
	}
}

func flagAlarm(name, severity string, flags, mask byte, lane int) transceiver.Alarm {
	return transceiver.Alarm{
		Name:     name,
		Severity: severity,
		Active:   flags&mask != 0,
		Lane:     lane,
	}
}

type alarmKey struct {
	name     string
	severity string
	lane     int
}

func mergeAlarms(alarms []transceiver.Alarm) []transceiver.Alarm {
	merged := make([]transceiver.Alarm, 0, len(alarms))
	seen := map[alarmKey]int{}
	for _, alarm := range alarms {
		key := alarmKey{name: alarm.Name, severity: alarm.Severity, lane: alarm.Lane}
		if idx, ok := seen[key]; ok {
			merged[idx].Active = merged[idx].Active || alarm.Active
			continue
		}
		seen[key] = len(merged)
		merged = append(merged, alarm)
	}
	return merged
}
