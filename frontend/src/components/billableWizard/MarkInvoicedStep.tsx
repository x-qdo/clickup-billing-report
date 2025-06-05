import React from 'react';
import {
  CurrencyDollarIcon,
  ExclamationTriangleIcon,
  ArrowPathIcon,
  ArrowRightIcon,
  CloudArrowDownIcon,
  ArrowLeftIcon,
  CheckCircleIcon, // Added
} from '@heroicons/react/24/outline';

interface MarkInvoicedStepProps {
  clientName: string;
  isLoading: boolean;
  error: string | null;
  finalReportGenerated: boolean;
  onGenerateFinalReport: () => void;
  onDownloadFinalExcel: () => void;
  onBack: () => void;
}

const MarkInvoicedStep: React.FC<MarkInvoicedStepProps> = ({
  clientName,
  isLoading,
  error,
  finalReportGenerated,
  onGenerateFinalReport,
  onDownloadFinalExcel,
  onBack,
}) => {
  return (
    <div className="space-y-6">
      <div className="text-center">
        <CurrencyDollarIcon className="mx-auto h-12 w-12 text-orange-600" />
        <h3 className="mt-2 text-lg font-medium text-gray-900">
          Mark as Invoiced
        </h3>
        <p className="mt-1 text-sm text-gray-600">
          Update InvoicedHours field in ClickUp for <strong>{clientName || 'the selected client'}</strong> to track billing progress.
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

      <div className="bg-orange-50 border border-orange-200 rounded-md p-4">
        <h4 className="text-sm font-medium text-orange-900 mb-2">
          This action will:
        </h4>
        <ul className="text-sm text-orange-800 space-y-1 list-disc list-inside">
          <li>Set InvoicedHours = BillableHours for reported non-internal tasks.</li>
          <li>Mark tasks as invoiced for future reporting periods.</li>
          <li>Generate a final report with updated data.</li>
          <li>Help track billing progress across months.</li>
        </ul>
      </div>

      <div className="text-center pt-4">
        {finalReportGenerated ? (
          <div className="space-y-4">
            <div className="text-green-600 font-medium flex items-center justify-center">
              <CheckCircleIcon className="h-6 w-6 mr-2 text-green-500" />
              InvoicedHours updated successfully!
            </div>
            <button
              type="button"
              onClick={onDownloadFinalExcel}
              disabled={isLoading}
              className="inline-flex items-center justify-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <CloudArrowDownIcon className="h-5 w-5 mr-2" />
              Download Final Excel
            </button>
          </div>
        ) : (
          <button
            type="button"
            onClick={onGenerateFinalReport}
            disabled={isLoading || !clientName}
            className="inline-flex items-center justify-center px-6 py-3 border border-transparent text-base font-medium rounded-md shadow-sm text-white bg-orange-600 hover:bg-orange-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-orange-500 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isLoading ? (
              <>
                <ArrowPathIcon className="animate-spin -ml-1 mr-3 h-5 w-5" />
                Updating ClickUp...
              </>
            ) : (
              <>
                Mark as Invoiced & Generate Final Report
                <ArrowRightIcon className="ml-2 h-4 w-4" />
              </>
            )}
          </button>
        )}
      </div>

      <div className="flex justify-start pt-6">
        <button
          type="button"
          onClick={onBack}
          className="inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
        >
          <ArrowLeftIcon className="mr-2 h-4 w-4" />
          Back
        </button>
      </div>
    </div>
  );
};

export default MarkInvoicedStep;
