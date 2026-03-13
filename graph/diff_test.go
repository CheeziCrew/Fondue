package graph

import (
	"strings"
	"testing"
)

func TestComputeDiffNoChanges(t *testing.T) {
	old := []DiffService{{Name: "a", Version: "1.0", Dependencies: []string{"b"}}}
	new := []DiffService{{Name: "a", Version: "1.0", Dependencies: []string{"b"}}}

	entries := ComputeDiff(old, new)
	if len(entries) != 0 {
		t.Errorf("expected no changes, got %d", len(entries))
	}
}

func TestComputeDiffServiceAdded(t *testing.T) {
	old := []DiffService{{Name: "a"}}
	new := []DiffService{{Name: "a"}, {Name: "b"}}

	entries := ComputeDiff(old, new)

	found := false
	for _, e := range entries {
		if e.Kind == DiffAdded && e.Service == "b" {
			found = true
		}
	}
	if !found {
		t.Error("expected DiffAdded for service 'b'")
	}
}

func TestComputeDiffServiceRemoved(t *testing.T) {
	old := []DiffService{{Name: "a"}, {Name: "b"}}
	new := []DiffService{{Name: "a"}}

	entries := ComputeDiff(old, new)

	found := false
	for _, e := range entries {
		if e.Kind == DiffRemoved && e.Service == "b" {
			found = true
		}
	}
	if !found {
		t.Error("expected DiffRemoved for service 'b'")
	}
}

func TestComputeDiffDepAdded(t *testing.T) {
	old := []DiffService{{Name: "a", Dependencies: []string{"b"}}}
	new := []DiffService{{Name: "a", Dependencies: []string{"b", "c"}}}

	entries := ComputeDiff(old, new)

	found := false
	for _, e := range entries {
		if e.Kind == DiffDepAdded && e.Service == "a" && e.Detail == "c" {
			found = true
		}
	}
	if !found {
		t.Error("expected DiffDepAdded for dep 'c' on service 'a'")
	}
}

func TestComputeDiffDepRemoved(t *testing.T) {
	old := []DiffService{{Name: "a", Dependencies: []string{"b", "c"}}}
	new := []DiffService{{Name: "a", Dependencies: []string{"b"}}}

	entries := ComputeDiff(old, new)

	found := false
	for _, e := range entries {
		if e.Kind == DiffDepRemoved && e.Service == "a" && e.Detail == "c" {
			found = true
		}
	}
	if !found {
		t.Error("expected DiffDepRemoved for dep 'c' on service 'a'")
	}
}

func TestComputeDiffVersionChanged(t *testing.T) {
	old := []DiffService{{Name: "a", Version: "1.0"}}
	new := []DiffService{{Name: "a", Version: "2.0"}}

	entries := ComputeDiff(old, new)

	found := false
	for _, e := range entries {
		if e.Kind == DiffVersionChanged && e.Service == "a" {
			found = true
			if !strings.Contains(e.Detail, "1.0") || !strings.Contains(e.Detail, "2.0") {
				t.Errorf("expected version details, got %q", e.Detail)
			}
		}
	}
	if !found {
		t.Error("expected DiffVersionChanged for service 'a'")
	}
}

func TestComputeDiffVersionAddedFromEmpty(t *testing.T) {
	old := []DiffService{{Name: "a", Version: ""}}
	new := []DiffService{{Name: "a", Version: "1.0"}}

	entries := ComputeDiff(old, new)

	found := false
	for _, e := range entries {
		if e.Kind == DiffVersionChanged && e.Service == "a" {
			found = true
			if !strings.Contains(e.Detail, "(none)") {
				t.Errorf("expected '(none)' in detail, got %q", e.Detail)
			}
		}
	}
	if !found {
		t.Error("expected DiffVersionChanged when version added from empty")
	}
}

func TestComputeDiffBothEmptyVersions(t *testing.T) {
	old := []DiffService{{Name: "a", Version: ""}}
	new := []DiffService{{Name: "a", Version: ""}}

	entries := ComputeDiff(old, new)
	if len(entries) != 0 {
		t.Errorf("expected no changes for both empty versions, got %d", len(entries))
	}
}

func TestComputeDiffSorted(t *testing.T) {
	old := []DiffService{{Name: "a"}, {Name: "b"}}
	new := []DiffService{{Name: "c"}} // removed a and b, added c

	entries := ComputeDiff(old, new)
	if len(entries) < 3 {
		t.Fatalf("expected at least 3 entries, got %d", len(entries))
	}

	// DiffAdded (0) < DiffRemoved (1) — wait, actually DiffAdded=0, DiffRemoved=1
	// So added should come first
	for i := 1; i < len(entries); i++ {
		if entries[i].Kind < entries[i-1].Kind {
			t.Errorf("entries not sorted by kind: %v before %v", entries[i-1].Kind, entries[i].Kind)
		}
	}
}

func TestComputeDiffEmpty(t *testing.T) {
	entries := ComputeDiff(nil, nil)
	if len(entries) != 0 {
		t.Errorf("expected no changes for nil inputs, got %d", len(entries))
	}
}

func TestFormatDiffEmpty(t *testing.T) {
	got := FormatDiff(nil)
	if got != "No changes detected." {
		t.Errorf("FormatDiff(nil) = %q, want %q", got, "No changes detected.")
	}
}

func TestFormatDiffWithEntries(t *testing.T) {
	entries := []DiffEntry{
		{Kind: DiffAdded, Service: "new-svc"},
		{Kind: DiffRemoved, Service: "old-svc"},
		{Kind: DiffDepAdded, Service: "svc", Detail: "dep"},
		{Kind: DiffDepRemoved, Service: "svc", Detail: "old-dep"},
		{Kind: DiffVersionChanged, Service: "svc", Detail: "1.0 → 2.0"},
	}
	got := FormatDiff(entries)
	if !strings.Contains(got, "+ new-svc") {
		t.Error("expected '+ new-svc' in output")
	}
	if !strings.Contains(got, "- old-svc") {
		t.Error("expected '- old-svc' in output")
	}
	if !strings.Contains(got, "+dep dep") {
		t.Error("expected '+dep dep' in output")
	}
	if !strings.Contains(got, "-dep old-dep") {
		t.Error("expected '-dep old-dep' in output")
	}
	if !strings.Contains(got, "version 1.0") {
		t.Error("expected version info in output")
	}
	if !strings.Contains(got, "5 change(s)") {
		t.Error("expected '5 change(s)' in output")
	}
}

func TestToSet(t *testing.T) {
	s := toSet([]string{"a", "b", "a"})
	if len(s) != 2 {
		t.Errorf("expected 2 unique items, got %d", len(s))
	}
	if !s["a"] || !s["b"] {
		t.Error("expected a and b in set")
	}
}

func TestToSetNil(t *testing.T) {
	s := toSet(nil)
	if len(s) != 0 {
		t.Errorf("expected empty set for nil, got %d", len(s))
	}
}

func TestVersionStr(t *testing.T) {
	if versionStr("") != "(none)" {
		t.Error("expected (none) for empty version")
	}
	if versionStr("1.0") != "1.0" {
		t.Error("expected 1.0")
	}
}
