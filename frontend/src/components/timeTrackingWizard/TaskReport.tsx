import React from "react";
import type { FinalReportItem } from "../../types/api/timeTrackingTypes";

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

  const renderTaskTable = (
    title: string,
    tasks: FinalReportItem[],
    isInternalSection: boolean = false,
  ) => {
    if (tasks.length === 0) {
      return null; // Don't render table if no tasks for this section
    }
    return (
      <div
        className={`shadow overflow-hidden border-b border-gray-200 sm:rounded-lg ${isInternalSection ? "mt-8" : ""}`}
      >
        <div className="px-4 py-5 sm:px-6 bg-white">
          <h3 className="text-lg leading-6 font-medium text-gray-900">
            {title}
          </h3>
          <p className="mt-1 max-w-2xl text-sm text-gray-500">
            Detailed hours and status by task.
          </p>
        </div>
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th
                scope="col"
                className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
              >
                Task ID
              </th>
              <th
                scope="col"
                className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
              >
                Task Name
              </th>
              <th
                scope="col"
                className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
              >
                Client
              </th>
              <th
                scope="col"
                className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
              >
                Tags
              </th>
              <th
                scope="col"
                className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
              >
                Adjusted Hours
              </th>
              <th
                scope="col"
                className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
              >
                Invoiced Hours
              </th>
              <th
                scope="col"
                className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
              >
                Billable Hours
              </th>
              <th
                scope="col"
                className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
              >
                Status
              </th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {tasks.map((item) => (
              <tr key={item.task_id}>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-blue-600 hover:text-blue-800">
                  <a href={item.url} target="_blank" rel="noopener noreferrer">
                    {item.custom_id || item.task_id}
                  </a>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                  {item.task_name}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {item.client}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {item.tags.join(", ")}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-right">
                  {item.adjusted_hours.toFixed(2)}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-right">
                  {item.invoiced_hours.toFixed(2)}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-right">
                  {item.calculated_billable_hours.toFixed(2)}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {item.status}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  };

  const scrollAndMarginsDivClass = isExpanded
    ? "-my-2 overflow-x-auto"
    : "-my-2 overflow-x-auto sm:-mx-6 lg:-mx-8";
  const paddingAndMinWidthDivClass = isExpanded
    ? "py-2 align-middle inline-block min-w-full"
    : "py-2 align-middle inline-block min-w-full sm:px-6 lg:px-8";

  // If both are empty, and finalReport was not, this means all tasks were filtered into one or the other category,
  // or the tag logic needs review. The individual renderTaskTable will return null if its list is empty.
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
    <div className="mt-6 flex flex-col">
      <div className={scrollAndMarginsDivClass}>
        <div className={paddingAndMinWidthDivClass}>
          {renderTaskTable(
            "Client Task Report - Detailed Breakdown",
            clientTasks,
          )}
          {renderTaskTable(
            "Internal Task Report - Detailed Breakdown",
            internalTasks,
            true, // Add margin top for the second table
          )}
        </div>
      </div>
    </div>
  );
};

export default TaskReport;
