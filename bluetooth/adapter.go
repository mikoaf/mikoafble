package bluetooth

import (
	"errors"
	"fmt"

	"github.com/godbus/dbus/v5"
)

const defaultAdapter = "hci0"

type Adapter struct {
	id                   string
	scanCancelChan       chan struct{}
	bus                  *dbus.Conn     //object at /
	bluez                dbus.BusObject //object at /org/bluez/hcix
	adapter              dbus.BusObject
	address              string
	defaultAdvertisement *Advertisement

	connectHandler func(device Device, connected bool)
}

func NewAdapter(id string) *Adapter {
	return &Adapter{
		id:             id,
		connectHandler: func(device Device, connected bool) {},
	}
}

var DefaultAdapter = NewAdapter(defaultAdapter)

func (a *Adapter) Enable() (err error) {
	bus, err := dbus.SystemBus()
	if err != nil {
		return err
	}

	a.bus = bus
	a.bluez = a.bus.Object("org.bluez", dbus.ObjectPath("/"))
	a.adapter = a.bus.Object("org.bluez", dbus.ObjectPath("/org/bluez/"+a.id))
	fmt.Println("Adapter path:", a.adapter.Path())
	addr, err := a.adapter.GetProperty("org.bluez.Adapter1.Address")
	// fmt.Println(addr)
	if err != nil {
		if err, ok := err.(dbus.Error); ok && err.Name == "org.freedesktop.DBus.Error.UnknownObject" {
			return fmt.Errorf("bluetooth: adapter %s does not exist", a.adapter.Path())
		}
		return fmt.Errorf("could not activate BlueZ adapter: %w", err)
	}
	addr.Store(&a.address)
	fmt.Printf("DBus Variant: %v\n", addr.Value())
	return nil
}

func (a *Adapter) Address() (MACAddress, error) {
	if a.address == "" {
		return MACAddress{}, errors.New("adapter not enabled")
	}
	mac, err := ParseMAC(a.address)
	if err != nil {
		return MACAddress{}, err
	}
	return MACAddress{MAC: mac}, nil
}
