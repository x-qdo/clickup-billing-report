export interface PersonalReportItem {
  username: string;
  client: string;
  adjusted_hours: number;
  internal_hours: number;
  total_hours: number;
}

export interface FinalReportItem {
  task_id: string;
  custom_id: string;
  tags: string[];
  task_name: string;
  client: string;
  adjusted_hours: number;
  invoiced_hours: number;
  calculated_billable_hours: number;
  status: string;
  url: string;
}

export interface TotalItem {
  client: string;
  adjusted_hours: number;
}

export interface TimeTrackingData {
  personal_report: PersonalReportItem[];
  final_report: FinalReportItem[];
  totals: TotalItem[];
}
