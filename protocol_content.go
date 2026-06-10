package wire

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// UserInput can be plain text or an array of content parts.
type UserInput struct {
	Text  string        `json:"text,omitempty"`
	Parts []ContentPart `json:"parts,omitempty"`
}

// MarshalJSON serializes UserInput as either a string or an array of ContentPart.
func (u UserInput) MarshalJSON() ([]byte, error) {
	if u.Text != "" && len(u.Parts) > 0 {
		return nil, fmt.Errorf("user_input cannot have both text and parts")
	}
	if u.Text != "" {
		return json.Marshal(u.Text)
	}
	if len(u.Parts) > 0 {
		return json.Marshal(u.Parts)
	}
	return json.Marshal("")
}

// UnmarshalJSON parses UserInput from either a string or an array of ContentPart.
func (u *UserInput) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("user_input must be a string or an array of content parts, got null")
	}
	*u = UserInput{}
	trimmed := bytes.TrimLeft(data, " \t\r\n")
	if len(trimmed) == 0 {
		return fmt.Errorf("user_input must be a string or an array of content parts, got %s", string(data))
	}
	switch trimmed[0] {
	case '"':
		return json.Unmarshal(data, &u.Text)
	case '[':
		return json.Unmarshal(data, &u.Parts)
	case 'n':
		return fmt.Errorf("user_input must be a string or an array of content parts, got null")
	}
	return fmt.Errorf("user_input must be a string or an array of content parts, got %s", string(data))
}

// ContentPart is a content part in a message.
type ContentPart struct {
	Type     string        `json:"type"`
	Text     *TextPart     `json:"text,omitempty"`
	Think    *ThinkPart    `json:"think,omitempty"`
	ImageURL *ImageURLPart `json:"image_url,omitempty"`
	AudioURL *AudioURLPart `json:"audio_url,omitempty"`
	VideoURL *VideoURLPart `json:"video_url,omitempty"`
}

// ContentPartType values.
const (
	ContentPartTypeText     = "text"
	ContentPartTypeThink    = "think"
	ContentPartTypeImageURL = "image_url"
	ContentPartTypeAudioURL = "audio_url"
	ContentPartTypeVideoURL = "video_url"
)

// TextPart is plain text content.
type TextPart struct {
	Text string `json:"text"`
}

// ThinkPart is thinking / reasoning content.
type ThinkPart struct {
	Think     string `json:"think"`
	Encrypted string `json:"encrypted,omitempty"`
}

// ImageURLPart is an image referenced by URL.
type ImageURLPart struct {
	ImageURL MediaURL `json:"image_url"`
}

// AudioURLPart is audio referenced by URL.
type AudioURLPart struct {
	AudioURL MediaURL `json:"audio_url"`
}

// VideoURLPart is video referenced by URL.
type VideoURLPart struct {
	VideoURL MediaURL `json:"video_url"`
}

// MediaURL is a media URL with an optional ID.
type MediaURL struct {
	URL string `json:"url"`
	ID  string `json:"id,omitempty"`
}

// DisplayBlockType is the discriminator for DisplayBlock.
type DisplayBlockType string

const (
	DisplayBlockTypeBrief   DisplayBlockType = "brief"
	DisplayBlockTypeDiff    DisplayBlockType = "diff"
	DisplayBlockTypeTodo    DisplayBlockType = "todo"
	DisplayBlockTypeShell   DisplayBlockType = "shell"
	DisplayBlockTypeUnknown DisplayBlockType = "unknown"
)

// DisplayBlock is a display block shown to the user.
type DisplayBlock struct {
	Type      DisplayBlockType  `json:"type"`
	Text      string            `json:"text,omitempty"`
	Path      string            `json:"path,omitempty"`
	OldText   string            `json:"old_text,omitempty"`
	NewText   string            `json:"new_text,omitempty"`
	IsSummary *bool             `json:"is_summary,omitempty"`
	Items     []TodoDisplayItem `json:"items,omitempty"`
	Language  string            `json:"language,omitempty"`
	Command   string            `json:"command,omitempty"`
	Data      json.RawMessage   `json:"data,omitempty"`
}

// TodoDisplayItem is a single item in a todo display block.
type TodoDisplayItem struct {
	Title  string     `json:"title"`
	Status TodoStatus `json:"status"`
}

// TodoStatus is the status of a todo item.
type TodoStatus string

const (
	TodoStatusPending    TodoStatus = "pending"
	TodoStatusInProgress TodoStatus = "in_progress"
	TodoStatusDone       TodoStatus = "done"
)

// ToolReturnValue is the result of a tool execution.
type ToolReturnValue struct {
	IsError bool            `json:"is_error"`
	Output  ToolOutput      `json:"output"`
	Message string          `json:"message"`
	Display []DisplayBlock  `json:"display,omitempty"`
	Extras  json.RawMessage `json:"extras,omitempty"`
}

// ToolOutput can be plain text or an array of content parts.
type ToolOutput struct {
	Text  string        `json:"text,omitempty"`
	Parts []ContentPart `json:"parts,omitempty"`
}

// MarshalJSON serializes ToolOutput as either a string or an array of ContentPart.
func (o ToolOutput) MarshalJSON() ([]byte, error) {
	if o.Text != "" && len(o.Parts) > 0 {
		return nil, fmt.Errorf("tool_output cannot have both text and parts")
	}
	if o.Text != "" {
		return json.Marshal(o.Text)
	}
	if len(o.Parts) > 0 {
		return json.Marshal(o.Parts)
	}
	return json.Marshal("")
}

// UnmarshalJSON parses ToolOutput from either a string or an array of ContentPart.
func (o *ToolOutput) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("tool_output must be a string or an array of content parts, got null")
	}
	*o = ToolOutput{}
	trimmed := bytes.TrimLeft(data, " \t\r\n")
	if len(trimmed) == 0 {
		return fmt.Errorf("tool_output must be a string or an array of content parts, got %s", string(data))
	}
	switch trimmed[0] {
	case '"':
		return json.Unmarshal(data, &o.Text)
	case '[':
		return json.Unmarshal(data, &o.Parts)
	case 'n':
		return fmt.Errorf("tool_output must be a string or an array of content parts, got null")
	}
	return fmt.Errorf("tool_output must be a string or an array of content parts, got %s", string(data))
}

// DisplayBlock builders.

// NewDisplayBlockBrief creates a brief text display block.
func NewDisplayBlockBrief(text string) DisplayBlock {
	return DisplayBlock{Type: DisplayBlockTypeBrief, Text: text}
}

// NewDisplayBlockDiff creates a diff display block.
func NewDisplayBlockDiff(path, oldText, newText string) DisplayBlock {
	return DisplayBlock{Type: DisplayBlockTypeDiff, Path: path, OldText: oldText, NewText: newText}
}

// NewDisplayBlockTodo creates a todo list display block.
func NewDisplayBlockTodo(items []TodoDisplayItem) DisplayBlock {
	return DisplayBlock{Type: DisplayBlockTypeTodo, Items: items}
}

// NewDisplayBlockShell creates a shell command display block.
func NewDisplayBlockShell(command, language string) DisplayBlock {
	return DisplayBlock{Type: DisplayBlockTypeShell, Command: command, Language: language}
}
