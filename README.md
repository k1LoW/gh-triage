# gh-triage

`gh-triage` is a tool that helps you manage and triage GitHub issues and pull requests through unread notifications.

Key features of `gh-triage` are:

- **Read**: Automatically mark Issues and Pull Requests that match specified conditions as read
- **Open**: Open Issues and Pull Requests that match specified conditions in a browser
- **List**: Display Issues and Pull Requests that match specified conditions in a list

## Usage

```bash
$ gh triage
```

## Install

```bash
$ gh extension install k1LoW/gh-triage
```

## Configuration

The configuration file is located at:

- `${XDG_DATA_HOME}/gh-triage/config.yml`
- OR `~/.local/share/gh-triage/config.yml`

It will be automatically created on first run.

### Default configuration

```yaml
read:
  max: 1000
  conditions: # Auto-mark merged PRs as read
    - "merged"

open:
  max: 1
  conditions: # Open PRs awaiting my review
    - "is_pull_request && me in reviewers && passed && !approved && !draft && !closed && !merged"

list:
  max: 1000
  conditions: # List all unread notifications
    - "*"
```

### Options

- `read`: Conditions and maximum number for marking as read
- `open`: Conditions and maximum number for opening in browser
- `list`: Conditions and maximum number for listing

Each action has the following parameters:
- `max`: Maximum number of items to process at once
- `conditions`: Processing conditions (no processing if empty array)

## Available Fields

gh-triage retrieves the following information for each notification, which can be used in condition evaluation:

| Field | Type | Pull Request Description | Issue Description |
|-------|------|-------------------------|-------------------|
| `is_pull_request` | `bool` | Always `true` for Pull Requests | Always `false` for Issues |
| `me` | `string` | Username of authenticated user | Username of authenticated user |
| `title` | `string` | The title of the Pull Request | The title of the Issue |
| `owner` | `string` | Repository owner name | Repository owner name |
| `repo` | `string` | Repository name | Repository name |
| `number` | `int` | Pull Request number | Issue number |
| `state` | `string` | State of the PR (`open`, `closed`) | State of the Issue (`open`, `closed`) |
| `closed` | `bool` | Whether the PR is closed | Whether the Issue is closed |
| `labels` | `[]string` | List of labels attached to the PR | List of labels attached to the Issue |
| `assignees` | `[]string` | List of assigned users | List of assigned users |
| `author` | `string` | Username of the PR author | Username of the Issue author |
| `html_url` | `string` | GitHub URL of the PR | GitHub URL of the Issue |
| `draft` | `bool` | Whether the PR is draft | N/A |
| `merged` | `bool` | Whether the PR has been merged | N/A |
| `mergeable` | `bool` | Whether the PR is mergeable | N/A |
| `mergeable_state` | `string` | Mergeable state of the PR | N/A |
| `reviewers` | `[]string` | List of requested reviewers | N/A |
| `review_teams` | `[]string` | List of requested review teams | N/A |
| `approved` | `bool` | Whether the PR has been approved | N/A |
| `review_states` | `[]string` | History of review states | N/A |
| `status_passed` | `bool` | Whether status checks have passed | N/A |
| `checks_passed` | `bool` | Whether checks have passed | N/A |
| `passed` | `bool` | Whether both status checks and checks have passed | N/A |

## Condition Evaluation System

Conditions are evaluated using the [expr-lang](https://expr-lang.org/) library. You can write conditions such as:

### Basic Conditions

```yaml
conditions:
  - "merged"                    # Merged Pull Request
  - "closed"                    # Closed Issue/PR
  - "approved"                  # Approved Pull Request
  - "is_pull_request"           # Pull Request
  - "is_issue"                  # Issue
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

### Automatically mark merged Pull Requests as read

```yaml
read:
  max: 1000
  conditions:
    - "merged"
```

### Open Issues and PRs related to yourself in browser

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
