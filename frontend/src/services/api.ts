import axios, { type AxiosInstance, type AxiosResponse } from "axios";

// --- Type Definitions ---

// Based on Go config.Client struct
export interface ApiClient {
  name: string;
  clickup_list_id: string;
  clickup_team_id: string;
  contract_included: number;
  toggl_sync_enabled: boolean;
  toggl_workspace_id?: string;
  created_at: string; // Dates are typically serialized as ISO strings
  updated_at: string;
}

// Job-related types
export type JobType = "timetrack" | "billable";
export type JobStatus = "pending" | "running" | "completed" | "failed";

export interface JobCreateRequest {
  type: JobType;
  params: Record<string, unknown>;
}

export interface JobCreateResponse {
  job_id: string;
  status: JobStatus;
}

export interface JobStatusResponse {
  job_id: string;
  type: JobType;
  status: JobStatus;
  created_at: string;
  updated_at: string;
  result?: unknown;
  error?: string;
}

export interface JobFileResult {
  download_url: string;
  filename: string;
}

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "";

class ApiService {
  private client: AxiosInstance;

  constructor() {
    this.client = axios.create({
      baseURL: API_BASE_URL,
      timeout: 30000,
      withCredentials: true,
      headers: {
        "Content-Type": "application/json",
      },
    });

    // Request interceptor
    this.client.interceptors.request.use(
      (config) => {
        return config;
      },
      (error) => {
        return Promise.reject(error);
      },
    );

    // Response interceptor
    this.client.interceptors.response.use(
      (response) => {
        return response;
      },
      (error) => {
        if (error.response?.status === 401) {
          // Clear any stored auth state and redirect to login
          localStorage.removeItem("timeTrackingWizardState");
          localStorage.removeItem("billableWizardState");

          // Use React Router navigation instead of direct redirect
          if (window.location.pathname !== "/login") {
            window.location.href = "/login";
          }
        }
        return Promise.reject(error);
      },
    );
  }
  // Specific API methods for the ClickUp Reporter

  // Fetch list of clients
  async listClients(): Promise<AxiosResponse<ApiClient[]>> {
    return this.client.get<ApiClient[]>("/api/clients");
  }

  // Generic HTTP methods
  async get<T = any>(url: string, params?: any): Promise<AxiosResponse<T>> {
    return this.client.get(url, { params });
  }

  async post<T = any>(url: string, data?: any): Promise<AxiosResponse<T>> {
    return this.client.post(url, data);
  }

  async postForm<T = any>(
    url: string,
    data: FormData | URLSearchParams,
  ): Promise<AxiosResponse<T>> {
    return this.client.post(url, data, {
      headers: {
        "Content-Type": "application/x-www-form-urlencoded",
      },
    });
  }

  async put<T = any>(url: string, data?: any): Promise<AxiosResponse<T>> {
    return this.client.put(url, data);
  }

  async delete<T = any>(url: string): Promise<AxiosResponse<T>> {
    return this.client.delete(url);
  }

  // Specific API methods for the ClickUp Reporter

  // Time Tracking Report
  async generateTimeTrackingReport(params: {
    report_date: string;
    refresh_billable?: boolean;
    format?: "json" | "excel";
  }): Promise<AxiosResponse> {
    const formData = new URLSearchParams();
    formData.append("report_date", params.report_date);

    if (params.refresh_billable) {
      formData.append("refresh_billable", "on");
    }

    if (params.format === "excel") {
      formData.append("format", "excel");

      // For Excel downloads, we need to handle blob response
      return this.client.post("/report/timetrack", formData, {
        headers: {
          "Content-Type": "application/x-www-form-urlencoded",
        },
        responseType: "blob",
      });
    }

    return this.client.post("/report/timetrack", formData, {
      headers: {
        "Content-Type": "application/x-www-form-urlencoded",
      },
    });
  }

  // Billable Report
  async generateBillableReport(params: {
    client_name: string;
    refresh_invoiced?: boolean;
    format?: "json" | "excel";
  }): Promise<AxiosResponse> {
    const formData = new URLSearchParams();
    formData.append("client_name", params.client_name);

    if (params.refresh_invoiced) {
      formData.append("refresh_invoiced", "on");
    }

    if (params.format === "excel") {
      formData.append("format", "excel");

      // For Excel downloads, we need to handle blob response
      return this.client.post("/report/billable", formData, {
        headers: {
          "Content-Type": "application/x-www-form-urlencoded",
        },
        responseType: "blob",
      });
    }

    return this.client.post("/report/billable", formData, {
      headers: {
        "Content-Type": "application/x-www-form-urlencoded",
      },
    });
  }

  // Auth methods
  async checkAuthStatus(): Promise<AxiosResponse> {
    try {
      return await this.get("/auth/me");
    } catch (error: any) {
      // Don't redirect on auth check failures to avoid infinite loops
      throw error;
    }
  }

  async logout(): Promise<AxiosResponse> {
    try {
      const response = await this.post("/auth/logout");
      // Clear wizard states on logout
      localStorage.removeItem("timeTrackingWizardState");
      localStorage.removeItem("billableWizardState");
      return response;
    } catch (error: any) {
      // Even if logout fails on server, clear local state
      localStorage.removeItem("timeTrackingWizardState");
      localStorage.removeItem("billableWizardState");
      throw error;
    }
  }

  // Helper method to initiate ClickUp OAuth flow
  initiateLogin(): void {
    window.location.href = `${API_BASE_URL}/auth/clickup`;
  }

  // Utility method for downloading files
  downloadFile(blob: Blob, filename: string): void {
    try {
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = filename;
      link.style.display = "none";
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);
    } catch (error) {
      console.error("Failed to download file:", error);
      throw new Error("Failed to download file");
    }
  }

  // Helper method to format error messages
  getErrorMessage(error: any): string {
    if (error.response?.data?.message) {
      return error.response.data.message;
    }
    if (error.response?.data?.error) {
      return error.response.data.error;
    }
    if (error.message) {
      return error.message;
    }
    return "An unexpected error occurred";
  }

  // --- Job API Methods ---

  // Create a new async job
  async createJob(
    type: JobType,
    params: Record<string, unknown>,
  ): Promise<AxiosResponse<JobCreateResponse>> {
    return this.client.post<JobCreateResponse>("/api/jobs", { type, params });
  }

  // Get job status
  async getJobStatus(jobId: string): Promise<AxiosResponse<JobStatusResponse>> {
    return this.client.get<JobStatusResponse>(`/api/jobs/${jobId}`);
  }

  // Helper to download file from presigned URL
  async downloadFromUrl(url: string, filename: string): Promise<void> {
    const response = await fetch(url);
    const blob = await response.blob();
    this.downloadFile(blob, filename);
  }
}

export const apiService = new ApiService();
