export interface BaseTask {
  task_id: string;
  custom_id: string;
  name: string;
  task_name?: string;
  url: string;
  tags?: string[];
  priority?: string;
}

export interface BillableTask extends BaseTask {
  billable_hours: number;
  invoiced_hours: number;
  monthly_reported: number;
  reporter?: string;
}

export interface TimeTrackingTask extends BaseTask {
  client: string;
  adjusted_hours: number;
  invoiced_hours: number;
  calculated_billable_hours: number;
  status: string;
}

export type Task = BillableTask | TimeTrackingTask;

export interface TaskDetailsProps {
  title: string;
  tasks: Task[];
  type: "billable" | "internal" | "timetracking";
  totalHours?: number;
  className?: string;
}
