package moduleeeprom

import (
	"bytes"
	"errors"
	"testing"
)

func TestReaderModuleEEPROMFallbacks(t *testing.T) {
	ioctlErr := errors.New("ioctl failed")
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
			netlink:  &fakeNetlink{data: []byte{0x18}},
			expected: []byte{0x03},
		},
		{
			name:     "cmis prefers netlink pages",
			ioctl:    &fakeIOCTL{data: []byte{0x18}},
			netlink:  &fakeNetlink{data: []byte{0x18, 0x01}},
			expected: []byte{0x18, 0x01},
		},
		{
			name:     "cmis falls back to ioctl when netlink fails",
			ioctl:    &fakeIOCTL{data: []byte{0x18}},
			netlink:  &fakeNetlink{err: netlinkErr},
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
			netlink:  &fakeNetlink{data: []byte{0x18}},
			expected: []byte{0x18},
		},
		{
			name:          "ioctl and netlink errors are joined",
			ioctl:         &fakeIOCTL{err: ioctlErr},
			netlink:       &fakeNetlink{err: netlinkErr},
			expectedError: netlinkErr,
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
}

func (f *fakeIOCTL) ModuleEeprom(string) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.data, nil
}

func (f *fakeIOCTL) Close() {
	f.closed = true
}

type fakeNetlink struct {
	data     []byte
	err      error
	closeErr error
	closed   bool
}

func (f *fakeNetlink) ModuleEEPROM(string) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.data, nil
}

func (f *fakeNetlink) Close() error {
	f.closed = true
	return f.closeErr
}
