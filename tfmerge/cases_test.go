package tfmerge

var cases = []struct {
	name      string
	dir       string
	baseState string
	hasError  bool
}{
	{
		name: "Resource Only (no base state)",
		dir:  "resource_only",
	},
	{
		name: "Resource Only (base state)",
		dir:  "resource_only",
		baseState: `{
"version": 4,
"terraform_version": "1.2.8",
"serial": 1,
"lineage": "00000000-0000-0000-0000-000000000000",
"outputs": {},
"resources": []
}
`,
	},
	{
		name: "Module no cross (no base state)",
		dir:  "module_no_cross",
	},
	{
		name: "Module no cross (base state)",
		dir:  "module_no_cross",
		baseState: `{
"version": 4,
"terraform_version": "1.2.8",
"serial": 1,
"lineage": "00000000-0000-0000-0000-000000000000",
"outputs": {},
"resources": []
}
`,
	},
	{
		name: "Module cross (no base state)",
		dir:  "module_cross",
	},
	{
		name: "Module cross (base state)",
		dir:  "module_cross",
		baseState: `{
"version": 4,
"terraform_version": "1.2.8",
"serial": 1,
"lineage": "00000000-0000-0000-0000-000000000000",
"outputs": {},
"resources": []
}
`,
	},
	{
		name: "Module instance",
		dir:  "module_instance",
	},
	{
		name:     "Resource conflict",
		dir:      "resource_conflict",
		hasError: true,
	},
	{
		name: "Resource conflict with base state",
		dir:  "resource_only",
		baseState: `{
"version": 4,
"terraform_version": "1.2.8",
"serial": 1,
"lineage": "00000000-0000-0000-0000-000000000000",
"outputs": {},
"resources": [
{
  "mode": "managed",
  "type": "null_resource",
  "name": "test1",
  "provider": "provider[\"registry.terraform.io/hashicorp/null\"]",
  "instances": [
	{
	  "schema_version": 0,
	  "attributes": {},
	  "sensitive_attributes": [],
	  "private": "bnVsbA=="
	}
  ]
}
]
}
`,
		hasError: true,
	},
	{
		name:     "Module conflict",
		dir:      "module_conflict",
		hasError: true,
	},
}
