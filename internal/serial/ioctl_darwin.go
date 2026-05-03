//go:build darwin
// +build darwin

package serial

// macOS/Darwin termios constants
// These are from <sys/ttycom.h> and <sys/ioctl.h> on macOS
const (
	TCGETS    = 0x40000000 | ((8 & 0x1fff) << 16) | ('t' << 8) | 19  // TIOCGETA
	TCSETS    = 0x80000000 | ((8 & 0x1fff) << 16) | ('t' << 8) | 20  // TIOCSETA
	TCIOFLUSH = 0x20007466                                             // TIOCFLUSH
	TIOCMGET  = 0x4004746a                                             // TIOCMGET
	TIOCMBIS  = 0x8004746c                                             // TIOCMBIS
	FIONREAD  = 0x4004667f                                             // FIONREAD
)

func getTermios() uint {
	return TCGETS
}

func setTermios() uint {
	return TCSETS
}

func getTCIOFLUSH() uint {
	return TCIOFLUSH
}

func getTIOCMGET() uint {
	return TIOCMGET
}

func getTIOCMBIS() uint {
	return TIOCMBIS
}

func getFIONREAD() uint {
	return FIONREAD
}
