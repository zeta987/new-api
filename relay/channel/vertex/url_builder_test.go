package vertex

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildGoogleModelURLUsesTheLocationHost(t *testing.T) {
	tests := []struct {
		region string
		want   string
	}{
		{"global", "https://aiplatform.googleapis.com/v1/projects/my-project/locations/global/publishers/google/models/gemini-4-argon:generateContent"},
		{"us-central1", "https://us-central1-aiplatform.googleapis.com/v1/projects/my-project/locations/us-central1/publishers/google/models/gemini-4-argon:generateContent"},
		{"us", "https://aiplatform.us.rep.googleapis.com/v1/projects/my-project/locations/us/publishers/google/models/gemini-4-argon:generateContent"},
		{"eu", "https://aiplatform.eu.rep.googleapis.com/v1/projects/my-project/locations/eu/publishers/google/models/gemini-4-argon:generateContent"},
	}
	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			got := BuildGoogleModelURL("", DefaultAPIVersion, "my-project", tt.region, "gemini-4-argon", "generateContent")
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetModelRegionFallsBackToUpstreamBase(t *testing.T) {
	regions := `{"default":"asia-east1","gemini-4-hydrogen":"global","gemini-4-hydrogen-low":"us"}`
	tests := []struct {
		origin, upstream, want string
	}{
		{"gemini-4-hydrogen-xhigh", "gemini-4-hydrogen", "global"},
		{"gemini-4-hydrogen-low", "gemini-4-hydrogen", "us"},
		{"gemini-2.5-pro", "gemini-2.5-pro", "asia-east1"},
	}
	for _, tt := range tests {
		t.Run(tt.origin, func(t *testing.T) {
			assert.Equal(t, tt.want, GetModelRegion(regions, tt.origin, tt.upstream))
		})
	}
	assert.Equal(t, "global", GetModelRegion(`{"gemini-4-carbon":"us"}`, "gemini-4-argon", "gemini-4-argon"))
	assert.Equal(t, "us-central1", GetModelRegion("us-central1", "gemini-4-argon", "gemini-4-argon"))
}
