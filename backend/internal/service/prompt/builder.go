package prompt

import (
	"fmt"
	"strings"

	"github.com/imagegen/backend/internal/model"
)

type PromptBuilder struct {
	Profile *model.StyleProfile
	// Runtime overrides (from task)
	SizeOverride    string
	QualityOverride string
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
	parts = append(parts, b.Profile.ExtraTokens...)

	stylePart := strings.Join(parts, ", ")
	cleaned := strings.TrimSpace(userInput)

	if stylePart == "" {
		return cleaned
	}
	return fmt.Sprintf("%s. %s", stylePart, cleaned)
}

// BuildAPIRequest constructs the API request body appropriate for the selected model.
func (b *PromptBuilder) BuildAPIRequest(userInput string) map[string]interface{} {
	profile := b.Profile
	prompt := b.Build(userInput)
	modelID := profile.Model
	if modelID == "" {
		modelID = "gpt-4o-image"
	}

	body := map[string]interface{}{
		"model":  modelID,
		"prompt": prompt,
	}

	size := b.SizeOverride
	if size == "" {
		size = "auto"
	}
	apiQuality := b.QualityOverride
	if apiQuality == "" {
		apiQuality = "medium"
	}

	switch model.GetModelType(modelID) {
	case model.ModelTypeGPT:
		if size != "auto" {
			body["size"] = size
		}
	case model.ModelTypeGemini, model.ModelTypeBanana:
		b.buildGeminiParams(body, profile, size, apiQuality)
	case model.ModelTypeMJ:
		// MJ only needs model + prompt
	}

	return body
}

func (b *PromptBuilder) buildGeminiParams(body map[string]interface{}, profile *model.StyleProfile, size, apiQuality string) {
	// Convert pixel size to aspect ratio
	aspectRatio := "1:1"
	if ar, ok := model.PixelSizeToAspectRatio[size]; ok {
		aspectRatio = ar
	}
	body["aspectRatio"] = aspectRatio

	rf := "url"
	body["response_format"] = rf

	if profile.ReferenceImageURL != "" {
		body["image"] = profile.ReferenceImageURL
	}

	// Image size only for gemini-3-pro and nano-banana-2 variants
	if strings.Contains(profile.Model, "gemini-3-pro") ||
		strings.Contains(profile.Model, "nano-banana-2") {
		if apiQuality == "high" {
			body["imageSize"] = "4K"
		} else if apiQuality == "medium" {
			body["imageSize"] = "2K"
		} else {
			body["imageSize"] = "1K"
		}
	}
}
