# gh-triage

`gh-triage` is a tool that helps you manage and triage GitHub issues, pull requests, and discussions through [unread notifications](https://github.com/notifications?query=is%3Aunread).

Key features of `gh-triage` are:

- **Done**: Mark Issues, Pull Requests, and Discussions that match specified conditions as done
- **Read**: Mark Issues, Pull Requests, and Discussions that match specified conditions as read
- **Unsubscribe**: Unsubscribe from notifications for Issues, Pull Requests, and Discussions that match specified conditions
- **Open**: Open Issues, Pull Requests, and Discussions that match specified conditions in a browser
- **List**: Display Issues, Pull Requests, and Discussions that match specified conditions in a list

ref: [Managing notifications from your inbox](https://docs.github.com/en/account-and-profile/managing-subscriptions-and-notifications-on-github/viewing-and-triaging-notifications/managing-notifications-from-your-inbox)

## Usage

```bash
$ gh triage
```

### Profile Support

`gh-triage` supports multiple configuration profiles. You can create different profiles for different workflows or environments.

#### Using Profiles

```bash
# Use default profile
$ gh triage

# Use specific profile
$ gh triage --profile work
$ gh triage -p personal
```

#### Profile Configuration Files

- Default profile: `default.yml`
- Named profiles: `{profile-name}.yml`

For example:
- `~/.local/share/gh-triage/default.yml` (default profile)
- `~/.local/share/gh-triage/work.yml` (work profile)
- `~/.local/share/gh-triage/personal.yml` (personal profile)

## Install

```bash
$ gh extension install k1LoW/gh-triage
```

## Configuration

The configuration file is located at:

- `${XDG_DATA_HOME}/gh-triage/default.yml` (if `XDG_DATA_HOME` is set)
- OR `~/.local/share/gh-triage/default.yml` (default location)

It will be automatically created on first run.

### Configuration File Migration

If you are upgrading from a previous version, your existing `config.yml` will be automatically migrated to `default.yml` on first run. The migration process:

1. Checks if `default.yml` exists
2. If not, looks for the old `config.yml` file
3. If found, copies the content to `default.yml`
4. Removes the old `config.yml` file
5. Logs the migration process

This ensures a smooth transition without losing your existing configuration.

### Default configuration

```yaml
done:
  max: 1000
  conditions: # Auto-mark merged and closed PRs / issues as done
    - "merged"
    - "closed"

open:
  max: 1
  conditions: # Open PRs awaiting my review
    - "is_pull_request && me in reviewers && passed && !approved && open && !draft"

list:
  max: 1000
  conditions: # List all unread notifications
    - "*"
```

### Options

- `done`: Conditions and maximum number for marking as done
- `read`: Conditions and maximum number for marking as read
- `unsubscribe`: Conditions and maximum number for unsubscribing from notifications
- `open`: Conditions and maximum number for opening in browser
- `list`: Conditions and maximum number for listing

Each action has the following parameters:
- `max`: Maximum number of items to process at once
- `conditions`: Processing conditions (no processing if empty array)

## Available Fields

gh-triage retrieves the following information for each notification, which can be used in condition evaluation:

| Field | Type | Pull Request Description | Issue Description | Discussion Description |
|-------|------|-------------------------|-------------------|------------------------|
| `is_pull_request` | `bool` | Always `true` for Pull Requests | Always `false` for Issues | Always `false` for Discussions |
| `is_issue` | `bool` | Always `false` for Pull Requests | Always `true` for Issues | Always `false` for Discussions |
| `is_discussion` | `bool` | Always `false` for Pull Requests | Always `false` for Issues | Always `true` for Discussions |
| `me` | `string` | Username of authenticated user | Username of authenticated user | Username of authenticated user |
| `title` | `string` | The title of the Pull Request | The title of the Issue | The title of the Discussion |
| `owner` | `string` | Repository owner name | Repository owner name | Repository owner name |
| `repo` | `string` | Repository name | Repository name | Repository name |
| `number` | `int` | Pull Request number | Issue number | Discussion number |
| `state` | `string` | State of the PR (`open`, `closed`) | State of the Issue (`open`, `closed`) | State of the Discussion (`open`, `closed`) |
| `open` | `bool` | Whether the PR is open | Whether the Issue is open | Whether the Discussion is open |
| `closed` | `bool` | Whether the PR is closed | Whether the Issue is closed | Whether the Discussion is closed |
| `labels` | `[]string` | List of labels attached to the PR | List of labels attached to the Issue | List of labels attached to the Discussion |
| `assignees` | `[]string` | List of assigned users | List of assigned users | N/A |
| `author` | `string` | Username of the PR author | Username of the Issue author | Username of the Discussion author |
| `html_url` | `string` | GitHub URL of the PR | GitHub URL of the Issue | GitHub URL of the Discussion |
| `draft` | `bool` | Whether the PR is draft | N/A | N/A |
| `merged` | `bool` | Whether the PR has been merged | N/A | N/A |
| `mergeable` | `bool` | Whether the PR is mergeable | N/A | N/A |
| `mergeable_state` | `string` | Mergeable state of the PR | N/A | N/A |
| `reviewers` | `[]string` | List of requested reviewers | N/A | N/A |
| `review_teams` | `[]string` | List of requested review teams | N/A | N/A |
| `approved` | `bool` | Whether the PR has been approved | N/A | N/A |
| `review_states` | `[]string` | History of review states | N/A | N/A |
| `status_passed` | `bool` | Whether status checks have passed | N/A | N/A |
| `checks_passed` | `bool` | Whether checks have passed | N/A | N/A |
| `passed` | `bool` | Whether both status checks and checks have passed | N/A | N/A |
| `failed` | `bool` | Whether status checks or checks have failed | N/A | N/A |
| `in_progress` | `bool` | Whether status checks or checks are in progress | N/A | N/A |
| `answered` | `bool` | N/A | N/A | Whether the Discussion has been answered |
| `unread` | `bool` | Whether the PR is not marked as read | Whether the Issue is not marked as read | Whether the Discussion is not marked as read |

## Condition Evaluation System

Conditions are evaluated using the [expr-lang](https://expr-lang.org/) library. You can write conditions such as:

### Basic Conditions

```yaml
conditions:
  - "merged"                    # Merged Pull Request
  - "closed"                    # Closed Issue/PR/Discussion
  - "approved"                  # Approved Pull Request
  - "is_pull_request"           # Pull Request
  - "is_issue"                  # Issue
  - "is_discussion"             # Discussion
  - "answered"                  # Answered Discussion
  - "state == 'open'"           # Open state
  - "passed"                    # All checks passed
```

### Complex Conditions

```yaml
conditions:
  - "merged && approved"                   # Merged and approved
  - "is_pull_request && state == 'open'"   # Open Pull Request
  - "author == 'username'"                 # Created by specific user
  - "len(labels) > 0"                      # Has labels
```

### Array Operations

```yaml
conditions:
  - "'bug' in labels"                      # Has bug label
  - "'hotfix' in labels"                   # Has hotfix label
  - "len(assignees) > 0"                   # Has assignees
  - "'APPROVED' in review_states"          # Review states include approved
```

### Special Conditions

```yaml
conditions:
  - "*"                                    # Match all (always true)
```

## Usage Examples

### Automatically mark merged Pull Requests and closed Issues as done

```yaml
done:
  max: 1000
  conditions:
    - "merged"
    - "closed"
```

### Mark failed CI Pull Requests as done

```yaml
done:
  max: 100
  conditions:
    - "is_pull_request && failed"
```

### Automatically mark merged Pull Requests as read

```yaml
read:
  max: 1000
  conditions:
    - "merged"
```

### Open Issues and PRs related to yourself in browser


## Command Line Options

| Option | Short | Description |
|--------|-------|-------------|
| `--profile` | `-p` | Specify profile name for configuration file |

### Examples

```bash
# Use default profile
$ gh triage

# Use work profile
$ gh triage --profile work

# Use personal profile
$ gh triage -p personal
```

```yaml
open:
  max: 5
  conditions:
    - "author == me"
    - "me in assignees"
```

### List items with specific labels

```yaml
list:
  max: 100
  conditions:
    - "'urgent' in labels"
    - "'bug' in labels"
```

### Process only Pull Requests that need review

```yaml
read:
  max: 1000
  conditions:
    - "is_pull_request && state == 'open' && !approved"
```

### Unsubscribe from closed/merged Issues/Pull Requests

```yaml
unsubscribe:
  max: 100
  conditions:
    - "closed" # Closed Issues/Discussions
    - "merged" # Merged Pull Requests
```

### Mark answered Discussions as done

```yaml
done:
  max: 100
  conditions:
    - "is_discussion && answered"
```

### List unanswered Discussions

```yaml
list:
  max: 100
  conditions:
    - "is_discussion && !answered && open"
```

## Contributing

To use this project from source, instead of a release:

    go build .
    gh extension remove triage
    gh extension install .
