import React, { useState } from "react";
import {
  ChevronDownIcon,
  ChevronRightIcon,
  ArrowTopRightOnSquareIcon,
} from "@heroicons/react/24/outline";
import type {
  Task,
  BillableTask,
  TimeTrackingTask,
  TaskDetailsProps,
} from "../../types/shared/taskTypes";

const TaskDetails: React.FC<TaskDetailsProps> = ({
  title,
  tasks,
  type,
  totalHours,
  className = "",
}) => {
  const [isExpanded, setIsExpanded] = useState(false);

  if (!tasks || tasks.length === 0) {
    return null;
  }

  const getTaskName = (task: Task) => {
    return "name" in task ? task.name : task.task_name;
  };

  const getHourValue = (task: Task) => {
    if (type === "timetracking") {
      return (task as TimeTrackingTask).adjusted_hours;
    }
    return (task as BillableTask).billable_hours;
  };

  const getTypeColor = () => {
    switch (type) {
      case "billable":
        return "bg-green-50 border-green-200 text-green-800";
      case "internal":
        return "bg-blue-50 border-blue-200 text-blue-800";
      case "timetracking":
        return "bg-purple-50 border-purple-200 text-purple-800";
      default:
        return "bg-gray-50 border-gray-200 text-gray-800";
    }
  };

  return (
    <div className={`border rounded-lg ${getTypeColor()} ${className}`}>
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="w-full px-4 py-3 flex items-center justify-between hover:bg-opacity-80 transition-colors"
      >
        <div className="flex items-center space-x-2">
          {isExpanded ? (
            <ChevronDownIcon className="h-5 w-5" />
          ) : (
            <ChevronRightIcon className="h-5 w-5" />
          )}
          <span className="font-medium text-sm">{title}</span>
          <span className="text-xs bg-white bg-opacity-70 px-2 py-1 rounded-full">
            {tasks.length} tasks
          </span>
        </div>
        {totalHours !== undefined && (
          <span className="text-sm font-semibold">
            {totalHours.toFixed(1)} hours
          </span>
        )}
      </button>

      {isExpanded && (
        <div className="border-t border-current border-opacity-20">
          <div className="p-4 bg-white bg-opacity-50">
            <div className="space-y-3">
              {tasks.map((task) => (
                <div
                  key={task.task_id}
                  className="bg-white rounded-md p-3 border border-gray-200 shadow-sm"
                >
                  <div className="flex justify-between items-start">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center space-x-2 mb-1">
                        <span className="text-xs font-mono text-gray-500 bg-gray-100 px-2 py-1 rounded">
                          {task.custom_id}
                        </span>
                        {task.tags && task.tags.length > 0 && (
                          <div className="flex flex-wrap gap-1">
                            {task.tags.map((tag, index) => (
                              <span
                                key={index}
                                className="text-xs bg-blue-100 text-blue-800 px-2 py-1 rounded-full"
                              >
                                {tag}
                              </span>
                            ))}
                          </div>
                        )}
                      </div>
                      <h4 className="text-sm font-medium text-gray-900 mb-2">
                        {getTaskName(task)}
                      </h4>
                      <div className="flex flex-wrap gap-4 text-xs text-gray-600">
                        <span>
                          <strong>Hours:</strong>{" "}
                          {getHourValue(task).toFixed(1)}
                        </span>
                        {type === "timetracking" && (
                          <>
                            <span>
                              <strong>Client:</strong>{" "}
                              {(task as TimeTrackingTask).client}
                            </span>
                            <span>
                              <strong>Billable:</strong>{" "}
                              {(
                                task as TimeTrackingTask
                              ).calculated_billable_hours.toFixed(1)}
                            </span>
                            <span>
                              <strong>Status:</strong>{" "}
                              {(task as TimeTrackingTask).status}
                            </span>
                          </>
                        )}
                        {type !== "timetracking" && (
                          <>
                            <span>
                              <strong>Invoiced:</strong>{" "}
                              {(task as BillableTask).invoiced_hours.toFixed(1)}
                            </span>
                            <span>
                              <strong>Reported:</strong>{" "}
                              {(task as BillableTask).monthly_reported.toFixed(
                                1,
                              )}
                            </span>
                          </>
                        )}
                      </div>
                    </div>
                    <a
                      href={task.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="ml-2 text-gray-400 hover:text-gray-600 transition-colors"
                    >
                      <ArrowTopRightOnSquareIcon className="h-4 w-4" />
                    </a>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default TaskDetails;
