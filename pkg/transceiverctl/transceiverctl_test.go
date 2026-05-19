package transceiverctl

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"syscall"
	"testing"

	"github.com/vexxhost/transceiver_exporter/pkg/transceiver"
)

func TestParseFlags(t *testing.T) {
	cfg, err := ParseFlags([]string{"-json", "-interface", "eth0", "-interface", "eth1", "-interface", "ens2"})
	if err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	if !cfg.JSON {
		t.Fatal("json = false, want true")
	}
	if want := []string{"eth0", "eth1", "ens2"}; !slices.Equal(cfg.Interfaces, want) {
		t.Fatalf("interfaces = %v, want %v", cfg.Interfaces, want)
	}
}

func TestParseFlagsDoesNotSplitCommaSeparatedInterfaces(t *testing.T) {
	cfg, err := ParseFlags([]string{"-interface", "eth0,eth1"})
	if err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	if want := []string{"eth0,eth1"}; !slices.Equal(cfg.Interfaces, want) {
		t.Fatalf("interfaces = %v, want %v", cfg.Interfaces, want)
	}
}

func TestRunSkipsUnsupportedAutodiscoveredInterfaces(t *testing.T) {
	var stdout, stderr bytes.Buffer
	eepromReader := &fakeReader{
		data: map[string][]byte{
			"eth1": {0x03},
		},
		errs: map[string]error{
			"eth0": syscall.EOPNOTSUPP,
		},
	}
	deps := dependencies{
		NewReader: func() (reader, error) {
			return eepromReader, nil
		},
		CandidateNames: func() ([]string, error) {
			return []string{"eth0", "eth1"}, nil
		},
		Decode: fakeDecode,
	}

	if code := run([]string{"-json"}, &stdout, &stderr, deps); code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if strings.Contains(stderr.String(), "eth0") {
		t.Fatalf("unsupported autodiscovered interface was reported: %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), `"interface": "eth1"`) {
		t.Fatalf("stdout missing eth1 observation: %q", stdout.String())
	}
	if !eepromReader.closed {
		t.Fatal("reader was not closed")
	}
}

func TestRunReturnsFailureWhenAllExplicitReadsFail(t *testing.T) {
	var stdout, stderr bytes.Buffer
	deps := dependencies{
		NewReader: func() (reader, error) {
			return &fakeReader{
				errs: map[string]error{"eth0": errors.New("device failed")},
			}, nil
		},
		Decode: fakeDecode,
	}

	if code := run([]string{"-interface", "eth0"}, &stdout, &stderr, deps); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "read eth0") {
		t.Fatalf("stderr missing read failure: %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}

func TestPrintObservationWritesHumanReadableOutput(t *testing.T) {
	var stdout bytes.Buffer
	observation := fakeObservation("eth0")
	observation.Module.Diagnostics.TemperatureCelsius = transceiver.NewReading(32.5)
	observation.Module.Diagnostics.Lanes = []transceiver.Lane{
		{
			Index:             1,
			TXBiasMilliAmps:   transceiver.NewReading(7.2),
			TXPowerMilliWatts: transceiver.NewReading(0.0123),
			RXPowerMilliWatts: transceiver.NewReading(0.0456),
		},
	}
	observation.Module.Diagnostics.Alarms = []transceiver.Alarm{
		{Name: "temperature_high", Severity: "warning", Active: true},
		{Name: "rx_power_low", Severity: "alarm", Active: true, Lane: 1},
	}

	if err := printObservation(&stdout, observation); err != nil {
		t.Fatalf("print observation: %v", err)
	}

	for _, want := range []string{
		"eth0: sff8472 sfp",
		"temperature: 32.50 C",
		"lane 1:",
		"tx bias: 7.200 mA",
		"warning temperature_high: active",
		"lane 1 alarm rx_power_low: active",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

type fakeReader struct {
	data   map[string][]byte
	errs   map[string]error
	closed bool
}

func (f *fakeReader) ModuleEEPROM(interfaceName string) ([]byte, error) {
	if err := f.errs[interfaceName]; err != nil {
		return nil, err
	}
	return f.data[interfaceName], nil
}

func (f *fakeReader) Close() error {
	f.closed = true
	return nil
}

func fakeDecode(interfaceName string, _ []byte) (transceiver.Observation, error) {
	return fakeObservation(interfaceName), nil
}

func fakeObservation(interfaceName string) transceiver.Observation {
	return transceiver.Observation{
		Interface: interfaceName,
		Module: transceiver.Module{
			MemoryMap:  transceiver.MemoryMapSFF8472,
			FormFactor: transceiver.FormFactorSFP,
			Vendor: transceiver.Vendor{
				Name:         "vendor",
				PartNumber:   "part",
				SerialNumber: "serial",
			},
		},
	}
}
