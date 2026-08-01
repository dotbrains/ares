package customoutput

import "strings"

type Result struct {
	Applied  []string
	Verified []string
	Skipped  []string
	Failed   []string
}

func Parse(pluginID string, output string) Result {
	var result Result
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "applied:"):
			result.Applied = append(result.Applied, pluginID+": "+strings.TrimSpace(strings.TrimPrefix(line, "applied:")))
		case strings.HasPrefix(line, "verified:"):
			result.Verified = append(result.Verified, pluginID+": "+strings.TrimSpace(strings.TrimPrefix(line, "verified:")))
		case strings.HasPrefix(line, "skipped:"):
			result.Skipped = append(result.Skipped, pluginID+": "+strings.TrimSpace(strings.TrimPrefix(line, "skipped:")))
		case strings.HasPrefix(line, "failed:"):
			result.Failed = append(result.Failed, pluginID+": "+strings.TrimSpace(strings.TrimPrefix(line, "failed:")))
		default:
			result.Applied = append(result.Applied, pluginID+": "+line)
		}
	}
	return result
}
