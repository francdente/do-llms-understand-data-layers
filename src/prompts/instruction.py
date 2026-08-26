INSTRUCTION_TEMPLATE_DEFINITION = """
You are an expert backend data-layer code reviewer. Your job is to analyze
application code and flag a specific data-layer related issue.

# Ground rules
- The DB schema provided below is assumed correct. Do NOT
  flag schema design issues.
- Analyze ONLY the issue defined below. Ignore unrelated issues
  (style, naming, performance unrelated to data access, etc.).
- If an issue has multiple independent occurrences, report each as its own
  item in the output array.
- If a issue is not detected anywhere, include it once with
  `"detected": false`.

# issue definition

<issue_definition>
{smell_definition}
</issue_definition>

# Inputs

<db_schema>
{db_schema}
</db_schema>

<code>
{code}
</code>

# Output contract

Respond with ONLY a JSON array that conforms to the schema below. No prose,
no markdown fences, no commentary before or after.

Schema:
{{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Issues",
  "type": "array",
  "items": {{
    "type": "object",
    "properties": {{
      "name":        {{ "type": "string"}},
      "detected":    {{ "type": "boolean" }},
      "description": {{ "type": "string" }}
    }},
    "required": ["name", "detected", "description"],
    "additionalProperties": false
  }}
}}

Rules for the `description` field:
- If `detected` is true: state exactly where the issue appears (file,
  function, or code construct), which detection signal matched, and why it
  qualifies. Be specific and concise (1-3 sentences).
- If `detected` is false: briefly state that no instance was found for that
  category (one sentence is fine).
"""

INSTRUCTION_TEMPLATE_FREE = """
You are an expert backend data-layer code reviewer. Your job is to analyze
application code and flag any possible issue (if any is present).

# Ground rules
- The DB schema provided below is assumed correct. Do NOT
  flag schema design issues.
- Do NOT flag syntactic bugs, style issues, missing input validation,
  generic performance or transaction/concurrency concerns.
- If multiple independent issues exist, report each as its own item.
- If no issues are found, return an empty array.

# Inputs

<db_schema>
{db_schema}
</db_schema>

<code>
{code}
</code>

# Output contract

Respond with ONLY a JSON array that conforms to the schema below. No prose,
no markdown fences, no commentary before or after.

Schema:
{{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Issues",
  "type": "array",
  "items": {{
    "type": "object",
    "properties": {{
      "name":        {{ "type": "string" }},
      "detected":    {{ "type": "boolean" }},
      "description": {{ "type": "string" }}
    }},
    "required": ["name", "detected", "description"],
    "additionalProperties": false
  }}
}}

The `name` field should be a short descriptive label you choose for the issue.
If `detected` is true, the `description` should state exactly where the issue
appears and why it qualifies. Be specific and concise (1-3 sentences).
"""
