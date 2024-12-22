import datetime
import io
import os
import pandas as pd

from flask import Flask, render_template, request, redirect, url_for, session, Response
import csv
import requests

from demo_bot import generate_demo_list, LIST_ID, TEAM_ID
from disable_logging import disable_logging
from report import generate_report
from timetracking_report import generate_timetracking_report
from toggl_sync import sync_clickup_to_toggl

app = Flask(__name__)
app.secret_key = os.environ['CLICKUP_SESSION_SECRET']
app.config['SERVER_NAME'] = os.environ['CLICKUP_SERVER_NAME']
app.config['PREFERRED_URL_SCHEME'] = os.environ['CLICKUP_URL_SCHEME']


def generate_csv_data(data, fieldnames):
    output = io.StringIO()
    writer = csv.DictWriter(output, fieldnames=fieldnames)
    writer.writeheader()
    for row in data:
        writer.writerow(row)
    return output.getvalue()


def is_token_valid():
    if 'access_token' not in session:
        return False

    token = session['access_token']
    response = requests.get(
        'https://api.clickup.com/api/v2/user',
        headers={'Authorization': token, 'Content-Type': 'application/json'}
    )
    return response.status_code == 200


@app.route('/')
def home():
    if is_token_valid():
        return redirect(url_for('reports_list_route'))
    client_id = os.environ['CLICKUP_CLIENT_ID']
    callback_url = url_for('callback', _external=True)
    return render_template('index.html', client_id=client_id, callback_url=callback_url)


@app.route('/report/demo', methods=['GET'])
def generate_demo_report_route():
    if not is_token_valid():
        return redirect(url_for('home'))

    today = datetime.datetime.now()
    week_number = today.strftime('%U')
    title = f"{today.strftime('%Y-%m-%d')}: Week {week_number} Demo List"
    slack_message = (f"@here Today is {today.strftime('%A')}! Get ready to demo your work today.\n"
                     f"Make sure your work is included in {title}!")

    md = generate_demo_list(session['access_token'], LIST_ID, TEAM_ID)
    return render_template(
        'demo_report.html',
        md=md,
        title=title,
        slack_message=slack_message,
    )


@app.route('/report/billable', methods=['GET', 'POST'])
def generate_report_route():
    if not is_token_valid():
        return redirect(url_for('home'))

    if request.method == 'POST':
        list_id = request.form.get('list_id', '10940440')
        refresh_invoiced = request.form.get('refresh_invoiced') == 'on'

        token = session['access_token']
        report_data = generate_report(list_id, token, refresh_invoiced=refresh_invoiced)

        full_report = ['custom_id', 'name', 'priority', 'tags', 'billable', 'reporter', 'invoiced', 'monthly_reported']
        csv_data = generate_csv_data(report_data, full_report)

        response = Response(csv_data, content_type='text/csv')
        response.headers.set('Content-Disposition', 'attachment', filename="report.csv")
        return response

    return render_template('billable_report.html', title="Generate Billable Report")


@app.route('/report/timetrack', methods=['GET', 'POST'])
def generate_timetrack_route():
    if not is_token_valid():
        return redirect(url_for('home'))

    if request.method == 'POST':
        selected_date = datetime.datetime.strptime(request.form['report_date'], '%Y-%m')
        refresh_billable = request.form.get('refresh_billable') == 'on'
        token = session['access_token']

        report_data = generate_timetracking_report(token, selected_date, refresh_billable)

        return render_template(
            'time_tracking_report.html',
            final_report=report_data['final_report'].to_html(classes='table table-striped table-hover', index=False, float_format=lambda x: '%.2f' % x),
            personal_timereport=report_data['personal_timereport'].to_html(classes='table table-striped table-hover', index=False, float_format=lambda x: '%.2f' % x),
            total=report_data['total'].to_html(classes='table table-striped table-hover', index=False, float_format=lambda x: '%.2f' % x)
        )

    # Generate list of available dates (current month and 3 months back)
    current_date = datetime.datetime.now().replace(day=1)
    available_dates = []
    for i in range(3, -1, -1):
        date = current_date - datetime.timedelta(days=i*30)
        date_str = date.strftime('%Y-%m')
        label = date.strftime('%B %Y')
        available_dates.append((date_str, label))
    
    return render_template(
        'time_tracking.html',
        title="Generate Time Tracking Report",
        available_dates=available_dates,
        current_date=current_date.strftime('%Y-%m')
    )


@app.route('/sync/toggl', methods=['GET', 'POST'])
def sync_toggl_route():
    if not is_token_valid():
        return redirect(url_for('home'))

    if request.method == 'POST':
        start_date = int(datetime.datetime.strptime(request.form['start_date'], '%Y-%m-%d').timestamp() * 1000)
        end_date = int(datetime.datetime.strptime(request.form['end_date'], '%Y-%m-%d').timestamp() * 1000)
        toggl_api_token = request.form['toggl_api_token']

        token = session['access_token']
        result = sync_clickup_to_toggl(token, toggl_api_token, start_date, end_date)

        if isinstance(result, pd.DataFrame):
            result_table = result.to_html(classes='table table-striped', index=False, escape=False, render_links=True)
            return render_template('toggl_sync_results.html', result_table=result_table)
        else:
            return result  # This will be the "All entries synced successfully" message

    return render_template('toggl_sync.html', title="ClickUp to Toggl Time Sync")


@app.route('/report', methods=['GET'])
def reports_list_route():
    return render_template('reports_list.html', title="Report Links")


@app.route('/callback')
def callback():
    code = request.args.get('code')

    if not code:
        return "Error: No code provided."

    client_id = os.environ['CLICKUP_CLIENT_ID']
    client_secret = os.environ['CLICKUP_CLIENT_SECRET']
    redirect_uri = url_for('callback', _external=True)

    r = requests.post(
        url='https://api.clickup.com/api/v2/oauth/token',
        data={
            'code': code,
            'client_id': client_id,
            'client_secret': client_secret,
            'redirect_uri': redirect_uri,
        },
    )

    if r.status_code != 200:
        return "Error: Failed to get access token."

    access_token = r.json()['access_token']
    session['access_token'] = access_token

    return redirect(url_for('reports_list_route'))


@app.route("/healthz", methods=["GET"])
@disable_logging
def health_check():
    return "OK", 200


if __name__ == '__main__':
    app.run()
