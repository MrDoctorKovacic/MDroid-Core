package main

import (
	"fmt"
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/format"
	"github.com/MrDoctorKovacic/MDroid-Core/mserial"
	"github.com/MrDoctorKovacic/MDroid-Core/sessions"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/rs/zerolog/log"
)

// Define temporary holding struct for power values
type power struct {
	isOn     bool
	target   string
	settings settingType
	errors   errorType
}

type settingType struct {
	component string
	name      string
}

type errorType struct {
	target error
	on     error
}

// Read the target action based on current ACC Power value
var (
	_wireless = power{settings: settingType{component: "WIRELESS", name: "POWER"}}
	_angel    = power{settings: settingType{component: "ANGEL_EYES", name: "POWER"}}
	_tablet   = power{settings: settingType{component: "TABLET", name: "POWER"}}
	_board    = power{settings: settingType{component: "BOARD", name: "POWER"}}
)

// Evaluates if the angel eyes should be on, and then passes that struct along as generic power module
func evalAngelEyesPower(keyIsIn string) {
	_angel.isOn, _angel.errors.on = sessions.GetBool("ANGEL_EYES_POWER")
	_angel.target, _angel.errors.target = settings.Get(_angel.settings.component, _angel.settings.name)
	lightSensor := sessions.GetBoolDefault("LIGHT_SENSOR_ON", false)

	shouldTrigger := !lightSensor && keyIsIn != "FALSE"

	// Pass angel module to generic power trigger
	genericPowerTrigger(shouldTrigger, "Angel", &_angel)
}

// Evaluates if the video boards should be on, and then passes that struct along as generic power module
func evalVideoPower(keyIsIn string, accOn bool, wifiOn bool) {
	_board.isOn, _board.errors.on = sessions.GetBool("BOARD_POWER")
	_board.target, _board.errors.target = settings.Get(_board.settings.component, _board.settings.name)

	shouldTrigger := accOn && !wifiOn || wifiOn && keyIsIn != "FALSE"

	// Pass angel module to generic power trigger
	genericPowerTrigger(shouldTrigger, "Board", &_board)
}

// Evaluates if the tablet should be on, and then passes that struct along as generic power module
func evalTabletPower(keyIsIn string, accOn bool, wifiOn bool) {
	_tablet.isOn, _tablet.errors.on = sessions.GetBool("TABLET_POWER")
	_tablet.target, _tablet.errors.target = settings.Get(_tablet.settings.component, _tablet.settings.name)

	shouldTrigger := accOn && !wifiOn || wifiOn && keyIsIn != "FALSE"

	// Pass angel module to generic power trigger
	genericPowerTrigger(shouldTrigger, "Tablet", &_tablet)
}

// Evaluates if the wireless boards should be on, and then passes that struct along as generic power module
func evalWirelessPower(keyIsIn string, accOn bool, wifiOn bool) {
	_wireless.isOn, _wireless.errors.on = sessions.GetBool("WIRELESS_POWER")
	_wireless.target, _wireless.errors.target = settings.Get(_wireless.settings.component, _wireless.settings.name)

	// Wireless is most likely supposed to be on, only one case where it should not be
	shouldTrigger := true
	if wifiOn && keyIsIn == "FALSE" {
		shouldTrigger = false
	}

	// Pass wireless module to generic power trigger
	genericPowerTrigger(shouldTrigger, "Wireless", &_wireless)
}

// Error check against module's status fetches, then check if we're powering on or off
func genericPowerTrigger(shouldBeOn bool, name string, module *power) {
	// Handle error in fetches
	if module.errors.target != nil {
		log.Error().Msg(fmt.Sprintf("Setting Error: %s", module.errors.target.Error()))
		if module.settings.component != "" && module.settings.name != "" {
			log.Error().Msg(fmt.Sprintf("Setting read error for %s. Resetting to AUTO", name))
			settings.Set(module.settings.component, module.settings.name, "AUTO")
		}
		return
	}
	if module.errors.on != nil {
		log.Debug().Msg(fmt.Sprintf("Session Error: %s", module.errors.on.Error()))
		return
	}

	// Evaluate power target with trigger and settings info
	if (module.target == "AUTO" && !module.isOn && shouldBeOn) || (module.target == "ON" && !module.isOn) {
		log.Info().Msg(fmt.Sprintf("Powering on %s, because target is %s", name, module.target))
		mserial.Push(mserial.Writer, fmt.Sprintf("powerOn%s", name))
	} else if (module.target == "AUTO" && module.isOn && !shouldBeOn) || (module.target == "OFF" && module.isOn) {
		log.Info().Msg(fmt.Sprintf("Powering off %s, because target is %s", name, module.target))
		gracefulShutdown(name)
	}
}

// Some shutdowns are more complicated than others, ensure we shut down safely
func gracefulShutdown(name string) {
	serialCommand := fmt.Sprintf("powerOff%s", name)

	if name == "Board" || name == "Wireless" {
		sendServiceCommand(format.Name(name), "shutdown")
		time.Sleep(time.Second * 10)
		mserial.Push(mserial.Writer, serialCommand)
	} else {
		mserial.Push(mserial.Writer, serialCommand)
	}
}
