package idx

import (
	"context"
	"testing"
)

// Trimmed from a captured live response (BBCA, July 2026).
const announcementJSON = `{"ResultCount":45,"Replies":[
 {"pengumuman":{"NoPengumuman":"0053/ESG/2026","TglPengumuman":"2026-06-05T16:48:01","JudulPengumuman":"Jadwal Dividen Tunai Interim","JenisPengumuman":"STOCK","Kode_Emiten":"BBCA                                ","PerihalPengumuman":"Jadwal Dividen Tunai Interim"},
  "attachments":[{"FullSavePath":"https://www.idx.co.id/StaticData/NewsAndAnnouncement/ANNOUNCEMENTSTOCK/From_EREP/202606/x.pdf","OriginalFilename":"20260605_BBCA_Jadwal Dividen.pdf"},{"FullSavePath":"","OriginalFilename":"empty path is skipped"}]},
 {"pengumuman":{"NoPengumuman":"0054/ESG/2026","TglPengumuman":"2026-06-08T17:30:00","JudulPengumuman":"Laporan Bulanan Registrasi Pemegang Efek","JenisPengumuman":"STOCK","Kode_Emiten":"BBCA","PerihalPengumuman":"Laporan Bulanan"},
  "attachments":[]}
]}`

func TestAnnouncements(t *testing.T) {
	f := &fakeFetcher{responses: map[string]string{"GetAnnouncement": announcementJSON}}
	c := New(f, nil)

	list, err := c.Announcements(context.Background(), "bbca", "", 30, 20)
	if err != nil {
		t.Fatalf("Announcements: %v", err)
	}
	if list.Code != "BBCA" || list.Total != 45 || !list.Truncated {
		t.Errorf("list meta = code %q total %d truncated %v", list.Code, list.Total, list.Truncated)
	}
	if len(list.Announcements) != 2 {
		t.Fatalf("got %d announcements, want 2", len(list.Announcements))
	}
	a := list.Announcements[0]
	if a.Title != "Jadwal Dividen Tunai Interim" || a.Code != "BBCA" {
		t.Errorf("announcement = %+v (space-padded code must be trimmed)", a)
	}
	if a.Date != "2026-06-05 16:48:01" {
		t.Errorf("date = %q", a.Date)
	}
	// Empty FullSavePath attachments are dropped.
	if len(a.Attachments) != 1 || a.Attachments[0].Filename != "20260605_BBCA_Jadwal Dividen.pdf" {
		t.Errorf("attachments = %+v", a.Attachments)
	}

	if _, err := c.Announcements(context.Background(), "", "", 0, 0); err == nil {
		t.Error("expected error for empty code")
	}
}
