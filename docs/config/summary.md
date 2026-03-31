---
title: Summary Plugin
description: "MR/PR plan summary comments: configuration and options"
outline: deep
---

# Summary Plugin

The summary plugin posts plan summaries as MR/PR comments. It is **enabled by default** and collects results from all other plugins (cost, policy) into a single comment.

## Configuration

```yaml
plugins:
  summary:
    enabled: true            # default: true (opt out with false)
    on_changes_only: false   # only comment when there are changes
    include_details: true    # include full plan output in expandable sections
```

## Options

### enabled

Enable or disable the summary plugin. Since summary uses `EnabledByDefault` policy, it is active unless explicitly disabled.

```yaml
plugins:
  summary:
    enabled: false   # disable plan summary comments
```

### on_changes_only

Only post a comment when the plan contains changes (add/change/destroy).

```yaml
plugins:
  summary:
    on_changes_only: true
```

### include_details

Include full plan output in expandable `<details>` sections within the comment.

```yaml
plugins:
  summary:
    include_details: true   # default
```

## CLI Command

```bash
terraci summary
```

The `terraci summary` command scans for plan results in the service directory, loads plugin reports (cost, policy), composes a markdown comment, and posts it to the MR/PR via the active CI provider's comment service.

## See Also

- [terraci summary CLI](/cli/summary)
- [GitLab MR Integration](/config/gitlab-mr)
- [Plugin System](/plugins/)
