import React, { useState, useEffect } from 'react';
import { useAuth } from '../contexts/AuthContext';
import {
  UserIcon,
  BuildingOfficeIcon,
  UsersIcon,
  CogIcon,
  PlusIcon,
  PencilIcon,
  TrashIcon,
} from '@heroicons/react/24/outline';

interface Client {
  id: string;
  name: string;
  description?: string;
  created_at: string;
}

interface Developer {
  id: string;
  name: string;
  coefficient: number;
  email?: string;
}

const Settings: React.FC = () => {
  const { user, logout } = useAuth();
  const [activeTab, setActiveTab] = useState('profile');
  const [clients, setClients] = useState<Client[]>([]);
  const [developers, setDevelopers] = useState<Developer[]>([]);
  const [isLoading] = useState(false);

  const tabs = [
    {
      id: 'profile',
      name: 'Profile',
      icon: UserIcon,
    },
    {
      id: 'clients',
      name: 'Clients',
      icon: BuildingOfficeIcon,
    },
    {
      id: 'developers',
      name: 'Developers',
      icon: UsersIcon,
    },
    {
      id: 'system',
      name: 'System',
      icon: CogIcon,
    },
  ];

  // Mock data for demonstration - in real app, this would come from API
  useEffect(() => {
    setClients([
      { id: '1', name: 'Acme Corp', description: 'Main client project', created_at: '2024-01-15' },
      { id: '2', name: 'Tech Solutions', description: 'Web development services', created_at: '2024-02-01' },
    ]);
    setDevelopers([
      { id: '1', name: 'John Doe', coefficient: 1.0, email: 'john@example.com' },
      { id: '2', name: 'Jane Smith', coefficient: 1.2, email: 'jane@example.com' },
      { id: '3', name: 'Bob Wilson', coefficient: 0.8, email: 'bob@example.com' },
    ]);
  }, []);

  const handleLogout = () => {
    logout();
  };

  const renderProfileTab = () => (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium text-gray-900">User Information</h3>
        <p className="mt-1 text-sm text-gray-600">Your ClickUp account details</p>
      </div>

      <div className="bg-white shadow rounded-lg p-6">
        <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
          <div>
            <label className="block text-sm font-medium text-gray-700">Username</label>
            <div className="mt-1 text-sm text-gray-900">{user?.username || 'N/A'}</div>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">Email</label>
            <div className="mt-1 text-sm text-gray-900">{user?.email || 'N/A'}</div>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">User ID</label>
            <div className="mt-1 text-sm text-gray-900 font-mono">{user?.id || 'N/A'}</div>
          </div>
        </div>

        <div className="mt-6 pt-6 border-t border-gray-200">
          <button
            onClick={handleLogout}
            className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-red-600 hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500"
          >
            Sign Out
          </button>
        </div>
      </div>
    </div>
  );

  const renderClientsTab = () => (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h3 className="text-lg font-medium text-gray-900">Client Configuration</h3>
          <p className="mt-1 text-sm text-gray-600">Manage clients for billable reporting</p>
        </div>
        <button className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500">
          <PlusIcon className="h-4 w-4 mr-2" />
          Add Client
        </button>
      </div>

      <div className="bg-white shadow rounded-lg overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Name
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Description
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Created
              </th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                Actions
              </th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {clients.map((client) => (
              <tr key={client.id}>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                  {client.name}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {client.description}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {new Date(client.created_at).toLocaleDateString()}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                  <button className="text-blue-600 hover:text-blue-900 mr-4">
                    <PencilIcon className="h-4 w-4" />
                  </button>
                  <button className="text-red-600 hover:text-red-900">
                    <TrashIcon className="h-4 w-4" />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );

  const renderDevelopersTab = () => (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h3 className="text-lg font-medium text-gray-900">Developer Configuration</h3>
          <p className="mt-1 text-sm text-gray-600">Manage developer coefficients for time calculations</p>
        </div>
        <button className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500">
          <PlusIcon className="h-4 w-4 mr-2" />
          Add Developer
        </button>
      </div>

      <div className="bg-white shadow rounded-lg overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Name
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Email
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Coefficient
              </th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                Actions
              </th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {developers.map((developer) => (
              <tr key={developer.id}>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                  {developer.name}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {developer.email}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                    developer.coefficient === 1.0 
                      ? 'bg-green-100 text-green-800'
                      : developer.coefficient > 1.0
                      ? 'bg-blue-100 text-blue-800'
                      : 'bg-yellow-100 text-yellow-800'
                  }`}>
                    {developer.coefficient}×
                  </span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                  <button className="text-blue-600 hover:text-blue-900 mr-4">
                    <PencilIcon className="h-4 w-4" />
                  </button>
                  <button className="text-red-600 hover:text-red-900">
                    <TrashIcon className="h-4 w-4" />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="bg-blue-50 rounded-lg p-4">
        <h4 className="text-sm font-medium text-blue-900 mb-2">About Coefficients</h4>
        <p className="text-sm text-blue-800">
          Coefficients are multipliers applied to developer time entries. Use values greater than 1.0 for senior developers 
          and less than 1.0 for junior developers to adjust billable hours based on experience level.
        </p>
      </div>
    </div>
  );

  const renderSystemTab = () => (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium text-gray-900">System Configuration</h3>
        <p className="mt-1 text-sm text-gray-600">Application settings and preferences</p>
      </div>

      <div className="bg-white shadow rounded-lg p-6 space-y-6">
        <div>
          <h4 className="text-sm font-medium text-gray-900 mb-3">DynamoDB Tables</h4>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div className="bg-gray-50 p-3 rounded-md">
              <div className="text-sm font-medium text-gray-700">Clients</div>
              <div className="text-xs text-gray-500">Stores client configurations</div>
            </div>
            <div className="bg-gray-50 p-3 rounded-md">
              <div className="text-sm font-medium text-gray-700">Developers</div>
              <div className="text-xs text-gray-500">Stores developer coefficients</div>
            </div>
            <div className="bg-gray-50 p-3 rounded-md">
              <div className="text-sm font-medium text-gray-700">Settings</div>
              <div className="text-xs text-gray-500">Application settings</div>
            </div>
            <div className="bg-gray-50 p-3 rounded-md">
              <div className="text-sm font-medium text-gray-700">Sessions</div>
              <div className="text-xs text-gray-500">User session data</div>
            </div>
          </div>
        </div>

        <div className="border-t pt-6">
          <h4 className="text-sm font-medium text-gray-900 mb-3">Application Info</h4>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div>
              <div className="text-sm font-medium text-gray-700">Version</div>
              <div className="text-sm text-gray-500">1.0.0</div>
            </div>
            <div>
              <div className="text-sm font-medium text-gray-700">Environment</div>
              <div className="text-sm text-gray-500">Production</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );

  const renderTabContent = () => {
    switch (activeTab) {
      case 'profile':
        return renderProfileTab();
      case 'clients':
        return renderClientsTab();
      case 'developers':
        return renderDevelopersTab();
      case 'system':
        return renderSystemTab();
      default:
        return renderProfileTab();
    }
  };

  return (
    <div className="px-4 sm:px-6 lg:px-8">
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900">Settings</h1>
        <p className="mt-1 text-sm text-gray-600">
          Manage your account, clients, and system configuration
        </p>
      </div>

      <div className="bg-white shadow rounded-lg">
        {/* Tab Navigation */}
        <div className="border-b border-gray-200">
          <nav className="-mb-px flex space-x-8 px-6" aria-label="Tabs">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`${
                  activeTab === tab.id
                    ? 'border-blue-500 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                } whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm flex items-center`}
              >
                <tab.icon className="h-4 w-4 mr-2" />
                {tab.name}
              </button>
            ))}
          </nav>
        </div>

        {/* Tab Content */}
        <div className="p-6">
          {isLoading ? (
            <div className="flex items-center justify-center py-12">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
            </div>
          ) : (
            renderTabContent()
          )}
        </div>
      </div>
    </div>
  );
};

export default Settings;