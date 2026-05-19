package sff

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/vexxhost/transceiver_exporter/pkg/transceiver"
)

func TestDecodeRejectsUnsupportedAndShortData(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		wantError string
	}{
		{name: "empty", data: nil, wantError: ErrUnsupportedFormat.Error()},
		{name: "unknown identifier", data: []byte{0xff}, wantError: ErrUnsupportedFormat.Error()},
		{name: "short sff8472", data: []byte{0x03}, wantError: "sff8472 eeprom too short"},
		{name: "short sff8636", data: []byte{0x0d}, wantError: "sff8636 eeprom too short"},
		{name: "short cmis", data: []byte{0x18}, wantError: "cmis eeprom too short"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decode(tt.data)
			if err == nil {
				t.Fatal("decode succeeded, want error")
			}
			if errors.Is(err, ErrUnsupportedFormat) {
				if tt.wantError != ErrUnsupportedFormat.Error() {
					t.Fatalf("error = %v, want %q", err, tt.wantError)
				}
				return
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want substring %q", err, tt.wantError)
			}
		})
	}
}

func TestDecodeFixturesMatchesEthtoolOutput(t *testing.T) {
	for _, dir := range fixtureDirs(t) {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			ethtoolOutput, err := os.ReadFile(filepath.Join(dir, "ethtool.txt"))
			if err != nil {
				t.Fatalf("read ethtool fixture: %v", err)
			}
			rawDump, rawDumpOK, err := parseEthtoolHexDump(string(ethtoolOutput))
			if err != nil {
				t.Fatalf("parse ethtool hex dump: %v", err)
			}
			data := readEEPROMFixture(t, dir, rawDump, rawDumpOK)

			module, err := Decode(data)
			if err != nil {
				t.Fatalf("decode fixture: %v", err)
			}

			if rawDumpOK {
				if !bytes.Equal(data, rawDump) {
					t.Fatalf("EEPROM fixture bytes do not match ethtool hex dump")
				}
				assertRawDumpModule(t, module, data)
				return
			}

			if module.MemoryMap != transceiver.MemoryMapSFF8472 {
				t.Fatalf("memory map = %q, want %q", module.MemoryMap, transceiver.MemoryMapSFF8472)
			}

			want := parseEthtoolOutput(t, string(ethtoolOutput))
			got := moduleEthtoolOutput(module)
			if !maps.EqualFunc(got, want, slices.Equal) {
				t.Fatalf("decoded fixture does not match ethtool output\nwant:\n%s\ngot:\n%s", formatEthtoolValues(want), formatEthtoolValues(got))
			}
			assertDiagnosticsSupport(t, module, want)
		})
	}
}

func TestCMISFixturesAdvertiseFourMediaLanes(t *testing.T) {
	for _, dir := range fixtureDirs(t) {
		name := filepath.Base(dir)
		if !strings.HasPrefix(name, "cmis-") {
			continue
		}

		t.Run(name, func(t *testing.T) {
			data := readRawDumpFixture(t, dir)
			if got, want := cmisAdvertisedMediaLaneCount(data), 4; got != want {
				t.Fatalf("advertised CMIS media lanes = %d, want %d", got, want)
			}
		})
	}
}

func TestCMISLaneMonitorsUseAdvertisedMediaLanes(t *testing.T) {
	data := make([]byte, cmisPage11Bank0Offset+cmisPageSize)
	copy(data, readRawDumpFixture(t, filepath.Join("testdata", "cmis-qsfp-plus-lower-page-a")))

	page01 := data[cmisPage01Offset : cmisPage01Offset+cmisPageSize]
	page01[cmisDiagAdvertOffset] = cmisTxBiasSupported | cmisTxPowerSupported | cmisRxPowerSupported

	page11 := data[cmisPage11Bank0Offset : cmisPage11Bank0Offset+cmisPageSize]
	page11[cmisDataPathStateOffset] = 0x44
	for lane := 0; lane < 4; lane++ {
		putU16(page11[cmisTxBiasOffset+lane*2:cmisTxBiasOffset+lane*2+2], uint16(4500+lane))
		putU16(page11[cmisTxPowerOffset+lane*2:cmisTxPowerOffset+lane*2+2], uint16(12000+lane))
		putU16(page11[cmisRxPowerOffset+lane*2:cmisRxPowerOffset+lane*2+2], uint16(11000+lane))
	}

	module, err := Decode(data)
	if err != nil {
		t.Fatalf("decode fixture: %v", err)
	}

	lanes := module.Diagnostics.Lanes
	if got, want := len(lanes), 4; got != want {
		t.Fatalf("CMIS lanes = %d, want %d", got, want)
	}
	for i, lane := range lanes {
		if got, want := lane.Index, i+1; got != want {
			t.Fatalf("lane index %d = %d, want %d", i, got, want)
		}
	}
}

func TestSFF8636AlarmsAndThresholds(t *testing.T) {
	data := make([]byte, sff8636TXPowerThresholdOffset+8)
	data[0] = byte(transceiver.FormFactorQSFPPlus)
	data[sff8636TempFlagsOffset] = 0x20
	data[sff8636LOSFlagsOffset] = 0x02
	data[sff8636FaultFlagsOffset] = 0x08
	putU16(data[22:24], uint16(30*256))
	putU16(data[26:28], uint16(3.3*10000))
	for lane := 0; lane < 4; lane++ {
		putU16(data[sff8636RXPowerOffset+lane*2:sff8636RXPowerOffset+lane*2+2], uint16(12000+lane))
		putU16(data[sff8636TXBiasOffset+lane*2:sff8636TXBiasOffset+lane*2+2], uint16(4500+lane))
		putU16(data[sff8636TXPowerOffset+lane*2:sff8636TXPowerOffset+lane*2+2], uint16(11000+lane))
	}
	putU16(data[sff8636TemperatureThresholdOffset:sff8636TemperatureThresholdOffset+2], uint16(70*256))
	putU16(data[sff8636TemperatureThresholdOffset+4:sff8636TemperatureThresholdOffset+6], uint16(60*256))
	putU16(data[sff8636VoltageThresholdOffset:sff8636VoltageThresholdOffset+2], uint16(37000))
	putU16(data[sff8636VoltageThresholdOffset+2:sff8636VoltageThresholdOffset+4], uint16(30000))
	putU16(data[sff8636VoltageThresholdOffset+4:sff8636VoltageThresholdOffset+6], uint16(36000))
	putU16(data[sff8636VoltageThresholdOffset+6:sff8636VoltageThresholdOffset+8], uint16(31000))
	putU16(data[sff8636RXPowerThresholdOffset:sff8636RXPowerThresholdOffset+2], uint16(20000))
	putU16(data[sff8636RXPowerThresholdOffset+2:sff8636RXPowerThresholdOffset+4], uint16(1000))
	putU16(data[sff8636RXPowerThresholdOffset+4:sff8636RXPowerThresholdOffset+6], uint16(18000))
	putU16(data[sff8636RXPowerThresholdOffset+6:sff8636RXPowerThresholdOffset+8], uint16(2000))

	module, err := Decode(data)
	if err != nil {
		t.Fatalf("decode sff8636: %v", err)
	}

	for _, want := range []transceiver.Alarm{
		{Name: "temperature_high", Severity: "warning", Active: true},
		{Name: "rx_los", Severity: "fault", Active: true, Lane: 2},
		{Name: "tx_fault", Severity: "fault", Active: true, Lane: 4},
	} {
		if !hasAlarm(module.Diagnostics.Alarms, want) {
			t.Fatalf("missing alarm %+v in %+v", want, module.Diagnostics.Alarms)
		}
	}
	if !hasThreshold(module.Diagnostics.Thresholds, transceiver.Threshold{
		Metric:   "temperature_celsius",
		Boundary: "high",
		Severity: "warning",
		Value:    60,
	}) {
		t.Fatalf("missing temperature threshold in %+v", module.Diagnostics.Thresholds)
	}
	if !hasThreshold(module.Diagnostics.Thresholds, transceiver.Threshold{
		Metric:   "rx_power_milliwatts",
		Boundary: "low",
		Severity: "warning",
		Value:    0.2,
		Lane:     1,
	}) {
		t.Fatalf("missing rx power threshold in %+v", module.Diagnostics.Thresholds)
	}
}

func TestDecodeObservationAttachesInterfaceName(t *testing.T) {
	data := readRawDumpFixture(t, filepath.Join("testdata", "cmis-qsfp-plus-lower-page-a"))

	observation, err := DecodeObservation("eth0", data)
	if err != nil {
		t.Fatalf("decode observation: %v", err)
	}
	if got, want := observation.Interface, "eth0"; got != want {
		t.Fatalf("interface = %q, want %q", got, want)
	}
	if observation.Module.MemoryMap != transceiver.MemoryMapCMIS {
		t.Fatalf("memory map = %q, want %q", observation.Module.MemoryMap, transceiver.MemoryMapCMIS)
	}
}

func TestMergeAlarmsCombinesDuplicateKeys(t *testing.T) {
	alarms := mergeAlarms([]transceiver.Alarm{
		{Name: "temperature_high", Severity: "warning"},
		{Name: "temperature_high", Severity: "warning", Active: true},
		{Name: "rx_power_low", Severity: "alarm", Lane: 1},
	})

	if got, want := len(alarms), 2; got != want {
		t.Fatalf("merged alarms = %d, want %d", got, want)
	}
	if !alarms[0].Active {
		t.Fatal("duplicate alarm active state was not merged")
	}
	if got, want := alarms[1].Lane, 1; got != want {
		t.Fatalf("lane = %d, want %d", got, want)
	}
}

func TestCMISLengthScales(t *testing.T) {
	tests := []struct {
		name     string
		value    byte
		expected int
	}{
		{name: "tenths rounded up", value: 0x05, expected: 1},
		{name: "meters", value: 0x41, expected: 1},
		{name: "tens of meters", value: 0x81, expected: 10},
		{name: "hundreds of meters", value: 0xc1, expected: 100},
		{name: "maximum", value: 0xff, expected: 6300},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cmisCableAssemblyLengthMeters(tt.value); got != tt.expected {
				t.Fatalf("cmis cable assembly length = %d, want %d", got, tt.expected)
			}
		})
	}
}

func fixtureDirs(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	var dirs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join("testdata", entry.Name())
		if _, err := os.Stat(filepath.Join(dir, "ethtool.txt")); err != nil {
			t.Fatalf("fixture %s missing ethtool.txt: %v", entry.Name(), err)
		}
		dirs = append(dirs, dir)
	}

	if len(dirs) == 0 {
		t.Fatal("no fixtures found in testdata")
	}

	sort.Strings(dirs)
	return dirs
}

func readRawDumpFixture(t *testing.T, dir string) []byte {
	t.Helper()

	ethtoolOutput, err := os.ReadFile(filepath.Join(dir, "ethtool.txt"))
	if err != nil {
		t.Fatalf("read ethtool fixture: %v", err)
	}
	rawDump, rawDumpOK, err := parseEthtoolHexDump(string(ethtoolOutput))
	if err != nil {
		t.Fatalf("parse ethtool hex dump: %v", err)
	}
	if !rawDumpOK {
		t.Fatalf("fixture %s is not a raw ethtool hex dump", filepath.Base(dir))
	}
	return rawDump
}

func readEEPROMFixture(t *testing.T, dir string, rawDump []byte, rawDumpOK bool) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(dir, "eeprom.bin"))
	if err == nil {
		return data
	}
	if !os.IsNotExist(err) {
		t.Fatalf("read EEPROM fixture: %v", err)
	}
	if rawDumpOK {
		return rawDump
	}

	t.Fatalf("fixture %s missing eeprom.bin and ethtool.txt is not a raw hex dump", filepath.Base(dir))
	return nil
}

func putU16(data []byte, value uint16) {
	binary.BigEndian.PutUint16(data, value)
}

func hasAlarm(alarms []transceiver.Alarm, want transceiver.Alarm) bool {
	return slices.ContainsFunc(alarms, func(got transceiver.Alarm) bool {
		return got.Name == want.Name &&
			got.Severity == want.Severity &&
			got.Active == want.Active &&
			got.Lane == want.Lane
	})
}

func hasThreshold(thresholds []transceiver.Threshold, want transceiver.Threshold) bool {
	return slices.ContainsFunc(thresholds, func(got transceiver.Threshold) bool {
		return got.Metric == want.Metric &&
			got.Boundary == want.Boundary &&
			got.Severity == want.Severity &&
			got.Value == want.Value &&
			got.Lane == want.Lane
	})
}

func parseEthtoolHexDump(output string) ([]byte, bool, error) {
	var data []byte
	found := false

	for _, line := range strings.Split(output, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok || !strings.HasPrefix(key, "0x") {
			continue
		}

		offset, err := strconv.ParseInt(strings.TrimPrefix(key, "0x"), 16, 32)
		if err != nil {
			return nil, false, fmt.Errorf("parse offset %q: %w", key, err)
		}
		if int(offset) != len(data) {
			return nil, false, fmt.Errorf("offset %s follows %d decoded bytes", key, len(data))
		}

		for _, field := range strings.Fields(value) {
			byteValue, err := strconv.ParseUint(field, 16, 8)
			if err != nil {
				return nil, false, fmt.Errorf("parse byte %q at %s: %w", field, key, err)
			}
			data = append(data, byte(byteValue))
		}
		found = true
	}

	return data, found, nil
}

func parseEthtoolOutput(t *testing.T, output string) map[string][]string {
	t.Helper()

	values := map[string][]string{}
	for _, line := range strings.Split(output, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		values[key] = append(values[key], strings.TrimSpace(value))
	}

	return values
}

func moduleEthtoolOutput(module transceiver.Module) map[string][]string {
	values := map[string][]string{
		"Identifier":          {formatCodedByte(byte(module.FormFactor), strings.ToUpper(module.FormFactor.String()))},
		"Extended identifier": {formatCodedByte(module.Raw.ExtendedIdentifier, module.ExtendedIdentifier)},
		"Connector":           {formatCodedByte(byte(module.Connector), module.Connector.String())},
		"Transceiver codes":   {formatHexBytes(module.Raw.TransceiverCodes)},
		"Transceiver type":    capabilityEthtoolValues(module.Capabilities),
		"Encoding":            {formatCodedByte(byte(module.Encoding), module.Encoding.String())},
		"BR, Nominal":         {fmt.Sprintf("%dMBd", module.BitRate.NominalMBd)},
		"Rate identifier":     {formatCodedByte(byte(module.RateIdentifier), module.RateIdentifier.String())},
		complianceEthtoolName(module.CableCompliance.Kind): {
			fmt.Sprintf("%s [SFF-8472 rev10.4 only]", formatCodedByte(module.Raw.CableCompliance, module.CableCompliance.Standard)),
		},
		"Vendor name":    {module.Vendor.Name},
		"Vendor OUI":     {module.Vendor.OUI},
		"Vendor PN":      {module.Vendor.PartNumber},
		"Vendor rev":     {module.Vendor.Revision},
		"Option values":  {formatHexBytes(module.Raw.OptionValues)},
		"BR margin, max": {fmt.Sprintf("%d%%", module.BitRate.MaxMarginPercent)},
		"BR margin, min": {fmt.Sprintf("%d%%", module.BitRate.MinMarginPercent)},
		"Vendor SN":      {module.Vendor.SerialNumber},
		"Date code":      {module.Vendor.DateCode},
	}
	if len(module.Options) > 0 {
		values["Option"] = optionEthtoolValues(module.Options)
	}
	if module.Raw.DiagnosticsSupportReported {
		support := "No"
		if module.Raw.DiagnosticsSupported {
			support = "Yes"
		}
		values["Optical diagnostics support"] = []string{support}
	}
	addLengthValues(values, module.Lengths)
	return values
}

func addLengthValues(values map[string][]string, lengths []transceiver.Length) {
	byMedium := map[transceiver.LengthMedium]int{}
	for _, length := range lengths {
		byMedium[length.Medium] += length.Meters
	}

	values["Length (SMF,km)"] = []string{fmt.Sprintf("%dkm", byMedium[transceiver.LengthMediumSingleMode]/1000)}
	values["Length (SMF)"] = []string{fmt.Sprintf("%dm", byMedium[transceiver.LengthMediumSingleMode]%1000)}
	values["Length (50um)"] = []string{fmt.Sprintf("%dm", byMedium[transceiver.LengthMediumMultimode50um])}
	values["Length (62.5um)"] = []string{fmt.Sprintf("%dm", byMedium[transceiver.LengthMediumMultimode625um])}
	values["Length (Copper)"] = []string{fmt.Sprintf("%dm", byMedium[transceiver.LengthMediumCopper])}
	values["Length (OM3)"] = []string{fmt.Sprintf("%dm", byMedium[transceiver.LengthMediumOM3])}
}

func formatCodedByte(code byte, label string) string {
	return fmt.Sprintf("0x%02x (%s)", code, label)
}

func capabilityEthtoolValues(capabilities []transceiver.Capability) []string {
	values := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		switch capability {
		case transceiver.CapabilityInfiniBand1XCopperPassive:
			values = append(values, "Infiniband: 1X Copper Passive")
		case transceiver.CapabilityInfiniBand1XCopperActive:
			values = append(values, "Infiniband: 1X Copper Active")
		case transceiver.CapabilityEthernet1000BaseCX:
			values = append(values, "Ethernet: 1000BASE-CX")
		case transceiver.CapabilityFCShortDistance:
			values = append(values, "FC: short distance (S)")
		case transceiver.CapabilityFCElectricalInterEnclosure:
			values = append(values, "FC: Electrical inter-enclosure (EL)")
		case transceiver.CapabilityFCElectricalIntraEnclosure:
			values = append(values, "FC: Electrical intra-enclosure (EL)")
		case transceiver.CapabilityPassiveCable:
			values = append(values, "Passive Cable")
		case transceiver.CapabilityActiveCable:
			values = append(values, "Active Cable")
		case transceiver.CapabilityFCTwinAxialPair:
			values = append(values, "FC: Twin Axial Pair (TW)")
		case transceiver.CapabilityFC1200MBytes:
			values = append(values, "FC: 1200 MBytes/sec")
		case transceiver.CapabilityFC800MBytes:
			values = append(values, "FC: 800 MBytes/sec")
		case transceiver.CapabilityFC400MBytes:
			values = append(values, "FC: 400 MBytes/sec")
		case transceiver.CapabilityFC200MBytes:
			values = append(values, "FC: 200 MBytes/sec")
		case transceiver.CapabilityFC100MBytes:
			values = append(values, "FC: 100 MBytes/sec")
		case transceiver.CapabilityExtended25GBaseCRCAS:
			values = append(values, "Extended: 25G Base-CR CA-S")
		}
	}
	return values
}

func assertDiagnosticsSupport(t *testing.T, module transceiver.Module, want map[string][]string) {
	t.Helper()

	values, ok := want["Optical diagnostics support"]
	if !ok {
		return
	}
	if len(values) != 1 {
		t.Fatalf("fixture has %d Optical diagnostics support values, want 1", len(values))
	}

	switch values[0] {
	case "No":
		if module.Diagnostics.Supported {
			t.Fatal("diagnostics support = true, want false")
		}
		if module.Diagnostics.TemperatureCelsius.Valid || module.Diagnostics.VoltageVolts.Valid || len(module.Diagnostics.Lanes) > 0 || len(module.Diagnostics.Alarms) > 0 {
			t.Fatal("diagnostics readings populated for module without optical diagnostics support")
		}
	case "Yes":
		if !module.Diagnostics.Supported {
			t.Fatal("diagnostics support = false, want true")
		}
	default:
		t.Fatalf("unexpected Optical diagnostics support value %q", values[0])
	}
}

func assertRawDumpModule(t *testing.T, module transceiver.Module, data []byte) {
	t.Helper()

	if got, want := module.MemoryMap, detect(data); got != want {
		t.Fatalf("memory map = %q, want %q", got, want)
	}
	if got, want := module.FormFactor, transceiver.FormFactor(data[0]); got != want {
		t.Fatalf("form factor = 0x%02x, want 0x%02x", byte(got), byte(want))
	}
	if module.MemoryMap != transceiver.MemoryMapCMIS {
		return
	}

	if !module.Diagnostics.TemperatureCelsius.Valid {
		t.Fatal("CMIS temperature reading is invalid")
	}
	if !module.Diagnostics.VoltageVolts.Valid {
		t.Fatal("CMIS voltage reading is invalid")
	}
	if got, want := module.Diagnostics.TemperatureCelsius.Value, temp(data[14:16]); got != want {
		t.Fatalf("CMIS temperature = %v, want %v", got, want)
	}
	if got, want := module.Diagnostics.VoltageVolts.Value, voltage(data[16:18]); got != want {
		t.Fatalf("CMIS voltage = %v, want %v", got, want)
	}
}

func optionEthtoolValues(options []transceiver.Option) []string {
	values := make([]string, 0, len(options))
	for _, option := range options {
		switch option {
		case transceiver.OptionRXLOS:
			values = append(values, "RX_LOS implemented")
		case transceiver.OptionTXDisable:
			values = append(values, "TX_DISABLE implemented")
		}
	}
	return values
}

func complianceEthtoolName(kind transceiver.CableKind) string {
	if kind == transceiver.CableKindActive {
		return "Active Cu cmplnce."
	}
	return "Passive Cu cmplnce."
}

func formatHexBytes(data []byte) string {
	parts := make([]string, 0, len(data))
	for _, value := range data {
		parts = append(parts, fmt.Sprintf("0x%02x", value))
	}
	return strings.Join(parts, " ")
}

func formatEthtoolValues(values map[string][]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var builder strings.Builder
	for _, key := range keys {
		for _, value := range values[key] {
			builder.WriteString(key)
			builder.WriteString(": ")
			builder.WriteString(value)
			builder.WriteString("\n")
		}
	}
	return builder.String()
}
