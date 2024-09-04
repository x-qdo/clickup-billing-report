from pprint import pprint
from typing import List, Dict

import pandas as pd
import ast
import requests
import datetime

from client import Client, clients

developer_coefficients = {
    'Yauheni Batsianouski': 1,
    'Evgeny Goroshko': 1,
    'Vladimir Kuznichenkov': 1,
    'Vlad Nikiforov': 1,
    'Linke Dmitry': 3,
    'Alexander Pavlov': 4,
    'Alexey Gorovenko': 3,
}


def extract_custom_field_value(custom_fields, field_name):
    for field in custom_fields:
        if field['name'] == field_name:
            return field.get('value')
    return None


def extract_custom_field_id(custom_fields, field_name):
    for field in custom_fields:
        if field['name'] == field_name:
            return field.get('id')
    return None


def fetch_and_process_tasks(token: str, client: Client) -> pd.DataFrame:
    # Loading full tasks list
    tasks_data = []
    url = f"https://api.clickup.com/api/v2/list/{client.list_id}/task"
    last_page = False
    page = 0

    while not last_page:
        print(f"fetching tasks for {client.name}, page: {page}")
        response = requests.get(
            url,
            headers={'Authorization': token, 'Content-Type': 'application/json'},
            params={
                "archived": "false", "page": page, "subtasks": "true", "include_closed": "true",
            }
        )
        page += 1
        # TODO add status check
        tasks_data.extend(response.json()['tasks'])
        last_page = response.json()['last_page']

    tasks_data = pd.json_normalize(tasks_data)

    # List of columns to drop if they exist
    columns_to_drop = [
        'text_content', 'description', 'orderindex', 'status.status', 'status.color', 'status.type',
        'status.orderindex', 'creator.id', 'creator.username', 'creator.color',
        'creator.email', 'creator.profilePicture', 'sharing.public',
        'sharing.public_share_expires_on', 'sharing.public_fields',
        'sharing.token', 'sharing.seo_optimized', 'permission_level',
        'list.id', 'list.name', 'list.access', 'project.id', 'project.name',
        'project.hidden', 'project.access', 'folder.id', 'folder.name',
        'folder.hidden', 'folder.access', 'space.id', 'priority.color',
        'priority.id', 'priority.orderindex', 'priority.priority', 'checklists',
        'watchers', 'url', 'team_id'
    ]

    # Drop columns if they exist, ignore if they don't
    tasks_data = tasks_data.drop(columns=columns_to_drop, errors='ignore')

    pprint(tasks_data.axes)

    tasks_data['InvoicedHours'] = tasks_data['custom_fields'].apply(
        lambda x: extract_custom_field_value(x, 'InvoicedHours'))
    tasks_data['BillableHours'] = tasks_data['custom_fields'].apply(
        lambda x: extract_custom_field_value(x, 'BillableHours'))
    tasks_data['InvoicedHours'] = pd.to_numeric(tasks_data['InvoicedHours'], errors='coerce').fillna(0)

    # tasks_data = tasks_data.drop(columns=['custom_fields'])
    tasks_data['client'] = client.name

    return tasks_data


def fetch_and_process_time_report(token: str, selected_month: int, tasks_data: pd.DataFrame,
                                  client: Client) -> pd.DataFrame:
    def calculate_adjusted_duration(row):
        if row['user.username'] in developer_coefficients:
            return row['duration'] / 60 / 60 / 1000 / developer_coefficients[row['user.username']]
        else:
            return row['duration'] / 60 / 60 / 1000 / 1  # Default coefficient is 1 if username not found

    all_assignees = [assignee for sublist in tasks_data['assignees'].tolist() for assignee in sublist]
    assignees_df = pd.DataFrame(all_assignees)
    unique_assignees_df = assignees_df.drop_duplicates()

    id_list = unique_assignees_df['id'].tolist()

    first_day_of_current_month = datetime.datetime.now().replace(
        day=1, hour=23, minute=59, second=59, microsecond=0,
        month=selected_month)
    last_day_of_prev_month = first_day_of_current_month - datetime.timedelta(days=1)
    first_day_of_prev_month = last_day_of_prev_month.replace(day=1)

    url = f"https://api.clickup.com/api/v2/team/{client.team_id}/time_entries"
    response = requests.get(url, headers={'Authorization': token, 'Content-Type': 'application/json'},
                            params={
                                "start_date": int(first_day_of_prev_month.timestamp() * 1000),
                                "end_date": int(last_day_of_prev_month.timestamp() * 1000),
                                "assignee": ','.join([str(x) for x in id_list]),
                                "include_task_tags": "true",
                                "list_id": client.list_id,
                            })
    time_report = response.json()
    time_report_data = (pd.json_normalize(time_report['data']).drop(columns=[
        'id',
        'task.status.status', 'task.status.color',
        'task.status.type',
        'task.status.orderindex',
        'task.custom_type',
        'user.email', 'user.color', 'user.initials',
        'user.profilePicture',
        'task_location.list_id',
        'task_location.folder_id',
        'task_location.space_id', 'billable',
        'description', 'tags', 'source', 'at',
        'start',
        'end', 'task_url']))
    time_report_data['duration'] = pd.to_numeric(time_report_data['duration'], errors='coerce')
    time_report_data['TotalDuration'] = time_report_data.apply(lambda row: row['duration'] / 60 / 60 / 1000, axis=1)
    time_report_data['AdjustedDuration'] = time_report_data.apply(calculate_adjusted_duration, axis=1)
    time_report_data['client'] = client.name

    return time_report_data


def calculate_personal_timereport(time_report_data: pd.DataFrame) -> pd.DataFrame:
    final_report = time_report_data.groupby(['user.username', 'client'])[
        ['AdjustedDuration', 'TotalDuration']].sum().reset_index()
    return final_report.sort_values(['client', 'AdjustedDuration'], ascending=[True, False])


def generate_final_report(tasks_data: pd.DataFrame, time_report_data: pd.DataFrame) -> pd.DataFrame:
    merged_df = pd.merge(tasks_data, time_report_data, left_on=['id', 'client'], right_on=['task.id', 'client'],
                         how='left')
    merged_df['parent'].fillna(merged_df['id'], inplace=True)

    grouped_df = merged_df.groupby(['parent', 'id', 'client'])['AdjustedDuration'].sum().reset_index()

    final_report = pd.merge(grouped_df, tasks_data[['id', 'client']], left_on=['parent', 'client'],
                            right_on=['id', 'client'])
    final_report = final_report.rename(columns={'id_y': 'id'})
    final_report = final_report.drop(columns=['parent', 'id_x'])
    final_report = final_report[final_report['AdjustedDuration'] != 0]
    final_report['AdjustedDuration'] = (round(final_report['AdjustedDuration'] * 2) / 2)
    final_report = final_report.groupby(['id', 'client'])['AdjustedDuration'].sum().reset_index()

    final_report = pd.merge(final_report, tasks_data[['id', 'name', 'custom_id', 'InvoicedHours', 'client']],
                            left_on=['id', 'client'], right_on=['id', 'client'])
    final_report = final_report.sort_values(['client', 'AdjustedDuration'], ascending=[True, False])

    return final_report


# Define function to update custom fields
def update_custom_fields(token: str, final_report: pd.DataFrame, clients: List[Client], tasks_data: pd.DataFrame):
    print("Updating tasks")
    for index, row in final_report.iterrows():
        client = next((c for c in clients if c.name == row['client']), None)
        if client:
            task_data = tasks_data[tasks_data['id'] == row['id']].iloc[0]
            billable_id = extract_custom_field_id(task_data['custom_fields'], 'BillableHours')
            if billable_id:
                print(
                    f"updating {row['custom_id']} to {row['InvoicedHours'] + row['AdjustedDuration']} for client {client.name}")
                url = f"https://api.clickup.com/api/v2/task/{row['id']}/field/{billable_id}"
                response = requests.post(url, headers={'Authorization': token, 'Content-Type': 'application/json'},
                                         json={"value": str(row['InvoicedHours'] + row['AdjustedDuration'])})
                print(response.json())
            else:
                print(f"BillableHours field not found for task {row['custom_id']}")
        else:
            print(f"Client not found for task {row['custom_id']}")


def generate_timetracking_report(token: str, selected_month: int, refresh_billable: bool = False) -> Dict[
    str, pd.DataFrame]:
    all_tasks_data = pd.DataFrame()
    all_time_report_data = pd.DataFrame()

    for client in clients:
        tasks_data = fetch_and_process_tasks(token, client)
        time_report_data = fetch_and_process_time_report(token, selected_month, tasks_data, client)

        all_tasks_data = pd.concat([all_tasks_data, tasks_data])
        all_time_report_data = pd.concat([all_time_report_data, time_report_data])

    personal_timereport = calculate_personal_timereport(all_time_report_data)
    final_report = generate_final_report(all_tasks_data, all_time_report_data)

    if refresh_billable:
        update_custom_fields(token, final_report, clients, all_tasks_data)

    return {
        'final_report': final_report,
        'personal_timereport': personal_timereport,
        'total': final_report.groupby('client')['AdjustedDuration'].sum().reset_index()
    }
