# Transceiver Exporter Alerts

These alerts follow the Google SRE alerting model: critical alerts are intended
for symptoms that need prompt human action, while warning alerts are
ticket-level signals for missing telemetry, degraded module access, or early
hardware warning thresholds. Lane-scoped alarm alerts are suppressed unless the
lane datapath is activated, so intentionally unused CMIS lanes do not page.

## Alert Name: TransceiverExporterDown

Prometheus cannot scrape the exporter. Check that the service is running, the
listen address is reachable, and any exporter-toolkit TLS or authentication
configuration matches the scrape job. This is a telemetry-loss warning, not a
direct hardware failure signal.

## Alert Name: TransceiverScrapeFailed

The exporter found an interface to scrape but could not read or decode its
module EEPROM. Check whether the interface still has a module installed,
whether the NIC driver supports module EEPROM reads, and whether the exporter
has enough privileges to use the ethtool API.

## Alert Name: TransceiverWarningActive

The module reports a warning threshold condition. Inspect the affected interface
and lane, compare current optical power, temperature, or voltage with the
exported threshold values, and check the peer optic and patching path. Treat
this as ticket-level unless it correlates with link errors or service impact.

## Alert Name: TransceiverAlarmActive

The module reports an alarm threshold condition. Treat this as likely service
impact or imminent loss of signal. Check the affected lane, replace the optic or
cable if levels remain outside threshold, and verify that the peer is
transmitting as expected.

## Alert Name: TransceiverFaultActive

The module reports a fault such as loss of signal, loss of lock, or transmitter
fault. Check link state, peer transmit state, lane mapping, cabling, and module
health.

## Alert Name: TransceiverModuleFault

The module itself reports a fault state. Check module firmware/status fields,
power mode, seating, and compatibility with the NIC.

## Alert Name: TransceiverModuleLowPower

The module reports a low-power request. Check whether this is expected for the
platform policy. If the link should be active, verify host firmware, module
power class support, and NIC configuration.
