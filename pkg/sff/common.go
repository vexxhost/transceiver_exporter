package sff

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	"github.com/vexxhost/transceiver_exporter/pkg/transceiver"
)

func oui(data []byte) string {
	parts := make([]string, 0, len(data))
	for _, value := range data {
		parts = append(parts, fmt.Sprintf("%02x", value))
	}
	return strings.Join(parts, ":")
}

func cleanString(data []byte) string {
	s := strings.TrimRight(string(data), " \x00")
	return strings.TrimSpace(s)
}

type scaler func([]byte) float64

func thresholdAlarms(reading transceiver.Reading, name string, scale scaler, data []byte) []transceiver.Alarm {
	return thresholdAlarmsWithLane(reading, name, scale, data, 0)
}

func thresholdAlarmsWithLane(reading transceiver.Reading, name string, scale scaler, data []byte, lane int) []transceiver.Alarm {
	if !reading.Valid || len(data) < 8 {
		return nil
	}
	if allZero(data) {
		return nil
	}

	return []transceiver.Alarm{
		thresholdAlarm(reading, name+"_high", "alarm", scale, data[0:2], true, lane),
		thresholdAlarm(reading, name+"_low", "alarm", scale, data[2:4], false, lane),
		thresholdAlarm(reading, name+"_high", "warning", scale, data[4:6], true, lane),
		thresholdAlarm(reading, name+"_low", "warning", scale, data[6:8], false, lane),
	}
}

func thresholds(metric string, scale scaler, data []byte) []transceiver.Threshold {
	return thresholdsWithLane(metric, scale, data, 0)
}

func thresholdsWithLane(metric string, scale scaler, data []byte, lane int) []transceiver.Threshold {
	if len(data) < 8 || allZero(data) {
		return nil
	}

	return []transceiver.Threshold{
		threshold(metric, "high", "alarm", scale, data[0:2], lane),
		threshold(metric, "low", "alarm", scale, data[2:4], lane),
		threshold(metric, "high", "warning", scale, data[4:6], lane),
		threshold(metric, "low", "warning", scale, data[6:8], lane),
	}
}

func allZero(data []byte) bool {
	for _, value := range data {
		if value != 0 {
			return false
		}
	}
	return true
}

func allEqual(data []byte, value byte) bool {
	for _, got := range data {
		if got != value {
			return false
		}
	}
	return true
}

func thresholdAlarm(reading transceiver.Reading, name, severity string, scale scaler, raw []byte, high bool, lane int) transceiver.Alarm {
	threshold := scale(raw)
	active := false
	if high {
		active = reading.Value > threshold
	} else {
		active = reading.Value < threshold
	}

	return transceiver.Alarm{Name: name, Severity: severity, Active: active, Lane: lane}
}

func threshold(metric, boundary, severity string, scale scaler, raw []byte, lane int) transceiver.Threshold {
	return transceiver.Threshold{
		Metric:   metric,
		Boundary: boundary,
		Severity: severity,
		Value:    scale(raw),
		Lane:     lane,
	}
}

func temp(data []byte) float64 {
	return float64(int16(binary.BigEndian.Uint16(data))) / 256
}

func voltage(data []byte) float64 {
	return float64(binary.BigEndian.Uint16(data)) / 10000
}

func bias(data []byte) float64 {
	return float64(binary.BigEndian.Uint16(data)) * 0.002
}

func power(data []byte) float64 {
	return round(float64(binary.BigEndian.Uint16(data))*0.0001, 6)
}

func round(value float64, precision int) float64 {
	scale := math.Pow10(precision)
	return math.Round(value*scale) / scale
}
