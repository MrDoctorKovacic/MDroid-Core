package main

import (
	"time"

	"github.com/MrDoctorKovacic/MDroid-Core/logging"
)

// MainStatus will control logging and reporting of status / warnings / errors
var MainStatus = logging.NewStatus("Main")

// Timezone location for session last used and logging
var Timezone *time.Location

// define our router and subsequent routes here
func main() {

	// Run through the config file and set up some global variables
	parseConfig()

	// Define routes and begin routing
	startRouter()
}
