package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"mikoafble/bluetooth"
)

var adapter = bluetooth.DefaultAdapter

var notifyChar bluetooth.Characteristic //notify characteristic

var connectedDevices = make(map[bluetooth.Address]bluetooth.Device)

func callAPI(path string, payload interface{}) ([]byte, error) {
	var resp *http.Response
	var err error
	if payload == nil {
		resp, err = http.Get("http://localhost:9000" + path)
	} else {
		body, _ := json.Marshal(payload)
		resp, err = http.Post("http://localhost:9000"+path, "application/json", bytes.NewBuffer(body))
	}
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	return buf.Bytes(), nil
}

func setupPeripheral() error {
	err := adapter.Enable()
	if err != nil {
		return err
	}

	//Connection handler
	adapter.SetConnectionHandler(func(device bluetooth.Device, connected bool) {
		if connected {
			log.Printf("Device connected: %s\n", device.Address.String())
			connectedDevices[device.Address] = device
		} else {
			log.Printf("Device disconnected: %s\n", device.Address)
			delete(connectedDevices, device.Address)
		}
	})

	// Write handler
	writeHandler := func(conn bluetooth.Connection, offset int, value []byte) {
		qs := string(value)
		fmt.Println("Write received:", qs)
		params, _ := url.ParseQuery(qs)
		cmd := params.Get("cmd")
		var resp []byte
		switch cmd {
		case "hello":
			resp, _ = callAPI("/hello", nil)
		case "user":
			resp, _ = callAPI("/user", map[string]string{"name": params.Get("name"), "age": params.Get("age")})
		case "greeting":
			resp, _ = callAPI("/greeting", map[string]string{"name": params.Get("name")})
		default:
			resp = []byte("invalid cmd")
		}

		notifyChar.Write(resp) // Send back via notification
	}

	// Define service and characteristics
	svc := bluetooth.Service{
		UUID: bluetooth.NewUUID([16]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0}),
		Characteristics: []bluetooth.CharacteristicConfig{
			{
				UUID:       bluetooth.NewUUID([16]byte{0xab, 0xcd, 0xef, 0x01, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0}),
				Flags:      bluetooth.CharacteristicWritePermission,
				WriteEvent: writeHandler,
			},
			{
				UUID:   bluetooth.NewUUID([16]byte{0xab, 0xcd, 0xef, 0x03, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0}),
				Flags:  bluetooth.CharacteristicNotifyPermission,
				Handle: &notifyChar, // Simpan pointer agar bisa kita gunakan nanti
			},
		},
	}

	// Tambahkan service ke adapter
	err = adapter.AddService(&svc)
	if err != nil {
		return err
	}

	// Mulai advertising
	adv := adapter.DefaultAdvertisement()
	adv.Configure(bluetooth.AdvertisementOptions{
		LocalName:    "GoBLE",
		ServiceUUIDs: []bluetooth.UUID{svc.UUID},
	})
	err = adv.Start()
	if err != nil {
		return err
	}

	log.Println("Advertising...")
	// address, _ := adapter.Address()
	// fmt.Println(address)
	// <-context.Background().Done()
	// adv.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v. Initiating shutdown...", sig)
		cancel()
	}()

	<-ctx.Done()
	log.Println("Shutting down BLE services")
	log.Println("Stopping advertisement...")
	if err := adv.Stop(); err != nil {
		log.Printf("Error stopping advertisement: %v\n", err)
	}

	log.Println("Advertisement stopped")
	log.Println("Disconnecting any active client connections...")
	for addr, dev := range connectedDevices {
		dev.Disconnect()
		delete(connectedDevices, addr)
	}
	return nil
}

func main() {
	if err := setupPeripheral(); err != nil {
		log.Fatal(err)
	}
}
