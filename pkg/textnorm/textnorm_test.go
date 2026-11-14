package textnorm

import "testing"

func TestNormalizeID(t *testing.T) {
	tests := []struct {
		in, want string
		opts     Options
	}{
		// number - existing logic
		{"1.000", "seribu", Options{Language: "id", NumberMode: "auto"}},
		{"08123456789", "nol delapan satu dua tiga empat lima enam tujuh delapan sembilan", Options{Language: "id"}},
		{"1234", "seribu dua ratus tiga puluh empat", Options{Language: "id"}},
		// date YYYY-MM-DD
		{"2024-01-15", "lima belas Januari dua ribu dua puluh empat", Options{Language: "id"}},
		{"Tanggal 2024-12-31", "Tanggal tiga puluh satu Desember dua ribu dua puluh empat", Options{Language: "id"}},
		// time - new spec: 10:30 -> sepuluh lewat tiga puluh menit, 10:00 -> sepuluh
		{"22:22", "dua puluh dua lewat dua puluh dua menit", Options{Language: "id"}},
		{"09:30", "sembilan lewat tiga puluh menit", Options{Language: "id"}},
		{"10:00", "sepuluh", Options{Language: "id"}},
		{"19:45:30", "sembilan belas lewat empat puluh lima menit tiga puluh detik", Options{Language: "id"}},
		// abbr
		{"WIB", "W I B", Options{Language: "id"}},
		{"GMT dan UTC", "G M T dan U T C", Options{Language: "id"}},
		// email
		{"a@gmail.com", "a at gmail dot com", Options{Language: "id"}},
		{"first.last@gmail.com", "first dot last at gmail dot com", Options{Language: "id"}},
		{"hubungi budi@example.co.id", "hubungi budi at example dot co dot id", Options{Language: "id"}},
		// domain / url
		{"https://google.com", "google dot com", Options{Language: "id"}},
		{"http://example.com/path", "example dot com", Options{Language: "id"}},
		{"google.com", "google dot com", Options{Language: "id"}},
		{"example.co.id", "example dot co dot id", Options{Language: "id"}},
		// combined
		{"Beli 2 apel di https://toko.com pada 2024-01-15 jam 10:30 hubungi a@gmail.com WIB",
			"Beli dua apel di toko dot com pada lima belas Januari dua ribu dua puluh empat jam sepuluh lewat tiga puluh menit hubungi a at gmail dot com W I B",
			Options{Language: "id"}},
		{"Jam 10:00 WIB", "Jam sepuluh W I B", Options{Language: "id"}},
		// number mode none should not convert
		{"1.000", "1.000", Options{Language: "id", NumberMode: "none"}},
	}
	for _, tt := range tests {
		got := Normalize(tt.in, tt.opts)
		if got != tt.want {
			t.Errorf("Normalize(%q, %+v) = %q, want %q", tt.in, tt.opts, got, tt.want)
		}
	}
}

func TestNormalizeEN(t *testing.T) {
	tests := []struct {
		in, want string
		opts     Options
	}{
		{"2024-01-15", "January fifteen two thousand twenty four", Options{Language: "en"}},
		{"22:22", "twenty two twenty two", Options{Language: "en"}},
		{"a@gmail.com", "a at gmail dot com", Options{Language: "en"}},
		{"https://google.com", "google dot com", Options{Language: "en"}},
		{"NASA", "N A S A", Options{Language: "en"}},
	}
	for _, tt := range tests {
		got := Normalize(tt.in, tt.opts)
		if got != tt.want {
			t.Errorf("Normalize EN %q = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestOrder(t *testing.T) {
	// ensure number inside url/email is verbalized and not double processed as date
	opts := Options{Language: "id"}
	in := "https://example123.com"
	got := Normalize(in, opts)
	// numbers inside domain should be spelled digit-by-digit to not miss them
	if got != "example satu dua tiga dot com" {
		t.Errorf("url with number: got %q want %q", got, "example satu dua tiga dot com")
	}
	// email with date-like local should NOT be parsed as date, but as digits
	in2 := "2024-01-15@example.com"
	got2 := Normalize(in2, opts)
	// after fix: digits spelled, dashes kept
	expected := "dua nol dua empat - nol satu - satu lima at example dot com"
	if got2 != expected {
		t.Errorf("email with date-like local: got %q want %q", got2, expected)
	}
}
