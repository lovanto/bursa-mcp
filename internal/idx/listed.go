package idx

import (
	"context"
	"sort"
	"strings"
	"time"
)

// ttlListed caches the listed-company directory. The list only changes on new
// listings/delistings (a handful per year), so a long TTL is safe.
const ttlListed = 7 * 24 * time.Hour

// maxListResults caps how many companies a single list_companies call returns,
// so an empty query doesn't dump all ~957 rows into the model's context.
const maxListResults = 100

// ListedCompany is one row of the IDX listed-company directory
// (ListedCompany/GetCompanyProfiles), the cleaned subset useful for ticker
// discovery.
type ListedCompany struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	ListingBoard string `json:"listing_board"`
	ListingDate  string `json:"listing_date"`
	Sector       string `json:"sector"`
	SubSector    string `json:"sub_sector"`
	Industry     string `json:"industry"`
	SubIndustry  string `json:"sub_industry"`
	Website      string `json:"website"`
}

// CompanyDirectory is the result of a directory search: how many companies
// matched, whether the returned slice was truncated, and the matches
// themselves.
type CompanyDirectory struct {
	Total     int             `json:"total"`     // total listed companies in the directory
	Matched   int             `json:"matched"`   // how many matched the query/sector filters
	Returned  int             `json:"returned"`  // how many are in Companies (<= maxListResults)
	Truncated bool            `json:"truncated"` // true if Matched > Returned
	Companies []ListedCompany `json:"companies"`
}

type rawListedCompanies struct {
	RecordsTotal int `json:"recordsTotal"`
	Data         []struct {
		KodeEmiten        string `json:"KodeEmiten"`
		NamaEmiten        string `json:"NamaEmiten"`
		PapanPencatatan   string `json:"PapanPencatatan"`
		TanggalPencatatan string `json:"TanggalPencatatan"`
		Sektor            string `json:"Sektor"`
		SubSektor         string `json:"SubSektor"`
		Industri          string `json:"Industri"`
		SubIndustri       string `json:"SubIndustri"`
		Website           string `json:"Website"`
	} `json:"data"`
}

// ListCompanies returns the IDX listed-company directory, optionally filtered by
// a case-insensitive substring query (matched against ticker code and company
// name) and/or a sector substring. Results are sorted by ticker and capped at
// maxListResults; Truncated reports whether more matched than were returned.
func (c *Client) ListCompanies(ctx context.Context, query, sector string) (*CompanyDirectory, error) {
	u := baseURL + "/primary/ListedCompany/GetCompanyProfiles?start=0&length=9999"
	var raw rawListedCompanies
	if err := c.getJSON(ctx, "listedcompanies:all", u, ttlListed, &raw); err != nil {
		return nil, err
	}

	q := strings.ToLower(strings.TrimSpace(query))
	sec := strings.ToLower(strings.TrimSpace(sector))

	matches := make([]ListedCompany, 0, len(raw.Data))
	for _, r := range raw.Data {
		if q != "" && !strings.Contains(strings.ToLower(r.KodeEmiten), q) &&
			!strings.Contains(strings.ToLower(r.NamaEmiten), q) {
			continue
		}
		if sec != "" && !strings.Contains(strings.ToLower(r.Sektor), sec) &&
			!strings.Contains(strings.ToLower(r.SubSektor), sec) {
			continue
		}
		matches = append(matches, ListedCompany{
			Code:         r.KodeEmiten,
			Name:         cleanText(r.NamaEmiten),
			ListingBoard: r.PapanPencatatan,
			ListingDate:  trimDate(r.TanggalPencatatan),
			Sector:       r.Sektor,
			SubSector:    r.SubSektor,
			Industry:     r.Industri,
			SubIndustry:  r.SubIndustri,
			Website:      r.Website,
		})
	}

	sort.SliceStable(matches, func(i, j int) bool { return matches[i].Code < matches[j].Code })

	dir := &CompanyDirectory{
		Total:   raw.RecordsTotal,
		Matched: len(matches),
	}
	if len(matches) > maxListResults {
		dir.Truncated = true
		matches = matches[:maxListResults]
	}
	dir.Returned = len(matches)
	dir.Companies = matches
	return dir, nil
}
