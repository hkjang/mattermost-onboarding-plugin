# Mattermost Onboarding Plugin

[![Build Status](https://github.com/mattermost/mattermost-plugin-starter-template/actions/workflows/ci.yml/badge.svg)](https://github.com/mattermost/mattermost-plugin-starter-template/actions/workflows/ci.yml)
[![E2E Status](https://github.com/mattermost/mattermost-plugin-starter-template/actions/workflows/e2e.yml/badge.svg)](https://github.com/mattermost/mattermost-plugin-starter-template/actions/workflows/e2e.yml)

This repository now contains a Mattermost onboarding plugin foundation that automatically sends a direct message to newly created users. The current implementation focuses on an operator-configurable first release that combines common guidance with department-specific guidance, applies exclusion rules, records send logs, and retries failures through a queue.

To learn more about plugins, see [our plugin documentation](https://developers.mattermost.com/extend/plugins/).

This template requires node v16 and npm v8. You can download and install nvm to manage your node versions by following the instructions [here](https://github.com/nvm-sh/nvm). Once you've setup the project simply run `nvm i` within the root folder to use the suggested version of node.

## Current Foundation

The server implementation currently supports the following:

- `UserHasBeenCreated` hook based automatic onboarding start
- Common template + department template composition
- Template variable replacement such as `{{user_name}}`, `{{department_name}}`, `{{start_date}}`
- Link ordering and per-link activation control
- Fallback to common-only or fallback department template
- Exclusion rules for email domain, username, user id, group, bot, and deleted users
- KV-backed send logs and duplicate prevention for automatic sends
- Retry queue processing through the background job
- System Console settings schema for runtime template and mapping changes`r`n- System Console operations panel for preview, resend, stats, and recent log checks
- Admin API endpoints for preview, resend, stats, and log lookup

## Admin API

All admin endpoints are under `/plugins/<plugin-id>/api/v1/admin` and require a logged-in system admin.

- `GET /health`
- `GET /admin/preview?user_id=<id>`
- `POST /admin/resend`
- `GET /admin/stats`
- `GET /admin/logs`

`POST /admin/resend` expects JSON such as:

```json
{
  "user_id": "target-user-id"
}
```

Preview also supports optional overrides for `dept_code`, `dept_name`, `organization_name`, `lang`, and `start_date`.

## Getting Started
Use GitHub's template feature to make a copy of this repository by clicking the "Use this template" button.

Alternatively shallow clone the repository matching your plugin name:
```
git clone --depth 1 https://github.com/mattermost/mattermost-plugin-starter-template com.example.my-plugin
```

Note that this project uses [Go modules](https://github.com/golang/go/wiki/Modules). Be sure to locate the project outside of `$GOPATH`.

Edit the following files:
1. `plugin.json` with your `id`, `name`, and `description`:
```json
{
    "id": "com.example.my-plugin",
    "name": "My Plugin",
    "description": "A plugin to enhance Mattermost."
}
```

2. `go.mod` with your Go module path, following the `<hosting-site>/<repository>/<module>` convention:
```
module github.com/example/my-plugin
```

3. Replace all occurrences of `github.com/mattermost/mattermost-plugin-starter-template` in the codebase with your Go module path:
```bash
sed -i '' 's|github.com/mattermost/mattermost-plugin-starter-template|github.com/example/my-plugin|g' server/*.go
```

4. Replace `.golangci.yml` `local-prefixes` attribute with your Go module path:
```yml
linters-settings:
  # [...]
  goimports:
    local-prefixes: github.com/example/my-plugin
```

5. Build your plugin:
```
make
```

This will produce a single plugin file (with support for multiple architectures) for upload to your Mattermost server:

```
dist/com.example.my-plugin.tar.gz
```

## Development

To avoid having to manually install your plugin, build and deploy your plugin using one of the following options. In order for the below options to work, you must first enable plugin uploads via your config.json or API and restart Mattermost.

```json
    "PluginSettings" : {
        ...
        "EnableUploads" : true
    }
```

### Development guidance

1. Fewer packages is better: default to the main package unless there's good reason for a new package.

2. Coupling implies same package: don't jump through hoops to break apart code that's naturally coupled.

3. New package for a new interface: a classic example is the sqlstore with layers for monitoring performance, caching and mocking.

4. New package for upstream integration: a discrete client package for interfacing with a 3rd party is often a great place to break out into a new package

### Server Components

The server side is centered on the following plugin layers:

#### Event Listener

`UserHasBeenCreated` starts onboarding delivery immediately or after an optional delay.

#### Template Resolver and Renderer

The onboarding service resolves a common template and an optional department template by language and department mapping, then renders a Markdown DM with ordered links.

#### Delivery Control

Automatic sends are protected by a KV-backed atomic state key so duplicate creation events do not generate multiple DMs.

#### Retry Queue

Failures are stored in a KV-backed retry queue and retried by the scheduled background job.

#### Admin APIs

Preview, manual resend, stats, and latest-log lookup are exposed through authenticated plugin HTTP endpoints for future admin UI integration.

### Deploying with Local Mode

If your Mattermost server is running locally, you can enable [local mode](https://docs.mattermost.com/administration/mmctl-cli-tool.html#local-mode) to streamline deploying your plugin. Edit your server configuration as follows:

```json
{
    "ServiceSettings": {
        ...
        "EnableLocalMode": true,
        "LocalModeSocketLocation": "/var/tmp/mattermost_local.socket"
    },
}
```

and then deploy your plugin:
```
make deploy
```

You may also customize the Unix socket path:
```bash
export MM_LOCALSOCKETPATH=/var/tmp/alternate_local.socket
make deploy
```

If developing a plugin with a webapp, watch for changes and deploy those automatically:
```bash
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_TOKEN=j44acwd8obn78cdcx7koid4jkr
make watch
```

### Deploying with credentials

Alternatively, you can authenticate with the server's API with credentials:
```bash
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_USERNAME=admin
export MM_ADMIN_PASSWORD=password
make deploy
```

or with a [personal access token](https://docs.mattermost.com/developer/personal-access-tokens.html):
```bash
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_TOKEN=j44acwd8obn78cdcx7koid4jkr
make deploy
```

### Releasing new versions

The version of a plugin is determined at compile time, automatically populating a `version` field in the [plugin manifest](plugin.json):
* If the current commit matches a tag, the version will match after stripping any leading `v`, e.g. `1.3.1`.
* Otherwise, the version will combine the nearest tag with `git rev-parse --short HEAD`, e.g. `1.3.1+d06e53e1`.
* If there is no version tag, an empty version will be combined with the short hash, e.g. `0.0.0+76081421`.

To disable this behaviour, manually populate and maintain the `version` field.

## How to Release

To trigger a release, follow these steps:

1. **For Patch Release:** Run the following command:
    ```
    make patch
    ```
   This will release a patch change.

2. **For Minor Release:** Run the following command:
    ```
    make minor
    ```
   This will release a minor change.

3. **For Major Release:** Run the following command:
    ```
    make major
    ```
   This will release a major change.

4. **For Patch Release Candidate (RC):** Run the following command:
    ```
    make patch-rc
    ```
   This will release a patch release candidate.

5. **For Minor Release Candidate (RC):** Run the following command:
    ```
    make minor-rc
    ```
   This will release a minor release candidate.

6. **For Major Release Candidate (RC):** Run the following command:
    ```
    make major-rc
    ```
   This will release a major release candidate.

## Q&A

### How do I make a server-only or web app-only plugin?

Simply delete the `server` or `webapp` folders and remove the corresponding sections from `plugin.json`. The build scripts will skip the missing portions automatically.

### How do I include assets in the plugin bundle?

Place them into the `assets` directory. To use an asset at runtime, build the path to your asset and open as a regular file:

```go
bundlePath, err := p.API.GetBundlePath()
if err != nil {
    return errors.Wrap(err, "failed to get bundle path")
}

profileImage, err := ioutil.ReadFile(filepath.Join(bundlePath, "assets", "profile_image.png"))
if err != nil {
    return errors.Wrap(err, "failed to read profile image")
}

if appErr := p.API.SetProfileImage(userID, profileImage); appErr != nil {
    return errors.Wrap(err, "failed to set profile image")
}
```

### How do I build the plugin with unminified JavaScript?
Setting the `MM_DEBUG` environment variable will invoke the debug builds. The simplist way to do this is to simply include this variable in your calls to `make` (e.g. `make dist MM_DEBUG=1`).
