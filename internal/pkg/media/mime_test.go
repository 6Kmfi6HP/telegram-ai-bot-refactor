package media

import "testing"

func TestMIMETypeFromExt(t *testing.T) {
	tests := map[string]string{
		"photo.jpg":  "image/jpeg",
		"photo.jpeg": "image/jpeg",
		"photo.png":  "image/png",
		"photo.gif":  "image/gif",
		"photo.webp": "image/webp",
		"photo.bin":  "image/jpeg",
	}
	for path, want := range tests {
		if got := MIMETypeFromExt(path); got != want {
			t.Fatalf("MIMETypeFromExt(%q)=%q, want %q", path, got, want)
		}
	}
}
