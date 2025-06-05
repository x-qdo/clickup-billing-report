import React from "react";
import { CheckCircleIcon, ArrowRightIcon } from "@heroicons/react/24/outline";

interface CompleteStepProps {
  reportDate: string;
  onResetWizard: () => void;
  onNavigateToBillable: () => void;
}

const CompleteStep: React.FC<CompleteStepProps> = ({
  reportDate,
  onResetWizard,
  onNavigateToBillable,
}) => {
  return (
    <div className="space-y-6">
      <div className="text-center">
        <CheckCircleIcon className="mx-auto h-12 w-12 text-green-600" />
        <h3 className="mt-2 text-lg font-medium text-gray-900">
          Workflow Complete!
        </h3>
        <p className="mt-1 text-sm text-gray-600">
          Time tracking report generated and ClickUp fields updated successfully
        </p>
      </div>

      <div className="bg-green-50 border border-green-200 rounded-md p-4">
        <h4 className="text-sm font-medium text-green-900 mb-2">
          What was accomplished:
        </h4>
        <ul className="text-sm text-green-800 space-y-1">
          <li>• ✓ Time tracking report generated for {reportDate}</li>
          <li>• ✓ Developer coefficients applied to time entries</li>
          <li>• ✓ BillableHours custom fields updated in ClickUp</li>
          <li>• ✓ Reports available for download</li>
        </ul>
      </div>

      <div className="bg-blue-50 border border-blue-200 rounded-md p-4">
        <h4 className="text-sm font-medium text-blue-900 mb-2">
          Next Steps:
        </h4>
        <p className="text-sm text-blue-800">
          You can now proceed to generate billable reports for specific clients
          using the updated BillableHours data.
        </p>
      </div>

      <div className="flex flex-col sm:flex-row gap-3">
        <button
          onClick={onResetWizard}
          className="flex-1 inline-flex items-center justify-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
        >
          Start New Report
        </button>
        <button
          onClick={onNavigateToBillable}
          className="flex-1 inline-flex items-center justify-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
        >
          Generate Billable Report
          <ArrowRightIcon className="ml-2 h-4 w-4" />
        </button>
      </div>
    </div>
  );
};

export default CompleteStep;
