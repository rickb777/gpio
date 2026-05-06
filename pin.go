package gpio

import (
	"fmt"
	"time"

	"github.com/rickb777/gpio/sysfs"
	"golang.org/x/sys/unix"
)

// Pin represents a GPIO pin.
type Pin struct {
	number int
	dir    sysfs.Path
	value  sysfs.Path
}

func newPin(pinNumber int, activeLow bool) (*Pin, error) {
	dir, err := pinDirectory(pinNumber)
	if err != nil {
		return nil, err
	}

	value := dir.Join("value")
	exists, err := value.IsExistingFile()
	if err != nil || !exists {
		return nil, err
	}

	err = sysfs.SetBool(dir.Join("active_low"), activeLow)
	if err != nil {
		return nil, err
	}

	return &Pin{number: pinNumber, dir: dir, value: value}, nil
}

func (pin *Pin) Read() (bool, error) {
	return sysfs.GetBool(pin.value)
}

func (pin *Pin) Write(value bool) error {
	return sysfs.SetBool(pin.value, value)
}

// Wait waits with the given timeout for a GPIO input pin to become active.
func (pin *Pin) Wait(timeout time.Duration) error {
	fd, err := unix.Open(string(pin.value), unix.O_NONBLOCK|unix.O_RDONLY, 0)
	if err != nil {
		return err
	}

	defer func() { _ = unix.Close(fd) }()

	// This must be big enough to read the entire value file (0 or 1 and newline).
	var valueBuf = make([]byte, 4)
	_, err = unix.Read(fd, valueBuf)
	// Return immediately if the value is already active.
	if err != nil || valueBuf[0] == '1' {
		return err
	}

	fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLPRI}}
	n, err := unix.Poll(fds, int(timeout/time.Millisecond))
	if err != nil {
		return err
	}

	switch n {
	case 1:
		return nil
	case 0:
		return TimeoutError{pin: pin, timeout: timeout}
	default:
		return fmt.Errorf("gpio%d.Select returned %d", pin.number, n)
	}
}

//-------------------------------------------------------------------------------------------------

// TimeoutError indicates that a Wait operation on a GPIO input pin timed out.
type TimeoutError struct {
	pin     *Pin
	timeout time.Duration
}

func (t TimeoutError) Error() string {
	return fmt.Sprintf("gpio%d.Wait timeout after %v", t.pin.number, t.timeout)
}
