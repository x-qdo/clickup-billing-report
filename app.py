import io
import os

from flask import Flask, render_template, request, redirect, url_for, session, Response
import csv
import requests

from disable_logging import disable_logging
from report import generate_report

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


@app.route('/')
def home():
    if 'access_token' in session:
        return redirect(url_for('generate_report_route'))
    client_id = os.environ['CLICKUP_CLIENT_ID']
    callback_url = url_for('callback', _external=True)
    return render_template('index.html', client_id=client_id, callback_url=callback_url)


@app.route('/report', methods=['GET', 'POST'])
def generate_report_route():
    if 'access_token' not in session:
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

    return render_template('report.html')


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

    return redirect(url_for('generate_report_route'))


@app.route("/healthz", methods=["GET"])
@disable_logging
def health_check():
    return "OK", 200


if __name__ == '__main__':
    app.run()
