package paths

import (
	"regexp"

	"mio9/mtx-monitor/internal/constants"
	"mio9/mtx-monitor/internal/mediamtx"
)

// PublishingPath is a Path with a non-nil source.
type PublishingPath struct {
	Name         string
	Source       mediamtx.PathSource
	InboundBytes int64
	Online       bool
}

// IsPublishingPath returns true if the path has an active publisher source.
func IsPublishingPath(p mediamtx.Path) bool {
	if p.Source == nil || !p.Online {
		return false
	}
	_, isPublisher := constants.PublisherSourceTypes[p.Source.Type]
	return isPublisher
}

// SplitPublishingPaths filters publishing paths and splits them into enforced vs other based on regex.
func SplitPublishingPaths(paths []mediamtx.Path, regex *regexp.Regexp) (enforced, other []PublishingPath) {
	for _, p := range paths {
		if !IsPublishingPath(p) {
			continue
		}

		pp := PublishingPath{
			Name:         p.Name,
			Source:       *p.Source,
			InboundBytes: p.InboundBytes,
			Online:       p.Online,
		}

		if regex == nil {
			enforced = append(enforced, pp)
			continue
		}

		if regex.MatchString(p.Name) {
			enforced = append(enforced, pp)
		} else {
			other = append(other, pp)
		}
	}

	return enforced, other
}
