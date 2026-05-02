package prompt

import (
	"fmt"
	"strings"

	"github.com/imagegen/backend/internal/model"
)

type PromptBuilder struct {
	Profile *model.StyleProfile
}

type ImageRequest struct {
	Model        string `json:"model"`
	Prompt       string `json:"prompt"`
	Size         string `json:"size"`
	Quality      string `json:"quality"`
	Background   string `json:"background,omitempty"`
	OutputFormat string `json:"output_format"`
	N            int    `json:"n"`
}

func NewPromptBuilder(profile *model.StyleProfile) *PromptBuilder {
	return &PromptBuilder{Profile: profile}
}

func (b *PromptBuilder) Build(userInput string) string {
	parts := []string{}
	appendIfNotEmpty := func(s string) {
		if strings.TrimSpace(s) != "" {
			parts = append(parts, s)
		}
	}

	appendIfNotEmpty(b.Profile.ArtStyle)
	appendIfNotEmpty(b.Profile.ColorTone)
	appendIfNotEmpty(b.Profile.Lighting)
	appendIfNotEmpty(b.Profile.QualityTags)
	appendIfNotEmpty(b.Profile.Composition)
	parts = append(parts, b.Profile.ExtraTokens...)

	stylePart := strings.Join(parts, ", ")
	cleaned := sanitizeUserInput(userInput)

	if stylePart == "" {
		return cleaned
	}
	return fmt.Sprintf("%s. %s", stylePart, cleaned)
}

func (b *PromptBuilder) BuildAPIRequest(userInput string) ImageRequest {
	size := b.Profile.Size
	if preset, ok := model.SizePresets[size]; ok {
		size = preset
	}

	return ImageRequest{
		Model:        "gpt-image-2",
		Prompt:       b.Build(userInput),
		Size:         size,
		Quality:      b.Profile.APIQuality,
		Background:   b.Profile.Background,
		OutputFormat: b.Profile.OutputFormat,
		N:            1,
	}
}

func sanitizeUserInput(input string) string {
	blocked := []string{
		"realistic", "photorealistic", "3d render", "3d cg",
		"oil painting", "watercolor", "sketch", "pixel art",
	}
	result := input
	for _, word := range blocked {
		result = strings.ReplaceAll(strings.ToLower(result), word, "")
	}
	return strings.TrimSpace(result)
}
