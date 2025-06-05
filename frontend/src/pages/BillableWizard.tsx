import { useState, useEffect } from "react";
import { apiService } from "../services/api";
import { CheckCircleIcon } from "@heroicons/react/24/outline";
import SelectClientStep from "../components/billableWizard/SelectClientStep";
import InitialReportStep from "../components/billableWizard/InitialReportStep";
import ReviewBillableStep from "../components/billableWizard/ReviewBillableStep";
import MarkInvoicedStep from "../components/billableWizard/MarkInvoicedStep";
import BillableCompleteStep from "../components/billableWizard/BillableCompleteStep";
import type { ApiClient } from "../services/api";

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
  availableClients: ApiClient[];
  clientsLoading: boolean;
  clientsError: string | null;
}

const BillableWizard: React.FC = () => {
  const [state, setState] = useState<WizardState>({
    currentStep: 0,
    clientName: "",
    initialReportGenerated: false,
    finalReportGenerated: false,
    reportData: null,
    isLoading: false,
    error: null,
    availableClients: [],
    clientsLoading: true,
    clientsError: null,
  });

  const steps: WizardStep[] = [
    {
      id: "select-client",
      title: "Select Client",
      completed: state.currentStep > 0 && state.clientName !== "",
    },
    {
      id: "initial-report",
      title: "Generate Billable Report",
      completed: state.initialReportGenerated,
    },
    {
      id: "review-billable",
      title: "Review Billable Tasks",
      completed: state.currentStep > 2 && state.initialReportGenerated,
    },
    {
      id: "final-report",
      title: "Mark as Invoiced",
      completed: state.finalReportGenerated,
    },
    {
      id: "complete",
      title: "Complete",
      completed: state.finalReportGenerated && state.currentStep >= 4,
    },
  ];

  // Load state from localStorage on component mount
  useEffect(() => {
    const savedState = localStorage.getItem("billableWizardState");
    if (savedState) {
      try {
        const parsed = JSON.parse(savedState);
        setState((prevState) => ({
          ...prevState,
          ...parsed,
          isLoading: false,
          error: null,
        }));
      } catch (error) {
        console.error("Failed to parse saved wizard state:", error);
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
    localStorage.setItem("billableWizardState", JSON.stringify(stateToSave));
  }, [
    state.currentStep,
    state.clientName,
    state.initialReportGenerated,
    state.finalReportGenerated,
    state.reportData,
  ]);

  const updateState = (updates: Partial<WizardState>) => {
    setState((prevState) => ({ ...prevState, ...updates }));
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
        format: "json",
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
        error: error.response?.data?.message || "Failed to generate report",
      });
    }
  };

  const handleDownloadExcel = async () => {
    updateState({ isLoading: true, error: null });
    try {
      const response = await apiService.generateBillableReport({
        client_name: state.clientName,
        refresh_invoiced: false,
        format: "excel",
      });

      const filename = `billable_report_${state.clientName.replace(/\s+/g, "_")}.xlsx`;
      apiService.downloadFile(response.data, filename);

      updateState({ isLoading: false });
    } catch (error: any) {
      updateState({
        isLoading: false,
        error: error.response?.data?.message || "Failed to download Excel file",
      });
    }
  };

  const handleGenerateFinalReport = async () => {
    updateState({ isLoading: true, error: null });
    try {
      const response = await apiService.generateBillableReport({
        client_name: state.clientName,
        refresh_invoiced: true,
        format: "json",
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
        error:
          error.response?.data?.message || "Failed to generate final report",
      });
    }
  };

  const handleDownloadFinalExcel = async () => {
    updateState({ isLoading: true, error: null });
    try {
      const response = await apiService.generateBillableReport({
        client_name: state.clientName,
        refresh_invoiced: true,
        format: "excel",
      });

      const filename = `billable_report_final_${state.clientName.replace(/\s+/g, "_")}.xlsx`;
      apiService.downloadFile(response.data, filename);

      updateState({ isLoading: false });
    } catch (error: any) {
      updateState({
        isLoading: false,
        error: error.response?.data?.message || "Failed to download Excel file",
      });
    }
  };

  const resetWizard = () => {
    updateState({
      currentStep: 0,
      clientName: "",
      initialReportGenerated: false,
      finalReportGenerated: false,
      reportData: null,
      isLoading: false,
      error: null,
    });
    localStorage.removeItem("billableWizardState");
  };

  // Fetch available clients on component mount
  useEffect(() => {
    const fetchClients = async () => {
      updateState({ clientsLoading: true, clientsError: null });
      try {
        const response = await apiService.listClients();
        updateState({
          availableClients: response.data || [],
          clientsLoading: false,
        });
      } catch (error: any) {
        console.error("Failed to fetch clients:", error);
        updateState({
          clientsError: apiService.getErrorMessage(error),
          clientsLoading: false,
          availableClients: [], // Ensure it's an empty array on error
        });
      }
    };

    fetchClients();
  }, []); // Empty dependency array ensures this runs only once on mount

  const renderStepContent = () => {
    const currentStepData = steps[state.currentStep];

    switch (currentStepData.id) {
      case "select-client":
        return (
          <SelectClientStep
            availableClients={state.availableClients}
            clientName={state.clientName}
            onClientNameChange={(name: string) =>
              updateState({ clientName: name })
            }
            onNext={handleNext}
            clientsLoading={state.clientsLoading}
            clientsError={state.clientsError}
          />
        );

      case "initial-report":
        return (
          <InitialReportStep
            clientName={state.clientName}
            isLoading={state.isLoading}
            error={state.error}
            onGenerateInitialReport={handleGenerateInitialReport}
          />
        );

      case "review-billable":
        return (
          <ReviewBillableStep
            reportData={state.reportData}
            clientName={state.clientName}
            isLoading={state.isLoading}
            error={state.error}
            onDownloadExcel={handleDownloadExcel}
            onNext={handleNext}
            onBack={handleBack}
          />
        );

      case "final-report":
        return (
          <MarkInvoicedStep
            clientName={state.clientName}
            isLoading={state.isLoading}
            error={state.error}
            finalReportGenerated={state.finalReportGenerated}
            onGenerateFinalReport={handleGenerateFinalReport}
            onDownloadFinalExcel={handleDownloadFinalExcel}
            onBack={handleBack}
          />
        );

      case "complete":
        return (
          <BillableCompleteStep
            clientName={state.clientName}
            onResetWizard={resetWizard}
            onNavigateToTimeTracking={() =>
              (window.location.href = "/timetracking")
            }
          />
        );

      default:
        return null;
    }
  };

  return (
    <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900">
          Billable Report Wizard
        </h1>
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
                          ? "bg-green-600 text-white"
                          : index === state.currentStep
                            ? "bg-blue-600 text-white"
                            : "bg-gray-200 text-gray-500"
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
                    <div
                      className={`text-sm font-medium ${
                        step.completed || index === state.currentStep
                          ? "text-gray-900"
                          : "text-gray-500"
                      }`}
                    >
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
