package report

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/x-qdo/clickup-billing-report/internal/clickup"
	"github.com/x-qdo/clickup-billing-report/internal/config"
)

const (
	BillableHoursField = "BillableHours"
	InvoicedHoursField = "InvoicedHours"
	ReportedByField    = "Requested by:"
)

// Service handles report generation logic.
type Service struct {
	appConfig *config.AppConfig
	log       *logrus.Entry
}

// NewService creates a new report service.
func NewService(appConf *config.AppConfig) *Service {
	return &Service{
		appConfig: appConf,
		log:       appConf.Logger.WithField("component", "ReportService"),
	}
}

// getCustomFieldValueByID extracts the value of a custom field by its ID from a task.
func getCustomFieldValueByID(task clickup.Task, fieldID string) (interface{}, error) {
	for _, cf := range task.CustomFields {
		if cf.ID == fieldID {
			var value interface{}
			if cf.Value != nil && len(cf.Value) > 0 && string(cf.Value) != "null" {
				// Attempt to unmarshal as the most common types first
				var numVal float64
				var strVal string
				// var arrVal []interface{} // Add if array types are needed
				// var boolVal bool // Add if boolean types are needed

				// Try number
				if err := json.Unmarshal(cf.Value, &numVal); err == nil {
					value = numVal
				} else if err := json.Unmarshal(cf.Value, &strVal); err == nil {
					// Try string (might capture numbers as strings too)
					value = strVal
				} else {
					// Fallback: return raw JSON if specific types fail
					// Check if it's already a simple string representation in RawMessage
					rawStr := string(cf.Value)
					if !strings.HasPrefix(rawStr, "{") && !strings.HasPrefix(rawStr, "[") {
						value = strings.Trim(rawStr, `"`) // Treat as string if not JSON object/array
					} else {
						value = cf.Value // Keep as RawMessage if complex JSON
					}
				}
			} else {
				return nil, fmt.Errorf("field ID '%s' has no value or is null", fieldID)
			}
			return value, nil
		}
	}
	return nil, fmt.Errorf("field ID '%s' not found in task %s", fieldID, task.CustomID)
}

// parseValueAsFloat tries to convert an interface{} value to float64.
// Handles float64 directly, attempts to parse strings, defaults to 0.0 on failure.
func parseValueAsFloat(value interface{}, fieldName, taskCustomID string, log *logrus.Entry) float64 {
	if value == nil {
		return 0.0
	}
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number: // Handles numbers that might be unmarshalled as json.Number
		f, err := v.Float64()
		if err == nil {
			return f
		}
		log.WithError(err).WithFields(logrus.Fields{"task_id": taskCustomID, "field": fieldName, "value": v}).Warn("Could not parse json.Number as float64")
		return 0.0
	case string:
		// Try parsing the string as a float
		f, err := strconv.ParseFloat(v, 64)
		if err == nil {
			return f
		}
		return 0.0
	default:
		log.WithFields(logrus.Fields{"task_id": taskCustomID, "field": fieldName, "value": value, "type": fmt.Sprintf("%T", value)}).Warn("Unsupported type for float64 conversion")
		return 0.0
	}
}

// parseValueAsString tries to convert an interface{} value to string.
// Handles string directly, uses fmt.Sprintf for others, defaults to "-" on failure/nil.
func parseValueAsString(value interface{}, fieldName, taskCustomID string, log *logrus.Entry) string {
	if value == nil {
		return "-"
	}
	switch v := value.(type) {
	case string:
		if v == "" {
			return "-"
		}
		return v
	case json.RawMessage:
		// Try unmarshalling as string first
		var strVal string
		if err := json.Unmarshal(v, &strVal); err == nil {
			if strVal == "" {
				return "-"
			}
			return strVal
		}
		// Fallback to string representation of the raw message
		rawStr := string(v)
		if rawStr == "" || rawStr == "null" {
			return "-"
		}
		return strings.Trim(rawStr, `"`) // Attempt to clean up quotes
	case float64: // Handle numbers that might be stored in string-like fields
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case bool:
		return strconv.FormatBool(v)
	default:
		str := fmt.Sprintf("%v", v)
		if str == "" {
			return "-"
		}
		return str
	}
}
