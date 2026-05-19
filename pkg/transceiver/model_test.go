package transceiver

import (
	"encoding/json"
	"testing"
)

func TestStringEnumsMarshalAsJSONStrings(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected string
	}{
		{name: "form factor", value: FormFactorSFP, expected: `"sfp"`},
		{name: "connector", value: ConnectorCopperPigtail, expected: `"Copper pigtail"`},
		{name: "encoding", value: EncodingUnspecified, expected: `"unspecified"`},
		{name: "rate identifier", value: RateIdentifierUnspecified, expected: `"unspecified"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatalf("marshal json: %v", err)
			}
			if string(got) != tt.expected {
				t.Fatalf("json = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestFormFactorString(t *testing.T) {
	tests := []struct {
		value    FormFactor
		expected string
	}{
		{value: FormFactorSFP, expected: "sfp"},
		{value: FormFactorQSFP, expected: "qsfp"},
		{value: FormFactorQSFPPlus, expected: "qsfp+"},
		{value: FormFactorQSFP28, expected: "qsfp28"},
		{value: FormFactorQSFPDD, expected: "qsfp-dd"},
		{value: FormFactorOSFP, expected: "osfp"},
		{value: FormFactorQSFPCMIS, expected: "qsfp_cmis"},
		{value: FormFactorSFPDD, expected: "sfp-dd"},
		{value: FormFactorSFPCMIS, expected: "sfp_cmis"},
		{value: FormFactorUnknown, expected: "unknown"},
	}

	for _, tt := range tests {
		if got := tt.value.String(); got != tt.expected {
			t.Fatalf("FormFactor(%d).String() = %q, want %q", tt.value, got, tt.expected)
		}
	}
}

func TestConnectorString(t *testing.T) {
	tests := []struct {
		value    Connector
		expected string
	}{
		{value: ConnectorLC, expected: "LC"},
		{value: ConnectorOpticalPigtail, expected: "Optical pigtail"},
		{value: ConnectorMPO1x12, expected: "MPO 1x12"},
		{value: ConnectorMPO2x16, expected: "MPO 2x16"},
		{value: ConnectorCopperPigtail, expected: "Copper pigtail"},
		{value: ConnectorRJ45, expected: "RJ45"},
		{value: ConnectorNoSeparable, expected: "No separable connector"},
		{value: ConnectorMXC2x16, expected: "MXC 2x16"},
		{value: ConnectorCS, expected: "CS"},
		{value: ConnectorSN, expected: "SN"},
		{value: ConnectorMPO2x12, expected: "MPO 2x12"},
		{value: ConnectorMPO1x16, expected: "MPO 1x16"},
		{value: ConnectorUnknown, expected: "unknown"},
	}

	for _, tt := range tests {
		if got := tt.value.String(); got != tt.expected {
			t.Fatalf("Connector(%d).String() = %q, want %q", tt.value, got, tt.expected)
		}
	}
}

func TestEncodingAndRateIdentifierString(t *testing.T) {
	if got, want := EncodingUnspecified.String(), "unspecified"; got != want {
		t.Fatalf("encoding string = %q, want %q", got, want)
	}
	if got, want := Encoding(0xff).String(), "unknown"; got != want {
		t.Fatalf("unknown encoding string = %q, want %q", got, want)
	}
	if got, want := RateIdentifierUnspecified.String(), "unspecified"; got != want {
		t.Fatalf("rate identifier string = %q, want %q", got, want)
	}
	if got, want := RateIdentifier(0xff).String(), "unknown"; got != want {
		t.Fatalf("unknown rate identifier string = %q, want %q", got, want)
	}
}

func TestNewReadingMarksValueValid(t *testing.T) {
	reading := NewReading(12.34)

	if !reading.Valid {
		t.Fatal("reading valid = false, want true")
	}
	if got, want := reading.Value, 12.34; got != want {
		t.Fatalf("reading value = %v, want %v", got, want)
	}
}
