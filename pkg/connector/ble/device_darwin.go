package ble

import (
	"github.com/k-katsuda/ble"
	"github.com/k-katsuda/ble/darwin"
)

func newDevice() (ble.Device, error) {
	return darwin.NewDevice()
}
