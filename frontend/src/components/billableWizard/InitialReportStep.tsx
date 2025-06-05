import React from 'react';
import {
  DocumentTextIcon,
  ExclamationTriangleIcon,
  ArrowPathIcon,
  ArrowRightIcon,
} from '@heroicons/react/24/outline';

interface InitialReportStepProps {
  clientName: string;
  isLoading: boolean;
  error: string | null;
  onGenerateInitialReport: () => void;
  // Consider adding onBack if users can go back from this step before generating.
  // onBack?: () => void;
}

const InitialReportStep: React.FC<InitialReportStepProps> = ({
  clientName,
  isLoading,
  error,
  onGenerateInitialReport,
}) => {
  return (
    <div className="space-y-6">
      <div className="text-center">
        <DocumentTextIcon className="mx-auto h-12 w-12 text-blue-600" />
        <h3 className="mt-2 text-lg font-medium text-gray-900">
          Generate Billable Report
        </h3>
        <p className="mt-1 text-sm text-gray-600">
          Create a billable report for <strong>{clientName || 'the selected client'}</strong> without updating ClickUp fields.
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

      <div className="bg-blue-50 border border-blue-200 rounded-md p-4">
        <h4 className="text-sm font-medium text-blue-900 mb-2">
          Report criteria:
        </h4>
        <ul className="text-sm text-blue-800 space-y-1 list-disc list-inside">
          <li>Tasks with BillableHours &gt; 0</li>
          <li>Tasks with MonthlyReported ≠ 0</li>
          <li>Separates internal vs. client tasks</li>
          <li>No ClickUp fields will be updated yet</li>
        </ul>
      </div>

      <div className="text-center pt-4">
        <button
          type="button"
          onClick={onGenerateInitialReport}
          disabled={isLoading || !clientName}
          className="inline-flex items-center justify-center px-6 py-3 border border-transparent text-base font-medium rounded-md shadow-sm text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {isLoading ? (
            <>
              <ArrowPathIcon className="animate-spin -ml-1 mr-3 h-5 w-5" />
              Generating Report...
            </>
          ) : (
            <>
              Generate Report
              <ArrowRightIcon className="ml-2 h-4 w-4" />
            </>
          )}
        </button>
      </div>
      {/* If onBack prop is added:
      <div className="mt-6 flex justify-start">
        <button
          type="button"
          onClick={onBack}
          className="text-sm font-medium text-blue-600 hover:text-blue-500"
        >
          Back
        </button>
      </div>
      */}
    </div>
  );
};

export default InitialReportStep;
