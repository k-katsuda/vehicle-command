package ble

import (
	"github.com/k-katsuda/ble"
	"github.com/k-katsuda/ble/linux"
)

func newDevice() (ble.Device, error) {
	return linux.NewDevice()
}
