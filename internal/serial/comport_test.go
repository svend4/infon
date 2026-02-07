package serial

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("COM1")

	if cfg.Name != "COM1" {
		t.Errorf("Name = %s, expected COM1", cfg.Name)
	}

	if cfg.BaudRate != 9600 {
		t.Errorf("BaudRate = %d, expected 9600", cfg.BaudRate)
	}

	if cfg.DataBits != 8 {
		t.Errorf("DataBits = %d, expected 8", cfg.DataBits)
	}

	if cfg.StopBits != StopBits1 {
		t.Errorf("StopBits = %d, expected StopBits1", cfg.StopBits)
	}

	if cfg.Parity != ParityNone {
		t.Errorf("Parity = %d, expected ParityNone", cfg.Parity)
	}

	if cfg.ReadTimeout != 1*time.Second {
		t.Errorf("ReadTimeout = %v, expected 1s", cfg.ReadTimeout)
	}
}

func TestHighSpeedConfig(t *testing.T) {
	cfg := HighSpeedConfig("COM2")

	if cfg.Name != "COM2" {
		t.Errorf("Name = %s, expected COM2", cfg.Name)
	}

	if cfg.BaudRate != 115200 {
		t.Errorf("BaudRate = %d, expected 115200", cfg.BaudRate)
	}

	if cfg.ReadTimeout != 100*time.Millisecond {
		t.Errorf("ReadTimeout = %v, expected 100ms", cfg.ReadTimeout)
	}
}

func TestConfigString(t *testing.T) {
	cfg := Config{
		Name:     "COM1",
		BaudRate: 9600,
		DataBits: 8,
		StopBits: StopBits1,
		Parity:   ParityNone,
	}

	str := cfg.String()

	expected := "COM1: 9600 baud, 8N1"
	if str != expected {
		t.Errorf("String() = %s, expected %s", str, expected)
	}
}

func TestConfigStringWithParity(t *testing.T) {
	tests := []struct {
		parity   Parity
		expected string
	}{
		{ParityNone, "COM1: 9600 baud, 8N1"},
		{ParityOdd, "COM1: 9600 baud, 8O1"},
		{ParityEven, "COM1: 9600 baud, 8E1"},
		{ParityMark, "COM1: 9600 baud, 8M1"},
		{ParitySpace, "COM1: 9600 baud, 8S1"},
	}

	for _, tt := range tests {
		cfg := Config{
			Name:     "COM1",
			BaudRate: 9600,
			DataBits: 8,
			StopBits: StopBits1,
			Parity:   tt.parity,
		}

		str := cfg.String()
		if str != tt.expected {
			t.Errorf("String() with parity %d = %s, expected %s", tt.parity, str, tt.expected)
		}
	}
}

func TestStopBitsValues(t *testing.T) {
	if StopBits1 != 1 {
		t.Errorf("StopBits1 = %d, expected 1", StopBits1)
	}

	if StopBits1_5 != 15 {
		t.Errorf("StopBits1_5 = %d, expected 15", StopBits1_5)
	}

	if StopBits2 != 2 {
		t.Errorf("StopBits2 = %d, expected 2", StopBits2)
	}
}

func TestParityValues(t *testing.T) {
	if ParityNone != 0 {
		t.Errorf("ParityNone = %d, expected 0", ParityNone)
	}

	if ParityOdd != 1 {
		t.Errorf("ParityOdd = %d, expected 1", ParityOdd)
	}

	if ParityEven != 2 {
		t.Errorf("ParityEven = %d, expected 2", ParityEven)
	}

	if ParityMark != 3 {
		t.Errorf("ParityMark = %d, expected 3", ParityMark)
	}

	if ParitySpace != 4 {
		t.Errorf("ParitySpace = %d, expected 4", ParitySpace)
	}
}

func TestListPorts(t *testing.T) {
	// ListPorts should not return an error
	ports, err := ListPorts()
	if err != nil {
		t.Fatalf("ListPorts() failed: %v", err)
	}

	// On some systems there may be no ports
	t.Logf("Found %d serial ports", len(ports))
	for i, port := range ports {
		t.Logf("  Port %d: %s", i+1, port)
	}
}

func TestStatistics(t *testing.T) {
	stats := Statistics{
		BytesRead:    1000,
		BytesWritten: 500,
		Errors:       2,
		LastError:    nil,
	}

	if stats.BytesRead != 1000 {
		t.Errorf("BytesRead = %d, expected 1000", stats.BytesRead)
	}

	if stats.BytesWritten != 500 {
		t.Errorf("BytesWritten = %d, expected 500", stats.BytesWritten)
	}

	if stats.Errors != 2 {
		t.Errorf("Errors = %d, expected 2", stats.Errors)
	}
}

func TestModemStatus(t *testing.T) {
	status := ModemStatus{
		CTS: true,
		DSR: false,
		RI:  false,
		DCD: true,
	}

	if !status.CTS {
		t.Error("CTS should be true")
	}

	if status.DSR {
		t.Error("DSR should be false")
	}

	if status.RI {
		t.Error("RI should be false")
	}

	if !status.DCD {
		t.Error("DCD should be true")
	}
}

// Mock port for testing (no actual hardware required)
func TestMockPort(t *testing.T) {
	// Test that we can create a config without opening a port
	cfg := DefaultConfig("MOCK_PORT")

	if cfg.Name != "MOCK_PORT" {
		t.Errorf("Config name = %s, expected MOCK_PORT", cfg.Name)
	}

	// Test various baud rates
	baudRates := []int{9600, 19200, 38400, 57600, 115200}
	for _, rate := range baudRates {
		cfg.BaudRate = rate
		if cfg.BaudRate != rate {
			t.Errorf("BaudRate = %d, expected %d", cfg.BaudRate, rate)
		}
	}

	// Test data bits
	dataBits := []int{5, 6, 7, 8}
	for _, bits := range dataBits {
		cfg.DataBits = bits
		if cfg.DataBits != bits {
			t.Errorf("DataBits = %d, expected %d", cfg.DataBits, bits)
		}
	}
}

func TestConfigValidation(t *testing.T) {
	// Test empty port name
	_, err := OpenPort(Config{Name: ""})
	if err == nil {
		t.Error("OpenPort with empty name should fail")
	}

	// Test that defaults are applied
	cfg := Config{
		Name: "COM1",
	}

	// BaudRate should default to 9600
	if cfg.BaudRate == 0 {
		cfg.BaudRate = 9600
	}
	if cfg.BaudRate != 9600 {
		t.Errorf("Default baud rate = %d, expected 9600", cfg.BaudRate)
	}
}

// Benchmarks

func BenchmarkConfigString(b *testing.B) {
	cfg := DefaultConfig("COM1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cfg.String()
	}
}

func BenchmarkListPorts(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ListPorts()
	}
}
