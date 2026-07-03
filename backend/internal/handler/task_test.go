package handler

import (
	"strings"
	"testing"

	"github.com/imagegen/backend/internal/model"
)

func TestSummarizeTasksForListOmitsSourceImagePayloads(t *testing.T) {
	tasks := []model.Task{
		{
			UserInput:       "draw a cat",
			SourceImageURLs: []string{"data:image/png;base64," + strings.Repeat("a", 1024), "https://example.com/ref.png"},
			ResultImageURL:  "https://example.com/out.png",
		},
	}

	summaries := summarizeTasksForList(tasks)

	if len(summaries) != 1 {
		t.Fatalf("len(summaries) = %d, want 1", len(summaries))
	}
	if summaries[0].UserInput != "draw a cat" || summaries[0].ResultImageURL != "https://example.com/out.png" {
		t.Fatalf("summary lost normal task fields: %#v", summaries[0])
	}
	if summaries[0].SourceImageURLs != nil {
		t.Fatalf("SourceImageURLs = %#v, want nil for list response", summaries[0].SourceImageURLs)
	}
	if tasks[0].SourceImageURLs == nil {
		t.Fatalf("summarizeTasksForList mutated original task")
	}
}
