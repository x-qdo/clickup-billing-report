import React from "react";
import type { FinalReportItem } from "../../types/api/timeTrackingTypes";
import TaskDetails from "../shared/TaskDetails";

interface TaskReportProps {
  finalReport: FinalReportItem[];
  isExpanded: boolean;
  internalTag: string;
}

const TaskReport: React.FC<TaskReportProps> = ({
  finalReport,
  isExpanded,
  internalTag,
}) => {
  if (!finalReport || finalReport.length === 0) {
    return (
      <div className="bg-white shadow sm:rounded-lg p-4 mt-6">
        <h3 className="text-lg leading-6 font-medium text-gray-900">
          Task Report
        </h3>
        <p className="mt-1 text-sm text-gray-500">
          No task report data available.
        </p>
      </div>
    );
  }

  const clientTasks = finalReport.filter(
    (item) => !item.tags.includes(internalTag),
  );
  const internalTasks = finalReport.filter((item) =>
    item.tags.includes(internalTag),
  );

  const clientTasksTotal = clientTasks.reduce(
    (sum, task) => sum + task.adjusted_hours,
    0,
  );
  const internalTasksTotal = internalTasks.reduce(
    (sum, task) => sum + task.adjusted_hours,
    0,
  );

  if (
    clientTasks.length === 0 &&
    internalTasks.length === 0 &&
    finalReport.length > 0
  ) {
    return (
      <div className="bg-white shadow sm:rounded-lg p-4 mt-6">
        <h3 className="text-lg leading-6 font-medium text-gray-900">
          Task Report
        </h3>
        <p className="mt-1 text-sm text-gray-500">
          No tasks match the display criteria (client/internal).
        </p>
      </div>
    );
  }

  return (
    <div className={`${isExpanded ? "" : "mt-6"} space-y-6`}>
      {clientTasks.length > 0 && (
        <TaskDetails
          title="Client Tasks - Detailed Breakdown"
          tasks={clientTasks}
          type="timetracking"
          totalHours={clientTasksTotal}
        />
      )}

      {internalTasks.length > 0 && (
        <TaskDetails
          title="Internal Tasks - Detailed Breakdown"
          tasks={internalTasks}
          type="timetracking"
          totalHours={internalTasksTotal}
        />
      )}
    </div>
  );
};

export default TaskReport;
