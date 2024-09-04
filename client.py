from dataclasses import dataclass


@dataclass
class Client:
    name: str
    team_id: str
    list_id: str
    contract_included: float
    toggl_sync_enabled: bool = False
    toggl_workspace_id: str = None


clients = [
    Client(
        name="Insly",
        list_id="10940440",
        team_id="2454960",
        contract_included=130
    ),
    Client(
        name="CI",
        list_id="901503819155",
        team_id="2454960",
        contract_included=20,
        toggl_sync_enabled=True,
        toggl_workspace_id="328724"
    ),
]
