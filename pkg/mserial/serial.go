package mserial

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/sessions"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
	"github.com/tarm/serial"
)

// Measurement contains a simple X,Y,Z output from the IMU
type Measurement struct {
	X float64 `json:"X"`
	Y float64 `json:"Y"`
	Z float64 `json:"Z"`
}

var writerLock sync.Mutex

// parseSerialDevices parses through other serial devices, if enabled
/*
func parseSerialDevices(settingsData map[string]map[string]string) map[string]int {

	serialDevices, additionalSerialDevices := settingsData["Serial Devices"]
	var devices map[string]int

	if additionalSerialDevices {
		for deviceName, baudrateString := range serialDevices {
			deviceBaud, err := strconv.Atoi(baudrateString)
			if err != nil {
				log.Error().Msgf("Failed to convert given baudrate string to int. Found values: %s: %s", deviceName, baudrateString)
			} else {
				devices[deviceName] = deviceBaud
			}
		}
	}

	return devices
}*/

// openSerialPort will return a *serial.Port with the given arguments
func openSerialPort(deviceName string, baudrate int) (*serial.Port, error) {
	log.Info().Msgf("Opening serial device %s at baud %d", deviceName, baudrate)
	c := &serial.Config{Name: deviceName, Baud: baudrate, ReadTimeout: time.Second * 10}
	s, err := serial.OpenPort(c)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// loop reads serial data into the session
func loop(device *serial.Port, isWriter bool) {
	for {
		// Write to device if is necessary
		if isWriter {
			Pop(device)
		}

		err := readSerial(device)
		if err != nil {
			// The device is nil, break out of this read loop
			log.Error().Msg("Failed to read from serial port")
			log.Error().Msg(err.Error())
			return
		}
	}
}

// readSerial takes one line from the serial device and parses it into the session
func readSerial(device *serial.Port) error {
	response, err := read(device)
	// Return the read error, the device has gone offline
	if err != nil {
		return err
	}

	if response == nil {
		return nil
	}

	// Handle parse errors here instead of passing up
	err = parseJSON(response) // Parse serial data
	if err != nil {
		log.Error().Msg(err.Error())
		log.Error().Msgf("Could not parse from from serial device: \n\t%v", (response))
	}
	return nil
}

// read will continuously pull data from incoming serial
func read(serialDevice *serial.Port) (interface{}, error) {
	reader := bufio.NewReader(serialDevice)
	msg, _, err := reader.ReadLine()
	if err != nil {
		return nil, err
	}

	if len(msg) == 0 {
		return nil, nil
	}

	// Parse serial data
	var data interface{}
	json.Unmarshal(msg, &data)
	return data, nil
}

// write pushes out a message to the open serial port
func write(msg *Message) error {
	if msg.Device == nil {
		return fmt.Errorf("Serial port is not set, nothing to write to")
	}

	if len(msg.Text) == 0 {
		return fmt.Errorf("Empty message, not writing to serial")
	}

	writerLock.Lock()
	n, err := msg.Device.Write([]byte(msg.Text))
	writerLock.Unlock()
	if err != nil {
		return fmt.Errorf("Failed to write to serial port: %s", err.Error())
	}

	if msg.UUID == "" {
		log.Info().Msgf("Successfully wrote %s (%d bytes) to serial.", msg.Text, n)
	} else {
		log.Info().Msgf("[%s] Successfully wrote %s (%d bytes) to serial.", msg.UUID, msg.Text, n)
	}
	return nil
}

func parseGyros(name string, m Measurement) error {
	if name != "ACCELERATION" && name != "GYROSCOPE" && name != "MAGNETIC" {
		return fmt.Errorf("Measurement name %s not registered for input", name)
	}

	// Skip publishing values
	sessions.Set(fmt.Sprintf("gyros.%s.x", strings.ToLower(name)), m.X, false)
	sessions.Set(fmt.Sprintf("gyros.%s.y", strings.ToLower(name)), m.Y, false)
	sessions.Set(fmt.Sprintf("gyros.%s.z", strings.ToLower(name)), m.Z, false)
	return nil
}

func parseJSON(marshalledJSON interface{}) error {
	if marshalledJSON == nil {
		return fmt.Errorf("Marshalled JSON is nil")
	}

	data := marshalledJSON.(map[string]interface{})

	// Switch through various types of JSON data
	for key, value := range data {
		switch vv := value.(type) {
		case bool:
			sessions.Set(key, vv, true)
		case int:
			sessions.Set(key, vv, true)
		case float64:
			sessions.Set(key, vv, true)
		case string:
			sessions.Set(key, vv, true)
		case map[string]interface{}:
			var m Measurement
			err := mapstructure.Decode(value, &m)
			if err != nil {
				return err
			}
			err = parseGyros(key, m)
			if err != nil {
				return err
			}
		case []interface{}:
			log.Error().Msg(key + " is an array. Data: ")
			for i, u := range vv {
				fmt.Println(i, u)
			}
		case nil:
			break
		default:
			return fmt.Errorf("%s is of a type I don't know how to handle (%s: %s)", key, vv, value)
		}
	}
	return nil
}
