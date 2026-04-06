package output

import (
	"encoding/json"

	"github.com/balyakin/pgkernel/internal/checker"
)

func RenderJSON(report checker.Report) (string, error) {
	bytes, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
