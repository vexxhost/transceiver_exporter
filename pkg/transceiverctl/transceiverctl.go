package transceiverctl

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/vexxhost/transceiver_exporter/pkg/moduleeeprom"
	"github.com/vexxhost/transceiver_exporter/pkg/netdev"
	"github.com/vexxhost/transceiver_exporter/pkg/sff"
	"github.com/vexxhost/transceiver_exporter/pkg/transceiver"
)

type reader interface {
	ModuleEEPROM(interfaceName string) ([]byte, error)
	Close() error
}

// Config contains parsed transceiverctl command options.
type Config struct {
	// JSON selects machine-readable JSON output instead of the text format.
	JSON bool

	// Interfaces is the explicit interface allowlist. When empty,
	// transceiverctl discovers physical non-loopback interfaces.
	Interfaces []string
}

type dependencies struct {
	NewReader      func() (reader, error)
	CandidateNames func() ([]string, error)
	Decode         func(interfaceName string, data []byte) (transceiver.Observation, error)
}

// ParseFlags parses transceiverctl command-line arguments.
func ParseFlags(args []string) (Config, error) {
	interfaces := []string{}

	fs := flag.NewFlagSet("transceiverctl", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOutput := fs.Bool("json", false, "Print decoded modules as JSON.")
	fs.Func("interface", "Interface to inspect. May be specified multiple times. Defaults to physical non-loopback interfaces.", func(value string) error {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil
		}
		interfaces = append(interfaces, value)
		return nil
	})

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	return Config{
		JSON:       *jsonOutput,
		Interfaces: append([]string(nil), interfaces...),
	}, nil
}

// Run executes transceiverctl and returns a process-style exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	return run(args, stdout, stderr, dependencies{})
}

func run(args []string, stdout, stderr io.Writer, deps dependencies) int {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}

	cfg, err := ParseFlags(args)
	if err != nil {
		writeDiagnostic(stderr, "transceiverctl: %v\n", err)
		return 2
	}

	deps = deps.withDefaults()
	reader, err := deps.NewReader()
	if err != nil {
		writeDiagnostic(stderr, "transceiverctl: open ethtool client: %v\n", err)
		return 1
	}

	exitCode := runWithReader(cfg, reader, stdout, stderr, deps)
	if err := reader.Close(); err != nil {
		writeDiagnostic(stderr, "transceiverctl: close eeprom reader: %v\n", err)
		if exitCode == 0 {
			exitCode = 1
		}
	}
	return exitCode
}

func (d dependencies) withDefaults() dependencies {
	if d.NewReader == nil {
		d.NewReader = func() (reader, error) {
			return moduleeeprom.New()
		}
	}
	if d.CandidateNames == nil {
		d.CandidateNames = netdev.CandidateNames
	}
	if d.Decode == nil {
		d.Decode = sff.DecodeObservation
	}
	return d
}

func runWithReader(cfg Config, reader reader, stdout, stderr io.Writer, deps dependencies) int {
	interfaces := cfg.Interfaces
	autoDiscovered := len(interfaces) == 0
	if autoDiscovered {
		discovered, err := deps.CandidateNames()
		if err != nil {
			writeDiagnostic(stderr, "transceiverctl: discover interfaces: %v\n", err)
			return 1
		}
		interfaces = discovered
		if len(interfaces) == 0 {
			writeDiagnostic(stderr, "transceiverctl: no physical non-loopback interfaces discovered\n")
		}
	}

	observations := make([]transceiver.Observation, 0, len(interfaces))
	hadError := false
	for _, interfaceName := range interfaces {
		data, err := reader.ModuleEEPROM(interfaceName)
		if err != nil {
			if autoDiscovered && netdev.IsUnsupportedModuleError(err) {
				continue
			}
			writeDiagnostic(stderr, "transceiverctl: read %s: %v\n", interfaceName, err)
			hadError = true
			continue
		}

		observation, err := deps.Decode(interfaceName, data)
		if err != nil {
			writeDiagnostic(stderr, "transceiverctl: decode %s: %v\n", interfaceName, err)
			hadError = true
			continue
		}
		observations = append(observations, observation)
	}
	if autoDiscovered && len(observations) == 0 && !hadError && len(interfaces) > 0 {
		writeDiagnostic(stderr, "transceiverctl: no readable transceiver eeprom found on discovered interfaces: %s\n", strings.Join(interfaces, ", "))
	}

	if cfg.JSON {
		if err := writeJSON(stdout, observations); err != nil {
			writeDiagnostic(stderr, "transceiverctl: encode json: %v\n", err)
			return 1
		}
	} else {
		for _, observation := range observations {
			if err := printObservation(stdout, observation); err != nil {
				writeDiagnostic(stderr, "transceiverctl: write output: %v\n", err)
				return 1
			}
		}
	}

	if hadError && len(observations) == 0 {
		return 1
	}
	return 0
}

func writeJSON(w io.Writer, observations []transceiver.Observation) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(observations)
}

func writeDiagnostic(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

func printObservation(w io.Writer, observation transceiver.Observation) error {
	module := observation.Module
	diagnostics := module.Diagnostics

	if _, err := fmt.Fprintf(w, "%s: %s %s %s %s %s\n", observation.Interface, module.MemoryMap, module.FormFactor, module.Vendor.Name, module.Vendor.PartNumber, module.Vendor.SerialNumber); err != nil {
		return err
	}
	if diagnostics.TemperatureCelsius.Valid {
		if _, err := fmt.Fprintf(w, "  temperature: %.2f C\n", diagnostics.TemperatureCelsius.Value); err != nil {
			return err
		}
	}
	if diagnostics.VoltageVolts.Valid {
		if _, err := fmt.Fprintf(w, "  voltage: %.4f V\n", diagnostics.VoltageVolts.Value); err != nil {
			return err
		}
	}
	for _, lane := range diagnostics.Lanes {
		if _, err := fmt.Fprintf(w, "  lane %d:\n", lane.Index); err != nil {
			return err
		}
		if lane.TXBiasMilliAmps.Valid {
			if _, err := fmt.Fprintf(w, "    tx bias: %.3f mA\n", lane.TXBiasMilliAmps.Value); err != nil {
				return err
			}
		}
		if lane.TXPowerMilliWatts.Valid {
			if _, err := fmt.Fprintf(w, "    tx power: %.4f mW\n", lane.TXPowerMilliWatts.Value); err != nil {
				return err
			}
		}
		if lane.RXPowerMilliWatts.Valid {
			if _, err := fmt.Fprintf(w, "    rx power: %.4f mW\n", lane.RXPowerMilliWatts.Value); err != nil {
				return err
			}
		}
	}
	for _, alarm := range diagnostics.Alarms {
		if !alarm.Active {
			continue
		}
		if alarm.Lane > 0 {
			if _, err := fmt.Fprintf(w, "  lane %d %s %s: active\n", alarm.Lane, alarm.Severity, alarm.Name); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintf(w, "  %s %s: active\n", alarm.Severity, alarm.Name); err != nil {
			return err
		}
	}
	return nil
}
