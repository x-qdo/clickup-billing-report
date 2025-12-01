package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/sirupsen/logrus"

	"github.com/x-qdo/clickup-billing-report/internal/config"
	"github.com/x-qdo/clickup-billing-report/internal/job"
	"github.com/x-qdo/clickup-billing-report/internal/report"
	"github.com/x-qdo/clickup-billing-report/internal/storage"
)

var (
	appConf       *config.AppConfig
	jobService    *job.Service
	reportService *report.Service
	awsCfg        aws.Config
)

func init() {
	var err error
	ctx := context.Background()

	awsCfg, err = awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		panic(fmt.Sprintf("FATAL: Failed to load AWS config: %v", err))
	}

	initLogger := logrus.New()
	initLogger.SetLevel(logrus.InfoLevel)
	initLogger.SetFormatter(&logrus.JSONFormatter{})
	logEntry := initLogger.WithField("service", "job-worker-init")

	store, err := storage.NewDynamoDBStore(awsCfg, initLogger)
	if err != nil {
		panic(fmt.Sprintf("FATAL: Failed to initialize DynamoDB store: %v", err))
	}

	appConf, err = config.LoadConfig(ctx, store)
	if err != nil {
		logEntry.WithError(err).Error("Error loading config during init")
		if appConf == nil {
			panic(fmt.Sprintf("FATAL: Failed to load critical config: %v", err))
		}
	}
	if appConf.Store == nil {
		panic("FATAL: DataStore is nil after config load")
	}

	reportService = report.NewService(appConf)
	jobService = job.NewService(awsCfg, appConf.Store, appConf.Logger)
	appConf.Logger.Info("Job worker initialized successfully")
}

// SQSMessage represents the message body from SQS
type SQSMessage struct {
	JobID string `json:"job_id"`
}

func handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	log := appConf.Logger.WithField("handler", "job_worker")
	log.WithField("record_count", len(sqsEvent.Records)).Info("Processing SQS batch")

	for _, record := range sqsEvent.Records {
		if err := processRecord(ctx, record, log); err != nil {
			log.WithError(err).WithField("message_id", record.MessageId).Error("Failed to process record")
			// Return error to mark batch as failed for retry
			return err
		}
	}

	return nil
}

func processRecord(ctx context.Context, record events.SQSMessage, log *logrus.Entry) error {
	var msg SQSMessage
	if err := json.Unmarshal([]byte(record.Body), &msg); err != nil {
		return fmt.Errorf("failed to unmarshal SQS message: %w", err)
	}

	jobID := msg.JobID
	log = log.WithField("job_id", jobID)
	log.Info("Processing job")

	// Get job from DynamoDB
	jobData, err := jobService.GetJob(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	// Update status to running
	if err := jobService.UpdateJobRunning(ctx, jobID); err != nil {
		log.WithError(err).Warn("Failed to update job status to running")
	}

	// Process based on job type
	var outputJSON string
	var processErr error

	switch jobData.Type {
	case config.JobTypeTimetrack:
		outputJSON, processErr = processTimetrackJob(ctx, jobData, log)
	case config.JobTypeBillable:
		outputJSON, processErr = processBillableJob(ctx, jobData, log)
	default:
		processErr = fmt.Errorf("unknown job type: %s", jobData.Type)
	}

	if processErr != nil {
		log.WithError(processErr).Error("Job processing failed")
		if err := jobService.UpdateJobFailed(ctx, jobID, processErr.Error()); err != nil {
			log.WithError(err).Error("Failed to update job status to failed")
		}
		return nil // Don't return error to avoid retry for business logic failures
	}

	// Update job as completed
	if err := jobService.UpdateJobCompleted(ctx, jobID, outputJSON); err != nil {
		log.WithError(err).Error("Failed to update job status to completed")
		return err
	}

	log.Info("Job completed successfully")
	return nil
}

func processTimetrackJob(ctx context.Context, jobData *config.Job, log *logrus.Entry) (string, error) {
	var input config.TimetrackJobInput
	if err := json.Unmarshal([]byte(jobData.Input), &input); err != nil {
		return "", fmt.Errorf("failed to parse job input: %w", err)
	}

	log.WithFields(logrus.Fields{
		"report_date":      input.ReportDate,
		"refresh_billable": input.RefreshBillable,
		"format":           input.Format,
	}).Info("Processing timetrack job")

	selectedMonth, err := time.Parse("2006-01", input.ReportDate)
	if err != nil {
		return "", fmt.Errorf("invalid report_date format: %w", err)
	}

	reportInput := report.TimeTrackingInput{
		SelectedMonth:   selectedMonth,
		RefreshBillable: input.RefreshBillable,
		ClickUpToken:    input.ClickUpToken,
	}

	reportOutput, err := reportService.GenerateTimeTrackingReport(ctx, reportInput)
	if err != nil {
		return "", fmt.Errorf("failed to generate report: %w", err)
	}

	// Handle Excel format - upload to S3
	if input.Format == "excel" {
		excelFile, err := report.GenerateTimeTrackingExcel(reportOutput, selectedMonth)
		if err != nil {
			return "", fmt.Errorf("failed to generate Excel: %w", err)
		}

		var buf bytes.Buffer
		if err := excelFile.Write(&buf); err != nil {
			return "", fmt.Errorf("failed to write Excel: %w", err)
		}

		filename := fmt.Sprintf("time_tracking_report_%s.xlsx", selectedMonth.Format("2006-01"))
		contentType := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

		downloadURL, err := jobService.UploadFileToS3(ctx, jobData.JobID, filename, buf.Bytes(), contentType)
		if err != nil {
			return "", fmt.Errorf("failed to upload to S3: %w", err)
		}

		result := config.JobFileOutput{
			DownloadURL: downloadURL,
			Filename:    filename,
		}
		outputBytes, _ := json.Marshal(result)
		return string(outputBytes), nil
	}

	// JSON format - return report data directly
	outputBytes, err := json.Marshal(reportOutput)
	if err != nil {
		return "", fmt.Errorf("failed to marshal output: %w", err)
	}

	return string(outputBytes), nil
}

func processBillableJob(ctx context.Context, jobData *config.Job, log *logrus.Entry) (string, error) {
	var input config.BillableJobInput
	if err := json.Unmarshal([]byte(jobData.Input), &input); err != nil {
		return "", fmt.Errorf("failed to parse job input: %w", err)
	}

	log.WithFields(logrus.Fields{
		"client_name":      input.ClientName,
		"refresh_invoiced": input.RefreshInvoiced,
		"format":           input.Format,
	}).Info("Processing billable job")

	reportInput := report.BillableReportInput{
		ClientName:      input.ClientName,
		RefreshInvoiced: input.RefreshInvoiced,
		ClickUpToken:    input.ClickUpToken,
	}

	reportOutput, err := reportService.GenerateBillableReport(ctx, reportInput)
	if err != nil {
		return "", fmt.Errorf("failed to generate report: %w", err)
	}

	// Handle Excel format - upload to S3
	if input.Format == "excel" {
		excelFile, err := report.GenerateBillableExcel(reportOutput, input.ClientName)
		if err != nil {
			return "", fmt.Errorf("failed to generate Excel: %w", err)
		}

		var buf bytes.Buffer
		if err := excelFile.Write(&buf); err != nil {
			return "", fmt.Errorf("failed to write Excel: %w", err)
		}

		filename := fmt.Sprintf("billable_report_%s_%s.xlsx", input.ClientName, time.Now().Format("20060102"))
		contentType := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

		downloadURL, err := jobService.UploadFileToS3(ctx, jobData.JobID, filename, buf.Bytes(), contentType)
		if err != nil {
			return "", fmt.Errorf("failed to upload to S3: %w", err)
		}

		result := config.JobFileOutput{
			DownloadURL: downloadURL,
			Filename:    filename,
		}
		outputBytes, _ := json.Marshal(result)
		return string(outputBytes), nil
	}

	// JSON format - return report data directly
	outputBytes, err := json.Marshal(reportOutput)
	if err != nil {
		return "", fmt.Errorf("failed to marshal output: %w", err)
	}

	return string(outputBytes), nil
}

func main() {
	lambda.Start(handler)
}
