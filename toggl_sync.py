import base64

import requests
from typing import List
import pandas as pd
from client import Client, clients
from datetime import datetime


def extract_custom_field_value(custom_fields, field_name):
    for field in custom_fields:
        if field['name'] == field_name:
            return field.get('value')
    return None


def fetch_task_details(token: str, task_id: str) -> dict:
    url = f"https://api.clickup.com/api/v2/task/{task_id}"
    response = requests.get(
        url,
        headers={'Authorization': token, 'Content-Type': 'application/json'}
    )
    return response.json()


def fetch_clickup_time_entries(token: str, client: Client, start_date: int, end_date: int) -> pd.DataFrame:
    url = f"https://api.clickup.com/api/v2/team/{client.team_id}/time_entries"
    response = requests.get(
        url,
        headers={'Authorization': token, 'Content-Type': 'application/json'},
        params={
            "start_date": start_date,
            "end_date": end_date,
            "include_task_tags": "true",
            "list_id": client.list_id,
        }
    )
    time_entries = response.json()['data']

    df = pd.json_normalize(time_entries)

    # Fetch task details for each unique task
    unique_tasks = df['task.id'].unique()
    task_details = {}
    for task_id in unique_tasks:
        task_details[task_id] = fetch_task_details(token, task_id)

    # Add Toggl task name to the dataframe
    df['toggl_task_name'] = df['task.id'].apply(
        lambda x: extract_custom_field_value(task_details[x]['custom_fields'], 'Toggl Task Name'))

    return df[['id', 'task.id', 'task.name', 'user.username', 'duration', 'start', 'end', 'toggl_task_name']]


def fetch_toggl_tasks(toggl_api_token: str, workspace_id: str, page: int = 1, per_page: int = 1000,
                      sort_order: str = 'ASC', sort_field: str = 'name',
                      active: bool = True) -> List[dict]:
    url = f"https://api.track.toggl.com/api/v9/workspaces/{workspace_id}/tasks"
    headers = {
        "Authorization": f"Basic {base64.b64encode(f'{toggl_api_token}:api_token'.encode()).decode()}"
    }
    params = {
        'page': page,
        'per_page': per_page,
        'sort_order': sort_order,
        'sort_field': sort_field
    }
    if active is not None:
        params['active'] = 'both' if active == 'both' else str(active).lower()

    response = requests.get(url, headers=headers, params=params)
    response.raise_for_status()  # Raise an exception for bad status codes
    return response.json()['data']


def find_toggl_task(toggl_tasks: List[dict], task_name: str) -> dict:
    for task in toggl_tasks:
        if task['name'] == task_name:
            return task
    return None


def shift_overlaps(time_entries: pd.DataFrame) -> pd.DataFrame:
    time_entries = time_entries.sort_values('start')
    shifted_entries = []

    for i, entry in time_entries.iterrows():
        if not shifted_entries or entry['start'] >= shifted_entries[-1]['end']:
            shifted_entries.append(entry)
        else:
            # Shift the start time of the current entry
            new_start = shifted_entries[-1]['end']
            duration = entry['duration']
            new_end = new_start + duration

            shifted_entry = entry.copy()
            shifted_entry['start'] = new_start
            shifted_entry['end'] = new_end
            shifted_entries.append(shifted_entry)

    return pd.DataFrame(shifted_entries)


def sync_to_toggl(clickup_entries: pd.DataFrame, toggl_api_token: str, workspace_id: str, client_name: str):
    toggl_tasks = fetch_toggl_tasks(toggl_api_token, workspace_id)
    error_entries = []
    synced_entries = []

    for _, entry in clickup_entries.iterrows():
        if pd.isna(entry['toggl_task_name']) or entry['toggl_task_name'] == '':
            error_entries.append({
                'Client': client_name,
                'ClickUp Task': entry['task.name'],
                'ClickUp Link': f"https://app.clickup.com/t/{entry['task.id']}",
                'Toggl Task Name': 'Not specified',
                'Error': 'Toggl Task Name is not filled'
            })
            continue

        toggl_task = find_toggl_task(toggl_tasks, entry['toggl_task_name'])

        if not toggl_task:
            error_entries.append({
                'Client': client_name,
                'ClickUp Task': entry['task.name'],
                'ClickUp Link': f"https://app.clickup.com/t/{entry['task.id']}",
                'Toggl Task Name': entry['toggl_task_name'],
                'Error': 'No matching Toggl task found'
            })
            continue

        # Convert milliseconds to seconds
        duration = int(entry['duration']) / 1000
        start_time = datetime.fromtimestamp(int(entry['start']) / 1000)

        # Create time entry in Toggl
        url = f"https://api.track.toggl.com/api/v9/workspaces/{workspace_id}/time_entries"
        headers = {
            "Content-Type": "application/json",
            "Authorization": f"Basic {base64.b64encode(f'{toggl_api_token}:api_token'.encode()).decode()}"
        }

        # Create ClickUp task link
        clickup_task_link = f"https://app.clickup.com/t/{entry['task.id']}"

        data = {
            "description": f"{entry['task.name']} - {clickup_task_link}",
            "workspace_id": int(workspace_id),
            "project_id": toggl_task['project_id'],
            "task_id": toggl_task['id'],
            "duration": int(duration),
            "start": start_time.isoformat() + "Z",
            "created_with": "ClickUp Sync",
            "billable": False  # You can change this if needed
        }

        response = requests.post(url, json=data, headers=headers)

        if response.status_code == 200:
            synced_entries.append({
                'Client': client_name,
                'ClickUp Task': entry['task.name'],
                'ClickUp Link': clickup_task_link,
                'Toggl Task Name': entry['toggl_task_name'],
                'Status': 'Synced successfully'
            })
        else:
            error_entries.append({
                'Client': client_name,
                'ClickUp Task': entry['task.name'],
                'ClickUp Link': clickup_task_link,
                'Toggl Task Name': entry['toggl_task_name'],
                'Error': f"Failed to sync. Status code: {response.status_code}, Response: {response.text}"
            })

    return error_entries, synced_entries


def sync_clickup_to_toggl(token: str, toggl_api_token: str, start_date: int, end_date: int):
    all_error_entries = []
    all_synced_entries = []

    for client in clients:
        if client.toggl_sync_enabled:
            print(f"Syncing time entries for {client.name}")
            clickup_entries = fetch_clickup_time_entries(token, client, start_date, end_date)

            # Group entries by task
            grouped_entries = clickup_entries.groupby('task.id')

            shifted_entries = pd.DataFrame()
            for _, group in grouped_entries:
                shifted_group = shift_overlaps(group)
                shifted_entries = pd.concat([shifted_entries, shifted_group])

            error_entries, synced_entries = sync_to_toggl(shifted_entries, toggl_api_token, client.toggl_workspace_id, client.name)
            all_error_entries.extend(error_entries)
            all_synced_entries.extend(synced_entries)

    if all_error_entries or all_synced_entries:
        return pd.DataFrame(all_error_entries + all_synced_entries)
    else:
        return "All entries synced successfully"
