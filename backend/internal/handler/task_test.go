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

func TestParsePositiveInt(t *testing.T) {
	if got := parsePositiveInt("", 20); got != 20 {
		t.Fatalf("parsePositiveInt(empty) = %d, want 20", got)
	}
	if got := parsePositiveInt("3", 20); got != 3 {
		t.Fatalf("parsePositiveInt(valid) = %d, want 3", got)
	}
	if got := parsePositiveInt("-1", 20); got != 20 {
		t.Fatalf("parsePositiveInt(negative) = %d, want 20", got)
	}
	if got := parsePositiveInt("abc", 20); got != 20 {
		t.Fatalf("parsePositiveInt(invalid) = %d, want 20", got)
	}
}

func TestNewTaskListResponse(t *testing.T) {
	tasks := []model.Task{
		{UserInput: "task-a", SourceImageURLs: []string{"https://example.com/ref-a.png"}},
		{UserInput: "task-b", SourceImageURLs: []string{"https://example.com/ref-b.png"}},
	}

	resp := newTaskListResponse(tasks, 41, 2, 20)
	if resp.Total != 41 || resp.Page != 2 || resp.PageSize != 20 || resp.TotalPages != 3 {
		t.Fatalf("unexpected pagination response: %#v", resp)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("len(resp.Items) = %d, want 2", len(resp.Items))
	}
	if resp.Items[0].SourceImageURLs != nil || resp.Items[1].SourceImageURLs != nil {
		t.Fatalf("expected response items to omit SourceImageURLs: %#v", resp.Items)
	}
}
