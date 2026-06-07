package domain

import "strings"

type SocialTaskMediaRef struct {
	Source      string `json:"source,omitempty"`
	StorageKey  string `json:"storage_key,omitempty"`
	URL         string `json:"url,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	FileName    string `json:"file_name,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
	ByteSize    int64  `json:"byte_size,omitempty"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
}

func (m SocialTaskMediaRef) IsZero() bool {
	return strings.TrimSpace(m.Source) == "" &&
		strings.TrimSpace(m.StorageKey) == "" &&
		strings.TrimSpace(m.URL) == "" &&
		strings.TrimSpace(m.ContentType) == "" &&
		strings.TrimSpace(m.FileName) == "" &&
		strings.TrimSpace(m.SHA256) == "" &&
		m.ByteSize == 0 &&
		m.Width == 0 &&
		m.Height == 0
}

type SocialProfileUpdateParams struct {
	DisplayName string `json:"display_name,omitempty"`
	ScreenName  string `json:"screen_name,omitempty"`
	Description string `json:"description,omitempty"`
	Location    string `json:"location,omitempty"`
	URL         string `json:"url,omitempty"`
}

func (p SocialProfileUpdateParams) IsZero() bool {
	return strings.TrimSpace(p.DisplayName) == "" &&
		strings.TrimSpace(p.ScreenName) == "" &&
		strings.TrimSpace(p.Description) == "" &&
		strings.TrimSpace(p.Location) == "" &&
		strings.TrimSpace(p.URL) == ""
}

type SocialTaskTemplateParams struct {
	Targets      []string                   `json:"targets,omitempty"`
	Contents     []string                   `json:"contents,omitempty"`
	QuotePostURL string                     `json:"quote_post_url,omitempty"`
	Media        []SocialTaskMediaRef       `json:"media,omitempty"`
	Profile      *SocialProfileUpdateParams `json:"profile,omitempty"`
	Avatar       *SocialTaskMediaRef        `json:"avatar,omitempty"`
	Banner       *SocialTaskMediaRef        `json:"banner,omitempty"`
}

func (p SocialTaskTemplateParams) IsZero() bool {
	return len(p.Targets) == 0 &&
		len(p.Contents) == 0 &&
		strings.TrimSpace(p.QuotePostURL) == "" &&
		len(p.Media) == 0 &&
		(p.Profile == nil || p.Profile.IsZero()) &&
		(p.Avatar == nil || p.Avatar.IsZero()) &&
		(p.Banner == nil || p.Banner.IsZero())
}

type SocialPostPayload struct {
	Text         string               `json:"text,omitempty"`
	QuotePostURL string               `json:"quote_post_url,omitempty"`
	Media        []SocialTaskMediaRef `json:"media,omitempty"`
}

func (p SocialPostPayload) IsZero() bool {
	return strings.TrimSpace(p.Text) == "" &&
		strings.TrimSpace(p.QuotePostURL) == "" &&
		len(p.Media) == 0
}

type SocialTaskPayload struct {
	Target  string                     `json:"target,omitempty"`
	Post    *SocialPostPayload         `json:"post,omitempty"`
	Profile *SocialProfileUpdateParams `json:"profile,omitempty"`
	Avatar  *SocialTaskMediaRef        `json:"avatar,omitempty"`
	Banner  *SocialTaskMediaRef        `json:"banner,omitempty"`
}

func (p SocialTaskPayload) IsZero() bool {
	return strings.TrimSpace(p.Target) == "" &&
		(p.Post == nil || p.Post.IsZero()) &&
		(p.Profile == nil || p.Profile.IsZero()) &&
		(p.Avatar == nil || p.Avatar.IsZero()) &&
		(p.Banner == nil || p.Banner.IsZero())
}

type SocialTaskTemplateSnapshot struct {
	TemplateID   string                   `json:"template_id,omitempty"`
	TemplateName string                   `json:"template_name,omitempty"`
	TemplateType string                   `json:"template_type,omitempty"`
	Params       SocialTaskTemplateParams `json:"params,omitempty"`
}

func (s SocialTaskTemplateSnapshot) IsZero() bool {
	return strings.TrimSpace(s.TemplateID) == "" &&
		strings.TrimSpace(s.TemplateName) == "" &&
		strings.TrimSpace(s.TemplateType) == "" &&
		s.Params.IsZero()
}
