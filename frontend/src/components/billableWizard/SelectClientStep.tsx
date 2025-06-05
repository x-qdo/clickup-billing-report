import React from 'react';
import { BuildingOfficeIcon, ArrowRightIcon, ExclamationTriangleIcon, ArrowPathIcon } from '@heroicons/react/24/outline';
import type { ApiClient } from '../../services/api'; // Assuming ApiClient is exported from api.ts

interface SelectClientStepProps {
  availableClients: ApiClient[];
  clientName: string;
  onClientNameChange: (name: string) => void;
  onNext: () => void;
  clientsLoading: boolean;
  clientsError: string | null;
}

const SelectClientStep: React.FC<SelectClientStepProps> = ({
  availableClients,
  clientName,
  onClientNameChange,
  onNext,
  clientsLoading,
  clientsError,
}) => {
  return (
    <div className="space-y-6">
      <div className="text-center">
        <BuildingOfficeIcon className="mx-auto h-12 w-12 text-blue-600" />
        <h3 className="mt-2 text-lg font-medium text-gray-900">
          Select Client
        </h3>
        <p className="mt-1 text-sm text-gray-600">
          Choose the client for which you want to generate the billable report
        </p>
      </div>

      {clientsLoading && (
        <div className="flex items-center justify-center text-gray-600 py-4">
          <ArrowPathIcon className="animate-spin h-6 w-6 mr-3 text-blue-600" />
          <span className="text-sm">Loading clients...</span>
        </div>
      )}

      {clientsError && (
        <div className="bg-red-50 border border-red-300 rounded-md p-4 max-w-md mx-auto">
          <div className="flex">
            <div className="flex-shrink-0">
              <ExclamationTriangleIcon className="h-5 w-5 text-red-400" aria-hidden="true" />
            </div>
            <div className="ml-3">
              <h3 className="text-sm font-medium text-red-800">Error loading clients</h3>
              <div className="mt-2 text-sm text-red-700">
                <p>{clientsError}</p>
              </div>
            </div>
          </div>
        </div>
      )}

      {!clientsLoading && !clientsError && (
        <div className="max-w-md mx-auto">
          <label
            htmlFor="client-name"
            className="block text-sm font-medium text-gray-700 mb-2"
          >
            Client Name
          </label>
          <select
            id="client-name"
            name="client-name"
            value={clientName}
            onChange={(e) => onClientNameChange(e.target.value)}
            className="block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 disabled:bg-gray-50"
            disabled={availableClients.length === 0}
            aria-describedby="client-name-description"
          >
            <option value="">Select a client...</option>
            {availableClients.map((client) => (
              <option key={client.name} value={client.name}>
                {client.name}
              </option>
            ))}
          </select>
          <p className="mt-2 text-xs text-gray-500" id="client-name-description">
            Client name must match exactly as configured in the system.
          </p>
          {availableClients.length === 0 && !clientsLoading && (
            <p className="mt-1 text-xs text-red-600">
              No clients available. Please ensure clients are configured.
            </p>
          )}
        </div>
      )}

      <div className="text-center pt-4">
        <button
          type="button"
          onClick={onNext}
          disabled={!clientName || clientsLoading || !!clientsError || availableClients.length === 0}
          className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          Continue
          <ArrowRightIcon className="ml-2 h-4 w-4" />
        </button>
      </div>
    </div>
  );
};

export default SelectClientStep;
