import { useState, useEffect } from "react";
import { format, startOfMonth } from "date-fns";
import { apiService } from "../services/api";
import { useAsyncJob, isJobFileResult } from "../hooks/useAsyncJob";
import { CheckCircleIcon } from "@heroicons/react/24/outline";

import SelectDateStep from "../components/timeTrackingWizard/SelectDateStep";
import ReviewDataStep from "../components/timeTrackingWizard/ReviewDataStep";
import FinalReportStep from "../components/timeTrackingWizard/FinalReportStep";
import CompleteStep from "../components/timeTrackingWizard/CompleteStep";
import JobProgress from "../components/shared/JobProgress";

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

  // Async job hooks for report generation
  const initialReportJob = useAsyncJob<TimeTrackingData>();
  const finalReportJob = useAsyncJob<TimeTrackingData>();
  const excelJob = useAsyncJob();

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

  // Handle initial report job completion
  useEffect(() => {
    if (initialReportJob.status === "completed" && initialReportJob.result) {
      updateState({
        reportData: initialReportJob.result,
        initialReportGenerated: true,
        isLoading: false,
        error: null,
      });
      setTimeout(() => handleNext(), 500);
    } else if (initialReportJob.status === "failed") {
      updateState({
        isLoading: false,
        error: initialReportJob.error || "Failed to generate report",
      });
    }
  }, [initialReportJob.status, initialReportJob.result, initialReportJob.error]);

  // Handle final report job completion
  useEffect(() => {
    if (finalReportJob.status === "completed" && finalReportJob.result) {
      updateState({
        reportData: finalReportJob.result,
        finalReportGenerated: true,
        isLoading: false,
        error: null,
      });
      setTimeout(() => handleNext(), 500);
    } else if (finalReportJob.status === "failed") {
      updateState({
        isLoading: false,
        error: finalReportJob.error || "Failed to generate report",
      });
    }
  }, [finalReportJob.status, finalReportJob.result, finalReportJob.error]);

  // Handle Excel job completion
  useEffect(() => {
    if (excelJob.status === "completed" && excelJob.result) {
      if (isJobFileResult(excelJob.result)) {
        apiService.downloadFromUrl(excelJob.result.download_url, excelJob.result.filename);
      }
      updateState({ isLoading: false });
    } else if (excelJob.status === "failed") {
      updateState({
        isLoading: false,
        error: excelJob.error || "Failed to download Excel file",
      });
    }
  }, [excelJob.status, excelJob.result, excelJob.error]);

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
    await initialReportJob.startJob("timetrack", {
      report_date: state.reportDate,
      refresh_billable: false,
      format: "json",
    });
  };

  const handleDownloadExcel = async () => {
    updateState({ isLoading: true, error: null });
    await excelJob.startJob("timetrack", {
      report_date: state.reportDate,
      refresh_billable: false,
      format: "excel",
    });
  };

  const handleGenerateFinalReport = async () => {
    updateState({ isLoading: true, error: null });
    await finalReportJob.startJob("timetrack", {
      report_date: state.reportDate,
      refresh_billable: true,
      format: "json",
    });
  };

  const handleDownloadFinalExcel = async () => {
    updateState({ isLoading: true, error: null });
    await excelJob.startJob("timetrack", {
      report_date: state.reportDate,
      refresh_billable: true,
      format: "excel",
    });
  };

  const resetWizard = () => {
    initialReportJob.reset();
    finalReportJob.reset();
    excelJob.reset();
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

  // Determine if any job is in progress
  const isJobInProgress =
    initialReportJob.isLoading || finalReportJob.isLoading || excelJob.isLoading;
  const currentJobStatus =
    initialReportJob.status || finalReportJob.status || excelJob.status;
  const currentJobElapsed =
    initialReportJob.elapsedTime || finalReportJob.elapsedTime || excelJob.elapsedTime;

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

      {/* Job Progress */}
      {isJobInProgress && currentJobStatus && (
        <div className="mb-6">
          <JobProgress
            status={currentJobStatus}
            elapsedTime={currentJobElapsed}
            error={state.error}
          />
        </div>
      )}

      {/* Step Content */}
      <div className="bg-white shadow rounded-lg p-6">
        {renderStepContent()}
      </div>
    </div>
  );
};

export default TimeTrackingWizard;
