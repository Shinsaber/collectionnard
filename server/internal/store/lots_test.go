package store

import (
	"encoding/json"
	"testing"
)

// P7-1: the photos of a merged lot join those of the lot that receives its copies.
func TestMergeGalleries(t *testing.T) {
	into := json.RawMessage(`{"condition":"good","photos":{"condition":["/u/a.jpg"],"default_image":"/u/a.jpg"}}`)
	from := json.RawMessage(`{"condition":"poor","photos":{"condition":["/u/a.jpg","/u/b.jpg"],"back":["/u/c.jpg",3],"default_image":"/u/b.jpg"},"other":{"x":["/u/d.jpg"]}}`)
	got, err := mergeGalleries(into, from, []string{"photos"})
	if err != nil {
		t.Fatal(err)
	}
	var attrs map[string]any
	_ = json.Unmarshal(got, &attrs)
	photos := attrs["photos"].(map[string]any)
	if attrs["condition"] != "good" || attrs["other"] != nil {
		t.Errorf("other attributes must stay those of into: %s", got)
	}
	if c := photos["condition"].([]any); len(c) != 2 || c[1] != "/u/b.jpg" {
		t.Errorf("category without duplicates: %v", c)
	}
	if b := photos["back"].([]any); len(b) != 1 || b[0] != "/u/c.jpg" {
		t.Errorf("new category, URLs only: %v", b)
	}
	if photos["default_image"] != "/u/a.jpg" {
		t.Errorf("into keeps its default image: %v", photos["default_image"])
	}

	// No gallery in into: it receives from's, default image included.
	got, _ = mergeGalleries(json.RawMessage(`{}`), from, []string{"photos"})
	if err := json.Unmarshal(got, &attrs); err != nil || attrs["photos"].(map[string]any)["default_image"] != "/u/b.jpg" {
		t.Errorf("empty into: %s", got)
	}
	// Nothing to merge: into unchanged, byte for byte.
	if got, _ := mergeGalleries(into, json.RawMessage(`{"condition":"poor"}`), []string{"photos"}); string(got) != string(into) {
		t.Errorf("unchanged: %s", got)
	}
}
