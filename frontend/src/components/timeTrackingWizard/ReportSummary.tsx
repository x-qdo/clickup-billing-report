import React from "react";
import type { TotalItem } from "../../types/api/timeTrackingTypes";
import { UsersIcon, ClockIcon } from "@heroicons/react/24/outline";

interface ReportSummaryProps {
  totals: TotalItem[];
}

const ReportSummary: React.FC<ReportSummaryProps> = ({ totals }) => {
  if (!totals || totals.length === 0) {
    return (
      <div className="bg-white shadow sm:rounded-lg p-6">
        <h3 className="text-lg leading-6 font-medium text-gray-900">
          Report Summary
        </h3>
        <p className="mt-2 text-sm text-gray-500">No summary data available.</p>
      </div>
    );
  }

  return (
    <div className="bg-gray-50 py-6">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="mb-5">
          <h2 className="text-xl font-semibold text-gray-900">
            Report Summary - Totals by Client
          </h2>
          <p className="mt-1 text-sm text-gray-600">
            Total adjusted (billable) hours per client.
          </p>
        </div>

        <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
          {totals.map((item) => (
            <div
              key={item.client}
              className="bg-white overflow-hidden shadow rounded-lg"
            >
              <div className="p-5">
                <div className="flex items-center">
                  <div className="flex-shrink-0">
                    <UsersIcon
                      className="h-6 w-6 text-gray-400"
                      aria-hidden="true"
                    />
                  </div>
                  <div className="ml-3 w-0 flex-1">
                    <dt className="text-sm font-medium text-gray-500 truncate">
                      Client
                    </dt>
                    <dd className="text-lg font-semibold text-gray-900">
                      {item.client}
                    </dd>
                  </div>
                </div>
              </div>
              <div className="bg-gray-50 px-5 py-4">
                <div className="flex items-center">
                  <div className="flex-shrink-0">
                    <ClockIcon
                      className="h-6 w-6 text-indigo-500"
                      aria-hidden="true"
                    />
                  </div>
                  <div className="ml-3 w-0 flex-1">
                    <dd className="text-xl font-bold text-indigo-600">
                      {item.adjusted_hours.toFixed(2)} hours
                    </dd>
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

export default ReportSummary;
