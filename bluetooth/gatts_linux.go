package bluetooth

import (
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/prop"
)

var serviceID uint64

type Characteristic struct {
	char        *blueZChar
	permissions CharacteristicPermissions
}

type blueZChar struct {
	props      *prop.Properties
	writeEvent func(client Connection, offset int, value []byte)
}

type objectManager struct {
	objects map[dbus.ObjectPath]map[string]map[string]*prop.Prop
}

func (om *objectManager) GetManagedObjects() (map[dbus.ObjectPath]map[string]map[string]dbus.Variant, *dbus.Error) {
	objects := map[dbus.ObjectPath]map[string]map[string]dbus.Variant{}
	for path, object := range om.objects {
		obj := make(map[string]map[string]dbus.Variant)
		objects[path] = obj
		for iface, props := range object {
			ifaceObj := make(map[string]dbus.Variant)
			obj[iface] = ifaceObj
			for k, v := range props {
				ifaceObj[k] = dbus.MakeVariant(v.Value)
			}
		}
	}
	return objects, nil
}

func (a *Adapter) AddService(s *Service) error {
	id := atomic.AddUint64(&serviceID, 1)
	path := dbus.ObjectPath(fmt.Sprintf("/org/nbable/bluetooth/service%d", id))

	objects := map[dbus.ObjectPath]map[string]map[string]*prop.Prop{}

	serviceSpec := map[string]map[string]*prop.Prop{
		"org.bluez.GattService1": {
			"UUID":    {Value: s.UUID.String()},
			"Primary": {Value: true},
		},
	}
	objects[path] = serviceSpec

	for i, char := range s.Characteristics {
		bluzCharFlags := []string{
			"broadcast",              //bit 0
			"read",                   //bit 1
			"write-without-response", //bit 2
			"write",                  //bit 3
			"notify",                 //bit 4
			"indicate",               //bit 5
		}
		var flags []string
		for i := 0; i < len(bluzCharFlags); i++ {
			if (char.Flags>>i)&1 != 0 {
				flags = append(flags, bluzCharFlags[i])
			}
		}

		charPath := path + dbus.ObjectPath("/char"+strconv.Itoa(i))
		propSpec := map[string]map[string]*prop.Prop{
			"org.bluez.GattCharacteristic1": {
				"UUID":    {Value: char.UUID.String()},
				"Service": {Value: path},
				"Flags":   {Value: flags},
				"Value":   {Value: char.Value, Writable: true, Emit: prop.EmitTrue},
			},
		}

		objects[charPath] = propSpec
		props, err := prop.Export(a.bus, charPath, propSpec)
		if err != nil {
			return err
		}

		obj := &blueZChar{
			props:      props,
			writeEvent: char.WriteEvent,
		}

		err = a.bus.Export(obj, charPath, "org.bluez.GattCharacteristic1")
		if err != nil {
			return err
		}

		if char.Handle != nil {
			char.Handle.permissions = char.Flags
			char.Handle.char = obj
		}
	}

	om := &objectManager{
		objects: objects,
	}
	err := a.bus.Export(om, path, "org.freedesktop.DBus.ObjectManager")
	if err != nil {
		return err
	}
	return a.adapter.Call("org.bluez.GattManager1.RegisterApplication", 0, path, map[string]dbus.Variant(nil)).Err
}

func (c *Characteristic) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil //nothing to do
	}

	if c.char.writeEvent != nil {
		c.char.writeEvent(0, 0, p)
	}
	err = c.char.props.Set("org.bluez.GattCharacteristic1", "Value", dbus.MakeVariant(p)) //gatt error check
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (c *blueZChar) ReadValue(options map[string]dbus.Variant) ([]byte, *dbus.Error) {
	value := c.props.GetMust("org.bluez.GattCharacteristic1", "Value").([]byte)
	return value, nil
}

func (c *blueZChar) WriteValue(value []byte, options map[string]dbus.Variant) *dbus.Error {
	if c.writeEvent != nil {
		client := Connection(0)
		offset, _ := options["offset"].Value().(uint16)
		c.writeEvent(client, int(offset), value)
	}
	return nil
}
