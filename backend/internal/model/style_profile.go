package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type StyleProfile struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name      string             `json:"name" bson:"name"`
	CreatedBy string             `json:"created_by" bson:"created_by"`
	Version   int                `json:"version" bson:"version"`
	IsLocked  bool               `json:"is_locked" bson:"is_locked"`

	// Prompt 语义层
	ArtStyle    string   `json:"art_style" bson:"art_style"`
	ColorTone   string   `json:"color_tone" bson:"color_tone"`
	Lighting    string   `json:"lighting" bson:"lighting"`
	QualityTags string   `json:"quality_tags" bson:"quality_tags"`
	Composition string   `json:"composition" bson:"composition"`
	ExtraTokens []string `json:"extra_tokens" bson:"extra_tokens"`

	// API 参数层
	Size         string `json:"size" bson:"size"`
	APIQuality   string `json:"api_quality" bson:"api_quality"`
	Background   string `json:"background" bson:"background"`
	OutputFormat string `json:"output_format" bson:"output_format"`
	Compression  int    `json:"compression" bson:"compression"`

	// 风格参考图
	ReferenceImageURL string `json:"reference_image_url" bson:"reference_image_url"`

	// 锁定后的 Prompt 模板
	LockedPromptPrefix string    `json:"locked_prompt_prefix" bson:"locked_prompt_prefix"`
	CreatedAt          time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" bson:"updated_at"`
}

var SizePresets = map[string]string{
	"square_1k":     "1024x1024",
	"landscape_hd":  "1536x1024",
	"portrait_hd":   "1024x1536",
}
