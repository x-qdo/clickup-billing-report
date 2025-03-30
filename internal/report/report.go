package report

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/x-qdo/clickup-billing-report/internal/clickup"
	"github.com/x-qdo/clickup-billing-report/internal/config"
)

const (
	BillableHoursField = "BillableHours"
	InvoicedHoursField = "InvoicedHours"
	ReportedByField    = "Reported By"
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

// TimeTrackingInput defines parameters for the time tracking report.
type TimeTrackingInput struct {
	SelectedMonth   time.Time // Represents the month to report on (e.g., 2023-10-01)
	RefreshBillable bool      // Flag to update BillableHours custom field
	ClickUpToken    string    // User's ClickUp access token
}

// PersonalTime represents aggregated time per user per client.
type PersonalTime struct {
	Username         string  `json:"username"`
	Client           string  `json:"client"`
	AdjustedDuration float64 `json:"adjusted_hours"` // Total adjusted hours (including internal)
	InternalDuration float64 `json:"internal_hours"` // Portion of AdjustedDuration spent on 'internal' tasks
	TotalDuration    float64 `json:"total_hours"`    // Raw hours before coefficient adjustment
}

// FinalReportTask represents aggregated time per task for the time tracking report.
type FinalReportTask struct {
	TaskID             string   `json:"task_id"`
	CustomID           string   `json:"custom_id"`
	Tags               []string `json:"tags"`
	TaskName           string   `json:"task_name"`
	Client             string   `json:"client"`
	AdjustedDuration   float64  `json:"adjusted_hours"`            // Rounded to 0.5h for the reporting period
	InvoicedHours      float64  `json:"invoiced_hours"`            // Value at the time of report generation
	CalculatedBillable float64  `json:"calculated_billable_hours"` // Invoiced + Adjusted for the period
}

// ClientTotal represents total adjusted hours per client for the time tracking report.
type ClientTotal struct {
	Client           string  `json:"client"`
	AdjustedDuration float64 `json:"adjusted_hours"`
}

// TimeTrackingOutput holds the results of the time tracking report.
type TimeTrackingOutput struct {
	PersonalReport []PersonalTime    `json:"personal_report"`
	FinalReport    []FinalReportTask `json:"final_report"`
	Totals         []ClientTotal     `json:"totals"`
}

// GenerateTimeTrackingReport generates the time tracking report based on Python logic.
func (s *Service) GenerateTimeTrackingReport(ctx context.Context, input TimeTrackingInput) (*TimeTrackingOutput, error) {
	s.log.WithFields(logrus.Fields{
		"month":           input.SelectedMonth.Format("2006-01"),
		"refreshBillable": input.RefreshBillable,
	}).Info("Generating time tracking report")

	// 1. Prepare: Get clients, developers, date range
	clients := s.appConfig.Clients
	developers := s.appConfig.Developers
	if len(clients) == 0 {
		return nil, fmt.Errorf("no clients configured")
	}

	firstDayOfMonth := input.SelectedMonth.Truncate(time.Hour * 24)
	lastDayOfMonth := firstDayOfMonth.AddDate(0, 1, -1).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	s.log.Debugf("Report date range: %s to %s", firstDayOfMonth.Format(time.RFC3339), lastDayOfMonth.Format(time.RFC3339))

	// Create ClickUp client with user's token
	cuClient, err := clickup.NewClient(nil, s.appConfig.Logger, input.ClickUpToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create clickup client: %w", err)
	}

	allTasks := make(map[string][]clickup.Task)            // clientName -> []Task
	allTimeEntries := make(map[string][]clickup.TimeEntry) // clientName -> []TimeEntry
	processedTasks := make(map[string]clickup.Task)        // taskID -> Task (for quick lookup)
	taskInvoicedHours := make(map[string]float64)          // taskID -> InvoicedHours
	taskBillableFieldID := make(map[string]string)         // taskID -> BillableHours Field ID (maps task ID to field ID)
	listBillableFieldIDCache := make(map[string]string)    // listID -> BillableHours Field ID (cache per list)
	listInvoicedFieldIDCache := make(map[string]string)    // listID -> InvoicedHours Field ID (cache per list)

	// 2. Fetch Data per Client
	for clientName, clientCfg := range clients {
		s.log.WithField("client", clientName).Info("Processing client")

		// Fetch Tasks
		taskOpts := &clickup.GetTasksOptions{
			IncludeClosed: true,
			Subtasks:      true,
		}
		tasks, err := cuClient.GetTasks(ctx, clientCfg.ClickUpListID, taskOpts)
		if err != nil {
			s.log.WithError(err).WithField("client", clientName).Error("Failed to fetch tasks")
			continue // Continue with other clients
		}
		allTasks[clientName] = tasks
		s.log.WithFields(logrus.Fields{"client": clientName, "task_count": len(tasks)}).Debug("Tasks fetched")

		// Get Custom Field IDs for this client's list (cached)
		billableFieldID, okB := listBillableFieldIDCache[clientCfg.ClickUpListID]
		if !okB {
			id, err := cuClient.GetCustomFieldID(ctx, clientCfg.ClickUpListID, BillableHoursField)
			if err != nil {
				s.log.WithError(err).
					WithFields(logrus.Fields{"client": clientName, "list_id": clientCfg.ClickUpListID}).
					Warnf("Could not find '%s' custom field ID", BillableHoursField)
			} else {
				s.log.
					WithField("list_id", clientCfg.ClickUpListID).
					Debugf("Found '%s' field ID: %s", BillableHoursField, id)
				billableFieldID = id
				listBillableFieldIDCache[clientCfg.ClickUpListID] = id
			}
		}
		invoicedFieldID, okI := listInvoicedFieldIDCache[clientCfg.ClickUpListID]
		if !okI {
			id, err := cuClient.GetCustomFieldID(ctx, clientCfg.ClickUpListID, InvoicedHoursField)
			if err != nil {
				s.log.WithError(err).
					WithFields(logrus.Fields{"client": clientName, "list_id": clientCfg.ClickUpListID}).
					Warnf("Could not find '%s' custom field ID", InvoicedHoursField)
			} else {
				s.log.WithField("list_id", clientCfg.ClickUpListID).Debugf("Found '%s' field ID: %s", InvoicedHoursField, id)
				invoicedFieldID = id
				listInvoicedFieldIDCache[clientCfg.ClickUpListID] = id
			}
		}

		// Process tasks: store for lookup, extract invoiced hours, map billable field ID, collect assignees
		clientAssigneeIDs := make(map[string]struct{}) // Use map for unique IDs
		for _, task := range tasks {
			processedTasks[task.ID] = task

			// Collect assignees from this task
			for _, assignee := range task.Assignees {
				clientAssigneeIDs[strconv.Itoa(assignee.ID)] = struct{}{}
			}

			// Store the billable field ID for this task if found for the list
			if billableFieldID != "" {
				taskBillableFieldID[task.ID] = billableFieldID
			}

			// Extract InvoicedHours value using the found field ID
			if invoicedFieldID != "" {
				invoicedVal, err := getCustomFieldValueByID(task, invoicedFieldID)
				if err != nil {
					// Field might not exist on task or have no value, default to 0
					taskInvoicedHours[task.ID] = 0.0
					// s.log.WithError(err).WithField("task_id", task.CustomID).Tracef("Could not get value for field ID %s", invoicedFieldID)
				} else {
					taskInvoicedHours[task.ID] = parseValueAsFloat(invoicedVal, InvoicedHoursField, task.CustomID, s.log)
				}
			} else {
				taskInvoicedHours[task.ID] = 0.0 // Field ID not found for list
			}
		}

		// Convert collected assignee IDs map keys to slice
		assigneeIDsSlice := make([]string, 0, len(clientAssigneeIDs))
		for id := range clientAssigneeIDs {
			assigneeIDsSlice = append(assigneeIDsSlice, id)
		}
		s.log.WithFields(logrus.Fields{"client": clientName, "assignee_ids": assigneeIDsSlice}).Debug("Collected assignees for time entry fetching")

		// Fetch Time Entries for collected assignees
		timeOpts := &clickup.GetTimeEntriesOptions{
			StartDate:   firstDayOfMonth,
			EndDate:     lastDayOfMonth,
			ListID:      clientCfg.ClickUpListID,
			AssigneeIDs: assigneeIDsSlice,
		}
		timeEntries, err := cuClient.GetTimeEntries(ctx, clientCfg.ClickUpTeamID, timeOpts)
		if err != nil {
			s.log.WithError(err).WithField("client", clientName).Error("Failed to fetch time entries")
			continue
		}
		allTimeEntries[clientName] = timeEntries
		s.log.WithFields(logrus.Fields{"client": clientName, "entry_count": len(timeEntries)}).Debug("Time entries fetched")
	}

	// 3. Process Time Entries & Calculate Durations
	personalTimeMap := make(map[string]*PersonalTime) // key: username_clientName
	taskTimeMap := make(map[string]float64)           // key: taskID -> total AdjustedDuration (in hours)

	for clientName, entries := range allTimeEntries {
		for _, entry := range entries {
			// Calculate durations in hours
			durationMs, err := strconv.ParseInt(entry.Duration, 10, 64)
			if err != nil {
				s.log.WithError(err).WithFields(logrus.Fields{"entry_id": entry.ID, "duration_str": entry.Duration}).Warn("Could not parse time entry duration, skipping entry")
				continue
			}
			totalDurationHours := float64(durationMs) / (1000 * 60 * 60)

			// Get developer coefficient
			coeff := 1.0
			dev, devOk := developers[entry.User.Username]
			if !devOk {
				s.log.WithField("username", entry.User.Username).Warn("Developer not found in config, creating with coefficient 1.0")
				newDeveloper := config.Developer{
					Name:        entry.User.Username,
					Coefficient: 1.0,
					ClickUpID:   strconv.Itoa(entry.User.ID),
				}
				err := s.appConfig.Store.SaveDeveloper(ctx, newDeveloper)
				if err != nil {
					// Log error but continue report generation with coeff 1.0
					s.log.WithError(err).WithField("developer_name", newDeveloper.Name).Error("Failed to save newly created developer")
				} else {
					s.log.WithField("developer_name", newDeveloper.Name).Info("Successfully saved new developer")
					developers[newDeveloper.Name] = newDeveloper
					dev = newDeveloper
				}
			} else {
				if dev.Coefficient > 0 {
					coeff = dev.Coefficient
				} else {
					s.log.WithFields(logrus.Fields{"developer": dev.Name, "coefficient": dev.Coefficient}).Warn("Developer coefficient is zero or negative, using 1.0")
					coeff = 1.0
				}
			}

			adjustedDurationHours := totalDurationHours / coeff

			// Determine task ID (handle parent tasks)
			taskIDForAggregation := entry.Task.ID
			isInternalTask := false
			var taskTags []clickup.Tag

			if task, ok := processedTasks[entry.Task.ID]; ok {
				taskTags = task.Tags // Get tags from the original task

				// Check if it's a subtask and parent exists
				if task.Parent != "" {
					if parentTask, parentExists := processedTasks[task.Parent]; parentExists {
						taskIDForAggregation = task.Parent // Aggregate time under the parent task
						taskTags = parentTask.Tags         // Use parent task's tags for internal check
					} else {
						s.log.WithFields(logrus.Fields{"task_id": task.CustomID, "parent_id": task.Parent}).Warn("Parent task details not found for time entry, aggregating under original task ID and using its tags")
					}
				}

				for _, tag := range taskTags {
					if strings.ToLower(tag.Name) == "internal" {
						isInternalTask = true
						break
					}
				}
			} else {
				s.log.WithField("task_id", entry.Task.ID).Warn("Task details not found for time entry, aggregating under original task ID, cannot check for internal tag")
			}

			// Aggregate personal time
			personalKey := fmt.Sprintf("%s_%s", entry.User.Username, clientName)
			pt, ptOk := personalTimeMap[personalKey]
			if !ptOk {
				pt = &PersonalTime{
					Username: entry.User.Username,
					Client:   clientName,
				}
				personalTimeMap[personalKey] = pt
			}
			pt.AdjustedDuration += adjustedDurationHours
			pt.TotalDuration += totalDurationHours
			if isInternalTask {
				pt.InternalDuration += adjustedDurationHours // Add to internal counter if tagged
			}

			// Aggregate task time using the determined task ID (original or parent)
			taskTimeMap[taskIDForAggregation] += adjustedDurationHours
		}
	}

	// 4. Generate Final Report Data
	finalReportTasks := []FinalReportTask{}
	clientTotalsMap := make(map[string]float64) // clientName -> total AdjustedDuration

	for taskID, adjustedDuration := range taskTimeMap {
		if task, ok := processedTasks[taskID]; ok {
			// Round adjusted duration to nearest 0.5 hour (like Python code)
			roundedAdjustedDuration := math.Round(adjustedDuration*2) / 2

			if roundedAdjustedDuration == 0 {
				continue // Skip tasks with zero rounded time for the period
			}

			invoicedHours := taskInvoicedHours[taskID] // Already fetched
			calculatedBillable := invoicedHours + roundedAdjustedDuration

			clientName := ""
			// Find client name from task's list relationship
			for name, cfg := range clients {
				if cfg.ClickUpListID == task.List.ID {
					clientName = name
					break
				}
			}
			if clientName == "" {
				s.log.WithField("task_id", task.CustomID).Warn("Could not determine client for task, skipping in final report")
				continue
			}

			// Extract tags
			taskTags := make([]string, len(task.Tags))
			for i, t := range task.Tags {
				taskTags[i] = t.Name
			}

			finalReportTasks = append(finalReportTasks, FinalReportTask{
				TaskID:             task.ID,
				CustomID:           task.CustomID,
				TaskName:           task.Name,
				Client:             clientName,
				Tags:               taskTags,
				AdjustedDuration:   roundedAdjustedDuration,
				InvoicedHours:      invoicedHours,
				CalculatedBillable: calculatedBillable,
			})

			// 5. Update ClickUp if refreshBillable is true
			if input.RefreshBillable {
				if fieldID, idOk := taskBillableFieldID[taskID]; idOk {
					s.log.WithFields(logrus.Fields{
						"task_id":   task.CustomID,
						"field_id":  fieldID,
						"new_value": calculatedBillable,
					}).Info("Updating BillableHours custom field")

					err := cuClient.UpdateTaskCustomField(ctx, taskID, fieldID, calculatedBillable)
					if err != nil {
						s.log.WithError(err).WithFields(logrus.Fields{
							"task_id":  task.CustomID,
							"field_id": fieldID,
						}).Error("Failed to update BillableHours custom field")
						// Continue processing other tasks
					} else {
						s.log.WithField("task_id", task.CustomID).Debug("BillableHours updated successfully")
					}
				} else {
					s.log.WithFields(logrus.Fields{
						"task_id": task.CustomID,
						"client":  clientName,
					}).Warn("Cannot update BillableHours: Field ID not found for this task/list")
				}
			}
		} else {
			s.log.WithField("task_id", taskID).Warn("Task details not found for aggregated time, skipping task in final report")
		}
	}

	// 6. Calculate Client Totals (excluding internal tasks)
	clientTotalsMap = make(map[string]float64)
	for _, task := range finalReportTasks {
		isInternal := false
		for _, tagName := range task.Tags {
			if strings.ToLower(tagName) == "internal" {
				isInternal = true
				break
			}
		}
		if !isInternal {
			clientTotalsMap[task.Client] += task.AdjustedDuration
		}
	}

	// 7. Format Output
	output := &TimeTrackingOutput{
		PersonalReport: make([]PersonalTime, 0, len(personalTimeMap)),
		FinalReport:    finalReportTasks, // Already populated
		Totals:         make([]ClientTotal, 0, len(clientTotalsMap)),
	}

	for _, pt := range personalTimeMap {
		// Round durations in personal report for consistency
		pt.AdjustedDuration = math.Round(pt.AdjustedDuration*100) / 100
		pt.InternalDuration = math.Round(pt.InternalDuration*100) / 100 // Round internal hours too
		pt.TotalDuration = math.Round(pt.TotalDuration*100) / 100
		output.PersonalReport = append(output.PersonalReport, *pt)
	}
	// Sort personal report by client then adjusted duration desc
	sort.SliceStable(output.PersonalReport, func(i, j int) bool {
		if output.PersonalReport[i].Client != output.PersonalReport[j].Client {
			return output.PersonalReport[i].Client < output.PersonalReport[j].Client
		}
		return output.PersonalReport[i].AdjustedDuration > output.PersonalReport[j].AdjustedDuration
	})

	for clientName, totalDuration := range clientTotalsMap {
		output.Totals = append(output.Totals, ClientTotal{
			Client:           clientName,
			AdjustedDuration: math.Round(totalDuration*100) / 100,
		})
	}
	// Sort totals by client
	sort.SliceStable(output.Totals, func(i, j int) bool {
		return output.Totals[i].Client < output.Totals[j].Client
	})

	// Sort final report by client then duration desc
	sort.SliceStable(output.FinalReport, func(i, j int) bool {
		if output.FinalReport[i].Client != output.FinalReport[j].Client {
			return output.FinalReport[i].Client < output.FinalReport[j].Client
		}
		return output.FinalReport[i].AdjustedDuration > output.FinalReport[j].AdjustedDuration
	})

	s.log.Info("Time tracking report generation complete")
	return output, nil
}

// --- Billable Report ---

// BillableReportInput defines parameters for the billable report.
type BillableReportInput struct {
	ListID          string // ClickUp List ID to generate the report for
	RefreshInvoiced bool   // Flag to update InvoicedHours to match BillableHours
	ClickUpToken    string // User's ClickUp access token
}

// BillableReportTask represents a task included in the billable report.
// Defined in config/config.go

// BillableReportTotals holds the sum of hours for the billable report.
// Defined in config/config.go

// BillableReportOutput holds the results of the billable report.
// Defined in config/config.go

// GenerateBillableReport generates a report similar to report.py.
func (s *Service) GenerateBillableReport(ctx context.Context, input BillableReportInput) (*config.BillableReportOutput, error) {
	s.log.WithFields(logrus.Fields{
		"list_id":         input.ListID,
		"refreshInvoiced": input.RefreshInvoiced,
	}).Info("Generating billable report")

	// Create ClickUp client
	cuClient, err := clickup.NewClient(nil, s.appConfig.Logger, input.ClickUpToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create clickup client: %w", err)
	}

	// Get Custom Field IDs for the list
	billableFieldID, errB := cuClient.GetCustomFieldID(ctx, input.ListID, BillableHoursField)
	if errB != nil {
		return nil, fmt.Errorf("failed to find '%s' custom field ID for list %s: %w", BillableHoursField, input.ListID, errB)
	}
	invoicedFieldID, errI := cuClient.GetCustomFieldID(ctx, input.ListID, InvoicedHoursField)
	if errI != nil {
		return nil, fmt.Errorf("failed to find '%s' custom field ID for list %s: %w", InvoicedHoursField, input.ListID, errI)
	}
	reporterFieldID, errR := cuClient.GetCustomFieldID(ctx, input.ListID, ReportedByField)
	if errR != nil {
		// Log a warning but continue, reporter field might be optional
		s.log.WithError(errR).WithField("list_id", input.ListID).Warnf("Could not find '%s' custom field ID, reporter field will be empty.", ReportedByField)
		reporterFieldID = "" // Ensure it's empty if not found
	} else {
		s.log.WithField("list_id", input.ListID).Debugf("Found '%s' field ID: %s", ReportedByField, reporterFieldID)
	}

	// Fetch all relevant tasks (including closed)
	taskOpts := &clickup.GetTasksOptions{
		IncludeClosed: true,
		Subtasks:      true, // Assuming subtasks might have billable hours too
	}
	allTasks, err := cuClient.GetTasks(ctx, input.ListID, taskOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tasks for list %s: %w", input.ListID, err)
	}
	s.log.WithField("fetched_task_count", len(allTasks)).Debug("Tasks fetched for billable report")

	// Filter and process tasks
	reportedTasks := []config.BillableReportTask{}
	totals := config.BillableReportTotals{}

	for _, task := range allTasks {
		// Extract Billable and Invoiced hours
		billableVal, _ := getCustomFieldValueByID(task, billableFieldID)
		invoicedVal, _ := getCustomFieldValueByID(task, invoicedFieldID)

		billableHours := parseValueAsFloat(billableVal, BillableHoursField, task.CustomID, s.log)
		invoicedHours := parseValueAsFloat(invoicedVal, InvoicedHoursField, task.CustomID, s.log)

		// Filter 1: BillableHours > 0 (from report.py logic)
		if billableHours <= 0 {
			continue
		}

		monthlyReported := billableHours - invoicedHours

		// Filter 2: monthly_reported != 0 (from report.py logic)
		if monthlyReported == 0 {
			continue
		}

		// Extract other relevant data
		priority := "-"
		if task.Priority != nil {
			priority = task.Priority.Priority
		}
		tags := make([]string, len(task.Tags))
		for i, t := range task.Tags {
			tags[i] = t.Name
		}

		// Extract Reporter field value
		reporter := "-" // Default value
		if reporterFieldID != "" {
			reporterVal, err := getCustomFieldValueByID(task, reporterFieldID)
			if err == nil {
				// Assuming the reporter field stores a simple string value
				reporter = parseValueAsString(reporterVal, ReportedByField, task.CustomID, s.log)
			} else {
				// Field ID exists, but value might be missing on this task
				// s.log.WithError(err).WithField("task_id", task.CustomID).Tracef("Could not get value for reporter field ID %s", reporterFieldID)
			}
		}

		reportTask := config.BillableReportTask{
			TaskID:          task.ID,
			CustomID:        task.CustomID,
			Name:            task.Name,
			Priority:        priority,
			Tags:            tags,
			BillableHours:   billableHours,
			InvoicedHours:   invoicedHours,
			MonthlyReported: monthlyReported,
			Reporter:        reporter, // Assign extracted value
			Status:          task.Status.Status,
			URL:             task.URL,
		}
		reportedTasks = append(reportedTasks, reportTask)

		// Update totals
		totals.BillableHours += billableHours
		totals.InvoicedHours += invoicedHours
		totals.MonthlyReported += monthlyReported

		// Update ClickUp InvoicedHours if refreshInvoiced is true
		if input.RefreshInvoiced {
			s.log.WithFields(logrus.Fields{
				"task_id":   task.CustomID,
				"field_id":  invoicedFieldID,
				"new_value": billableHours, // Set Invoiced = Billable
			}).Info("Updating InvoicedHours custom field")

			err := cuClient.UpdateTaskCustomField(ctx, task.ID, invoicedFieldID, billableHours)
			if err != nil {
				s.log.WithError(err).WithFields(logrus.Fields{
					"task_id":  task.CustomID,
					"field_id": invoicedFieldID,
				}).Error("Failed to update InvoicedHours custom field")
				// Decide if this should halt the process or just log
				// return nil, fmt.Errorf("failed to update invoiced hours for task %s: %w", task.CustomID, err)
			} else {
				s.log.WithField("task_id", task.CustomID).Debug("InvoicedHours updated successfully")
			}
		}
	}

	// Round totals
	totals.BillableHours = math.Round(totals.BillableHours*100) / 100
	totals.InvoicedHours = math.Round(totals.InvoicedHours*100) / 100
	totals.MonthlyReported = math.Round(totals.MonthlyReported*100) / 100

	// Sort tasks (e.g., by MonthlyReported descending)
	sort.SliceStable(reportedTasks, func(i, j int) bool {
		return reportedTasks[i].MonthlyReported > reportedTasks[j].MonthlyReported
	})

	output := &config.BillableReportOutput{
		Tasks:  reportedTasks,
		Totals: totals,
	}

	s.log.WithField("reported_task_count", len(reportedTasks)).Info("Billable report generation complete")
	return output, nil
}

// --- Helpers ---

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
		log.WithError(err).WithFields(logrus.Fields{"task_id": taskCustomID, "field": fieldName, "value": v}).Warn("Could not parse string value as float64")
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
	default:
		str := fmt.Sprintf("%v", v)
		if str == "" {
			return "-"
		}
		return str
	}
}

// getCustomFieldValue (Deprecated by getCustomFieldValueByID if IDs are available)
// Extracts the value of a custom field by name from a task.
func getCustomFieldValue(task clickup.Task, fieldName string) (interface{}, error) {
	for _, cf := range task.CustomFields {
		if cf.Name == fieldName {
			// Need to unmarshal the json.RawMessage based on type
			var value interface{}
			if cf.Value != nil && len(cf.Value) > 0 && string(cf.Value) != "null" {
				// Simple heuristic: try unmarshalling as number first, then string, etc.
				var numVal float64
				if err := json.Unmarshal(cf.Value, &numVal); err == nil {
					value = numVal
				} else {
					// Try as string (remove quotes if present)
					strVal := strings.Trim(string(cf.Value), `"`)
					value = strVal
					// Add more type checks if needed (arrays, objects, etc.)
				}
			} else {
				// Field exists but has no value or is null
				return nil, fmt.Errorf("field '%s' has no value", fieldName)
			}
			return value, nil
		}
	}
	return nil, fmt.Errorf("field '%s' not found", fieldName)
}
