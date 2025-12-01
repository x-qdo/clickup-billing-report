import { useState, useCallback, useRef, useEffect } from "react";
import {
  apiService,
  type JobType,
  type JobStatus,
  type JobStatusResponse,
  type JobFileResult,
} from "../services/api";

interface UseAsyncJobState<T> {
  status: JobStatus | null;
  jobId: string | null;
  result: T | null;
  error: string | null;
  isLoading: boolean;
  elapsedTime: number;
}

interface UseAsyncJobReturn<T> extends UseAsyncJobState<T> {
  startJob: (type: JobType, params: Record<string, unknown>) => Promise<void>;
  reset: () => void;
}

const POLL_INTERVAL_MS = 2000;
const MAX_POLL_INTERVAL_MS = 10000;

export function useAsyncJob<T = unknown>(): UseAsyncJobReturn<T> {
  const [state, setState] = useState<UseAsyncJobState<T>>({
    status: null,
    jobId: null,
    result: null,
    error: null,
    isLoading: false,
    elapsedTime: 0,
  });

  const pollTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const pollIntervalRef = useRef(POLL_INTERVAL_MS);
  const startTimeRef = useRef<number | null>(null);
  const elapsedIntervalRef = useRef<NodeJS.Timeout | null>(null);

  const clearTimers = useCallback(() => {
    if (pollTimeoutRef.current) {
      clearTimeout(pollTimeoutRef.current);
      pollTimeoutRef.current = null;
    }
    if (elapsedIntervalRef.current) {
      clearInterval(elapsedIntervalRef.current);
      elapsedIntervalRef.current = null;
    }
  }, []);

  useEffect(() => {
    return () => clearTimers();
  }, [clearTimers]);

  const pollJobStatus = useCallback(
    async (jobId: string) => {
      try {
        const response = await apiService.getJobStatus(jobId);
        const jobData: JobStatusResponse = response.data;

        if (
          jobData.status === "completed" ||
          jobData.status === "failed"
        ) {
          clearTimers();

          setState((prev) => ({
            ...prev,
            status: jobData.status,
            isLoading: false,
            result: jobData.status === "completed" ? (jobData.result as T) : null,
            error: jobData.status === "failed" ? jobData.error || "Job failed" : null,
          }));
          return;
        }

        // Still pending or running, continue polling with exponential backoff
        setState((prev) => ({
          ...prev,
          status: jobData.status,
        }));

        pollIntervalRef.current = Math.min(
          pollIntervalRef.current * 1.5,
          MAX_POLL_INTERVAL_MS,
        );

        pollTimeoutRef.current = setTimeout(
          () => pollJobStatus(jobId),
          pollIntervalRef.current,
        );
      } catch (err: unknown) {
        clearTimers();
        setState((prev) => ({
          ...prev,
          isLoading: false,
          error: apiService.getErrorMessage(err),
        }));
      }
    },
    [clearTimers],
  );

  const startJob = useCallback(
    async (type: JobType, params: Record<string, unknown>) => {
      clearTimers();
      pollIntervalRef.current = POLL_INTERVAL_MS;
      startTimeRef.current = Date.now();

      setState({
        status: "pending",
        jobId: null,
        result: null,
        error: null,
        isLoading: true,
        elapsedTime: 0,
      });

      // Start elapsed time counter
      elapsedIntervalRef.current = setInterval(() => {
        if (startTimeRef.current) {
          setState((prev) => ({
            ...prev,
            elapsedTime: Math.floor((Date.now() - startTimeRef.current!) / 1000),
          }));
        }
      }, 1000);

      try {
        const response = await apiService.createJob(type, params);
        const { job_id, status } = response.data;

        setState((prev) => ({
          ...prev,
          jobId: job_id,
          status,
        }));

        // Start polling
        pollTimeoutRef.current = setTimeout(
          () => pollJobStatus(job_id),
          pollIntervalRef.current,
        );
      } catch (err: unknown) {
        clearTimers();
        setState((prev) => ({
          ...prev,
          isLoading: false,
          error: apiService.getErrorMessage(err),
        }));
      }
    },
    [clearTimers, pollJobStatus],
  );

  const reset = useCallback(() => {
    clearTimers();
    setState({
      status: null,
      jobId: null,
      result: null,
      error: null,
      isLoading: false,
      elapsedTime: 0,
    });
  }, [clearTimers]);

  return {
    ...state,
    startJob,
    reset,
  };
}

// Type guard for file result
export function isJobFileResult(result: unknown): result is JobFileResult {
  return (
    typeof result === "object" &&
    result !== null &&
    "download_url" in result &&
    "filename" in result
  );
}
