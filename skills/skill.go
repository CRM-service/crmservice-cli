package skills

import _ "embed"

// CRMServiceSkill contains the bundled crmservice Agent Skill.
//
//go:embed crmservice/SKILL.md
var CRMServiceSkill string
