# ClickUp Reporter (Go Refactor)

This application provides reporting and automation tools for ClickUp, refactored into Go for deployment on AWS Lambda.

## Features (Planned)

*   ClickUp OAuth2 Authentication
*   Time Billing Reporting:
    *   Generate monthly time reports based on ClickUp time entries.
    *   Apply developer coefficients.
    *   Update ClickUp custom fields for billable/invoiced hours.
*   Toggl Time Sync (Optional):
    *   Sync confirmed ClickUp time entries to Toggl.
*   Demo Preparation:
    *   Generate Markdown summaries of completed tasks for demos.
    *   (Future) Post to Slack and create Wiki pages.
*   Configuration Management via DynamoDB.

## Architecture

*   **Compute:** AWS Lambda functions triggered via API Gateway (or local HTTP server in debug mode).
*   **Storage:** AWS DynamoDB for configuration (Clients, Developers, Settings) and user sessions.
*   **Language:** Go
*   **Secrets:** Managed via Lambda environment variables (consider AWS Secrets Manager for production).

## Setup (Local Development)

1.  **Prerequisites:**
    *   Go (version 1.21 or later)
    *   AWS CLI configured with credentials (needed for DynamoDB access even locally, unless using DynamoDB Local)
    *   (Optional) Docker for local DynamoDB testing
    *   (Optional) `godotenv` for managing local environment variables (`go install github.com/joho/godotenv/cmd/godotenv@latest`)

2.  **Environment Variables:** Create a `.env` file in the root directory with necessary variables (see `internal/config/config.go`):
    ```dotenv
    # ClickUp App Credentials
    CLICKUP_CLIENT_ID=your_clickup_client_id
    CLICKUP_CLIENT_SECRET=your_clickup_client_secret

    # Application URL (for OAuth callback) - e.g., http://localhost:8080 for local debug server
    BASE_URL=http://localhost:8080

    # Strong secret for signing session cookies/tokens
    SESSION_SECRET=a_very_strong_random_secret_key_32_bytes_or_longer

    # Debug Mode (Optional)
    DEBUG_MODE=true # Set to true to run as local HTTP server
    DEBUG_PORT=8080 # Optional: Port for the debug server (defaults to 8080)

    # Log Level (Optional: trace, debug, info, warn, error, fatal, panic - defaults to info)
    LOG_LEVEL=debug

    # DynamoDB Table Names (Optional - defaults are defined in config)
    # DYNAMODB_CLIENTS_TABLE=YourClientsTable
    # DYNAMODB_DEVELOPERS_TABLE=YourDevelopersTable
    # DYNAMODB_SETTINGS_TABLE=YourSettingsTable
    # DYNAMODB_SESSIONS_TABLE=YourSessionsTable

    # Other API Tokens (Optional - needed for specific features)
    # SLACK_BOT_TOKEN=xoxb-...
    # TOGGL_API_TOKEN=...
    # WIKI_API_TOKEN=...
    # WIKI_BASE_URL=...
    ```
    **Note:** Ensure `BASE_URL` matches the address where your application will be running (e.g., `http://localhost:8080` if `DEBUG_MODE=true` and `DEBUG_PORT=8080`). This is crucial for the OAuth callback.

3.  **Build:**
    ```bash
    # Build a specific handler (e.g., auth_handler)
    go build -o build/auth_handler cmd/auth_handler/main.go
    # Build report handler
    go build -o build/report_handler cmd/report_handler/main.go
    # Build demo handler
    go build -o build/demo_handler cmd/demo_handler/main.go
    ```

4.  **Run Locally (Debug Mode):**
    *   Ensure `DEBUG_MODE=true` is set in your `.env` file (or exported).
    *   Run the desired handler binary directly:
        ```bash
        # Load .env variables (if using godotenv locally)
        # godotenv -f .env ./build/auth_handler
        # Or export variables manually: export $(grep -v '^#' .env | xargs)
        ./build/auth_handler
        ```
    *   Run other handlers similarly in separate terminals if needed (they will listen on the same port if `DEBUG_PORT` is the same, which might cause conflicts - consider different ports or running only one at a time for local testing).
    *   Access the server at `http://localhost:8080` (or your configured `DEBUG_PORT`).

5.  **Deploy to Lambda:**
    *   Ensure `DEBUG_MODE` is **not** set to `true` in the Lambda environment.
    *   Use AWS SAM CLI, Terraform, or CDK to deploy the Lambda functions, API Gateway, and DynamoDB tables defined in the `deploy/` directory (to be created).

## Project Structure

*   `cmd/`: Main packages for Lambda handlers (and debug servers).
    *   `auth_handler/`: Handles OAuth flow.
    *   `report_handler/`: Handles report generation endpoints.
    *   `demo_handler/`: Handles demo report generation.
*   `internal/`: Core application logic.
    *   `auth/`: Authentication logic (OAuth, sessions, crypto).
    *   `clickup/`: ClickUp API client and types.
    *   `config/`: Configuration loading and structs (including DataStore interface).
    *   `report/`: Report generation service.
    *   `server/`: HTTP server utilities (response writer, Lambda shims).
    *   `storage/`: DataStore implementation (DynamoDB).
    *   `toggl/`: (Future) Toggl API client.
    *   `demo/`: (Future) Demo report service.
*   `web/`: (Currently unused by Go handlers, templates are from Python version).
*   `deploy/`: Infrastructure as Code (IaC) definitions (to be created).
*   `build/`: Compiled binaries.

## TODO

*   Implement actual API calls in `internal/toggl`, `internal/wiki`, `internal/slack`.
*   Implement core logic in `internal/demo`.
*   Implement HTML template rendering in Go handlers if needed (currently returning JSON).
*   Add robust error handling and logging improvements.
*   Write unit and integration tests.
*   Define infrastructure in `deploy/`.
*   Consider using a router library (e.g., `httprouter`, `chi`) for more complex routing in debug mode.
