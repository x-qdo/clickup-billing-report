import React from "react";
import type { PersonalReportItem } from "../../types/api/timeTrackingTypes";

interface DeveloperReportProps {
  personalReport: PersonalReportItem[];
}

const DeveloperReport: React.FC<DeveloperReportProps> = ({
  personalReport,
}) => {
  if (!personalReport || personalReport.length === 0) {
    return (
      <div className="bg-white shadow sm:rounded-lg p-4 mt-6">
        <h3 className="text-lg leading-6 font-medium text-gray-900">
          Developer Report
        </h3>
        <p className="mt-1 text-sm text-gray-500">
          No developer report data available.
        </p>
      </div>
    );
  }

  return (
    <div className="mt-6 flex flex-col">
      <div className="-my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
        <div className="py-2 align-middle inline-block min-w-full sm:px-6 lg:px-8">
          <div className="shadow overflow-hidden border-b border-gray-200 sm:rounded-lg">
            <div className="px-4 py-5 sm:px-6 bg-white">
              <h3 className="text-lg leading-6 font-medium text-gray-900">
                Developer Report - Personal Time Summary
              </h3>
              <p className="mt-1 max-w-2xl text-sm text-gray-500">
                Breakdown of hours by developer and client.
              </p>
            </div>
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th
                    scope="col"
                    className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
                  >
                    Developer
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
                    Adjusted Hours
                  </th>
                  <th
                    scope="col"
                    className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
                  >
                    Internal Hours
                  </th>
                  <th
                    scope="col"
                    className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
                  >
                    Total Hours
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {personalReport.map((item, index) => (
                  <tr key={`${item.username}-${item.client}-${index}`}>
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                      {item.username}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {item.client}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {item.adjusted_hours.toFixed(2)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {item.internal_hours.toFixed(2)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {item.total_hours.toFixed(2)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
};

export default DeveloperReport;
