import React from "react";
import type { TotalItem } from "../../types/api/timeTrackingTypes";

interface ReportSummaryProps {
  totals: TotalItem[];
}

const ReportSummary: React.FC<ReportSummaryProps> = ({ totals }) => {
  if (!totals || totals.length === 0) {
    return (
      <div className="bg-white shadow sm:rounded-lg p-4">
        <h3 className="text-lg leading-6 font-medium text-gray-900">
          Report Summary
        </h3>
        <p className="mt-1 text-sm text-gray-500">No summary data available.</p>
      </div>
    );
  }

  return (
    <div className="bg-white shadow overflow-hidden sm:rounded-lg">
      <div className="px-4 py-5 sm:px-6">
        <h3 className="text-lg leading-6 font-medium text-gray-900">
          Report Summary - Totals by Client
        </h3>
        <p className="mt-1 max-w-2xl text-sm text-gray-500">
          Total adjusted hours per client.
        </p>
      </div>
      <div className="border-t border-gray-200">
        <dl>
          {totals.map((item, index) => (
            <div
              key={item.client}
              className={`${
                index % 2 === 0 ? "bg-gray-50" : "bg-white"
              } px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6`}
            >
              <dt className="text-sm font-medium text-gray-500">
                {item.client}
              </dt>
              <dd className="mt-1 text-sm text-gray-900 sm:mt-0 sm:col-span-2">
                {item.adjusted_hours.toFixed(2)} hours
              </dd>
            </div>
          ))}
        </dl>
      </div>
    </div>
  );
};

export default ReportSummary;
