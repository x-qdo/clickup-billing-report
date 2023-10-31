from pprint import pprint
import pandas as pd
import ast
import requests
import datetime

team_id = "2454960"
list_id = "10940440"
contract_included = 130
BILLABLE_ID = '074e1387-e7b8-41c6-92db-fbada8f8486c'
INVOICED_ID = '82aa8afc-dbd2-4a80-b4ae-cccd06ba774b'

developer_coefficients = {
    'Yauheni Batsianouski': 1,
    'Evgeny Goroshko': 1,
    'Vladimir Kuznichenkov': 1,
    'Linke Dmitry': 4,
    'Alexander Pavlov': 4,
    'Alex Pavlovsky': 4,
    'Alexey Gorovenko': 3,
}


def extract_custom_field_value(custom_fields, field_name):
    for field in custom_fields:
        if field['name'] == field_name:
            return field.get('value')
    return None


def fetch_and_process_tasks(token):
    # Loading full tasks list
    tasks_data = []
    url = "https://api.clickup.com/api/v2/list/" + list_id + "/task"
    last_page = False
    page = 0

    while not last_page:
        print(f"fetching tasks, page: {page}")
        response = requests.get(url,
                                headers={'Authorization': token, 'Content-Type': 'application/json'},
                                params={
                                    "archived": "false", "page": page, "subtasks": "true", "include_closed": "true",
                                })
        page += 1
        # TODO add status check
        tasks_data.extend(response.json()['tasks'])
        last_page = response.json()['last_page']

    tasks_data = pd.json_normalize(tasks_data).drop(columns=[
        'text_content', 'description', 'orderindex', 'status.status', 'status.color', 'status.type',
        'status.orderindex', 'creator.id', 'creator.username', 'creator.color',
        'creator.email', 'creator.profilePicture', 'sharing.public',
        'sharing.public_share_expires_on', 'sharing.public_fields',
        'sharing.token', 'sharing.seo_optimized', 'permission_level',
        'list.id', 'list.name', 'list.access', 'project.id', 'project.name',
        'project.hidden', 'project.access', 'folder.id', 'folder.name',
        'folder.hidden', 'folder.access', 'space.id', 'priority.color',
        'priority.id', 'priority.orderindex', 'priority.priority', 'checklists',
        'watchers', 'url', 'team_id'])

    pprint(tasks_data.axes)

    tasks_data['InvoicedHours'] = tasks_data['custom_fields'].apply(
        lambda x: extract_custom_field_value(x, 'InvoicedHours'))
    tasks_data['BillableHours'] = tasks_data['custom_fields'].apply(
        lambda x: extract_custom_field_value(x, 'BillableHours'))
    tasks_data['InvoicedHours'] = pd.to_numeric(tasks_data['InvoicedHours'], errors='coerce').fillna(0)

    tasks_data = tasks_data.drop(columns=['custom_fields'])

    return tasks_data


def fetch_and_process_time_report(token, selected_month, tasks_data):
    all_assignees = [assignee for sublist in tasks_data['assignees'].tolist() for assignee in sublist]
    assignees_df = pd.DataFrame(all_assignees)
    unique_assignees_df = assignees_df.drop_duplicates()

    id_list = unique_assignees_df['id'].tolist()

    first_day_of_current_month = datetime.datetime.now().replace(day=1, hour=23, minute=59, second=59, microsecond=0, month=selected_month)
    last_day_of_prev_month = first_day_of_current_month - datetime.timedelta(days=1)
    first_day_of_prev_month = last_day_of_prev_month.replace(day=1)

    url = "https://api.clickup.com/api/v2/team/" + team_id + "/time_entries"
    response = requests.get(url, headers={'Authorization': token, 'Content-Type': 'application/json'},
                            params={
                                "start_date": int(first_day_of_prev_month.timestamp() * 1000),
                                "end_date": int(last_day_of_prev_month.timestamp() * 1000),
                                "assignee": ','.join([str(x) for x in id_list]),
                                "include_task_tags": "true",
                                "list_id": list_id,
                            })
    time_report = response.json()
    time_report_data = pd.json_normalize(time_report['data']).drop(columns=['id',
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
                                                                            'end', 'task_url'])
    time_report_data['duration'] = pd.to_numeric(time_report_data['duration'], errors='coerce')
    time_report_data['AdjustedDuration'] = time_report_data.apply(
        lambda row: row['duration'] / 60 / 60 / 1000 / developer_coefficients[row['user.username']], axis=1)

    return time_report_data


def calculate_personal_timereport(time_report_data):
    final_report = time_report_data.groupby(['user.username'])['AdjustedDuration'].sum().reset_index()
    return final_report.sort_values('AdjustedDuration', ascending=False)


def generate_final_report(tasks_data, time_report_data):
    merged_df = pd.merge(tasks_data, time_report_data, left_on='id', right_on='task.id', how='left')
    merged_df['parent'].fillna(merged_df['id'], inplace=True)

    grouped_df = merged_df.groupby(['parent', 'id'])['AdjustedDuration'].sum().reset_index()

    final_report = pd.merge(grouped_df, tasks_data[['id']], left_on='parent', right_on='id')
    final_report = final_report.rename(columns={'id_y': 'id'})
    final_report = final_report.drop(columns=['parent', 'id_x'])
    final_report = final_report[final_report['AdjustedDuration'] != 0]
    final_report['AdjustedDuration'] = (round(final_report['AdjustedDuration'] * 2) / 2)
    final_report = final_report.groupby(['id'])['AdjustedDuration'].sum().reset_index()

    final_report = pd.merge(final_report, tasks_data[['id', 'name', 'custom_id', 'InvoicedHours']], left_on='id',
                            right_on='id')
    final_report = final_report.sort_values('AdjustedDuration', ascending=False)

    return final_report


# Define function to update custom fields
def update_custom_fields(token, final_report):
    print("Updating tasks")
    for index, row in final_report.iterrows():
        print("updating " + row['custom_id'] + " to " + str(row['InvoicedHours'] + row['AdjustedDuration']))
        url = "https://api.clickup.com/api/v2/task/" + row['id'] + "/field/" + BILLABLE_ID
        response = requests.post(url, headers={'Authorization': token, 'Content-Type': 'application/json'},
                                 json={"value": str(row['InvoicedHours'] + row['AdjustedDuration'])})
        print(response.json())
