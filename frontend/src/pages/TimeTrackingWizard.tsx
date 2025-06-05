import { useState, useEffect } from "react";
import { format, startOfMonth } from "date-fns";
import { apiService } from "../services/api";
import {
  CheckCircleIcon,
  // Unused icons can be removed if not needed by the main wizard layout anymore
  // For now, keeping them to avoid breaking anything if they are used outside renderStepContent
} from "@heroicons/react/24/outline";

import SelectDateStep from "../components/timeTrackingWizard/SelectDateStep";
// InitialReportStep is no longer used directly here
import ReviewDataStep from "../components/timeTrackingWizard/ReviewDataStep";
import FinalReportStep from "../components/timeTrackingWizard/FinalReportStep";
import CompleteStep from "../components/timeTrackingWizard/CompleteStep";

import type { TimeTrackingData } from "../types/api/timeTrackingTypes";

interface WizardStep {
  id: string;
  title: string;
  completed: boolean;
}

interface WizardState {
  currentStep: number;
  reportDate: string;
  initialReportGenerated: boolean;
  finalReportGenerated: boolean;
  reportData: TimeTrackingData | null;
  isLoading: boolean;
  error: string | null;
}

const TimeTrackingWizard: React.FC = () => {
  const [state, setState] = useState<WizardState>({
    currentStep: 0,
    reportDate: format(startOfMonth(new Date()), "yyyy-MM"),
    initialReportGenerated: false,
    finalReportGenerated: false,
    reportData: null,
    isLoading: false,
    error: null,
  });

  const steps: WizardStep[] = [
    {
      id: "select-date",
      title: "Select Month & Generate",
      completed: state.initialReportGenerated,
    },
    {
      id: "review-data",
      title: "Review & Verify",
      completed: state.currentStep > 1 && state.initialReportGenerated,
    },
    {
      id: "final-report",
      title: "Update ClickUp Fields",
      completed: state.finalReportGenerated,
    },
    {
      id: "complete",
      title: "Complete", // Index is now 3
      completed: state.finalReportGenerated && state.currentStep >= 3,
    },
  ];

  // Load state from localStorage on component mount
  useEffect(() => {
    const savedState = localStorage.getItem("timeTrackingWizardState");
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
      reportDate: state.reportDate,
      initialReportGenerated: state.initialReportGenerated,
      finalReportGenerated: state.finalReportGenerated,
      reportData: state.reportData,
    };
    localStorage.setItem(
      "timeTrackingWizardState",
      JSON.stringify(stateToSave),
    );
  }, [
    state.currentStep,
    state.reportDate,
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
      const response = await apiService.generateTimeTrackingReport({
        report_date: state.reportDate,
        refresh_billable: false,
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
      const response = await apiService.generateTimeTrackingReport({
        report_date: state.reportDate,
        refresh_billable: false,
        format: "excel",
      });

      const filename = `time_tracking_report_${state.reportDate}.xlsx`;
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
      const response = await apiService.generateTimeTrackingReport({
        report_date: state.reportDate,
        refresh_billable: true,
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
      const response = await apiService.generateTimeTrackingReport({
        report_date: state.reportDate,
        refresh_billable: true,
        format: "excel",
      });

      const filename = `time_tracking_report_final_${state.reportDate}.xlsx`;
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
      reportDate: format(startOfMonth(new Date()), "yyyy-MM"),
      initialReportGenerated: false,
      finalReportGenerated: false,
      reportData: null,
      isLoading: false,
      error: null,
    });
    localStorage.removeItem("timeTrackingWizardState");
  };

  const renderStepContent = () => {
    const currentStepData = steps[state.currentStep];

    switch (currentStepData.id) {
      case "select-date":
        return (
          <SelectDateStep
            reportDate={state.reportDate}
            onReportDateChange={(newDate) =>
              updateState({ reportDate: newDate })
            }
            // onNext is replaced by onGenerateReport, also pass isLoading and error
            onGenerateReport={handleGenerateInitialReport}
            isLoading={state.isLoading}
            error={state.error}
          />
        );

      // "initial-report" case is removed
      case "review-data":
        return (
          <ReviewDataStep
            reportData={state.reportData}
            isLoading={state.isLoading}
            error={state.error}
            onDownloadExcel={handleDownloadExcel}
            onNext={handleNext}
            onBack={handleBack}
          />
        );

      case "final-report":
        return (
          <FinalReportStep
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
          <CompleteStep
            reportDate={state.reportDate}
            onResetWizard={resetWizard}
            onNavigateToBillable={() => (window.location.href = "/billable")}
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
          Time Tracking Report Wizard
        </h1>
        <p className="mt-1 text-sm text-gray-600">
          Generate monthly time tracking reports with developer coefficients
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

export default TimeTrackingWizard;
