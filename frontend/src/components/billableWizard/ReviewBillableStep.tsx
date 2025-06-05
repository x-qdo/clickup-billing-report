import React from 'react';
import {
  CheckCircleIcon,
  CloudArrowDownIcon,
  ArrowLeftIcon,
  ArrowRightIcon,
  ExclamationTriangleIcon,
} from '@heroicons/react/24/outline';
import TaskDetails from '../shared/TaskDetails';

interface BillableReportTask {
  task_id: string;
  custom_id: string;
  name: string;
  priority?: string;
  tags?: string[];
  billable_hours: number;
  invoiced_hours: number;
  monthly_reported: number;
  reporter?: string;
  url: string;
}

interface BillableReportTotals {
  total_billable_hours?: number;
  total_amount?: number;
  billable_hours?: number;
  invoiced_hours?: number;
  monthly_reported?: number;
}

export interface BillableReportData {
  tasks?: BillableReportTask[];
  internal_tasks?: BillableReportTask[];
  totals?: BillableReportTotals;
}

interface ReviewBillableStepProps {
  reportData: BillableReportData | null;
  clientName: string;
  isLoading: boolean;
  error: string | null;
  onDownloadExcel: () => void;
  onNext: () => void;
  onBack: () => void;
}

const ReviewBillableStep: React.FC<ReviewBillableStepProps> = ({
  reportData,
  clientName,
  isLoading,
  error,
  onDownloadExcel,
  onNext,
  onBack,
}) => {
  const billableTasksCount = reportData?.tasks?.length || 0;
  const internalTasksCount = reportData?.internal_tasks?.length || 0;

  const totalBillableHours = reportData?.tasks?.reduce((sum, task) => sum + task.billable_hours, 0) || 0;
  const totalInternalHours = reportData?.internal_tasks?.reduce((sum, task) => sum + task.billable_hours, 0) || 0;

  return (
    <div className="space-y-6">
      <div className="text-center">
        <CheckCircleIcon className="mx-auto h-12 w-12 text-green-600" />
        <h3 className="mt-2 text-lg font-medium text-gray-900">
          Review Billable Tasks
        </h3>
        <p className="mt-1 text-sm text-gray-600">
          Review the billable tasks for <strong>{clientName || 'the selected client'}</strong> and verify totals.
        </p>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-300 rounded-md p-4">
          <div className="flex">
            <div className="flex-shrink-0">
              <ExclamationTriangleIcon
                className="h-5 w-5 text-red-400"
                aria-hidden="true"
              />
            </div>
            <div className="ml-3">
              <p className="text-sm text-red-700">{error}</p>
            </div>
          </div>
        </div>
      )}

      {isLoading && (
        <div className="text-center py-4">
          <p className="text-sm text-gray-500">Loading report data...</p>
          {/* You might want a spinner icon here */}
        </div>
      )}

      {!isLoading && reportData && (
        <>
          {/* Summary Section */}
          <div className="bg-white shadow sm:rounded-lg p-4 mb-6">
            <h3 className="text-base font-semibold leading-6 text-gray-900 mb-4">
              Report Totals
            </h3>
            <dl className="grid grid-cols-1 gap-x-4 gap-y-8 sm:grid-cols-2">
              <div className="sm:col-span-1">
                <dt className="text-sm font-medium text-gray-500">
                  Total Billable Hours
                </dt>
                <dd className="mt-1 text-2xl font-semibold tracking-tight text-gray-900">
                  {totalBillableHours.toFixed(1)}
                </dd>
              </div>
              <div className="sm:col-span-1">
                <dt className="text-sm font-medium text-gray-500">
                  Total Internal Hours
                </dt>
                <dd className="mt-1 text-2xl font-semibold tracking-tight text-gray-900">
                  {totalInternalHours.toFixed(1)}
                </dd>
              </div>
            </dl>
          </div>

          {/* Task Details Sections */}
          {(reportData.tasks && reportData.tasks.length > 0) ? (
            <TaskDetails
              title={`Billable Tasks (${billableTasksCount})`}
              tasks={reportData.tasks}
              type="billable"
              totalHours={totalBillableHours}
              className="mb-4"
            />
          ) : (
             <div className="bg-yellow-50 border border-yellow-300 rounded-md p-4 mb-4">
                <p className="text-sm text-yellow-700">No billable tasks found for this client.</p>
            </div>
          )}

          {(reportData.internal_tasks && reportData.internal_tasks.length > 0) ? (
            <TaskDetails
              title={`Internal Tasks (${internalTasksCount})`}
              tasks={reportData.internal_tasks}
              type="internal"
              totalHours={totalInternalHours}
              className="mb-4"
            />
          ) : (
            <div className="bg-gray-50 border border-gray-300 rounded-md p-4 mb-4">
                <p className="text-sm text-gray-700">No internal tasks found.</p>
            </div>
          )}
        </>
      )}

      {/* Action Buttons */}
      <div className="pt-6">
        <div className="flex justify-between">
          <button
            type="button"
            onClick={onBack}
            className="inline-flex items-center px-4 py-2 border border-gray-300 shadow-sm text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
          >
            <ArrowLeftIcon className="mr-2 h-5 w-5 text-gray-400" />
            Back
          </button>
          <div className="flex space-x-3">
            <button
              type="button"
              onClick={onDownloadExcel}
              disabled={isLoading || !reportData}
              className="inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
            >
              <CloudArrowDownIcon className="mr-2 h-5 w-5" />
              Download Excel
            </button>
            <button
              type="button"
              onClick={onNext}
              disabled={isLoading || !reportData}
              className="inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-green-600 hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-green-500 disabled:opacity-50"
            >
              Next
              <ArrowRightIcon className="ml-2 h-5 w-5" />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ReviewBillableStep;
