// Package formatting are common utilities used across the MDroid suite
package formatting

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// JSONResponse for common return value to API
type JSONResponse struct {
	Output interface{} `json:"output,omitempty"`
	Status string      `json:"status,omitempty"`
	OK     bool        `json:"ok"`
	Method string      `json:"method,omitempty"`
	ID     int         `json:"id,omitempty"`
}

// WriteResponse to an http writer, adding extra info and HTTP status as needed
func WriteResponse(w *http.ResponseWriter, response JSONResponse) {
	// Deref writer
	writer := *w

	// Add string Status if it doesn't exist, add appropriate headers
	if response.OK {
		if response.Status == "" {
			response.Status = "success"
		}
		writer.WriteHeader(http.StatusOK)
	} else {
		if response.Status == "" {
			response.Status = "fail"
			writer.WriteHeader(http.StatusBadRequest)
		} else if response.Status == "error" {
			writer.WriteHeader(http.StatusNoContent)
		} else {
			writer.WriteHeader(http.StatusBadRequest)
		}
	}

	// Log this to debug
	log.Debug().
		Str("Output", fmt.Sprintf("%v", response.Output)).
		Str("Status", response.Status).
		Bool("OK", response.OK).
		Msg("Full Response:")

	// Write out this response
	json.NewEncoder(writer).Encode(response)
}

// FormatName returns string in upper case with underscores replacing spaces
func FormatName(name string) string {
	spaceRemover := regexp.MustCompile(`\s+`)
	name = spaceRemover.ReplaceAllString(name, " ")
	return strings.ToUpper(strings.Replace(strings.TrimSpace(name), " ", "_", -1))
}

// IsValidName verifies the name is alphanumeric
func IsValidName(name string) bool {
	return name == FormatName(name)
}

// IsPositiveRequest helps translate UP or LOCK into true or false
func IsPositiveRequest(request string) (bool, error) {
	switch request {
	case "ON", "UP", "LOCK", "OPEN", "TOGGLE", "PUSH":
		return true, nil
	case "OFF", "DOWN", "UNLOCK", "CLOSE":
		return false, nil
	}

	// Command didn't match any of the above, get out of here
	return false, fmt.Errorf("Error: %s in an invalid command", request)
}

// CompareTimestamps assuming both timezones are the same
func CompareTimestamps(time1 string, time2 string) (time.Duration, error) {
	t, err := time.Parse("2006-01-02 15:04:05.999", time2)
	t2, err2 := time.Parse("2006-01-02 15:04:05.999", time2)

	if err != nil {
		return 0, err
	}
	if err2 != nil {
		return 0, err2
	}

	return t.Sub(t2), nil
}

// CompareTimeToNow given a properly formatted time and timezone
func CompareTimeToNow(time1 string, timezone *time.Location) (time.Duration, error) {
	t, err := time.Parse("2006-01-02 15:04:05.999", time1)

	if err != nil {
		return 0, err
	}
	return time.Now().In(timezone).Sub(t), nil
}

// StringInSlice iterates over a slice, determining if a given string is present
// https://stackoverflow.com/questions/15323767/does-go-have-if-x-in-construct-similar-to-python
func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
