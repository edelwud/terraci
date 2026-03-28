package terraform

import rego.v1

warn contains msg if {
    some r in input.resource_changes
    r.change.actions[0] == "create"
    msg := sprintf("New resource: %s", [r.address])
}
