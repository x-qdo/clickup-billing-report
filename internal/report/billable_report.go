package report

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/x-qdo/clickup-billing-report/internal/clickup"
	"github.com/x-qdo/clickup-billing-report/internal/config"
)

// BillableReportInput defines parameters for the billable report.
type BillableReportInput struct {
	ClientName      string // Client Name to generate the report for
	RefreshInvoiced bool   // Flag to update InvoicedHours to match BillableHours (only for non-internal tasks)
	ClickUpToken    string // User's ClickUp access token
}

// isInternalTask checks if a task has the "internal" tag (case-insensitive).
func isInternalTask(task clickup.Task) bool {
	for _, tag := range task.Tags {
		if strings.ToLower(tag.Name) == "internal" {
			return true
		}
	}
	return false
}

// GenerateBillableReport generates a report similar to report.py, separating internal tasks.
func (s *Service) GenerateBillableReport(ctx context.Context, input BillableReportInput) (*config.BillableReportOutput, error) {
	s.log.WithFields(logrus.Fields{
		"client_name":     input.ClientName,
		"refreshInvoiced": input.RefreshInvoiced,
	}).Info("Generating billable report")

	clientCfg, ok := s.appConfig.Clients[input.ClientName]
	if !ok {
		s.log.WithField("client_name", input.ClientName).Error("Client configuration not found")
		return nil, fmt.Errorf("client configuration not found for '%s'", input.ClientName)
	}
	if clientCfg.ClickUpListID == "" {
		s.log.WithField("client_name", input.ClientName).Error("Client configuration is missing ClickUp List ID")
		return nil, fmt.Errorf("client '%s' is missing ClickUp List ID in configuration", input.ClientName)
	}
	listID := clientCfg.ClickUpListID

	// Create ClickUp client
	cuClient, err := clickup.NewClient(nil, s.appConfig.Logger, input.ClickUpToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create clickup client: %w", err)
	}

	// Get Custom Field IDs for the list
	billableFieldID, errB := cuClient.GetCustomFieldID(ctx, listID, BillableHoursField)
	if errB != nil {
		// Billable field is crucial for non-internal tasks
		return nil, fmt.Errorf("failed to find '%s' custom field ID for list %s (client: %s): %w", BillableHoursField, listID, input.ClientName, errB)
	}
	invoicedFieldID, errI := cuClient.GetCustomFieldID(ctx, listID, InvoicedHoursField)
	if errI != nil {
		// Invoiced field is crucial for non-internal tasks
		return nil, fmt.Errorf("failed to find '%s' custom field ID for list %s (client: %s): %w", InvoicedHoursField, listID, input.ClientName, errI)
	}
	reporterFieldID, errR := cuClient.GetCustomFieldID(ctx, listID, ReportedByField)
	if errR != nil {
		// Log a warning but continue, reporter field might be optional
		s.log.WithError(errR).Warnf("Could not find '%s' custom field ID, reporter field will be empty.", ReportedByField)
		reporterFieldID = ""
	} else {
		s.log.Debugf("Found '%s' field ID: %s", ReportedByField, reporterFieldID)
	}

	taskOpts := &clickup.GetTasksOptions{
		IncludeClosed: true,
		Subtasks:      false, // Assuming we only report on parent tasks for billable report
	}
	allTasks, err := cuClient.GetTasks(ctx, listID, taskOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tasks for list %s (client: %s): %w", listID, input.ClientName, err)
	}
	s.log.WithField("fetched_task_count", len(allTasks)).Debug("Tasks fetched for billable report")

	reportedTasks := []config.BillableReportTask{}
	internalTasks := []config.BillableReportTask{}
	totals := config.BillableReportTotals{} // Totals for non-internal tasks only

	for _, task := range allTasks {
		// Common task details
		priority := "-"
		if task.Priority != nil {
			priority = task.Priority.Priority
		}
		tags := make([]string, len(task.Tags))
		for i, t := range task.Tags {
			tags[i] = t.Name
		}
		reporter := "-"
		if reporterFieldID != "" {
			reporterVal, err := getCustomFieldValueByID(task, reporterFieldID)
			if err == nil {
				reporterStr := parseValueAsString(reporterVal, ReportedByField, task.CustomID, s.log)
				if reporterStr != "-" {
					reporter = reporterStr
				}
			}
		}

		// Extract Billable and Invoiced hours (needed for both paths initially)
		billableVal, _ := getCustomFieldValueByID(task, billableFieldID)
		invoicedVal, _ := getCustomFieldValueByID(task, invoicedFieldID)
		billableHours := parseValueAsFloat(billableVal, BillableHoursField, task.CustomID, s.log)
		invoicedHours := parseValueAsFloat(invoicedVal, InvoicedHoursField, task.CustomID, s.log)
		monthlyReported := billableHours - invoicedHours // Calculate even for internal for potential display

		// Check if task is internal
		if isInternalTask(task) {
			s.log.WithField("task_id", task.CustomID).Debug("Identified internal task")
			// Update ClickUp InvoicedHours if refreshInvoiced is true for internal tasks
			if input.RefreshInvoiced {
				// Only update if the new value (billableHours) is different from the current invoicedHours
				// to avoid unnecessary API calls. The invoicedFieldID is guaranteed to be present here
				// due to earlier checks that would terminate the function.
				if math.Abs(billableHours-invoicedHours) > 0.001 {
					s.log.WithFields(logrus.Fields{
						"task_id":      task.CustomID,
						"field_id":     invoicedFieldID,
						"current_inv":  invoicedHours,
						"new_inv_val":  billableHours,   // Set Invoiced = Billable
						"monthly_diff": monthlyReported, // This is the diff *before* update
					}).Info("Updating InvoicedHours custom field to match BillableHours for internal task")

					errUpdate := cuClient.UpdateTaskCustomField(ctx, task.ID, invoicedFieldID, billableHours)
					if errUpdate != nil {
						s.log.WithError(errUpdate).WithFields(logrus.Fields{
							"task_id":  task.CustomID,
							"field_id": invoicedFieldID,
						}).Error("Failed to update InvoicedHours custom field for internal task")
					} else {
						s.log.WithField("task_id", task.CustomID).Debug("InvoicedHours updated successfully for internal task")
						// The report will reflect the state *before* this update.
						// The update ensures it's "cleared" for the next reporting cycle.
						// We do not modify local `invoicedHours` or `monthlyReported` here,
						// so the current report reflects values *before* this specific update,
						// consistent with non-internal task handling.
					}
				} else {
					s.log.WithFields(logrus.Fields{
						"task_id":     task.CustomID,
						"billable":    billableHours,
						"invoiced":    invoicedHours,
						"monthly_rep": monthlyReported,
					}).Debug("Skipping InvoicedHours update for internal task as it already matches BillableHours (or difference is negligible)")
				}
			}

			internalReportTask := config.BillableReportTask{
				TaskID:          task.ID,
				CustomID:        task.CustomID,
				Name:            task.Name,
				Priority:        priority,
				Tags:            tags,
				BillableHours:   billableHours,
				InvoicedHours:   invoicedHours,
				MonthlyReported: monthlyReported,
				Reporter:        reporter,
				URL:             task.URL,
			}
			internalTasks = append(internalTasks, internalReportTask)
			continue
		}

		// Filter 1: BillableHours > 0 (from original report.py logic)
		if billableHours <= 0 {
			s.log.WithField("task_id", task.CustomID).Debug("Skipping non-internal task: BillableHours <= 0")
			continue
		}

		// Filter 2: MonthlyReported != 0 (from original report.py logic)
		// Use a small tolerance for float comparison
		if math.Abs(monthlyReported) < 0.001 {
			s.log.WithField("task_id", task.CustomID).Debug("Skipping non-internal task: MonthlyReported is zero")
			continue
		}

		// Task passed filters, add to report and totals
		reportTask := config.BillableReportTask{
			TaskID:          task.ID,
			CustomID:        task.CustomID,
			Name:            task.Name,
			Priority:        priority,
			Tags:            tags,
			BillableHours:   billableHours,
			InvoicedHours:   invoicedHours,
			MonthlyReported: monthlyReported,
			Reporter:        reporter,
			URL:             task.URL,
		}
		reportedTasks = append(reportedTasks, reportTask)

		// Update totals (only for non-internal, filtered tasks)
		totals.BillableHours += billableHours
		totals.InvoicedHours += invoicedHours
		totals.MonthlyReported += monthlyReported

		// Update ClickUp InvoicedHours if refreshInvoiced is true (only for non-internal, filtered tasks)
		if input.RefreshInvoiced {
			// Only update if the new value (billableHours) is different from the current invoicedHours
			// to avoid unnecessary API calls and potential race conditions/errors.
			if math.Abs(billableHours-invoicedHours) > 0.001 {
				s.log.WithFields(logrus.Fields{
					"task_id":      task.CustomID,
					"field_id":     invoicedFieldID,
					"current_inv":  invoicedHours,
					"new_inv_val":  billableHours, // Set Invoiced = Billable
					"monthly_diff": monthlyReported,
				}).Info("Updating InvoicedHours custom field to match BillableHours for non-internal task")

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
			} else {
				s.log.WithFields(logrus.Fields{
					"task_id":     task.CustomID,
					"billable":    billableHours,
					"invoiced":    invoicedHours,
					"monthly_rep": monthlyReported,
				}).Debug("Skipping InvoicedHours update as it already matches BillableHours")
			}
		}
	}

	// Round totals
	totals.BillableHours = math.Round(totals.BillableHours*100) / 100
	totals.InvoicedHours = math.Round(totals.InvoicedHours*100) / 100
	totals.MonthlyReported = math.Round(totals.MonthlyReported*100) / 100

	// Sort reported tasks by MonthlyReported descending
	sort.SliceStable(reportedTasks, func(i, j int) bool {
		return reportedTasks[i].MonthlyReported > reportedTasks[j].MonthlyReported
	})

	// Sort internal tasks (e.g., by name or ID)
	sort.SliceStable(internalTasks, func(i, j int) bool {
		return internalTasks[i].Name < internalTasks[j].Name // Sort alphabetically by name
	})

	output := &config.BillableReportOutput{
		Tasks:         reportedTasks,
		InternalTasks: internalTasks,
		Totals:        totals,
	}

	s.log.WithFields(logrus.Fields{
		"reported_task_count": len(reportedTasks),
		"internal_task_count": len(internalTasks),
	}).Info("Billable report generation complete")
	return output, nil
}
