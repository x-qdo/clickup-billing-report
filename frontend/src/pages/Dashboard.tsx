import React from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../contexts/AuthContext";
import {
  ClockIcon,
  CurrencyDollarIcon,
  ArrowRightIcon,
} from "@heroicons/react/24/outline";

const Dashboard: React.FC = () => {
  const { user } = useAuth();

  const quickActions = [
    {
      name: "Time Tracking Report",
      description:
        "Generate monthly time tracking reports with developer coefficients",
      href: "/timetracking",
      icon: ClockIcon,
      color: "bg-blue-500 hover:bg-blue-600",
      steps: [
        "Select report month",
        "Review calculated hours",
        "Update ClickUp billable fields",
      ],
    },
    {
      name: "Billable Report",
      description:
        "Create client-specific billable reports and mark tasks as invoiced",
      href: "/billable",
      icon: CurrencyDollarIcon,
      color: "bg-green-500 hover:bg-green-600",
      steps: [
        "Select client",
        "Review billable tasks",
        "Update invoiced status",
      ],
    },
  ];

  return (
    <div className="px-4 sm:px-6 lg:px-8">
      {/* Welcome Section */}
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900">
          Welcome back, {user?.username || "User"}!
        </h1>
        <p className="mt-1 text-sm text-gray-600">
          Generate time tracking and billable reports from your ClickUp data
        </p>
      </div>

      {/* Quick Actions */}
      <div className="mb-8">
        <h2 className="text-lg font-medium text-gray-900 mb-4">
          Quick Actions
        </h2>
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          {quickActions.map((action) => (
            <div
              key={action.name}
              className="bg-white overflow-hidden shadow rounded-lg hover:shadow-md transition-shadow duration-200"
            >
              <div className="p-6">
                <div className="flex items-center">
                  <div
                    className={`flex-shrink-0 p-3 rounded-md ${action.color}`}
                  >
                    <action.icon className="h-6 w-6 text-white" />
                  </div>
                  <div className="ml-4 flex-1">
                    <h3 className="text-lg font-medium text-gray-900">
                      {action.name}
                    </h3>
                    <p className="text-sm text-gray-600 mt-1">
                      {action.description}
                    </p>
                  </div>
                </div>

                <div className="mt-4">
                  <h4 className="text-sm font-medium text-gray-700 mb-2">
                    Steps:
                  </h4>
                  <ol className="text-sm text-gray-600 space-y-1">
                    {action.steps.map((step, index) => (
                      <li key={index} className="flex items-center">
                        <span className="flex-shrink-0 w-5 h-5 bg-gray-100 rounded-full flex items-center justify-center text-xs font-medium text-gray-600 mr-2">
                          {index + 1}
                        </span>
                        {step}
                      </li>
                    ))}
                  </ol>
                </div>

                <div className="mt-6">
                  <Link
                    to={action.href}
                    className={`inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white transition-colors duration-200 ${action.color}`}
                  >
                    Start Wizard
                    <ArrowRightIcon className="ml-2 h-4 w-4" />
                  </Link>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Workflow Information */}
      <div className="bg-blue-50 rounded-lg p-6">
        <h3 className="text-lg font-medium text-blue-900 mb-3">
          Recommended Workflow
        </h3>
        <div className="prose prose-sm text-blue-800">
          <ol className="space-y-2">
            <li>
              <strong>Generate Time Tracking Report:</strong> Start with
              reviewing time entries without updating ClickUp fields
            </li>
            <li>
              <strong>Review & Verify:</strong> Check calculations, developer
              coefficients, and task assignments
            </li>
            <li>
              <strong>Update Billable Fields:</strong> Re-run the report with
              field updates enabled
            </li>
            <li>
              <strong>Generate Billable Report:</strong> Create client-specific
              reports for invoicing
            </li>
            <li>
              <strong>Mark as Invoiced:</strong> Update invoiced status to track
              billing progress
            </li>
          </ol>
        </div>
      </div>
    </div>
  );
};

export default Dashboard;
