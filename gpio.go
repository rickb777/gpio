package gpio

import (
	"fmt"
	"time"

	"github.com/rickb777/gpio/sysfs"
)

type (
	// InputPin is the interface satisfied by GPIO input pins.
	InputPin interface {
		Read() (bool, error)
	}

	// InterruptPin is the interface satisfied by GPIO interrupt pins.
	InterruptPin interface {
		InputPin
		Wait(time.Duration) error
	}

	// OutputPin is the interface satisfied by GPIO output pins.
	OutputPin interface {
		Write(bool) error
	}
)

// Input initializes a GPIO input pin with the given pin number.
func Input(pinNumber int, activeLow bool) (InputPin, error) {
	pin, err := newPin(pinNumber, activeLow)
	if err != nil {
		return nil, err
	}

	err = sysfs.SetString(pin.dir.Join("direction"), "in")
	return pin, err
}

// Interrupt initializes a GPIO interrupt pin with the given pin number.
// The edge parameter must be "rising", "falling", or "both".
func Interrupt(pinNumber int, activeLow bool, edge string) (InterruptPin, error) {
	pin, err := newPin(pinNumber, activeLow)
	if err != nil {
		return nil, err
	}

	err = sysfs.SetString(pin.dir.Join("direction"), "in")
	if err != nil {
		return pin, err
	}

	err = sysfs.SetString(pin.dir.Join("edge"), edge)
	return pin, err
}

var gpioDirection = map[bool]string{true: "high", false: "low"}

// Output initializes a GPIO output pin with the given pin number
// and initial logical value.
func Output(pinNumber int, activeLow bool, initialValue bool) (OutputPin, error) {
	pin, err := newPin(pinNumber, activeLow)
	if err != nil {
		return nil, err
	}

	// Set direction based on initial *logical* value.
	direction := gpioDirection[initialValue != activeLow]
	err = sysfs.SetString(pin.dir.Join("direction"), direction)
	return pin, err
}

func pinDirectory(pinNumber int) (sysfs.Path, error) {
	const gpioDir sysfs.Path = "/sys/class/gpio"
	dir := gpioDir.Join(fmt.Sprintf("gpio%d/", pinNumber))
	tried := false
	for {
		exists, err := dir.IsExistingDirectory()
		if err != nil || exists {
			return dir, err
		}
		if tried {
			return dir, fmt.Errorf("failed to export GPIO directory %s", dir)
		}

		err = sysfs.SetString(gpioDir.Join("export"), fmt.Sprintf("%d", pinNumber))
		if err != nil {
			return dir, err
		}

		tried = true
		// Give udev rules a chance to execute on newly-created gpio%d directory.
		time.Sleep(time.Second)
	}
}
