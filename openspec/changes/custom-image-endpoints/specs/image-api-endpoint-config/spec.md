## ADDED Requirements

### Requirement: Admin can configure image API endpoints in system settings
The system SHALL allow an administrator to configure image-related API endpoints from the system settings page, including text-to-image submission, image edit submission, task polling, model list retrieval, and chat completion.

#### Scenario: Administrator views configurable endpoint fields
- **WHEN** an administrator opens the system settings page
- **THEN** the page shows separate configurable fields for text-to-image submission, image edit submission, task polling, model list retrieval, and chat completion endpoints

#### Scenario: Configuration changes take effect immediately
- **WHEN** an administrator saves updated endpoint settings
- **THEN** the backend uses the updated values for subsequent requests without requiring a service restart

### Requirement: Endpoint settings support relative paths and full URLs
Each configurable endpoint SHALL accept either a relative path or a fully qualified URL.

#### Scenario: Relative path uses API base URL
- **WHEN** an endpoint setting contains a relative path such as `/v1/images/generations`
- **THEN** the backend resolves the final request URL by combining `api_base_url` with that path

#### Scenario: Full URL bypasses API base URL
- **WHEN** an endpoint setting contains a fully qualified URL beginning with `http://` or `https://`
- **THEN** the backend sends the request directly to that URL without prefixing `api_base_url`

### Requirement: Image generation and image editing endpoints are independently configurable
The system SHALL allow text-to-image requests and image edit requests to use different configured endpoints.

#### Scenario: Text-to-image uses generation endpoint
- **WHEN** a user submits a text-to-image task
- **THEN** the backend sends the request to the configured text-to-image endpoint

#### Scenario: Image edit uses edit endpoint
- **WHEN** a user submits an image edit task
- **THEN** the backend sends the request to the configured image edit endpoint, even if it differs from the text-to-image endpoint

### Requirement: Missing endpoint settings fall back to defaults or fail clearly
The system SHALL define deterministic behavior when new endpoint settings are absent or blank.

#### Scenario: Unconfigured optional endpoint falls back to default path
- **WHEN** a newly introduced endpoint setting is absent or blank
- **THEN** the backend uses the built-in default path for that endpoint if a compatible default exists

#### Scenario: Request cannot be resolved to a usable endpoint
- **WHEN** the backend cannot determine a valid request URL for a required endpoint
- **THEN** the request fails with a clear error message that indicates endpoint configuration is invalid

### Requirement: Existing deployments remain compatible during migration
The system SHALL preserve compatibility for deployments that upgrade before populating all new endpoint settings.

#### Scenario: Existing deployment upgrades without new settings
- **WHEN** an existing deployment upgrades to the new version and the newly added endpoint settings are not yet present in storage
- **THEN** the backend continues to serve requests using backward-compatible defaults where supported

#### Scenario: Existing stored base URL remains valid
- **WHEN** an existing deployment has only `api_base_url` and older endpoint settings configured
- **THEN** the backend continues to resolve requests successfully using those existing values and default fallbacks