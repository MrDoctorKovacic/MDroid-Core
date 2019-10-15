package main

import (
	"fmt"

	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/MrDoctorKovacic/MDroid-Core/mserial"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
)

func setupHooks() {
	settings.RegisterHook("ANGEL_EYES", angelEyesSettings)
}

func angelEyesSettings(settingName string, settingValue string) {
	if settingName == "" || settingValue == "" {
		mainStatus.Log(logging.Error(), "Fail")
	}

	switch settingName {
	case "POWER":
		if settingValue == "ON" {
			mserial.WriteSerial(settings.Config.SerialControlDevice, fmt.Sprintf("powerOnAngel"))
		} else {
			// Even if is AUTO, turn off and let session handlers deal with it
			mserial.WriteSerial(settings.Config.SerialControlDevice, fmt.Sprintf("powerOffAngel"))
		}
	}
}