import React from "react";
import type { JobStatus } from "../../services/api";
import {
  ClockIcon,
  CheckCircleIcon,
  XCircleIcon,
  ArrowPathIcon,
} from "@heroicons/react/24/outline";

interface JobProgressProps {
  status: JobStatus | null;
  elapsedTime: number;
  error?: string | null;
  onCancel?: () => void;
}

const formatElapsedTime = (seconds: number): string => {
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  if (mins > 0) {
    return `${mins}m ${secs}s`;
  }
  return `${secs}s`;
};

const statusConfig: Record<
  JobStatus,
  { label: string; color: string; icon: React.ElementType; animate?: boolean }
> = {
  pending: {
    label: "Starting...",
    color: "text-yellow-600",
    icon: ClockIcon,
  },
  running: {
    label: "Processing...",
    color: "text-blue-600",
    icon: ArrowPathIcon,
    animate: true,
  },
  completed: {
    label: "Completed",
    color: "text-green-600",
    icon: CheckCircleIcon,
  },
  failed: {
    label: "Failed",
    color: "text-red-600",
    icon: XCircleIcon,
  },
};

const JobProgress: React.FC<JobProgressProps> = ({
  status,
  elapsedTime,
  error,
  onCancel,
}) => {
  if (!status) return null;

  const config = statusConfig[status];
  const Icon = config.icon;

  return (
    <div className="bg-gray-50 rounded-lg p-4 border border-gray-200">
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-3">
          <Icon
            className={`w-6 h-6 ${config.color} ${config.animate ? "animate-spin" : ""}`}
          />
          <div>
            <p className={`font-medium ${config.color}`}>{config.label}</p>
            {(status === "pending" || status === "running") && (
              <p className="text-sm text-gray-500">
                Elapsed time: {formatElapsedTime(elapsedTime)}
              </p>
            )}
            {status === "failed" && error && (
              <p className="text-sm text-red-500 mt-1">{error}</p>
            )}
          </div>
        </div>

        {onCancel && (status === "pending" || status === "running") && (
          <button
            type="button"
            onClick={onCancel}
            className="text-sm text-gray-500 hover:text-gray-700 underline"
          >
            Cancel
          </button>
        )}
      </div>

      {(status === "pending" || status === "running") && (
        <div className="mt-3">
          <div className="w-full bg-gray-200 rounded-full h-2">
            <div
              className="bg-blue-600 h-2 rounded-full animate-pulse"
              style={{ width: "100%" }}
            />
          </div>
          <p className="text-xs text-gray-500 mt-2">
            This may take a few minutes depending on the amount of data...
          </p>
        </div>
      )}
    </div>
  );
};

export default JobProgress;
