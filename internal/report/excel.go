package report

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/x-qdo/clickup-billing-report/internal/config"
	"github.com/xuri/excelize/v2"
)

const (
	sheetNameBillable   = "Billable Report"
	sheetNameInternal   = "Internal Tasks" // New sheet name
	sheetNameTimeTrack  = "Time Tracking Details"
	sheetNamePersonal   = "Personal Time Summary"
	sheetNameClientTots = "Client Totals"
	floatFormat         = 4        // "#,##0.00" Excel format for 2 decimal places
	headerFillColor     = "C6E0B4" // Light green fill for headers
	totalFillColor      = "FCE4D6" // Light orange fill for totals
)

// --- Helper Function for writing rows ---
func writeTaskRow(f *excelize.File, sheet string, rowNum int, task config.BillableReportTask, floatStyle int) {
	colNum := 0
	cell := func(val interface{}) string {
		c, _ := excelize.CoordinatesToCellName(colNum+1, rowNum)
		_ = f.SetCellValue(sheet, c, val)
		colNum++
		return c
	}
	floatCell := func(val float64) string {
		c := cell(val)
		_ = f.SetCellStyle(sheet, c, c, floatStyle)
		return c
	}
	linkCell := func(url, display string) string {
		c := cell(display)
		if url != "" {
			_ = f.SetCellHyperLink(sheet, c, url, "External")
			// Optional: Add basic link styling
			// linkStyle, _ := f.NewStyle(`{"font":{"color":"#0563C1","underline":"single"}}`)
			// _ = f.SetCellStyle(sheet, c, c, linkStyle)
		}
		return c
	}

	cell(task.CustomID)
	cell(task.Name)
	cell(task.Priority)
	cell(strings.Join(task.Tags, ", "))
	cell(task.Reporter)
	floatCell(task.BillableHours)
	floatCell(task.InvoicedHours)
	floatCell(task.MonthlyReported)
	linkCell(task.URL, "Link") // Add link column
}

// --- Helper Function for writing headers ---
func writeHeaders(f *excelize.File, sheet string, headers []string, headerStyle int) {
	for i, h := range headers {
		cellName, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cellName, h)
		_ = f.SetCellStyle(sheet, cellName, cellName, headerStyle)
	}
}

// --- Helper Function for writing totals ---
func writeTotalsRow(f *excelize.File, sheet string, rowNum int, totals config.BillableReportTotals, totalStyle int) {
	totalHeaders := []string{"", "", "", "", "TOTALS:", "", "", "", ""} // Adjusted for link column
	totalCells := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I"}
	for i, th := range totalHeaders {
		cell := totalCells[i] + strconv.Itoa(rowNum)
		_ = f.SetCellValue(sheet, cell, th)
		_ = f.SetCellStyle(sheet, cell, cell, totalStyle) // Apply style to header cell too
	}
	// Set total values and apply style
	totalValCells := []string{"F", "G", "H"}
	totalVals := []float64{totals.BillableHours, totals.InvoicedHours, totals.MonthlyReported}
	for i, tv := range totalVals {
		cell := totalValCells[i] + strconv.Itoa(rowNum)
		_ = f.SetCellValue(sheet, cell, tv)
		_ = f.SetCellStyle(sheet, cell, cell, totalStyle)
	}
}

// --- Helper Function for setting column widths and panes ---
func formatSheet(f *excelize.File, sheet string) {
	// Set column widths (adjust as needed)
	_ = f.SetColWidth(sheet, "A", "A", 15) // Task ID
	_ = f.SetColWidth(sheet, "B", "B", 50) // Name
	_ = f.SetColWidth(sheet, "C", "C", 10) // Priority
	_ = f.SetColWidth(sheet, "D", "D", 25) // Tags
	_ = f.SetColWidth(sheet, "E", "E", 20) // Reporter
	_ = f.SetColWidth(sheet, "F", "H", 18) // Hours columns
	_ = f.SetColWidth(sheet, "I", "I", 10) // Link column

	// Freeze header row
	_ = f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1, // Freeze first row
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})
}

// GenerateBillableExcel creates an Excel file from BillableReportOutput data,
// separating internal tasks onto a new sheet.
func GenerateBillableExcel(reportData *config.BillableReportOutput, clientName string) (*excelize.File, error) {
	f := excelize.NewFile()
	defer func() {
		// Close the file to release resources; ignore errors as we return the object
		_ = f.Close()
	}()

	// --- Styles ---
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{headerFillColor}, Pattern: 1},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})
	totalStyle, _ := f.NewStyle(&excelize.Style{
		Font:   &excelize.Font{Bold: true},
		Fill:   excelize.Fill{Type: "pattern", Color: []string{totalFillColor}, Pattern: 1},
		NumFmt: floatFormat,
	})
	floatStyle, _ := f.NewStyle(&excelize.Style{
		NumFmt: floatFormat,
	})

	// --- Sheet 1: Billable Report (Non-Internal) ---
	sheet1 := sheetNameBillable
	index1, err := f.NewSheet(sheet1)
	if err != nil {
		return nil, fmt.Errorf("failed to create sheet '%s': %w", sheet1, err)
	}
	f.SetActiveSheet(index1)
	f.DeleteSheet("Sheet1") // Remove default sheet

	// Headers
	headers := []string{"Task ID", "Name", "Priority", "Tags", "Reporter", "Billable Hours", "Invoiced Hours", "Monthly Reported", "Link"}
	writeHeaders(f, sheet1, headers, headerStyle)

	// Data Rows
	rowNum1 := 2
	for _, task := range reportData.Tasks {
		writeTaskRow(f, sheet1, rowNum1, task, floatStyle)
		rowNum1++
	}

	// Totals Row (only if there are tasks)
	if len(reportData.Tasks) > 0 {
		rowNum1++ // Add a blank row before totals
		writeTotalsRow(f, sheet1, rowNum1, reportData.Totals, totalStyle)
	}

	// Formatting
	formatSheet(f, sheet1)

	// --- Sheet 2: Internal Tasks ---
	if len(reportData.InternalTasks) > 0 {
		sheet2 := sheetNameInternal
		_, err = f.NewSheet(sheet2)
		if err != nil {
			// Log error but don't fail the whole report generation
			logrus.WithError(err).Warnf("Failed to create sheet '%s'", sheet2)
		} else {
			// Headers (same as billable for consistency)
			writeHeaders(f, sheet2, headers, headerStyle)

			// Data Rows
			rowNum2 := 2
			internalTotals := config.BillableReportTotals{} // Calculate totals for this sheet if needed
			for _, task := range reportData.InternalTasks {
				writeTaskRow(f, sheet2, rowNum2, task, floatStyle)
				// Optionally sum hours for internal tasks if meaningful
				internalTotals.BillableHours += task.BillableHours
				internalTotals.InvoicedHours += task.InvoicedHours
				internalTotals.MonthlyReported += task.MonthlyReported
				rowNum2++
			}

			// Totals Row for Internal Tasks (Optional)
			// You might decide not to show totals for internal, or show different ones.
			// Here we show the same totals structure for consistency.
			rowNum2++ // Blank row
			internalTotals.BillableHours = math.Round(internalTotals.BillableHours*100) / 100
			internalTotals.InvoicedHours = math.Round(internalTotals.InvoicedHours*100) / 100
			internalTotals.MonthlyReported = math.Round(internalTotals.MonthlyReported*100) / 100
			writeTotalsRow(f, sheet2, rowNum2, internalTotals, totalStyle)

			// Formatting
			formatSheet(f, sheet2)
		}
	}

	// Set the first sheet (Billable Report) as active by default
	f.SetActiveSheet(index1)

	return f, nil
}

// GenerateTimeTrackingExcel creates an Excel file from TimeTrackingOutput data.
// (This function remains unchanged by the current request)
func GenerateTimeTrackingExcel(reportData *TimeTrackingOutput, month time.Time) (*excelize.File, error) {
	f := excelize.NewFile()
	defer func() {
		// Close the file to release resources; ignore errors as we return the object
		_ = f.Close()
	}()

	// --- Styles ---
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{headerFillColor}, Pattern: 1},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})
	totalStyle, _ := f.NewStyle(&excelize.Style{
		Font:   &excelize.Font{Bold: true},
		Fill:   excelize.Fill{Type: "pattern", Color: []string{totalFillColor}, Pattern: 1},
		NumFmt: floatFormat,
	})
	floatStyle, _ := f.NewStyle(&excelize.Style{
		NumFmt: floatFormat,
	})

	// --- Sheet 1: Final Report (Task Details) ---
	sheet1 := sheetNameTimeTrack
	index1, err := f.NewSheet(sheet1)
	if err != nil {
		return nil, fmt.Errorf("failed to create sheet '%s': %w", sheet1, err)
	}
	f.SetActiveSheet(index1)
	f.DeleteSheet("Sheet1") // Remove default sheet

	headers1 := []string{"Client", "Task ID", "Task Name", "Tags", "Status", "Adjusted Hours (Period)", "Invoiced Hours (Start)", "Calculated Billable (End)", "Task URL"}
	headerCells1 := []string{"A1", "B1", "C1", "D1", "E1", "F1", "G1", "H1", "I1"}
	for i, h := range headers1 {
		cell := headerCells1[i]
		_ = f.SetCellValue(sheet1, cell, h)
		_ = f.SetCellStyle(sheet1, cell, cell, headerStyle)
	}

	rowNum1 := 2
	for _, task := range reportData.FinalReport {
		colNum := 0
		cell := func(val interface{}) string {
			c, _ := excelize.CoordinatesToCellName(colNum+1, rowNum1)
			_ = f.SetCellValue(sheet1, c, val)
			colNum++
			return c
		}
		floatCell := func(val float64) string {
			c := cell(val)
			_ = f.SetCellStyle(sheet1, c, c, floatStyle)
			return c
		}
		linkCell := func(url, display string) string {
			c := cell(display)
			if url != "" {
				_ = f.SetCellHyperLink(sheet1, c, url, "External")
			}
			return c
		}

		cell(task.Client)
		cell(task.CustomID)
		cell(task.TaskName)
		cell(strings.Join(task.Tags, ", "))
		cell(task.Status)
		floatCell(task.AdjustedDuration)
		floatCell(task.InvoicedHours)
		floatCell(task.CalculatedBillable)
		linkCell(task.URL, "Link") // Display "Link" text for the URL

		rowNum1++
	}
	_ = f.SetColWidth(sheet1, "A", "A", 20) // Client
	_ = f.SetColWidth(sheet1, "B", "B", 15) // Task ID
	_ = f.SetColWidth(sheet1, "C", "C", 50) // Task Name
	_ = f.SetColWidth(sheet1, "D", "D", 25) // Tags
	_ = f.SetColWidth(sheet1, "E", "E", 15) // Status
	_ = f.SetColWidth(sheet1, "F", "H", 20) // Hours columns
	_ = f.SetColWidth(sheet1, "I", "I", 10) // URL Link
	_ = f.SetPanes(sheet1, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2"})

	// --- Sheet 2: Personal Time Summary ---
	sheet2 := sheetNamePersonal
	_, err = f.NewSheet(sheet2)
	if err != nil {
		return nil, fmt.Errorf("failed to create sheet '%s': %w", sheet2, err)
	}

	headers2 := []string{"Username", "Client", "Adjusted Hours (Total)", "Internal Hours", "Raw Hours"}
	headerCells2 := []string{"A1", "B1", "C1", "D1", "E1"}
	for i, h := range headers2 {
		cell := headerCells2[i]
		_ = f.SetCellValue(sheet2, cell, h)
		_ = f.SetCellStyle(sheet2, cell, cell, headerStyle)
	}

	rowNum2 := 2
	for _, pt := range reportData.PersonalReport {
		colNum := 0
		cell := func(val interface{}) string {
			c, _ := excelize.CoordinatesToCellName(colNum+1, rowNum2)
			_ = f.SetCellValue(sheet2, c, val)
			colNum++
			return c
		}
		floatCell := func(val float64) string {
			c := cell(val)
			_ = f.SetCellStyle(sheet2, c, c, floatStyle)
			return c
		}

		cell(pt.Username)
		cell(pt.Client)
		floatCell(pt.AdjustedDuration)
		floatCell(pt.InternalDuration)
		floatCell(pt.TotalDuration)

		rowNum2++
	}
	_ = f.SetColWidth(sheet2, "A", "B", 25) // Username, Client
	_ = f.SetColWidth(sheet2, "C", "E", 20) // Hours columns
	_ = f.SetPanes(sheet2, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2"})

	// --- Sheet 3: Client Totals (Excluding Internal) ---
	sheet3 := sheetNameClientTots
	_, err = f.NewSheet(sheet3)
	if err != nil {
		return nil, fmt.Errorf("failed to create sheet '%s': %w", sheet3, err)
	}

	headers3 := []string{"Client", "Total Adjusted Hours (Excl. Internal)"}
	headerCells3 := []string{"A1", "B1"}
	for i, h := range headers3 {
		cell := headerCells3[i]
		_ = f.SetCellValue(sheet3, cell, h)
		_ = f.SetCellStyle(sheet3, cell, cell, headerStyle)
	}

	rowNum3 := 2
	var grandTotal float64 = 0
	for _, ct := range reportData.Totals {
		colNum := 0
		cell := func(val interface{}) string {
			c, _ := excelize.CoordinatesToCellName(colNum+1, rowNum3)
			_ = f.SetCellValue(sheet3, c, val)
			colNum++
			return c
		}
		floatCell := func(val float64) string {
			c := cell(val)
			_ = f.SetCellStyle(sheet3, c, c, floatStyle)
			return c
		}

		cell(ct.Client)
		floatCell(ct.AdjustedDuration)
		grandTotal += ct.AdjustedDuration
		rowNum3++
	}
	// Add Grand Total row
	cellA := "A" + strconv.Itoa(rowNum3)
	cellB := "B" + strconv.Itoa(rowNum3)
	_ = f.SetCellValue(sheet3, cellA, "GRAND TOTAL:")
	_ = f.SetCellValue(sheet3, cellB, grandTotal)
	_ = f.SetCellStyle(sheet3, cellA, cellB, totalStyle)

	_ = f.SetColWidth(sheet3, "A", "A", 25) // Client
	_ = f.SetColWidth(sheet3, "B", "B", 35) // Total Hours
	_ = f.SetPanes(sheet3, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2"})

	// Set the first sheet as active by default
	f.SetActiveSheet(index1)

	return f, nil
}
