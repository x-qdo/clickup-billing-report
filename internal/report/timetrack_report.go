package report

import (
	"context"
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
	Status             string   `json:"status"`                    // Task status
	URL                string   `json:"url"`                       // Task URL
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

	firstDayOfMonth := time.Date(input.SelectedMonth.Year(), input.SelectedMonth.Month(), 1, 0, 0, 0, 0, input.SelectedMonth.Location())
	lastDayOfMonth := firstDayOfMonth.AddDate(0, 1, 0).Add(-time.Nanosecond) // End of the month
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
		if clientCfg.ClickUpListID == "" || clientCfg.ClickUpTeamID == "" {
			s.log.WithField("client", clientName).Warn("Client is missing ClickUpListID or ClickUpTeamID, skipping")
			continue
		}

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
					taskInvoicedHours[task.ID] = 0.0
				} else {
					taskInvoicedHours[task.ID] = parseValueAsFloat(invoicedVal, InvoicedHoursField, task.CustomID, s.log)
				}
			} else {
				taskInvoicedHours[task.ID] = 0.0
			}
		}

		// Convert collected assignee IDs map keys to slice
		assigneeIDsSlice := make([]string, 0, len(clientAssigneeIDs))
		for id := range clientAssigneeIDs {
			assigneeIDsSlice = append(assigneeIDsSlice, id)
		}
		s.log.WithFields(logrus.Fields{"client": clientName, "assignee_ids": assigneeIDsSlice}).Debug("Collected assignees for time entry fetching")

		// Fetch Time Entries for collected assignees within the date range
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
	personalTimeMap := make(map[string]*PersonalTime)
	taskTimeMap := make(map[string]float64)

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
				// Attempt to create and save the developer if not found
				newDeveloper := config.Developer{
					Name:        entry.User.Username,
					Coefficient: 1.0,
					ClickUpID:   strconv.Itoa(entry.User.ID),
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
				// Ensure the store is available before trying to save
				if s.appConfig.Store != nil {
					err := s.appConfig.Store.SaveDeveloper(ctx, newDeveloper)
					if err != nil {
						// Log error but continue report generation with coeff 1.0
						s.log.WithError(err).WithField("developer_name", newDeveloper.Name).Error("Failed to save newly created developer")
					} else {
						s.log.WithField("developer_name", newDeveloper.Name).Info("Successfully saved new developer")
						// Update the local map for subsequent lookups in this run
						developers[newDeveloper.Name] = newDeveloper
						dev = newDeveloper // Use the newly created developer info
						devOk = true       // Mark as found now
					}
				} else {
					s.log.Error("DataStore is nil, cannot save new developer")
					// Proceed with coeff 1.0 without saving
				}
			}

			// If developer was found (either initially or after creation)
			if devOk {
				if dev.Coefficient > 0 {
					coeff = dev.Coefficient
				} else {
					s.log.WithFields(logrus.Fields{"developer": dev.Name, "coefficient": dev.Coefficient}).Warn("Developer coefficient is zero or negative, using 1.0")
					coeff = 1.0
				}
			}
			// If developer wasn't found and couldn't be created, coeff remains 1.0

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
		task, ok := processedTasks[taskID]
		if !ok {
			s.log.WithField("task_id", taskID).Warn("Task details not found for aggregated time, skipping task in final report")
			continue
		}

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
			Status:             task.Status.Status,
			URL:                task.URL,
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
