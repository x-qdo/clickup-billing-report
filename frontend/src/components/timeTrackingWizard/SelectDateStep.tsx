import React from "react";
import {
  ArrowRightIcon,
  ArrowPathIcon,
  ExclamationTriangleIcon,
  DocumentTextIcon,
} from "@heroicons/react/24/outline";

interface SelectDateStepProps {
  reportDate: string;
  isLoading: boolean;
  error: string | null;
  onReportDateChange: (newDate: string) => void;
  onGenerateReport: () => Promise<void>;
}

const SelectDateStep: React.FC<SelectDateStepProps> = ({
  reportDate,
  isLoading,
  error,
  onReportDateChange,
  onGenerateReport,
}) => {
  return (
    <div className="space-y-6">
      <div className="text-center">
        <DocumentTextIcon className="mx-auto h-12 w-12 text-blue-600" />
        <h3 className="mt-2 text-lg font-medium text-gray-900">
          Select Month & Generate Initial Report
        </h3>
        <p className="mt-1 text-sm text-gray-600">
          Choose the month and generate an initial time tracking report.
        </p>
        <p className="mt-1 text-sm text-gray-500">
          This report will use existing ClickUp data without making any updates
          yet.
        </p>
      </div>

      <div className="max-w-xs mx-auto">
        <label
          htmlFor="report-date"
          className="block text-sm font-medium text-gray-700 mb-2"
        >
          Report Month
        </label>
        <input
          type="month"
          id="report-date"
          value={reportDate}
          onChange={(e) => onReportDateChange(e.target.value)}
          className="block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
        />
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 rounded-md p-4 max-w-md mx-auto">
          <div className="flex">
            <ExclamationTriangleIcon className="h-5 w-5 text-red-400" />
            <div className="ml-3">
              <p className="text-sm text-red-800">{error}</p>
            </div>
          </div>
        </div>
      )}

      <div className="bg-blue-50 border border-blue-200 rounded-md p-4 max-w-md mx-auto">
        <h4 className="text-sm font-medium text-blue-900 mb-2">
          What happens next:
        </h4>
        <ul className="text-sm text-blue-800 space-y-1">
          <li>
            • Time entries will be aggregated by user and task for {reportDate}
          </li>
          <li>• Developer coefficients will be applied</li>
          <li>• Billable hours will be calculated</li>
          <li>• No ClickUp fields will be updated at this stage</li>
        </ul>
      </div>

      <div className="text-center">
        <button
          onClick={onGenerateReport}
          disabled={isLoading}
          className="inline-flex items-center px-6 py-3 border border-transparent text-base font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {isLoading ? (
            <>
              <ArrowPathIcon className="animate-spin -ml-1 mr-2 h-5 w-5" />
              Generating Report...
            </>
          ) : (
            <>
              Generate Report for {reportDate}
              <ArrowRightIcon className="ml-2 h-4 w-4" />
            </>
          )}
        </button>
      </div>
    </div>
  );
};

export default SelectDateStep;
