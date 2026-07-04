package idx

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/lovanto/idx-mcp/internal/xbrl"
)

// FinancialReport is the parsed XBRL financial report for one period.
type FinancialReport struct {
	Code   string       `json:"code"`
	Year   string       `json:"year"`
	Period string       `json:"period"` // TW1..TW3 (quarterly) or Audit (annual)
	Report *xbrl.Report `json:"report"`
}

// rawFinancialResponse maps GetFinancialReport. reportType=rdf yields the XBRL
// bundle whose Attachments include instance.zip (verified in the spike).
type rawFinancialResponse struct {
	Results []struct {
		KodeEmiten   string `json:"KodeEmiten"`
		ReportYear   string `json:"Report_Year"`
		ReportPeriod string `json:"Report_Period"`
		Attachments  []struct {
			FileName string `json:"File_Name"`
			FilePath string `json:"File_Path"`
		} `json:"Attachments"`
	} `json:"Results"`
}

// FinancialReport fetches and parses the XBRL financial report for `code` in
// the given `year` and `period`. period accepts "tw1".."tw3" for quarterly or
// "audit" for the audited annual report; it is normalised to the API's casing.
func (c *Client) FinancialReport(ctx context.Context, code, year, period string) (*FinancialReport, error) {
	code = normalizeCode(code)
	if code == "" {
		return nil, fmt.Errorf("empty emiten code")
	}
	period = normalizePeriod(period)
	if year == "" {
		return nil, fmt.Errorf("empty year")
	}

	// 1. Resolve the instance.zip download path via the report listing.
	listURL := fmt.Sprintf(
		"%s/primary/ListedCompany/GetFinancialReport?indexFrom=1&pageSize=12&year=%s&reportType=rdf&EmitenType=s&periode=%s&kodeEmiten=%s&SortColumn=KodeEmiten&SortOrder=asc",
		baseURL, url.QueryEscape(year), url.QueryEscape(strings.ToLower(period)), url.QueryEscape(code))
	listKey := fmt.Sprintf("finlist:%s:%s:%s", code, year, period)

	var raw rawFinancialResponse
	if err := c.getJSON(ctx, listKey, listURL, ttlFinList, &raw); err != nil {
		return nil, err
	}

	instancePath := findInstanceZip(raw, code)
	if instancePath == "" {
		return nil, fmt.Errorf("no instance.zip for %s %s %s (report may not be published yet)", code, year, period)
	}

	// 2. Download the (immutable) zip, cached forever.
	zipURL := buildStaticURL(instancePath)
	zipKey := fmt.Sprintf("finzip:%s:%s:%s", code, year, period)
	zipBytes, err := c.getRaw(ctx, zipKey, zipURL, ttlFinData)
	if err != nil {
		return nil, fmt.Errorf("download instance.zip: %w", err)
	}

	// 3. Extract instance.xbrl and parse it.
	xbrlBytes, err := extractInstanceXBRL(zipBytes)
	if err != nil {
		return nil, err
	}
	report, err := xbrl.ParseBytes(xbrlBytes)
	if err != nil {
		return nil, fmt.Errorf("parse xbrl: %w", err)
	}

	return &FinancialReport{
		Code:   code,
		Year:   year,
		Period: period,
		Report: report,
	}, nil
}

// findInstanceZip returns the File_Path of the instance.zip attachment for the
// matching emiten, or "" if absent.
func findInstanceZip(raw rawFinancialResponse, code string) string {
	for _, res := range raw.Results {
		if !strings.EqualFold(res.KodeEmiten, code) {
			continue
		}
		for _, a := range res.Attachments {
			if strings.EqualFold(a.FileName, "instance.zip") {
				return a.FilePath
			}
		}
	}
	return ""
}

// buildStaticURL turns an IDX File_Path (which contains spaces and a stray
// double slash) into a properly encoded absolute URL.
func buildStaticURL(filePath string) string {
	u := &url.URL{Scheme: "https", Host: "www.idx.co.id", Path: filePath}
	return u.String()
}

// extractInstanceXBRL reads instance.xbrl out of the zip archive bytes.
func extractInstanceXBRL(zipBytes []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	for _, f := range zr.File {
		if strings.EqualFold(f.Name, "instance.xbrl") {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("open instance.xbrl: %w", err)
			}
			defer rc.Close()
			b, err := io.ReadAll(rc)
			if err != nil {
				return nil, fmt.Errorf("read instance.xbrl: %w", err)
			}
			return b, nil
		}
	}
	return nil, fmt.Errorf("instance.xbrl not found in archive")
}

// normalizePeriod maps user input to IDX period tokens. "audit"/"annual"/"tw4"
// all mean the audited full-year report.
func normalizePeriod(p string) string {
	p = strings.ToLower(strings.TrimSpace(p))
	switch p {
	case "", "tw1", "q1", "1":
		return "TW1"
	case "tw2", "q2", "2", "semester1", "s1":
		return "TW2"
	case "tw3", "q3", "3":
		return "TW3"
	case "tw4", "q4", "4", "audit", "annual", "audited":
		return "Audit"
	default:
		return strings.ToUpper(p)
	}
}
