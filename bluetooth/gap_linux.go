package bluetooth

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/prop"
)

const (
	// Match rule constants for D-Bus signals.
	//
	// See [DBusPropertiesLink] for more information.

	dbusPropertiesChangedInterfaceName = 0
	dbusPropertiesChangedDictionary    = 1
	dbusPropertiesChangedInvalidated   = 2

	dbusInterfacesAddedDictionary = 1

	dbusSignalInterfacesAdded   = "org.freedesktop.DBus.ObjectManager.InterfacesAdded"
	dbusSignalPropertiesChanged = "org.freedesktop.DBus.Properties.PropertiesChanged"

	bluezDevice1Interface = "org.bluez.Device1"
	bluezDevice1Address   = "Address"
	bluezDevice1Connected = "Connected"
)

var errAdvertisementNotStarted = errors.New("bluetooth: advertisement is not started")
var errAdvertisementAlreadyStarted = errors.New("bluetooth: advertisement is already started")
var errAdaptorNotPowered = errors.New("bluetooth: adaptor is not powered")

var advertisementID uint64

var (
	// See [DBusPropertiesLink] for more information.
	matchOptionsPropertiesChanged = []dbus.MatchOption{dbus.WithMatchInterface("org.freedesktop.DBus.Properties"),
		dbus.WithMatchMember("PropertiesChanged"),
		dbus.WithMatchArg(dbusPropertiesChangedInterfaceName, "org.bluez.Device1")}

	// See [DBusObjectManagerLink] for more information.
	matchOptionsInterfacesAdded = []dbus.MatchOption{dbus.WithMatchInterface("org.freedesktop.DBus.ObjectManager"),
		dbus.WithMatchMember("InterfacesAdded")}
)

type Address struct {
	MACAddress
}

type Device struct {
	Address Address

	device  dbus.BusObject
	adapter *Adapter
}

type Advertisement struct {
	adapter    *Adapter
	properties *prop.Properties
	path       dbus.ObjectPath
	started    bool

	//D-bus signals
	sigCh chan *dbus.Signal
}

func (a *Adapter) DefaultAdvertisement() *Advertisement {
	if a.defaultAdvertisement == nil {
		a.defaultAdvertisement = &Advertisement{
			adapter: a,
		}
	}
	return a.defaultAdvertisement
}

func (a *Advertisement) Configure(options AdvertisementOptions) error {
	if a.started {
		return errAdvertisementAlreadyStarted
	}

	var serviceUUIDs []string
	for _, uuid := range options.ServiceUUIDs {
		serviceUUIDs = append(serviceUUIDs, uuid.String())
	}
	var serviceData = make(map[string]interface{})
	for _, element := range options.ServiceData {
		serviceData[element.UUID.String()] = element.Data
	}

	manufacturerData := map[uint16]any{}
	for _, element := range options.ManufacturerData {
		manufacturerData[element.CompanyID] = element.Data
	}

	id := atomic.AddUint64(&advertisementID, 1)
	a.path = dbus.ObjectPath(fmt.Sprintf("/org/tinygo/bluetooth/advertisement%d", id))
	propsSpec := map[string]map[string]*prop.Prop{
		"org.bluez.LEAdvertisement1": {
			"Type":             {Value: "broadcast"},
			"ServiceUUIDs":     {Value: serviceUUIDs},
			"ManufacturerData": {Value: manufacturerData},
			"LocalName":        {Value: options.LocalName},
			"ServiceData":      {Value: serviceData, Writable: true},
			"Timeout":          {Value: uint16(0)},
		},
	}

	props, err := prop.Export(a.adapter.bus, a.path, propsSpec)
	if err != nil {
		return err
	}
	a.properties = props

	if options.LocalName != "" {
		call := a.adapter.adapter.Call("org.freedesktop.DBus.Properties.Set", 0, "org.bluez.Adapter1", "Alias", dbus.MakeVariant((options.LocalName)))
		if call.Err != nil {
			return fmt.Errorf("set adapter alias: %w", call.Err)
		}
	}

	return nil
}

func (a *Advertisement) handleDBusSignals() {
	for {
		select {
		case sig, ok := <-a.sigCh:
			if !ok {
				return // channel closed
			}

			device := Device{
				device:  a.adapter.bus.Object("org.bluez", sig.Path),
				adapter: a.adapter,
			}

			switch sig.Name {
			case dbusSignalInterfacesAdded:
				interfaces := sig.Body[dbusInterfacesAddedDictionary].(map[string]map[string]dbus.Variant)

				props, ok := interfaces[bluezDevice1Interface]
				if !ok {
					continue
				}

				if err := device.parseProperties(&props); err != nil {
					continue
				}

				if connected, ok := props[bluezDevice1Connected].Value().(bool); ok {
					a.adapter.connectHandler(device, connected)
				}
			case dbusSignalPropertiesChanged:
				// Skip any signals that are not the Device1 interface.
				if interfaceName, ok := sig.Body[dbusPropertiesChangedInterfaceName].(string); !ok || interfaceName != bluezDevice1Interface {
					continue
				}

				// Get all changed properties and skip any signals that are not
				// compliant with the Device1 interface.
				changes, ok := sig.Body[dbusPropertiesChangedDictionary].(map[string]dbus.Variant)
				if !ok {
					continue
				}

				// Call the connect handler if the Connected property has changed.
				if connected, ok := changes[bluezDevice1Connected].Value().(bool); ok {
					// The only property received is the changed property "Connected",
					// so we have to get the other properties from D-Bus.
					var props map[string]dbus.Variant
					if err := device.device.Call("org.freedesktop.DBus.Properties.GetAll",
						0,
						bluezDevice1Interface).Store(&props); err != nil {
						continue
					}

					if err := device.parseProperties(&props); err != nil {
						continue
					}

					a.adapter.connectHandler(device, connected)
				}
			}
		}
	}
}

// Start advertisement. May only be called after it has been configured.
func (a *Advertisement) Start() error {
	// Register our advertisement object to start advertising.
	err := a.adapter.adapter.Call("org.bluez.LEAdvertisingManager1.RegisterAdvertisement", 0, a.path, map[string]interface{}{}).Err
	if err != nil {
		if err, ok := err.(dbus.Error); ok && err.Name == "org.bluez.Error.AlreadyExists" {
			return errAdvertisementAlreadyStarted
		}
		return fmt.Errorf("bluetooth: could not start advertisement: %w", err)
	}

	if a.adapter.connectHandler != nil {
		a.sigCh = make(chan *dbus.Signal)
		a.adapter.bus.Signal(a.sigCh)

		if err := a.adapter.bus.AddMatchSignal(matchOptionsPropertiesChanged...); err != nil {
			return fmt.Errorf("bluetooth: add dbus match signal: PropertiesChanged: %w", err)
		}

		if err := a.adapter.bus.AddMatchSignal(matchOptionsInterfacesAdded...); err != nil {
			return fmt.Errorf("bluetooth: add dbus match signal: InterfacesAdded: %w", err)
		}

		go a.handleDBusSignals()
	}

	// Make us discoverable.
	err = a.adapter.adapter.SetProperty("org.bluez.Adapter1.Discoverable", dbus.MakeVariant(true))
	if err != nil {
		return fmt.Errorf("bluetooth: could not start advertisement: %w", err)
	}
	a.started = true
	return nil
}

// Stop advertisement. May only be called after it has been started.
func (a *Advertisement) Stop() error {
	err := a.adapter.adapter.Call("org.bluez.LEAdvertisingManager1.UnregisterAdvertisement", 0, a.path).Err
	if err != nil {
		if err, ok := err.(dbus.Error); ok && err.Name == "org.bluez.Error.DoesNotExist" {
			return errAdvertisementNotStarted
		}
		return fmt.Errorf("bluetooth: could not stop advertisement: %w", err)
	}
	a.started = false

	if a.sigCh != nil {
		defer close(a.sigCh)
		if err := a.adapter.bus.RemoveMatchSignal(matchOptionsPropertiesChanged...); err != nil {
			return fmt.Errorf("bluetooth: remove dbus match signal: PropertiesChanged: %w", err)
		}
		if err := a.adapter.bus.RemoveMatchSignal(matchOptionsInterfacesAdded...); err != nil {
			return fmt.Errorf("bluetooth: remove dbus match signal: InterfacesAdded: %w", err)
		}
		a.adapter.bus.RemoveSignal(a.sigCh)
	}
	return nil
}

func (d *Device) parseProperties(props *map[string]dbus.Variant) error {
	for prop, v := range *props {
		switch prop {
		case bluezDevice1Address:
			if addrStr, ok := v.Value().(string); ok {
				mac, err := ParseMAC(addrStr)
				if err != nil {
					return fmt.Errorf("ParseMAC: %w", err)
				}
				d.Address = Address{MACAddress: MACAddress{MAC: mac}}
			}
		}
	}

	return nil
}
