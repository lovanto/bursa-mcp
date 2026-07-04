package idx

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// CompanyProfile is the cleaned subset of GetCompanyProfilesDetail most useful
// for analysis. The raw endpoint also returns directors, shareholders,
// dividends, subsidiaries, etc. — exposed via Dividends here, others omitted
// for now.
type CompanyProfile struct {
	Code         string     `json:"code"`
	Name         string     `json:"name"`
	Address      string     `json:"address"`
	Sector       string     `json:"sector"`
	SubSector    string     `json:"sub_sector"`
	Industry     string     `json:"industry"`
	SubIndustry  string     `json:"sub_industry"`
	ListingBoard string     `json:"listing_board"`
	ListingDate  string     `json:"listing_date"`
	Website      string     `json:"website"`
	Email        string     `json:"email"`
	Phone        string     `json:"phone"`
	MainBusiness string     `json:"main_business"`
	Dividends    []Dividend `json:"dividends"`
}

// Dividend is one entry from the profile's Dividen array (corporate action
// data — the GetDividend endpoint itself returned 503 in the spike, but the
// same data rides along in the profile payload).
type Dividend struct {
	BookYear string  `json:"book_year"`
	Type     string  `json:"type"`
	CashPer  float64 `json:"cash_per_share"`
	Currency string  `json:"currency"`
	ExDate   string  `json:"ex_date"`
	PayDate  string  `json:"payment_date"`
}

type rawProfileResponse struct {
	Profiles []struct {
		KodeEmiten         string `json:"KodeEmiten"`
		NamaEmiten         string `json:"NamaEmiten"`
		Alamat             string `json:"Alamat"`
		Sektor             string `json:"Sektor"`
		SubSektor          string `json:"SubSektor"`
		Industri           string `json:"Industri"`
		SubIndustri        string `json:"SubIndustri"`
		PapanPencatatan    string `json:"PapanPencatatan"`
		TanggalPencatatan  string `json:"TanggalPencatatan"`
		Website            string `json:"Website"`
		Email              string `json:"Email"`
		Telepon            string `json:"Telepon"`
		KegiatanUsahaUtama string `json:"KegiatanUsahaUtama"`
	} `json:"Profiles"`
	Dividen []struct {
		Jenis                        string  `json:"Jenis"`
		TahunBuku                    string  `json:"TahunBuku"`
		CashDividenPerSaham          float64 `json:"CashDividenPerSaham"`
		CashDividenPerSahamMU        string  `json:"CashDividenPerSahamMU"`
		TanggalExRegulerDanNegosiasi string  `json:"TanggalExRegulerDanNegosiasi"`
		TanggalPembayaran            string  `json:"TanggalPembayaran"`
	} `json:"Dividen"`
}

// CompanyProfile returns the profile for an emiten code (e.g. "BBCA").
func (c *Client) CompanyProfile(ctx context.Context, code string) (*CompanyProfile, error) {
	code = normalizeCode(code)
	if code == "" {
		return nil, fmt.Errorf("empty emiten code")
	}

	u := fmt.Sprintf("%s/primary/ListedCompany/GetCompanyProfilesDetail?KodeEmiten=%s&language=en-us",
		baseURL, url.QueryEscape(code))
	key := "profile:" + code

	var raw rawProfileResponse
	if err := c.getJSON(ctx, key, u, ttlProfile, &raw); err != nil {
		return nil, err
	}
	if len(raw.Profiles) == 0 {
		return nil, fmt.Errorf("no profile found for %q", code)
	}

	p := raw.Profiles[0]
	out := &CompanyProfile{
		Code:         p.KodeEmiten,
		Name:         p.NamaEmiten,
		Address:      strings.TrimSpace(p.Alamat),
		Sector:       p.Sektor,
		SubSector:    p.SubSektor,
		Industry:     p.Industri,
		SubIndustry:  p.SubIndustri,
		ListingBoard: p.PapanPencatatan,
		ListingDate:  trimDate(p.TanggalPencatatan),
		Website:      p.Website,
		Email:        p.Email,
		Phone:        p.Telepon,
		MainBusiness: strings.TrimSpace(p.KegiatanUsahaUtama),
	}
	for _, d := range raw.Dividen {
		out.Dividends = append(out.Dividends, Dividend{
			BookYear: d.TahunBuku,
			Type:     d.Jenis,
			CashPer:  d.CashDividenPerSaham,
			Currency: d.CashDividenPerSahamMU,
			ExDate:   trimDate(d.TanggalExRegulerDanNegosiasi),
			PayDate:  trimDate(d.TanggalPembayaran),
		})
	}
	return out, nil
}
