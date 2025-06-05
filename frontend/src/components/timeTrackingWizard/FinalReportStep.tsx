import React from "react";
import {
  ArrowPathIcon,
  ExclamationTriangleIcon,
  CloudArrowDownIcon,
  ArrowRightIcon,
  ArrowLeftIcon,
} from "@heroicons/react/24/outline";

interface FinalReportStepProps {
  isLoading: boolean;
  error: string | null;
  finalReportGenerated: boolean;
  onGenerateFinalReport: () => Promise<void>;
  onDownloadFinalExcel: () => void;
  onBack: () => void;
}

const FinalReportStep: React.FC<FinalReportStepProps> = ({
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
        <ArrowPathIcon className="mx-auto h-12 w-12 text-orange-600" />
        <h3 className="mt-2 text-lg font-medium text-gray-900">
          Update ClickUp Fields
        </h3>
        <p className="mt-1 text-sm text-gray-600">
          Generate the final report and update BillableHours custom fields in
          ClickUp
        </p>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 rounded-md p-4">
          <div className="flex">
            <ExclamationTriangleIcon className="h-5 w-5 text-red-400" />
            <div className="ml-3">
              <p className="text-sm text-red-800">{error}</p>
            </div>
          </div>
        </div>
      )}

      <div className="bg-orange-50 border border-orange-200 rounded-md p-4">
        <h4 className="text-sm font-medium text-orange-900 mb-2">
          This will:
        </h4>
        <ul className="text-sm text-orange-800 space-y-1">
          <li>• Calculate BillableHours = InvoicedHours + AdjustedDuration</li>
          <li>• Update the BillableHours custom field in ClickUp tasks</li>
          <li>• Generate a final report with updated data</li>
        </ul>
      </div>

      <div className="text-center">
        {finalReportGenerated ? (
          <div className="space-y-4">
            <div className="text-green-600 font-medium">
              ✓ ClickUp fields updated successfully
            </div>
            <button
              onClick={onDownloadFinalExcel}
              disabled={isLoading}
              className="inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            >
              <CloudArrowDownIcon className="h-4 w-4 mr-2" />
              Download Final Excel
            </button>
          </div>
        ) : (
          <button
            onClick={onGenerateFinalReport}
            disabled={isLoading}
            className="inline-flex items-center px-6 py-3 border border-transparent text-base font-medium rounded-md text-white bg-orange-600 hover:bg-orange-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-orange-500 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isLoading ? (
              <>
                <ArrowPathIcon className="animate-spin -ml-1 mr-2 h-5 w-5" />
                Updating ClickUp...
              </>
            ) : (
              <>
                Update ClickUp Fields
                <ArrowRightIcon className="ml-2 h-4 w-4" />
              </>
            )}
          </button>
        )}
      </div>

      <div className="flex justify-start"> {/* Changed justify-between to justify-start as only one button */}
        <button
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

export default FinalReportStep;
