package idx

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Announcement caps.
const (
	defaultAnnounceDays  = 30
	maxAnnounceDays      = 365
	defaultAnnounceLimit = 20
	maxAnnounceLimit     = 50
)

// AnnouncementAttachment is one document attached to an announcement.
type AnnouncementAttachment struct {
	Filename string `json:"filename"`
	URL      string `json:"url"`
}

// Announcement is one IDX disclosure (keterbukaan informasi) entry.
type Announcement struct {
	Number      string                   `json:"number"`
	Date        string                   `json:"date"`
	Title       string                   `json:"title"`
	Type        string                   `json:"type"` // e.g. STOCK
	Code        string                   `json:"code"`
	Attachments []AnnouncementAttachment `json:"attachments,omitempty"`
}

// AnnouncementList is the disclosure feed for one emiten.
type AnnouncementList struct {
	Code          string         `json:"code"`
	From          string         `json:"from"`
	To            string         `json:"to"`
	Total         int            `json:"total"` // matches on IDX's side, before the limit
	Announcements []Announcement `json:"announcements"`
	Truncated     bool           `json:"truncated"`
}

// rawAnnouncementResponse maps GetAnnouncement (modelled on a captured
// response; Kode_Emiten is space-padded to 100 chars).
type rawAnnouncementResponse struct {
	ResultCount int `json:"ResultCount"`
	Replies     []struct {
		Pengumuman struct {
			NoPengumuman      string `json:"NoPengumuman"`
			TglPengumuman     string `json:"TglPengumuman"`
			JudulPengumuman   string `json:"JudulPengumuman"`
			JenisPengumuman   string `json:"JenisPengumuman"`
			KodeEmiten        string `json:"Kode_Emiten"`
			PerihalPengumuman string `json:"PerihalPengumuman"`
		} `json:"pengumuman"`
		Attachments []struct {
			FullSavePath     string `json:"FullSavePath"`
			OriginalFilename string `json:"OriginalFilename"`
		} `json:"attachments"`
	} `json:"Replies"`
}

// Announcements returns official IDX disclosures for `code` over the last
// `days` days (default 30, max 365), newest first, optionally filtered by a
// keyword understood by IDX's own search. limit caps the rows (default 20,
// max 50); Total reports how many matched overall.
func (c *Client) Announcements(ctx context.Context, code, keyword string, days, limit int) (*AnnouncementList, error) {
	code = normalizeCode(code)
	if code == "" {
		return nil, fmt.Errorf("empty emiten code")
	}
	if days < 1 {
		days = defaultAnnounceDays
	}
	if days > maxAnnounceDays {
		days = maxAnnounceDays
	}
	if limit < 1 {
		limit = defaultAnnounceLimit
	}
	if limit > maxAnnounceLimit {
		limit = maxAnnounceLimit
	}

	now := time.Now()
	from := now.AddDate(0, 0, -days)
	u := fmt.Sprintf(
		"%s/primary/ListedCompany/GetAnnouncement?kodeEmiten=%s&emitenType=*&indexFrom=0&pageSize=%d&dateFrom=%s&dateTo=%s&lang=id&keyword=%s",
		baseURL, url.QueryEscape(code), limit,
		from.Format("20060102"), now.Format("20060102"), url.QueryEscape(keyword))
	key := fmt.Sprintf("announce:%s:%s:%d:%d:%s", code, now.Format("20060102"), days, limit, keyword)

	var raw rawAnnouncementResponse
	if err := c.getJSON(ctx, key, u, ttlAnnounce, &raw); err != nil {
		return nil, err
	}

	list := &AnnouncementList{
		Code:  code,
		From:  from.Format("2006-01-02"),
		To:    now.Format("2006-01-02"),
		Total: raw.ResultCount,
	}
	for _, r := range raw.Replies {
		a := Announcement{
			Number: r.Pengumuman.NoPengumuman,
			Date:   strings.Replace(r.Pengumuman.TglPengumuman, "T", " ", 1),
			Title:  r.Pengumuman.JudulPengumuman,
			Type:   r.Pengumuman.JenisPengumuman,
			Code:   normalizeCode(r.Pengumuman.KodeEmiten),
		}
		for _, att := range r.Attachments {
			if att.FullSavePath == "" {
				continue
			}
			a.Attachments = append(a.Attachments, AnnouncementAttachment{
				Filename: att.OriginalFilename,
				URL:      att.FullSavePath,
			})
		}
		list.Announcements = append(list.Announcements, a)
	}
	list.Truncated = list.Total > len(list.Announcements)
	return list, nil
}
