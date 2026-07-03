package image

import (
	"encoding/json"
	"testing"
)

func TestTaskResultResponseAcceptsPercentProgressString(t *testing.T) {
	body := []byte(`{"task_id":"gpt-image-2-6af74006-388e-4b51-9c91-2e4cb842cb15","status":"SUCCESS","progress":"100%","data":[{"url":"https://cdnoss.jounery.vip/images/1781602096/1781602096588368523.png"}]}`)

	var result TaskResultResponse
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if result.Progress != 100 {
		t.Fatalf("Progress = %d, want 100", result.Progress)
	}
}

func TestJoinURLPath(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		path    string
		want    string
	}{
		{
			name:    "joins trimmed base and relative path",
			baseURL: "https://api.example.com/",
			path:    "/v1/images/generations/tasks",
			want:    "https://api.example.com/v1/images/generations/tasks",
		},
		{
			name:    "keeps absolute path when base empty",
			baseURL: "",
			path:    "/v1/images/tasks",
			want:    "/v1/images/tasks",
		},
		{
			name:    "keeps absolute url untouched",
			baseURL: "https://api.example.com",
			path:    "https://gateway.example.com/custom/tasks",
			want:    "https://gateway.example.com/custom/tasks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := joinURLPath(tt.baseURL, tt.path); got != tt.want {
				t.Fatalf("joinURLPath(%q, %q) = %q, want %q", tt.baseURL, tt.path, got, tt.want)
			}
		})
	}
}

func TestPickConfiguredURLPrefersFullURL(t *testing.T) {
	got := pickConfiguredURL("https://api.example.com", "https://gateway.example.com/custom/generate", "/v1/images/generations/tasks")
	want := "https://gateway.example.com/custom/generate"
	if got != want {
		t.Fatalf("pickConfiguredURL(...) = %q, want %q", got, want)
	}
}
