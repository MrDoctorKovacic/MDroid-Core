// Package bluetooth is a rudimentary interface between MDroid-Core and underlying BT dbus
package bluetooth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/formatting"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/gosimple/slug"
	"github.com/rs/zerolog/log"
)

// Regex expressions for parsing dbus output
var (
	BluetoothAddress string
	replySerialRegex *regexp.Regexp
	findStringRegex  *regexp.Regexp
	cleanRegex       *regexp.Regexp
)

func init() {
	replySerialRegex = regexp.MustCompile(`(.*reply_serial=2\n\s*variant\s*)array`)
	findStringRegex = regexp.MustCompile(`string\s"(.*)"|uint32\s(\d)+`)
	cleanRegex = regexp.MustCompile(`(string|uint32|\")+`)
}

// Setup bluetooth with address
func Setup(configAddr *map[string]string) {
	configMap := *configAddr
	bluetoothAddress, usingBluetooth := configMap["BLUETOOTH_ADDRESS"]
	if usingBluetooth {
		EnableAutoRefresh()
		SetAddress(bluetoothAddress)
		BluetoothAddress = bluetoothAddress
	}
	BluetoothAddress = ""
}

// Parse the variant output from DBus into map of string
func cleanDBusOutput(output string) map[string]string {
	outputMap := make(map[string]string, 0)

	// Start regex replacing for important values
	s := replySerialRegex.ReplaceAllString(output, "")
	outputArray := findStringRegex.FindAllString(s, -1)

	if outputArray == nil {
		log.Error().Msg("Error parsing dbus output")
	}

	var (
		key    string
		invert = 0
	)
	// The regex should cut things down to an alternating key:value after being trimmed
	// We add these to the map, and add a "Meta" key when it would normally be empty (as the first in the array)
	for i, value := range outputArray {
		newValue := strings.TrimSpace(cleanRegex.ReplaceAllString(value, ""))
		// Some devices have this meta value as the first entry (iOS mainly)
		// we should swap key/value pairs if so
		if i == 0 && (newValue == "Item" || newValue == "playing" || newValue == "paused") {
			invert = 1
			key = "Meta"
		}

		// Define key or insert into map if defined
		if i%2 == invert {
			key = newValue
		} else {
			outputMap[key] = newValue
		}
	}

	return outputMap
}

// EnableAutoRefresh continously refreshes bluetooth media devices
func EnableAutoRefresh() {
	log.Info().Msg("Enabling auto refresh of BT address")
	go startAutoRefresh()
}

// startAutoRefresh will begin go routine for refreshing bt media device address
func startAutoRefresh() {
	for {
		getConnectedAddress()
		time.Sleep(1000 * time.Millisecond)
	}
}

// ForceRefresh to immediately reload bt address
func ForceRefresh(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Forcing refresh of BT address")
	go getConnectedAddress()
}

// getConnectedAddress will find and replace the playing media device
// this should be run continuously to check for changes in connection
func getConnectedAddress() string {
	args := "busctl tree org.bluez | grep /fd | head -n 1 | sed -n 's/.*\\/org\\/bluez\\/hci0\\/dev_\\(.*\\)\\/.*/\\1/p'"
	out, err := exec.Command("bash", "-c", args).Output()

	if err != nil {
		log.Error().Msg(err.Error())
		return err.Error()
	}

	// Use new device if found
	newAddress := strings.TrimSpace(string(out))
	if newAddress != "" && BluetoothAddress != newAddress {
		log.Info().Msg("Found new connected media device with address: " + newAddress)
		SetAddress(newAddress)
	}

	return string(out)
}

// SetAddress makes address given in args available to all dbus functions
func SetAddress(address string) {
	// Format address for dbus
	if address != "" {
		BluetoothAddress = strings.Replace(strings.TrimSpace(address), ":", "_", -1)
		log.Info().Msg("Now routing Bluetooth commands to " + BluetoothAddress)

		// Set new address to persist in settings file
		settings.Set("CONFIG", "BLUETOOTH_ADDRESS", BluetoothAddress)
	}
}

// SendDBusCommand used as a general BT control function for these endpoints
func SendDBusCommand(args []string, hideOutput bool) (string, bool) {
	if BluetoothAddress == "" {
		log.Warn().Msg("No valid BT Address to run command")
		return "No valid BT Address to run command", false
	}

	// Fill in the meta nonsense
	args = append([]string{"--system", "--type=method_call", "--dest=org.bluez"}, args...)
	var stderr bytes.Buffer
	var out bytes.Buffer
	cmd := exec.Command("dbus-send", args...)
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Error().Msg(err.Error())
		log.Error().Msg(stderr.String())
		return stderr.String(), false
	}

	if !hideOutput {
		log.Info().Msg(out.String())
	}

	return out.String(), true
}

// Connect bluetooth device
func Connect(w http.ResponseWriter, r *http.Request) {
	//go SendDBusCommand([]string{"/org/bluez/hci0/dev_" + BluetoothAddress, "org.bluez.Device1.Connect"}, false)

	var stderr bytes.Buffer
	var out bytes.Buffer
	cmd := exec.Command("/bin/sh", "/home/pi/bluetooth/connect.sh")
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Error().Msg(err.Error())
		log.Error().Msg(stderr.String())
	}

	json.NewEncoder(w).Encode(formatting.JSONResponse{Output: "OK", Status: "success", OK: true})
}

// Disconnect bluetooth device
func Disconnect(w http.ResponseWriter, r *http.Request) {
	//go SendDBusCommand([]string{"/org/bluez/hci0/dev_" + BluetoothAddress, "org.bluez.Device1.Disconnect"}, false)

	var stderr bytes.Buffer
	var out bytes.Buffer
	cmd := exec.Command("/bin/sh", "/home/pi/bluetooth/disconnect.sh")
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Error().Msg(err.Error())
		log.Error().Msg(stderr.String())
	}

	json.NewEncoder(w).Encode(formatting.JSONResponse{Output: "OK", Status: "success", OK: true})
}

// GetDeviceInfo attempts to get metadata about connected device
func GetDeviceInfo(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Getting device info...")
	result, ok := SendDBusCommand([]string{"/org/bluez/hci0/dev_" + BluetoothAddress + "/player0", "org.freedesktop.DBus.Properties.Get", "string:org.bluez.MediaPlayer1", "string:Status"}, true)
	if !ok {
		json.NewEncoder(w).Encode(formatting.JSONResponse{Output: "Error getting device info", Status: "fail", OK: false})
		return
	}
	json.NewEncoder(w).Encode(formatting.JSONResponse{Output: cleanDBusOutput(result), Status: "success", OK: true})
}

// GetMediaInfo attempts to get metadata about current track
func GetMediaInfo(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Getting device info...")

	result, ok := SendDBusCommand([]string{"/org/bluez/hci0/dev_" + BluetoothAddress + "/player0", "org.freedesktop.DBus.Properties.Get", "string:org.bluez.MediaPlayer1", "string:Status"}, true)
	if !ok {
		json.NewEncoder(w).Encode(formatting.JSONResponse{Output: "Error getting media info", Status: "fail", OK: false})
		return
	}
	deviceStatus := cleanDBusOutput(result)

	log.Info().Msg("Getting media info...")
	result, ok = SendDBusCommand([]string{"/org/bluez/hci0/dev_" + BluetoothAddress + "/player0", "org.freedesktop.DBus.Properties.Get", "string:org.bluez.MediaPlayer1", "string:Track"}, true)
	if !ok {
		json.NewEncoder(w).Encode(formatting.JSONResponse{Output: "Error getting media info", Status: "fail", OK: false})
		return
	}

	// Append device status to media info
	cleanResult := cleanDBusOutput(result)
	cleanResult["Status"] = deviceStatus["Meta"]

	// Append Album / Artwork slug if both exist
	album, albumOK := cleanResult["Album"]
	artist, artistOK := cleanResult["Artist"]
	if albumOK && artistOK {
		cleanResult["Album_Artwork"] = slug.Make(artist) + "/" + slug.Make(album) + ".jpg"
	}

	// Echo back all info
	json.NewEncoder(w).Encode(formatting.JSONResponse{Output: cleanResult, Status: "success", OK: true})
}

// Prev skips to previous track
func Prev(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Going to previous track...")
	go SendDBusCommand([]string{"/org/bluez/hci0/dev_" + BluetoothAddress + "/player0", "org.bluez.MediaPlayer1.Previous"}, false)
	json.NewEncoder(w).Encode(formatting.JSONResponse{Output: "OK", Status: "success", OK: true})
}

// Next skips to next track
func Next(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Going to next track...")
	go SendDBusCommand([]string{"/org/bluez/hci0/dev_" + BluetoothAddress + "/player0", "org.bluez.MediaPlayer1.Next"}, false)
	json.NewEncoder(w).Encode(formatting.JSONResponse{Output: "OK", Status: "success", OK: true})
}

// Play attempts to play bluetooth media
func Play(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Attempting to play media...")
	go SendDBusCommand([]string{"/org/bluez/hci0/dev_" + BluetoothAddress + "/player0", "org.bluez.MediaPlayer1.Play"}, false)
	json.NewEncoder(w).Encode(formatting.JSONResponse{Output: "OK", Status: "success", OK: true})
}

// Pause attempts to pause bluetooth media
func Pause(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Attempting to pause media...")
	go SendDBusCommand([]string{"/org/bluez/hci0/dev_" + BluetoothAddress + "/player0", "org.bluez.MediaPlayer1.Pause"}, false)
	json.NewEncoder(w).Encode(formatting.JSONResponse{Output: "OK", Status: "success", OK: true})
}
