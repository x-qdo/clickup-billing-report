import csv
import requests
import argparse

BILLABLE_ID = '074e1387-e7b8-41c6-92db-fbada8f8486c'
INVOICED_ID = '82aa8afc-dbd2-4a80-b4ae-cccd06ba774b'
REPORTED_BY = 'eb30f61c-dbad-4ad4-896d-15d2a239cb69'

def generate_report(list_id, token, refresh_invoiced=False):
    r = requests.get(
        url='https://api.clickup.com/api/v2/list/%s/task' % list_id,
        params={
            'include_closed': True,
            'archived': False,
            'custom_fields': '[{"field_id": "%s", "operator": ">", "value": 0}]' % BILLABLE_ID,
        },
        headers={'Authorization': token, 'Content-Type': 'application/json'}
    )

    reported_tasks = []
    totals = {
        'billable': 0.0,
        'invoiced': 0.0,
        'monthly_reported': 0.0,
    }
    for task in r.json()['tasks']:
        custom_fields = {}
        for f in task['custom_fields']:
            if 'value' in f:
                custom_fields[f['id']] = f['value']
            else:
                custom_fields[f['id']] = 0

        reported_task = {
            'custom_id': task['custom_id'],
            'name': task['name'],
            'priority': task['priority']['priority'] if task['priority'] else '-',
            'tags': ", ".join([t['name'] for t in task['tags']]),
            'reporter': custom_fields[REPORTED_BY] if REPORTED_BY in custom_fields else '-',
            'billable': float(custom_fields[BILLABLE_ID] if BILLABLE_ID in custom_fields else 0),
            'invoiced': float(custom_fields[INVOICED_ID] if INVOICED_ID in custom_fields else 0),
        }
        reported_task['monthly_reported'] = reported_task['billable'] - reported_task['invoiced']

        # skip 0 invoced tasks
        if reported_task['monthly_reported'] == 0:
            continue
        
        reported_tasks.append(reported_task)

        if refresh_invoiced:
            # set invoiced equal billable
            u = requests.post(
                url='https://api.clickup.com/api/v2/task/%s/field/%s' % (task['id'], INVOICED_ID),
                headers={'Authorization': token, 'Content-Type': 'application/json'},
                json={'value': reported_task['billable']}
            )
            print('Task %s updated. R: %s' % (task['custom_id'], u.status_code))

        for key in totals:
            totals[key] += reported_task[key]

    reported_tasks.append(totals)
    return reported_tasks


parser = argparse.ArgumentParser(description='ClickUp report builder')
parser.add_argument('--token', required=True, help='Personal auth topic. Can be found '
                                                   'https://app.clickup.com/2454960/settings/apps')
parser.add_argument('--list-id', default='10940440')
parser.add_argument('--refresh-invoiced', action='store_true')

args = parser.parse_args()

data = generate_report(**vars(args))

full_report = ['custom_id', 'name', 'priority', 'tags', 'billable', 'reporter', 'invoiced', 'monthly_reported']
csv_file = "insly_report.csv"
try:
    with open(csv_file, 'w') as csvfile:
        writer = csv.DictWriter(csvfile, fieldnames=full_report)
        writer.writeheader()
        for data in data:
            writer.writerow(data)
except IOError:
    print("I/O error")
