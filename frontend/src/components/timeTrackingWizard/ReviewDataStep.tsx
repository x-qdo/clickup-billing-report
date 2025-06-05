import React, { useState } from "react";
import {
  CheckCircleIcon,
  ArrowLeftIcon,
  ArrowRightIcon,
  CloudArrowDownIcon,
  ExclamationTriangleIcon,
  ArrowsPointingOutIcon,
  ArrowsPointingInIcon,
} from "@heroicons/react/24/outline";
import type { TimeTrackingData } from "../../types/api/timeTrackingTypes";
import ReportSummary from "./ReportSummary";
import DeveloperReport from "./DeveloperReport";
import TaskReport from "./TaskReport";

interface ReviewDataStepProps {
  reportData: TimeTrackingData | null;
  isLoading: boolean;
  error: string | null;
  onDownloadExcel: () => Promise<void>;
  onNext: () => void;
  onBack: () => void;
}

const ReviewDataStep: React.FC<ReviewDataStepProps> = ({
  reportData,
  isLoading,
  error,
  onDownloadExcel,
  onNext,
  onBack,
}) => {
  const [taskReportExpanded, setTaskReportExpanded] = useState(false);

  const internalTagForTasks = "internal";

  // TODO: Implement actual data review components here.
  // For now, it only contains the checklist and navigation.
  // The "bug" about this section being empty might refer to missing data tables.

  return (
    <div className="space-y-6">
      <div className="text-center">
        <CheckCircleIcon className="mx-auto h-12 w-12 text-green-600" />
        <h3 className="mt-2 text-lg font-medium text-gray-900">
          Review & Verify Data
        </h3>
      </div>

      {/* Actual report data display */}
      {reportData ? (
        <div className="space-y-6">
          {reportData.totals && reportData.totals.length > 0 && (
            <ReportSummary totals={reportData.totals} />
          )}
          {reportData.personal_report &&
            reportData.personal_report.length > 0 && (
              <DeveloperReport personalReport={reportData.personal_report} />
            )}
          {reportData.final_report && reportData.final_report.length > 0 && (
            <div className={taskReportExpanded ? "col-span-full" : ""}>
              <div className="flex justify-end mb-2">
                <button
                  onClick={() => setTaskReportExpanded(!taskReportExpanded)}
                  className="p-1 text-gray-500 hover:text-gray-700 focus:outline-none"
                  aria-label={
                    taskReportExpanded
                      ? "Collapse task report"
                      : "Expand task report"
                  }
                >
                  {taskReportExpanded ? (
                    <ArrowsPointingInIcon className="h-5 w-5" />
                  ) : (
                    <ArrowsPointingOutIcon className="h-5 w-5" />
                  )}
                </button>
              </div>
              <TaskReport
                finalReport={reportData.final_report}
                isExpanded={taskReportExpanded}
                internalTag={internalTagForTasks}
              />
            </div>
          )}
          {(!reportData.totals || reportData.totals.length === 0) &&
            (!reportData.personal_report ||
              reportData.personal_report.length === 0) &&
            (!reportData.final_report ||
              reportData.final_report.length === 0) && (
              <div className="bg-white shadow sm:rounded-lg p-4">
                <h3 className="text-lg leading-6 font-medium text-gray-900">
                  No Data to Display
                </h3>
                <p className="mt-1 text-sm text-gray-500">
                  The generated report does not contain any data for totals,
                  personal reports, or final task reports.
                </p>
              </div>
            )}
        </div>
      ) : (
        <div className="bg-gray-50 p-4 rounded-md shadow text-center">
          <h4 className="text-md font-semibold text-gray-700 mb-2">
            No Report Data
          </h4>
          <p className="text-sm text-gray-600">
            Report data is not yet available or has not been loaded.
          </p>
        </div>
      )}

      <div className="mt-6 bg-yellow-50 border border-yellow-200 rounded-md p-4">
        <h4 className="text-sm font-medium text-yellow-900 mb-2">
          Review Checklist:
        </h4>
        <ul className="text-sm text-yellow-800 space-y-1">
          <li>
            • Verify personal time summary shows expected hours per developer
          </li>
          <li>• Check that coefficients were applied correctly</li>
          <li>• Ensure tasks are assigned to the correct clients</li>
          <li>• Make any necessary corrections in ClickUp before proceeding</li>
        </ul>
      </div>

      {/* Download Button and Error Display */}
      <div className="mt-6 flex flex-col sm:flex-row items-center justify-between space-y-4 sm:space-y-0 sm:space-x-4">
        <button
          onClick={onDownloadExcel}
          disabled={isLoading || !reportData}
          className="w-full sm:w-auto inline-flex items-center justify-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-green-600 hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-green-500 disabled:bg-gray-300 disabled:cursor-not-allowed"
        >
          <CloudArrowDownIcon className="mr-2 h-5 w-5" />
          Download Initial Excel
        </button>
        {error && (
          <div className="flex items-center text-sm text-red-600 bg-red-50 p-3 rounded-md">
            <ExclamationTriangleIcon className="mr-2 h-5 w-5 text-red-500" />
            <span>Error: {error}</span>
          </div>
        )}
      </div>

      <div className="mt-8 flex justify-between">
        <button
          onClick={onBack}
          className="inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
        >
          <ArrowLeftIcon className="mr-2 h-4 w-4" />
          Back
        </button>
        <button
          onClick={onNext}
          className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
        >
          Data Verified
          <ArrowRightIcon className="ml-2 h-4 w-4" />
        </button>
      </div>
    </div>
  );
};

export default ReviewDataStep;
