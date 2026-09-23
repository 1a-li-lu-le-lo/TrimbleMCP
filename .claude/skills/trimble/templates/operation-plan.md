# Trimble operation plan (planning aid; qualified review required)

- **Actor:** <who is requesting>
- **Reason:** <why>
- **Product:** <product ID from trimble_get_capabilities>
- **Project:** <name> (<project_id from trimble_list_projects>)
- **Resources:** <IDs, each from a tool result>
- **Expected current state:** <what trimble_get_file_metadata / trimble_list_folder_items returned, with version_id and revision>
- **Desired state:** <exact change>
- **Units / CRS:** <explicit, or "none">
- **Side effects:** <who sees it, notifications, processing jobs>
- **Idempotency key:** <to be issued by the server when mutation tools exist>
- **Approval required:** <level and approver>
- **Rollback:** <how to undo, or "not reversible">
- **Status:** not executed. This release of Trimble MCP Bridge has no mutation tools.
