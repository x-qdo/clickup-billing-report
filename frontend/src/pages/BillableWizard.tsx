import { useState, useEffect } from 'react';
import { apiService } from '../services/api';
import {
  CheckCircleIcon,
  CurrencyDollarIcon,
  DocumentTextIcon,
  ExclamationTriangleIcon,
  ArrowRightIcon,
  ArrowLeftIcon,
  CloudArrowDownIcon,
  ArrowPathIcon,
  BuildingOfficeIcon,
} from '@heroicons/react/24/outline';

interface WizardStep {
  id: string;
  title: string;
  completed: boolean;
}

interface BillableReportData {
  tasks: any[];
  internal_tasks: any[];
  totals: any;
}

interface WizardState {
  currentStep: number;
  clientName: string;
  initialReportGenerated: boolean;
  finalReportGenerated: boolean;
  reportData: BillableReportData | null;
  isLoading: boolean;
  error: string | null;
}

const BillableWizard: React.FC = () => {
  const [state, setState] = useState<WizardState>({
    currentStep: 0,
    clientName: '',
    initialReportGenerated: false,
    finalReportGenerated: false,
    reportData: null,
    isLoading: false,
    error: null,
  });

  // Mock client list - in real app, this would come from API
  const [availableClients] = useState([
    'Acme Corp',
    'Tech Solutions',
    'Global Industries',
    'Digital Ventures',
    'Innovation Labs',
  ]);

  const steps: WizardStep[] = [
    {
      id: 'select-client',
      title: 'Select Client',
      completed: state.currentStep > 0 && state.clientName !== '',
    },
    {
      id: 'initial-report',
      title: 'Generate Billable Report',
      completed: state.initialReportGenerated,
    },
    {
      id: 'review-billable',
      title: 'Review Billable Tasks',
      completed: state.currentStep > 2 && state.initialReportGenerated,
    },
    {
      id: 'final-report',
      title: 'Mark as Invoiced',
      completed: state.finalReportGenerated,
    },
    {
      id: 'complete',
      title: 'Complete',
      completed: state.finalReportGenerated && state.currentStep >= 4,
    },
  ];

  // Load state from localStorage on component mount
  useEffect(() => {
    const savedState = localStorage.getItem('billableWizardState');
    if (savedState) {
      try {
        const parsed = JSON.parse(savedState);
        setState(prevState => ({ ...prevState, ...parsed, isLoading: false, error: null }));
      } catch (error) {
        console.error('Failed to parse saved wizard state:', error);
      }
    }
  }, []);

  // Save state to localStorage whenever it changes
  useEffect(() => {
    const stateToSave = {
      currentStep: state.currentStep,
      clientName: state.clientName,
      initialReportGenerated: state.initialReportGenerated,
      finalReportGenerated: state.finalReportGenerated,
      reportData: state.reportData,
    };
    localStorage.setItem('billableWizardState', JSON.stringify(stateToSave));
  }, [state.currentStep, state.clientName, state.initialReportGenerated, state.finalReportGenerated, state.reportData]);

  const updateState = (updates: Partial<WizardState>) => {
    setState(prevState => ({ ...prevState, ...updates }));
  };

  const handleNext = () => {
    if (state.currentStep < steps.length - 1) {
      updateState({ currentStep: state.currentStep + 1 });
    }
  };

  const handleBack = () => {
    if (state.currentStep > 0) {
      updateState({ currentStep: state.currentStep - 1 });
    }
  };

  const handleGenerateInitialReport = async () => {
    updateState({ isLoading: true, error: null });
    try {
      const response = await apiService.generateBillableReport({
        client_name: state.clientName,
        refresh_invoiced: false,
        format: 'json',
      });
      
      updateState({
        isLoading: false,
        reportData: response.data,
        initialReportGenerated: true,
      });
      
      // Auto-advance to next step
      setTimeout(() => handleNext(), 500);
    } catch (error: any) {
      updateState({
        isLoading: false,
        error: error.response?.data?.message || 'Failed to generate report',
      });
    }
  };

  const handleDownloadExcel = async () => {
    updateState({ isLoading: true, error: null });
    try {
      const response = await apiService.generateBillableReport({
        client_name: state.clientName,
        refresh_invoiced: false,
        format: 'excel',
      });
      
      const filename = `billable_report_${state.clientName.replace(/\s+/g, '_')}.xlsx`;
      apiService.downloadFile(response.data, filename);
      
      updateState({ isLoading: false });
    } catch (error: any) {
      updateState({
        isLoading: false,
        error: error.response?.data?.message || 'Failed to download Excel file',
      });
    }
  };

  const handleGenerateFinalReport = async () => {
    updateState({ isLoading: true, error: null });
    try {
      const response = await apiService.generateBillableReport({
        client_name: state.clientName,
        refresh_invoiced: true,
        format: 'json',
      });
      
      updateState({
        isLoading: false,
        reportData: response.data,
        finalReportGenerated: true,
      });
      
      // Auto-advance to next step
      setTimeout(() => handleNext(), 500);
    } catch (error: any) {
      updateState({
        isLoading: false,
        error: error.response?.data?.message || 'Failed to generate final report',
      });
    }
  };

  const handleDownloadFinalExcel = async () => {
    updateState({ isLoading: true, error: null });
    try {
      const response = await apiService.generateBillableReport({
        client_name: state.clientName,
        refresh_invoiced: true,
        format: 'excel',
      });
      
      const filename = `billable_report_final_${state.clientName.replace(/\s+/g, '_')}.xlsx`;
      apiService.downloadFile(response.data, filename);
      
      updateState({ isLoading: false });
    } catch (error: any) {
      updateState({
        isLoading: false,
        error: error.response?.data?.message || 'Failed to download Excel file',
      });
    }
  };

  const resetWizard = () => {
    updateState({
      currentStep: 0,
      clientName: '',
      initialReportGenerated: false,
      finalReportGenerated: false,
      reportData: null,
      isLoading: false,
      error: null,
    });
    localStorage.removeItem('billableWizardState');
  };

  const renderStepContent = () => {
    const currentStepData = steps[state.currentStep];

    switch (currentStepData.id) {
      case 'select-client':
        return (
          <div className="space-y-6">
            <div className="text-center">
              <BuildingOfficeIcon className="mx-auto h-12 w-12 text-blue-600" />
              <h3 className="mt-2 text-lg font-medium text-gray-900">Select Client</h3>
              <p className="mt-1 text-sm text-gray-600">
                Choose the client for which you want to generate the billable report
              </p>
            </div>
            
            <div className="max-w-md mx-auto">
              <label htmlFor="client-name" className="block text-sm font-medium text-gray-700 mb-2">
                Client Name
              </label>
              <select
                id="client-name"
                value={state.clientName}
                onChange={(e) => updateState({ clientName: e.target.value })}
                className="block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
              >
                <option value="">Select a client...</option>
                {availableClients.map((client) => (
                  <option key={client} value={client}>
                    {client}
                  </option>
                ))}
              </select>
              <p className="mt-2 text-xs text-gray-500">
                Client name must match exactly as configured in DynamoDB
              </p>
            </div>
            
            <div className="text-center">
              <button
                onClick={handleNext}
                disabled={!state.clientName}
                className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Continue
                <ArrowRightIcon className="ml-2 h-4 w-4" />
              </button>
            </div>
          </div>
        );

      case 'initial-report':
        return (
          <div className="space-y-6">
            <div className="text-center">
              <DocumentTextIcon className="mx-auto h-12 w-12 text-blue-600" />
              <h3 className="mt-2 text-lg font-medium text-gray-900">Generate Billable Report</h3>
              <p className="mt-1 text-sm text-gray-600">
                Create a billable report for {state.clientName} without updating ClickUp fields
              </p>
            </div>

            {state.error && (
              <div className="bg-red-50 border border-red-200 rounded-md p-4">
                <div className="flex">
                  <ExclamationTriangleIcon className="h-5 w-5 text-red-400" />
                  <div className="ml-3">
                    <p className="text-sm text-red-800">{state.error}</p>
                  </div>
                </div>
              </div>
            )}

            <div className="bg-blue-50 border border-blue-200 rounded-md p-4">
              <h4 className="text-sm font-medium text-blue-900 mb-2">Report criteria:</h4>
              <ul className="text-sm text-blue-800 space-y-1">
                <li>• Tasks with BillableHours &gt; 0</li>
                <li>• Tasks with MonthlyReported ≠ 0</li>
                <li>• Separates internal vs. client tasks</li>
                <li>• No ClickUp fields will be updated yet</li>
              </ul>
            </div>

            <div className="text-center">
              <button
                onClick={handleGenerateInitialReport}
                disabled={state.isLoading}
                className="inline-flex items-center px-6 py-3 border border-transparent text-base font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {state.isLoading ? (
                  <>
                    <ArrowPathIcon className="animate-spin -ml-1 mr-2 h-5 w-5" />
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
          </div>
        );

      case 'review-billable':
        return (
          <div className="space-y-6">
            <div className="text-center">
              <CheckCircleIcon className="mx-auto h-12 w-12 text-green-600" />
              <h3 className="mt-2 text-lg font-medium text-gray-900">Review Billable Tasks</h3>
              <p className="mt-1 text-sm text-gray-600">
                Review the billable tasks for {state.clientName} and verify totals
              </p>
            </div>

            {state.reportData && (
              <div className="bg-white border border-gray-200 rounded-lg p-6">
                <h4 className="text-lg font-medium text-gray-900 mb-4">Report Summary</h4>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
                  <div className="bg-green-50 p-4 rounded-lg">
                    <div className="text-2xl font-bold text-green-600">
                      {state.reportData.tasks?.length || 0}
                    </div>
                    <div className="text-sm text-green-800">Billable Tasks</div>
                  </div>
                  <div className="bg-blue-50 p-4 rounded-lg">
                    <div className="text-2xl font-bold text-blue-600">
                      {state.reportData.internal_tasks?.length || 0}
                    </div>
                    <div className="text-sm text-blue-800">Internal Tasks</div>
                  </div>
                  <div className="bg-purple-50 p-4 rounded-lg">
                    <div className="text-2xl font-bold text-purple-600">
                      {state.reportData.totals?.total_billable_hours?.toFixed(1) || '0.0'}
                    </div>
                    <div className="text-sm text-purple-800">Total Hours</div>
                  </div>
                </div>

                {state.reportData.totals && (
                  <div className="bg-gray-50 p-4 rounded-lg mb-4">
                    <h5 className="text-sm font-medium text-gray-900 mb-2">Billing Totals (Non-Internal)</h5>
                    <div className="text-lg font-semibold text-gray-900">
                      ${state.reportData.totals.total_amount?.toFixed(2) || '0.00'}
                    </div>
                    <div className="text-sm text-gray-600">
                      {state.reportData.totals.total_billable_hours?.toFixed(1) || '0.0'} hours
                    </div>
                  </div>
                )}

                <div className="flex flex-col sm:flex-row gap-3">
                  <button
                    onClick={handleDownloadExcel}
                    disabled={state.isLoading}
                    className="flex-1 inline-flex items-center justify-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                  >
                    <CloudArrowDownIcon className="h-4 w-4 mr-2" />
                    Download Excel
                  </button>
                </div>
              </div>
            )}

            <div className="bg-yellow-50 border border-yellow-200 rounded-md p-4">
              <h4 className="text-sm font-medium text-yellow-900 mb-2">Review Checklist:</h4>
              <ul className="text-sm text-yellow-800 space-y-1">
                <li>• Verify all expected tasks are included in the billable report</li>
                <li>• Check that internal tasks are properly separated</li>
                <li>• Confirm billable hours and amounts are correct</li>
                <li>• This report is typically sent to the client for invoicing</li>
              </ul>
            </div>

            <div className="flex justify-between">
              <button
                onClick={handleBack}
                className="inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
              >
                <ArrowLeftIcon className="mr-2 h-4 w-4" />
                Back
              </button>
              <button
                onClick={handleNext}
                className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
              >
                Report Approved
                <ArrowRightIcon className="ml-2 h-4 w-4" />
              </button>
            </div>
          </div>
        );

      case 'final-report':
        return (
          <div className="space-y-6">
            <div className="text-center">
              <CurrencyDollarIcon className="mx-auto h-12 w-12 text-orange-600" />
              <h3 className="mt-2 text-lg font-medium text-gray-900">Mark as Invoiced</h3>
              <p className="mt-1 text-sm text-gray-600">
                Update InvoicedHours field in ClickUp to track billing progress
              </p>
            </div>

            {state.error && (
              <div className="bg-red-50 border border-red-200 rounded-md p-4">
                <div className="flex">
                  <ExclamationTriangleIcon className="h-5 w-5 text-red-400" />
                  <div className="ml-3">
                    <p className="text-sm text-red-800">{state.error}</p>
                  </div>
                </div>
              </div>
            )}

            <div className="bg-orange-50 border border-orange-200 rounded-md p-4">
              <h4 className="text-sm font-medium text-orange-900 mb-2">This will:</h4>
              <ul className="text-sm text-orange-800 space-y-1">
                <li>• Set InvoicedHours = BillableHours for reported non-internal tasks</li>
                <li>• Mark tasks as invoiced for future reporting periods</li>
                <li>• Generate a final report with updated data</li>
                <li>• Help track billing progress across months</li>
              </ul>
            </div>

            <div className="text-center">
              {state.finalReportGenerated ? (
                <div className="space-y-4">
                  <div className="text-green-600 font-medium">
                    ✓ InvoicedHours updated successfully
                  </div>
                  <button
                    onClick={handleDownloadFinalExcel}
                    disabled={state.isLoading}
                    className="inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                  >
                    <CloudArrowDownIcon className="h-4 w-4 mr-2" />
                    Download Final Excel
                  </button>
                </div>
              ) : (
                <button
                  onClick={handleGenerateFinalReport}
                  disabled={state.isLoading}
                  className="inline-flex items-center px-6 py-3 border border-transparent text-base font-medium rounded-md text-white bg-orange-600 hover:bg-orange-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-orange-500 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {state.isLoading ? (
                    <>
                      <ArrowPathIcon className="animate-spin -ml-1 mr-2 h-5 w-5" />
                      Updating ClickUp...
                    </>
                  ) : (
                    <>
                      Mark as Invoiced
                      <ArrowRightIcon className="ml-2 h-4 w-4" />
                    </>
                  )}
                </button>
              )}
            </div>

            <div className="flex justify-between">
              <button
                onClick={handleBack}
                className="inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
              >
                <ArrowLeftIcon className="mr-2 h-4 w-4" />
                Back
              </button>
            </div>
          </div>
        );

      case 'complete':
        return (
          <div className="space-y-6">
            <div className="text-center">
              <CheckCircleIcon className="mx-auto h-12 w-12 text-green-600" />
              <h3 className="mt-2 text-lg font-medium text-gray-900">Workflow Complete!</h3>
              <p className="mt-1 text-sm text-gray-600">
                Billable report generated and tasks marked as invoiced successfully
              </p>
            </div>

            <div className="bg-green-50 border border-green-200 rounded-md p-4">
              <h4 className="text-sm font-medium text-green-900 mb-2">What was accomplished:</h4>
              <ul className="text-sm text-green-800 space-y-1">
                <li>• ✓ Billable report generated for {state.clientName}</li>
                <li>• ✓ Tasks filtered based on BillableHours and MonthlyReported</li>
                <li>• ✓ Internal tasks separated from client tasks</li>
                <li>• ✓ InvoicedHours updated to match BillableHours</li>
                <li>• ✓ Tasks marked as invoiced for future reporting</li>
              </ul>
            </div>

            <div className="bg-blue-50 border border-blue-200 rounded-md p-4">
              <h4 className="text-sm font-medium text-blue-900 mb-2">Next Steps:</h4>
              <p className="text-sm text-blue-800">
                You can now generate reports for other clients or start a new time tracking 
                report for the next billing period.
              </p>
            </div>

            <div className="flex flex-col sm:flex-row gap-3">
              <button
                onClick={resetWizard}
                className="flex-1 inline-flex items-center justify-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
              >
                New Billable Report
              </button>
              <button
                onClick={() => window.location.href = '/timetracking'}
                className="flex-1 inline-flex items-center justify-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
              >
                Time Tracking Report
                <ArrowRightIcon className="ml-2 h-4 w-4" />
              </button>
            </div>
          </div>
        );

      default:
        return null;
    }
  };

  return (
    <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900">Billable Report Wizard</h1>
        <p className="mt-1 text-sm text-gray-600">
          Generate client-specific billable reports and mark tasks as invoiced
        </p>
      </div>

      {/* Progress Steps */}
      <div className="mb-8">
        <nav aria-label="Progress">
          <ol className="flex items-center justify-between">
            {steps.map((step, index) => (
              <li key={step.id} className="relative flex-1">
                <div className="flex items-center">
                  <div className="relative flex items-center justify-center">
                    <div
                      className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium ${
                        step.completed
                          ? 'bg-green-600 text-white'
                          : index === state.currentStep
                          ? 'bg-blue-600 text-white'
                          : 'bg-gray-200 text-gray-500'
                      }`}
                    >
                      {step.completed ? (
                        <CheckCircleIcon className="w-5 h-5" />
                      ) : (
                        <span>{index + 1}</span>
                      )}
                    </div>
                  </div>
                  <div className="ml-4 min-w-0 flex-1">
                    <div className={`text-sm font-medium ${
                      step.completed || index === state.currentStep ? 'text-gray-900' : 'text-gray-500'
                    }`}>
                      {step.title}
                    </div>
                  </div>
                </div>
              </li>
            ))}
          </ol>
        </nav>
      </div>

      {/* Step Content */}
      <div className="bg-white shadow rounded-lg p-6">
        {renderStepContent()}
      </div>
    </div>
  );
};

export default BillableWizard;