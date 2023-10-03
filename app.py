import datetime
import io
import os

from flask import Flask, render_template, request, redirect, url_for, session, Response
import csv
import requests

from disable_logging import disable_logging
from report import generate_report
from timetracking_report import calculate_personal_timereport, fetch_and_process_tasks, fetch_and_process_time_report, \
    generate_final_report, update_custom_fields

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

    return render_template('billable_report.html')


@app.route('/report/timetrack', methods=['GET', 'POST'])
def generate_timetrack_route():
    if not is_token_valid():
        return redirect(url_for('home'))

    if request.method == 'POST':
        selected_month = int(request.form['month'])
        refresh_billable = request.form.get('refresh_billable') == 'on'
        token = session['access_token']

        tasks_data = fetch_and_process_tasks(token)
        time_report_data = fetch_and_process_time_report(token, selected_month, tasks_data)
        personal_timereport = calculate_personal_timereport(time_report_data)
        final_report = generate_final_report(tasks_data, time_report_data)
        if refresh_billable:
            update_custom_fields(token, final_report)

        return render_template(
            'time_tracking_report.html',
            final_report=final_report.to_html(classes='table table-striped', index=False),
            personal_timereport=personal_timereport.to_html(classes='table table-striped', index=False),
            total=final_report['AdjustedDuration'].sum()
        )

    months = []
    for i in range(1, 13):
        month_name = datetime.date(1900, i, 1).strftime('%B')
        months.append((i, month_name))
    return render_template('time_tracking.html', months=months, current_month=datetime.datetime.now().strftime("%B"))


@app.route('/report', methods=['GET'])
def reports_list_route():
    return render_template('reports_list.html')


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
