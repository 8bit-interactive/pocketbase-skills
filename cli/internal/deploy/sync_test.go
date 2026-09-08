package deploy

import "testing"

func TestDiffDetectsCreateReplaceDeleteAndSame(t *testing.T) {
	local := []Entry{
		{Type: "folder", Name: "pb_public"},
		{Type: "file", Name: "pb_public/index.html", Hash: "new"},
		{Type: "file", Name: "pb_public/site.css", Hash: "same"},
		{Type: "file", Name: "pb_hooks/app.js", Hash: "created"},
	}
	remote := []Entry{
		{Type: "folder", Name: "pb_public"},
		{Type: "file", Name: "pb_public/index.html", Hash: "old"},
		{Type: "file", Name: "pb_public/site.css", Hash: "same"},
		{Type: "file", Name: "pb_public/removed.txt", Hash: "gone"},
	}
	result := diff(local, remote)
	if len(result.Uploaded) != 1 || result.Uploaded[0] != "pb_hooks/app.js" {
		t.Fatalf("uploaded = %#v", result.Uploaded)
	}
	if len(result.Replaced) != 1 || result.Replaced[0] != "pb_public/index.html" {
		t.Fatalf("replaced = %#v", result.Replaced)
	}
	if len(result.Deleted) != 1 || result.Deleted[0] != "pb_public/removed.txt" {
		t.Fatalf("deleted = %#v", result.Deleted)
	}
	if len(result.Unchanged) != 2 {
		t.Fatalf("unchanged = %#v", result.Unchanged)
	}
}
