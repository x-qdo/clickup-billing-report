import React from 'react';
import { CheckCircleIcon, ArrowRightIcon } from '@heroicons/react/24/outline';

interface BillableCompleteStepProps {
  clientName: string;
  onResetWizard: () => void;
  onNavigateToTimeTracking: () => void;
}

const BillableCompleteStep: React.FC<BillableCompleteStepProps> = ({
  clientName,
  onResetWizard,
  onNavigateToTimeTracking,
}) => {
  return (
    <div className="space-y-6">
      <div className="text-center">
        <CheckCircleIcon className="mx-auto h-16 w-16 text-green-500" />
        <h3 className="mt-4 text-2xl font-semibold text-gray-900">
          Workflow Complete!
        </h3>
        <p className="mt-2 text-md text-gray-600">
          Billable report generated and tasks marked as invoiced successfully for <strong>{clientName || 'the client'}</strong>.
        </p>
      </div>

      <div className="bg-green-50 border border-green-200 rounded-md p-6 shadow-sm">
        <h4 className="text-md font-medium text-green-800 mb-3">
          What was accomplished:
        </h4>
        <ul className="text-sm text-green-700 space-y-2 list-disc list-inside">
          <li>Billable report generated for {clientName || 'the client'}.</li>
          <li>Tasks filtered based on BillableHours and MonthlyReported.</li>
          <li>Internal tasks separated from client tasks.</li>
          <li>InvoicedHours updated to match BillableHours.</li>
          <li>Tasks marked as invoiced for future reporting periods.</li>
        </ul>
      </div>

      <div className="bg-blue-50 border border-blue-200 rounded-md p-6 shadow-sm">
        <h4 className="text-md font-medium text-blue-800 mb-3">
          Next Steps:
        </h4>
        <p className="text-sm text-blue-700">
          You can now generate reports for other clients or start a new time tracking
          report for the next billing period.
        </p>
      </div>

      <div className="flex flex-col sm:flex-row gap-4 pt-4">
        <button
          type="button"
          onClick={onResetWizard}
          className="flex-1 inline-flex items-center justify-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 shadow-sm"
        >
          New Billable Report
        </button>
        <button
          type="button"
          onClick={onNavigateToTimeTracking}
          className="flex-1 inline-flex items-center justify-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 shadow-sm"
        >
          Time Tracking Report
          <ArrowRightIcon className="ml-2 h-4 w-4" />
        </button>
      </div>
    </div>
  );
};

export default BillableCompleteStep;
