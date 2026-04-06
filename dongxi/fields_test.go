package dongxi

import "testing"

func TestNewNote(t *testing.T) {
	note := NewNote("hello world")
	if note[NoteKeyType] != "tx" {
		t.Errorf("_t = %v, want tx", note[NoteKeyType])
	}
	if note[NoteKeyLength] != 11 {
		t.Errorf("ch = %v, want 11", note[NoteKeyLength])
	}
	if note[NoteKeyValue] != "hello world" {
		t.Errorf("v = %v, want 'hello world'", note[NoteKeyValue])
	}
	if note[NoteKeyVersion] != 1 {
		t.Errorf("t = %v, want 1", note[NoteKeyVersion])
	}
}

func TestNewNoteEmpty(t *testing.T) {
	note := NewNote("")
	if note[NoteKeyLength] != 0 {
		t.Errorf("ch = %v, want 0", note[NoteKeyLength])
	}
	if note[NoteKeyValue] != "" {
		t.Errorf("v = %v, want empty", note[NoteKeyValue])
	}
}

func TestNoteText(t *testing.T) {
	note := map[string]any{"_t": "tx", "ch": 5, "v": "hello", "t": 1}
	text := NoteText(note)
	if text != "hello" {
		t.Errorf("NoteText = %q, want 'hello'", text)
	}
}

func TestNoteTextNil(t *testing.T) {
	if got := NoteText(nil); got != "" {
		t.Errorf("NoteText(nil) = %q, want empty", got)
	}
}

func TestNoteTextWrongType(t *testing.T) {
	if got := NoteText("not a map"); got != "" {
		t.Errorf("NoteText(string) = %q, want empty", got)
	}
}

func TestNoteTextMissingValue(t *testing.T) {
	note := map[string]any{"_t": "tx"}
	if got := NoteText(note); got != "" {
		t.Errorf("NoteText(no v) = %q, want empty", got)
	}
}
