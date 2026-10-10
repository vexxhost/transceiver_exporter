package moduleeeprom

import (
	"bytes"
	"errors"
	"testing"
)

func TestReaderModuleEEPROMFallbacks(t *testing.T) {
	ioctlErr := errors.New("ioctl failed")
	netlinkProbeErr := errors.New("netlink probe failed")
	netlinkErr := errors.New("netlink failed")

	tests := []struct {
		name          string
		ioctl         *fakeIOCTL
		netlink       *fakeNetlink
		expected      []byte
		expectedError error
	}{
		{
			name:     "ioctl success non-cmis",
			ioctl:    &fakeIOCTL{data: []byte{0x03}},
			netlink:  &fakeNetlink{identifier: 0x03, data: []byte{0x18}},
			expected: []byte{0x03},
		},
		{
			name:     "cmis prefers netlink pages before ioctl",
			ioctl:    &fakeIOCTL{data: []byte{0x18}},
			netlink:  &fakeNetlink{identifier: 0x1e, data: []byte{0x18, 0x01}},
			expected: []byte{0x18, 0x01},
		},
		{
			name:     "cmis falls back to ioctl when netlink fails",
			ioctl:    &fakeIOCTL{data: []byte{0x18}},
			netlink:  &fakeNetlink{identifier: 0x1e, err: netlinkErr},
			expected: []byte{0x18},
		},
		{
			name:          "ioctl error without netlink",
			ioctl:         &fakeIOCTL{err: ioctlErr},
			expectedError: ioctlErr,
		},
		{
			name:     "ioctl error falls back to netlink",
			ioctl:    &fakeIOCTL{err: ioctlErr},
			netlink:  &fakeNetlink{identifier: 0x03, data: []byte{0x18}},
			expected: []byte{0x18},
		},
		{
			name:          "ioctl and netlink errors are joined",
			ioctl:         &fakeIOCTL{err: ioctlErr},
			netlink:       &fakeNetlink{identifier: 0x03, err: netlinkErr},
			expectedError: netlinkErr,
		},
		{
			name:     "cmis after ioctl still prefers netlink when probe fails",
			ioctl:    &fakeIOCTL{data: []byte{0x1e}},
			netlink:  &fakeNetlink{identifierErr: netlinkProbeErr, data: []byte{0x1e, 0x01}},
			expected: []byte{0x1e, 0x01},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var netlink netlinkModuleReader
			if tt.netlink != nil {
				netlink = tt.netlink
			}
			reader := newReader(tt.ioctl, netlink)
			got, err := reader.ModuleEEPROM("eth0")
			if tt.expectedError != nil {
				if !errors.Is(err, tt.expectedError) {
					t.Fatalf("error = %v, want %v", err, tt.expectedError)
				}
				return
			}
			if err != nil {
				t.Fatalf("module eeprom: %v", err)
			}
			if !bytes.Equal(got, tt.expected) {
				t.Fatalf("data = %x, want %x", got, tt.expected)
			}

			switch tt.name {
			case "cmis prefers netlink pages before ioctl":
				if tt.ioctl.calls != 0 {
					t.Fatalf("ioctl calls = %d, want 0", tt.ioctl.calls)
				}
				if tt.netlink.identifierCalls != 1 {
					t.Fatalf("netlink identifier calls = %d, want 1", tt.netlink.identifierCalls)
				}
				if tt.netlink.eepromCalls != 1 {
					t.Fatalf("netlink EEPROM calls = %d, want 1", tt.netlink.eepromCalls)
				}
			case "ioctl success non-cmis":
				if tt.ioctl.calls != 1 {
					t.Fatalf("ioctl calls = %d, want 1", tt.ioctl.calls)
				}
				if tt.netlink.identifierCalls != 1 {
					t.Fatalf("netlink identifier calls = %d, want 1", tt.netlink.identifierCalls)
				}
				if tt.netlink.eepromCalls != 0 {
					t.Fatalf("netlink EEPROM calls = %d, want 0", tt.netlink.eepromCalls)
				}
			}
		})
	}
}

func TestReaderCloseClosesBothClients(t *testing.T) {
	ioctl := &fakeIOCTL{}
	netlink := &fakeNetlink{}

	if err := newReader(ioctl, netlink).Close(); err != nil {
		t.Fatalf("close reader: %v", err)
	}
	if !ioctl.closed {
		t.Fatal("ioctl client was not closed")
	}
	if !netlink.closed {
		t.Fatal("netlink client was not closed")
	}
}

func TestReaderCloseClosesPresenceProbe(t *testing.T) {
	presence := &fakePresenceProbe{}

	if err := newReaderWithPresence(&fakeIOCTL{}, &fakeNetlink{}, presence).Close(); err != nil {
		t.Fatalf("close reader: %v", err)
	}
	if !presence.closed {
		t.Fatal("presence probe was not closed")
	}
}

func TestReaderModuleEEPROMReturnsErrModuleAbsentBeforeRead(t *testing.T) {
	ioctl := &fakeIOCTL{data: []byte{0x03}}
	netlink := &fakeNetlink{identifier: 0x03, data: []byte{0x18}}
	reader := newReaderWithPresence(ioctl, netlink, &fakePresenceProbe{present: false})

	_, err := reader.ModuleEEPROM("eth0")
	if !errors.Is(err, ErrModuleAbsent) {
		t.Fatalf("module eeprom error = %v, want ErrModuleAbsent", err)
	}
	if ioctl.calls != 0 {
		t.Fatalf("ioctl calls = %d, want 0", ioctl.calls)
	}
	if netlink.identifierCalls != 0 {
		t.Fatalf("netlink identifier calls = %d, want 0", netlink.identifierCalls)
	}
}

func TestReaderModuleEEPROMContinuesWhenPresenceProbeErrors(t *testing.T) {
	reader := newReaderWithPresence(
		&fakeIOCTL{data: []byte{0x03}},
		nil,
		&fakePresenceProbe{err: errors.New("probe failed")},
	)

	got, err := reader.ModuleEEPROM("eth0")
	if err != nil {
		t.Fatalf("module eeprom: %v", err)
	}
	if !bytes.Equal(got, []byte{0x03}) {
		t.Fatalf("data = %x, want 03", got)
	}
}

func TestReaderCloseReturnsNetlinkError(t *testing.T) {
	closeErr := errors.New("close failed")
	reader := newReader(&fakeIOCTL{}, &fakeNetlink{closeErr: closeErr})

	if err := reader.Close(); !errors.Is(err, closeErr) {
		t.Fatalf("close error = %v, want %v", err, closeErr)
	}
}

func TestReaderTreatsTypedNilNetlinkAsUnavailable(t *testing.T) {
	var netlink *fakeNetlink

	reader := newReader(&fakeIOCTL{data: []byte{0x18}}, netlink)
	got, err := reader.ModuleEEPROM("eth0")
	if err != nil {
		t.Fatalf("module eeprom: %v", err)
	}
	if !bytes.Equal(got, []byte{0x18}) {
		t.Fatalf("data = %x, want 18", got)
	}
}

func TestIsCMIS(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{name: "empty", data: nil},
		{name: "sff8472", data: []byte{0x03}},
		{name: "qsfp dd", data: []byte{0x18}, expected: true},
		{name: "osfp", data: []byte{0x19}, expected: true},
		{name: "qsfp cmis", data: []byte{0x1e}, expected: true},
		{name: "sfp dd", data: []byte{0x1f}, expected: true},
		{name: "sfp cmis", data: []byte{0x20}, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isCMIS(tt.data); got != tt.expected {
				t.Fatalf("isCMIS(%x) = %t, want %t", tt.data, got, tt.expected)
			}
		})
	}
}

type fakeIOCTL struct {
	data   []byte
	err    error
	closed bool
	calls  int
}

func (f *fakeIOCTL) ModuleEeprom(string) ([]byte, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.data, nil
}

func (f *fakeIOCTL) Close() {
	f.closed = true
}

type fakeNetlink struct {
	identifier      uint8
	identifierErr   error
	identifierCalls int
	data            []byte
	err             error
	eepromCalls     int
	closeErr        error
	closed          bool
}

type fakePresenceProbe struct {
	present  bool
	err      error
	closeErr error
	closed   bool
}

func (f *fakeNetlink) ModuleIdentifier(string) (uint8, error) {
	f.identifierCalls++
	if f.identifierErr != nil {
		return 0, f.identifierErr
	}
	return f.identifier, nil
}

func (f *fakeNetlink) ModuleEEPROM(string) ([]byte, error) {
	f.eepromCalls++
	if f.err != nil {
		return nil, f.err
	}
	return f.data, nil
}

func (f *fakeNetlink) Close() error {
	f.closed = true
	return f.closeErr
}

func (f *fakePresenceProbe) ModulePresent(string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.present, nil
}

func (f *fakePresenceProbe) Close() error {
	f.closed = true
	return f.closeErr
}
