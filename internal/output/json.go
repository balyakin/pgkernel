package output

import (
	"encoding/json"

	"github.com/balyakin/pgkernel/internal/checker"
)

// FILE:internal/output/json.go
// VERSION:1.0.0
// START_MODULE_CONTRACT:
// PURPOSE:Serialize report into stable JSON contract.
// SCOPE:JSON rendering and indentation profile.
// INPUT:checker.Report structure.
// OUTPUT:JSON string suitable for CI and machine processing.
// KEYWORDS:[DOMAIN(Contract): schema stability; CONCEPT(CI): machine readability]
// LINKS:[USES_API(encoding/json): serialization]
// END_MODULE_CONTRACT

// START_CHANGE_SUMMARY:
// LAST_CHANGE:1.0.0 - Added JSON renderer.
// PREV_CHANGE_SUMMARY:none
// END_CHANGE_SUMMARY

func RenderJSON(report checker.Report) (string, error) {
	bytes, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
