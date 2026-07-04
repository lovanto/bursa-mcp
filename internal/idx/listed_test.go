package idx

import (
	"context"
	"testing"
)

const listedJSON = `{"recordsTotal":4,"recordsFiltered":4,"data":[
 {"KodeEmiten":"BBCA","NamaEmiten":"PT Bank Central Asia Tbk","PapanPencatatan":"Utama","TanggalPencatatan":"2000-05-31T00:00:00","Sektor":"Keuangan","SubSektor":"Bank","Industri":"Bank","SubIndustri":"Bank","Website":"www.bca.co.id"},
 {"KodeEmiten":"BBRI","NamaEmiten":"PT Bank Rakyat Indonesia (Persero) Tbk","PapanPencatatan":"Utama","TanggalPencatatan":"2003-11-10T00:00:00","Sektor":"Keuangan","SubSektor":"Bank","Industri":"Bank","SubIndustri":"Bank","Website":"www.bri.co.id"},
 {"KodeEmiten":"AADI","NamaEmiten":"PT Adaro Andalan Indonesia Tbk","PapanPencatatan":"Utama","TanggalPencatatan":"2024-12-05T00:00:00","Sektor":"Energi","SubSektor":"Minyak, Gas & Batu Bara","Industri":"Batu Bara","SubIndustri":"Produksi Batu Bara","Website":"www.adaroindonesia.com"},
 {"KodeEmiten":"TLKM","NamaEmiten":"PT Telkom Indonesia (Persero) Tbk","PapanPencatatan":"Utama","TanggalPencatatan":"1995-11-14T00:00:00","Sektor":"Infrastruktur","SubSektor":"Telekomunikasi","Industri":"Telekomunikasi","SubIndustri":"Telekomunikasi","Website":"www.telkom.co.id"}
]}`

func TestListCompaniesNoFilter(t *testing.T) {
	f := &fakeFetcher{responses: map[string]string{"GetCompanyProfiles": listedJSON}}
	c := New(f, nil)

	dir, err := c.ListCompanies(context.Background(), "", "")
	if err != nil {
		t.Fatalf("ListCompanies: %v", err)
	}
	if dir.Total != 4 || dir.Matched != 4 || dir.Returned != 4 || dir.Truncated {
		t.Fatalf("counts = %+v", *dir)
	}
	// Sorted by code: AADI, BBCA, BBRI, TLKM.
	if dir.Companies[0].Code != "AADI" || dir.Companies[3].Code != "TLKM" {
		t.Errorf("not sorted by code: %s ... %s", dir.Companies[0].Code, dir.Companies[3].Code)
	}
	if dir.Companies[1].ListingDate != "2000-05-31" {
		t.Errorf("listing date not trimmed: %q", dir.Companies[1].ListingDate)
	}
}

func TestListCompaniesQueryMatchesCodeAndName(t *testing.T) {
	f := &fakeFetcher{responses: map[string]string{"GetCompanyProfiles": listedJSON}}
	c := New(f, nil)

	// "bank" appears in two company names, not in any code.
	dir, err := c.ListCompanies(context.Background(), "bank", "")
	if err != nil {
		t.Fatalf("ListCompanies: %v", err)
	}
	if dir.Matched != 2 {
		t.Fatalf("query 'bank' matched %d, want 2", dir.Matched)
	}
	// "bbr" matches the BBRI code.
	dir, err = c.ListCompanies(context.Background(), "bbr", "")
	if err != nil {
		t.Fatalf("ListCompanies: %v", err)
	}
	if dir.Matched != 1 || dir.Companies[0].Code != "BBRI" {
		t.Errorf("query 'bbr' = %+v", dir.Companies)
	}
}

func TestListCompaniesSectorFilter(t *testing.T) {
	f := &fakeFetcher{responses: map[string]string{"GetCompanyProfiles": listedJSON}}
	c := New(f, nil)

	dir, err := c.ListCompanies(context.Background(), "", "energi")
	if err != nil {
		t.Fatalf("ListCompanies: %v", err)
	}
	if dir.Matched != 1 || dir.Companies[0].Code != "AADI" {
		t.Errorf("sector 'energi' = %+v", dir.Companies)
	}
}
