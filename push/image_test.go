package push

import "testing"

func TestRelativeImagePath(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"absolute is stripped", "https://storage.googleapis.com/olympsis-event-media/x.jpg", "event-media/x.jpg"},
		{"already relative unchanged", "event-media/x.jpg", "event-media/x.jpg"},
		{"empty stays empty", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := relativeImagePath(c.in); got != c.want {
				t.Errorf("relativeImagePath(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestAbsoluteImageURL(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"relative gets prefixed", "event-media/x.jpg", "https://storage.googleapis.com/olympsis-event-media/x.jpg"},
		{"already absolute unchanged", "https://storage.googleapis.com/olympsis-event-media/x.jpg", "https://storage.googleapis.com/olympsis-event-media/x.jpg"},
		{"empty stays empty", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := absoluteImageURL(c.in); got != c.want {
				t.Errorf("absoluteImageURL(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
