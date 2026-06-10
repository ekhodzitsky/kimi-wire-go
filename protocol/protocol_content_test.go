package protocol

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestUserInputTextRoundtrip(t *testing.T) {
	original := UserInput{Text: "hello"}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(data) != `"hello"` {
		t.Fatalf("expected wire string, got %s", data)
	}
	var parsed UserInput
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed.Text != "hello" {
		t.Fatalf("expected hello, got %s", parsed.Text)
	}
}

func TestUserInputPartsRoundtrip(t *testing.T) {
	original := UserInput{Parts: []ContentPart{{Type: ContentPartTypeText, Text: &TextPart{Text: "hi"}}}}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var parsed UserInput
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(parsed.Parts) != 1 || parsed.Parts[0].Type != ContentPartTypeText {
		t.Fatalf("roundtrip mismatch")
	}
}

func TestUserInputWireString(t *testing.T) {
	var ui UserInput
	if err := json.Unmarshal([]byte(`"hello"`), &ui); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ui.Text != "hello" {
		t.Fatalf("expected text hello, got %q", ui.Text)
	}
	out, err := json.Marshal(ui)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(out) != `"hello"` {
		t.Fatalf("expected wire string, got %s", out)
	}
}

func TestUserInputWireArray(t *testing.T) {
	var ui UserInput
	in := `[{"type":"text","text":{"text":"hi"}}]`
	if err := json.Unmarshal([]byte(in), &ui); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(ui.Parts) != 1 || ui.Parts[0].Text == nil || ui.Parts[0].Text.Text != "hi" {
		t.Fatalf("roundtrip mismatch: %+v", ui)
	}
}

func TestToolOutputWireString(t *testing.T) {
	var to ToolOutput
	if err := json.Unmarshal([]byte(`"result text"`), &to); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if to.Text != "result text" {
		t.Fatalf("expected result text, got %q", to.Text)
	}
	out, err := json.Marshal(to)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(out) != `"result text"` {
		t.Fatalf("expected wire string, got %s", out)
	}
}

func TestToolOutputWireArray(t *testing.T) {
	var to ToolOutput
	in := `[{"type":"text","text":{"text":"hi"}}]`
	if err := json.Unmarshal([]byte(in), &to); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(to.Parts) != 1 || to.Parts[0].Text == nil || to.Parts[0].Text.Text != "hi" {
		t.Fatalf("roundtrip mismatch: %+v", to)
	}
}

func TestContentPartTextRoundtrip(t *testing.T) {
	original := ContentPart{Type: ContentPartTypeText, Text: &TextPart{Text: "hello"}}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var parsed ContentPart
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed.Type != ContentPartTypeText || parsed.Text.Text != "hello" {
		t.Fatalf("roundtrip mismatch")
	}
}

func TestDisplayBlockBriefRoundtrip(t *testing.T) {
	original := DisplayBlock{Type: DisplayBlockTypeBrief, Text: "summary"}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var parsed DisplayBlock
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed.Type != DisplayBlockTypeBrief || parsed.Text != "summary" {
		t.Fatalf("roundtrip mismatch")
	}
}

func TestUserInputNullRejection(t *testing.T) {
	var ui UserInput
	if err := json.Unmarshal([]byte(`null`), &ui); err == nil {
		t.Fatal("expected error unmarshaling null into UserInput")
	}
}

func TestToolOutputNullRejection(t *testing.T) {
	var to ToolOutput
	if err := json.Unmarshal([]byte(`null`), &to); err == nil {
		t.Fatal("expected error unmarshaling null into ToolOutput")
	}
}

func TestContentPartAllVariants(t *testing.T) {
	cases := []struct {
		name string
		part ContentPart
	}{
		{"text", ContentPart{Type: ContentPartTypeText, Text: &TextPart{Text: "hi"}}},
		{"think", ContentPart{Type: ContentPartTypeThink, Think: &ThinkPart{Think: "..."}}},
		{"image_url", ContentPart{Type: ContentPartTypeImageURL, ImageURL: &ImageURLPart{ImageURL: MediaURL{URL: "https://x/img.png", ID: "i1"}}}},
		{"audio_url", ContentPart{Type: ContentPartTypeAudioURL, AudioURL: &AudioURLPart{AudioURL: MediaURL{URL: "https://x/a.mp3"}}}},
		{"video_url", ContentPart{Type: ContentPartTypeVideoURL, VideoURL: &VideoURLPart{VideoURL: MediaURL{URL: "https://x/v.mp4"}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.part)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var got ContentPart
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if !reflect.DeepEqual(tc.part, got) {
				t.Fatalf("roundtrip mismatch\noriginal: %+v\ngot: %+v", tc.part, got)
			}
		})
	}
}
