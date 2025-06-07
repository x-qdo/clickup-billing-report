export interface PersonalReportItem {
  username: string;
  client: string;
  adjusted_hours: number;
  internal_hours: number;
  total_hours: number;
}

import type { TimeTrackingTask } from "../shared/taskTypes";

export type FinalReportItem = TimeTrackingTask & {
  tags: string[];
};

export interface TotalItem {
  client: string;
  adjusted_hours: number;
}

export interface TimeTrackingData {
  personal_report: PersonalReportItem[];
  final_report: FinalReportItem[];
  totals: TotalItem[];
}
