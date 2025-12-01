package job

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/x-qdo/clickup-billing-report/internal/config"
)

const (
	DefaultJobTTLDays = 7
)

type Service struct {
	store      config.DataStore
	sqsClient  *sqs.Client
	s3Client   *s3.Client
	queueURL   string
	bucketName string
	log        *logrus.Entry
}

func NewService(awsCfg aws.Config, store config.DataStore, logger *logrus.Logger) *Service {
	queueURL := os.Getenv("SQS_JOBS_QUEUE_URL")
	bucketName := os.Getenv("S3_JOBS_BUCKET")

	return &Service{
		store:      store,
		sqsClient:  sqs.NewFromConfig(awsCfg),
		s3Client:   s3.NewFromConfig(awsCfg),
		queueURL:   queueURL,
		bucketName: bucketName,
		log:        logger.WithField("component", "JobService"),
	}
}

// CreateJob creates a new job record and sends it to SQS for processing.
func (s *Service) CreateJob(ctx context.Context, jobType, inputJSON, userID string) (*config.Job, error) {
	jobID := uuid.New().String()
	now := time.Now().UTC()
	ttl := now.AddDate(0, 0, DefaultJobTTLDays).Unix()

	job := config.Job{
		JobID:     jobID,
		Type:      jobType,
		Status:    config.JobStatusPending,
		Input:     inputJSON,
		UserID:    userID,
		CreatedAt: now,
		UpdatedAt: now,
		TTL:       ttl,
	}

	if err := s.store.SaveJob(ctx, job); err != nil {
		s.log.WithError(err).WithField("job_id", jobID).Error("Failed to save job")
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	if err := s.sendToQueue(ctx, jobID); err != nil {
		s.log.WithError(err).WithField("job_id", jobID).Error("Failed to send job to queue")
		// Update job status to failed
		_ = s.store.UpdateJobStatus(ctx, jobID, config.JobStatusFailed, "", "Failed to queue job")
		return nil, fmt.Errorf("failed to queue job: %w", err)
	}

	s.log.WithFields(logrus.Fields{
		"job_id":   jobID,
		"job_type": jobType,
		"user_id":  userID,
	}).Info("Job created and queued")

	return &job, nil
}

// GetJob retrieves a job by ID.
func (s *Service) GetJob(ctx context.Context, jobID string) (*config.Job, error) {
	return s.store.GetJob(ctx, jobID)
}

// UpdateJobRunning marks a job as running.
func (s *Service) UpdateJobRunning(ctx context.Context, jobID string) error {
	return s.store.UpdateJobStatus(ctx, jobID, config.JobStatusRunning, "", "")
}

// UpdateJobCompleted marks a job as completed with the result.
func (s *Service) UpdateJobCompleted(ctx context.Context, jobID, outputJSON string) error {
	return s.store.UpdateJobStatus(ctx, jobID, config.JobStatusCompleted, outputJSON, "")
}

// UpdateJobFailed marks a job as failed with an error message.
func (s *Service) UpdateJobFailed(ctx context.Context, jobID, errMsg string) error {
	return s.store.UpdateJobStatus(ctx, jobID, config.JobStatusFailed, "", errMsg)
}

func (s *Service) sendToQueue(ctx context.Context, jobID string) error {
	if s.queueURL == "" {
		return fmt.Errorf("SQS queue URL not configured")
	}

	message := map[string]string{"job_id": jobID}
	messageBody, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	input := &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.queueURL),
		MessageBody: aws.String(string(messageBody)),
	}

	_, err = s.sqsClient.SendMessage(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to send SQS message: %w", err)
	}

	s.log.WithField("job_id", jobID).Debug("Job sent to SQS queue")
	return nil
}

// UploadFileToS3 uploads a file to S3 and returns a presigned download URL.
func (s *Service) UploadFileToS3(ctx context.Context, jobID, filename string, data []byte, contentType string) (string, error) {
	if s.bucketName == "" {
		return "", fmt.Errorf("S3 bucket not configured")
	}

	key := fmt.Sprintf("jobs/%s/%s", jobID, filename)

	_, err := s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(key),
		Body:        bytesReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	presignClient := s3.NewPresignClient(s.s3Client)
	presignedReq, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(24*time.Hour))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	s.log.WithFields(logrus.Fields{
		"job_id":   jobID,
		"filename": filename,
		"key":      key,
	}).Debug("File uploaded to S3")

	return presignedReq.URL, nil
}

// bytesReader wraps a byte slice to implement io.Reader
func bytesReader(data []byte) *bytesReaderWrapper {
	return &bytesReaderWrapper{data: data}
}

type bytesReaderWrapper struct {
	data   []byte
	offset int
}

func (r *bytesReaderWrapper) Read(p []byte) (n int, err error) {
	if r.offset >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.offset:])
	r.offset += n
	return n, nil
}
