//go:build linux
// +build linux

package serial

const (
	TCGETS     = 0x5401
	TCSETS     = 0x5402
	TCIOFLUSH  = 0x540B
	TIOCMGET   = 0x5415
	TIOCMBIS   = 0x5416
	FIONREAD   = 0x541B
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
