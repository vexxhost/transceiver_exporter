package transceiver

import "encoding/json"

// MemoryMap identifies the transceiver management memory map used by a module.
type MemoryMap string

// Supported transceiver memory maps.
const (
	MemoryMapUnknown MemoryMap = "unknown"
	MemoryMapSFF8472 MemoryMap = "sff8472"
	MemoryMapSFF8636 MemoryMap = "sff8636"
	MemoryMapCMIS    MemoryMap = "cmis"
)

// FormFactor identifies the physical module form factor from the EEPROM
// identifier byte.
type FormFactor uint8

// Supported transceiver form factors.
const (
	FormFactorUnknown  FormFactor = 0
	FormFactorSFP      FormFactor = 0x03
	FormFactorQSFP     FormFactor = 0x0c
	FormFactorQSFPPlus FormFactor = 0x0d
	FormFactorQSFP28   FormFactor = 0x11
	FormFactorQSFPDD   FormFactor = 0x18
	FormFactorOSFP     FormFactor = 0x19
	FormFactorQSFPCMIS FormFactor = 0x1e
	FormFactorSFPDD    FormFactor = 0x1f
	FormFactorSFPCMIS  FormFactor = 0x20
)

// String returns the stable label used in JSON output and Prometheus labels.
func (f FormFactor) String() string {
	switch f {
	case FormFactorSFP:
		return "sfp"
	case FormFactorQSFP:
		return "qsfp"
	case FormFactorQSFPPlus:
		return "qsfp+"
	case FormFactorQSFP28:
		return "qsfp28"
	case FormFactorQSFPDD:
		return "qsfp-dd"
	case FormFactorOSFP:
		return "osfp"
	case FormFactorQSFPCMIS:
		return "qsfp_cmis"
	case FormFactorSFPDD:
		return "sfp-dd"
	case FormFactorSFPCMIS:
		return "sfp_cmis"
	default:
		return "unknown"
	}
}

// MarshalJSON marshals the form factor as its stable string label.
func (f FormFactor) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.String())
}

// Connector identifies the module connector type.
type Connector uint8

// Supported connector types.
const (
	ConnectorUnknown        Connector = 0x00
	ConnectorLC             Connector = 0x07
	ConnectorOpticalPigtail Connector = 0x0b
	ConnectorMPO1x12        Connector = 0x0c
	ConnectorMPO2x16        Connector = 0x0d
	ConnectorCopperPigtail  Connector = 0x21
	ConnectorRJ45           Connector = 0x22
	ConnectorNoSeparable    Connector = 0x23
	ConnectorMXC2x16        Connector = 0x24
	ConnectorCS             Connector = 0x25
	ConnectorSN             Connector = 0x26
	ConnectorMPO2x12        Connector = 0x27
	ConnectorMPO1x16        Connector = 0x28
)

// String returns a human-readable connector label.
func (c Connector) String() string {
	switch c {
	case ConnectorLC:
		return "LC"
	case ConnectorOpticalPigtail:
		return "Optical pigtail"
	case ConnectorMPO1x12:
		return "MPO 1x12"
	case ConnectorMPO2x16:
		return "MPO 2x16"
	case ConnectorCopperPigtail:
		return "Copper pigtail"
	case ConnectorRJ45:
		return "RJ45"
	case ConnectorNoSeparable:
		return "No separable connector"
	case ConnectorMXC2x16:
		return "MXC 2x16"
	case ConnectorCS:
		return "CS"
	case ConnectorSN:
		return "SN"
	case ConnectorMPO2x12:
		return "MPO 2x12"
	case ConnectorMPO1x16:
		return "MPO 1x16"
	default:
		return "unknown"
	}
}

// MarshalJSON marshals the connector as its human-readable label.
func (c Connector) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.String())
}

// Encoding identifies the serial encoding advertised by the module.
type Encoding uint8

// Supported encoding values.
const (
	EncodingUnspecified Encoding = 0x00
)

// String returns a human-readable encoding label.
func (e Encoding) String() string {
	switch e {
	case EncodingUnspecified:
		return "unspecified"
	default:
		return "unknown"
	}
}

// MarshalJSON marshals the encoding as its human-readable label.
func (e Encoding) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

// RateIdentifier identifies extended rate select behavior advertised by the
// module.
type RateIdentifier uint8

// Supported rate identifier values.
const (
	RateIdentifierUnspecified RateIdentifier = 0x00
)

// String returns a human-readable rate identifier label.
func (r RateIdentifier) String() string {
	switch r {
	case RateIdentifierUnspecified:
		return "unspecified"
	default:
		return "unknown"
	}
}

// MarshalJSON marshals the rate identifier as its human-readable label.
func (r RateIdentifier) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

// Capability identifies an advertised transceiver capability or compliance
// code.
type Capability string

// Supported transceiver capability values.
const (
	CapabilityInfiniBand1XCopperPassive  Capability = "infiniband_1x_copper_passive"
	CapabilityInfiniBand1XCopperActive   Capability = "infiniband_1x_copper_active"
	CapabilityEthernet1000BaseCX         Capability = "ethernet_1000base_cx"
	CapabilityFCShortDistance            Capability = "fc_short_distance"
	CapabilityFCElectricalInterEnclosure Capability = "fc_electrical_inter_enclosure"
	CapabilityFCElectricalIntraEnclosure Capability = "fc_electrical_intra_enclosure"
	CapabilityPassiveCable               Capability = "passive_cable"
	CapabilityActiveCable                Capability = "active_cable"
	CapabilityFCTwinAxialPair            Capability = "fc_twin_axial_pair"
	CapabilityFC1200MBytes               Capability = "fc_1200_mbytes"
	CapabilityFC800MBytes                Capability = "fc_800_mbytes"
	CapabilityFC400MBytes                Capability = "fc_400_mbytes"
	CapabilityFC200MBytes                Capability = "fc_200_mbytes"
	CapabilityFC100MBytes                Capability = "fc_100_mbytes"
	CapabilityExtended25GBaseCRCAS       Capability = "extended_25g_base_cr_ca_s"
)

// Option identifies an advertised module option bit.
type Option string

// Supported module option values.
const (
	OptionRXLOS     Option = "rx_los"
	OptionTXDisable Option = "tx_disable"
)

// CableKind describes whether a copper cable assembly is passive or active.
type CableKind string

// Supported cable kinds.
const (
	CableKindPassive CableKind = "passive"
	CableKindActive  CableKind = "active"
)

// CableCompliance describes copper cable compliance information.
type CableCompliance struct {
	Kind     CableKind `json:"kind,omitempty"`
	Standard string    `json:"standard,omitempty"`
}

// Vendor describes module vendor identity fields.
type Vendor struct {
	Name         string `json:"name,omitempty"`
	OUI          string `json:"oui,omitempty"`
	PartNumber   string `json:"part_number,omitempty"`
	Revision     string `json:"revision,omitempty"`
	SerialNumber string `json:"serial_number,omitempty"`
	DateCode     string `json:"date_code,omitempty"`
}

// BitRate describes nominal signaling rate and advertised margin.
type BitRate struct {
	NominalMBd       int `json:"nominal_mbd,omitempty"`
	MaxMarginPercent int `json:"max_margin_percent,omitempty"`
	MinMarginPercent int `json:"min_margin_percent,omitempty"`
}

// LengthMedium identifies the medium a supported link length applies to.
type LengthMedium string

// Supported link length media.
const (
	LengthMediumSingleMode     LengthMedium = "single_mode"
	LengthMediumMultimode50um  LengthMedium = "multimode_50um"
	LengthMediumMultimode625um LengthMedium = "multimode_625um"
	LengthMediumCopper         LengthMedium = "copper"
	LengthMediumOM3            LengthMedium = "om3"
	LengthMediumOM4            LengthMedium = "om4"
	LengthMediumOM5            LengthMedium = "om5"
	LengthMediumOM2            LengthMedium = "om2"
	LengthMediumOM1            LengthMedium = "om1"
	LengthMediumCableAssembly  LengthMedium = "cable_assembly"
)

// Length describes a supported link length for a medium.
type Length struct {
	Medium LengthMedium `json:"medium"`
	Meters int          `json:"meters"`
}

// Reading is a numeric diagnostic value with an explicit validity bit.
//
// The zero value is invalid, which lets decoders distinguish an absent EEPROM
// field from a real value of zero.
type Reading struct {
	Value float64 `json:"value"`
	Valid bool    `json:"valid"`
}

// NewReading returns a valid diagnostic reading.
func NewReading(value float64) Reading {
	return Reading{Value: value, Valid: true}
}

// Lane contains per-lane diagnostic data.
type Lane struct {
	Index             int     `json:"index"`
	DataPathState     string  `json:"data_path_state,omitempty"`
	TXBiasMilliAmps   Reading `json:"tx_bias_milliamps,omitempty"`
	TXPowerMilliWatts Reading `json:"tx_power_milliwatts,omitempty"`
	RXPowerMilliWatts Reading `json:"rx_power_milliwatts,omitempty"`
}

// Alarm describes an alarm, warning, or fault condition.
type Alarm struct {
	Name     string `json:"name"`
	Severity string `json:"severity"`
	Active   bool   `json:"active"`
	Lane     int    `json:"lane,omitempty"`
}

// Threshold describes a diagnostic threshold value.
type Threshold struct {
	Metric   string  `json:"metric"`
	Boundary string  `json:"boundary"`
	Severity string  `json:"severity"`
	Value    float64 `json:"value"`
	Lane     int     `json:"lane,omitempty"`
}

// Diagnostics contains module-level and per-lane monitoring data.
type Diagnostics struct {
	Supported          bool        `json:"supported"`
	TemperatureCelsius Reading     `json:"temperature_celsius,omitempty"`
	VoltageVolts       Reading     `json:"voltage_volts,omitempty"`
	Lanes              []Lane      `json:"lanes,omitempty"`
	Alarms             []Alarm     `json:"alarms,omitempty"`
	Thresholds         []Threshold `json:"thresholds,omitempty"`
}

// ModuleStatus describes module state, fault, and low-power request fields.
type ModuleStatus struct {
	State                        string `json:"state,omitempty"`
	Fault                        string `json:"fault,omitempty"`
	LowPowerAllowRequestHardware bool   `json:"low_power_allow_request_hardware"`
	LowPowerRequestSoftware      bool   `json:"low_power_request_software"`
}

// ModulePower describes module power class information.
type ModulePower struct {
	Class    int     `json:"class,omitempty"`
	MaxWatts float64 `json:"max_watts,omitempty"`
}

// Media describes module media information.
type Media struct {
	Type                          string  `json:"type,omitempty"`
	InterfaceTechnology           string  `json:"interface_technology,omitempty"`
	WavelengthNanometers          float64 `json:"wavelength_nanometers,omitempty"`
	WavelengthToleranceNanometers float64 `json:"wavelength_tolerance_nanometers,omitempty"`
	TXCDR                         bool    `json:"tx_cdr"`
	RXCDR                         bool    `json:"rx_cdr"`
	TXCDRBypassControlSupported   bool    `json:"tx_cdr_bypass_control_supported"`
	RXCDRBypassControlSupported   bool    `json:"rx_cdr_bypass_control_supported"`
}

// Raw contains standards-specific decoder metadata.
//
// Callers should prefer the normalized Module fields when possible. Raw is
// exposed for compatibility checks and tools that need to compare decoded output
// with lower-level EEPROM utilities.
type Raw struct {
	ExtendedIdentifier         byte   `json:"extended_identifier,omitempty"`
	TransceiverCodes           []byte `json:"transceiver_codes,omitempty"`
	OptionValues               []byte `json:"option_values,omitempty"`
	CableCompliance            byte   `json:"cable_compliance,omitempty"`
	DiagnosticsSupportReported bool   `json:"diagnostics_support_reported,omitempty"`
	DiagnosticsSupported       bool   `json:"diagnostics_supported,omitempty"`
}

// Module is a decoded transceiver module.
type Module struct {
	MemoryMap          MemoryMap       `json:"memory_map"`
	FormFactor         FormFactor      `json:"form_factor"`
	RevisionCompliance string          `json:"revision_compliance,omitempty"`
	ExtendedIdentifier string          `json:"extended_identifier,omitempty"`
	Connector          Connector       `json:"connector,omitempty"`
	Encoding           Encoding        `json:"encoding,omitempty"`
	RateIdentifier     RateIdentifier  `json:"rate_identifier,omitempty"`
	Capabilities       []Capability    `json:"capabilities,omitempty"`
	CableCompliance    CableCompliance `json:"cable_compliance,omitempty"`
	Vendor             Vendor          `json:"vendor,omitempty"`
	BitRate            BitRate         `json:"bit_rate,omitempty"`
	Lengths            []Length        `json:"lengths,omitempty"`
	Options            []Option        `json:"options,omitempty"`
	Status             *ModuleStatus   `json:"status,omitempty"`
	Power              *ModulePower    `json:"power,omitempty"`
	Media              *Media          `json:"media,omitempty"`
	Diagnostics        Diagnostics     `json:"diagnostics,omitempty"`
	Raw                Raw             `json:"-"`
}

// Observation is a decoded module associated with a host network interface.
type Observation struct {
	Interface string `json:"interface"`
	Module    Module `json:"module"`
}
