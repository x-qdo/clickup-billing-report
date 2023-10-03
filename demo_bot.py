import re

import requests

LIST_ID = '7-2454960-1'
TEAM_ID = '2454960'


def get_tasks_list(list_id, token):
    page = 0
    all_tasks = []
    last_page = False
    while not last_page:
        response = requests.get(
            url='https://api.clickup.com/api/v2/view/%s/task' % list_id,
            headers={'Authorization': token, 'Content-Type': 'application/json'},
            params={
                'page': page
            }
        )

        # Check the response status code
        if response.status_code != 200:
            print(f"Error: {response.status_code}")
            exit()

        # Parse the JSON response
        data = response.json()
        tasks = data["tasks"]
        last_page = data["last_page"]

        # Add the tasks to the all_tasks list
        all_tasks += tasks
        # Increment the page number
        page += 1
    return all_tasks


def get_spaces(team_id, token):
    response = requests.get(
        url='https://api.clickup.com/api/v2/team/%s/space' % team_id,
        headers={'Authorization': token, 'Content-Type': 'application/json'},
        params={
            'archived': False
        }
    )
    spaces_dict = {}
    for space in response.json()["spaces"]:
        spaces_dict[space["id"]] = space["name"]
    return spaces_dict


def group_tasks(tasks, spaces):
    grouped_tasks = {}
    pattern = r'<.*?>'

    # Loop through each task
    for task in tasks:
        # Get the assignees
        assignees = ['@%s' % assignee["username"] for assignee in task["assignees"]]

        # Create a key for the assignees
        key = " && ".join(sorted(assignees))

        # Get the space name from the spaces dictionary
        space_name = spaces.get(task["space"]["id"])

        # Add the task to the corresponding group
        if key in grouped_tasks:
            grouped_tasks[key].append({
                "space_name": space_name,
                "task_id": task["custom_id"],
                "task_name": re.sub(pattern, '', task["name"]),
                "assignees": assignees
            })
        else:
            grouped_tasks[key] = [{
                "space_name": space_name,
                "task_id": task["custom_id"],
                "task_name": re.sub(pattern, '', task["name"]),
                "assignees": assignees
            }]

    return grouped_tasks


def generate_demo_list(token, list_id, team_id):
    spaces = get_spaces(team_id, token)
    tasks = get_tasks_list(list_id, token)
    grouped_tasks = group_tasks(tasks, spaces)

    md = "|Space|Presenter|Summary|Notes|\n"
    md += "| --- | --- | --- |--- |\n"
    for assignee, tasks in grouped_tasks.items():
        for task in tasks:
            md += '|{space_name}|{assignees}|[{task_id}: {task_name}](https://app.clickup.com/t/{team_id}/{task_id})| |\n'.format(
                space_name=task["space_name"], task_id=task["task_id"], task_name=task["task_name"],
                assignees=assignee, team_id=team_id)

    return md
