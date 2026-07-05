package idx

// Decoding of the 30-character Remarks string carried by the trading feeds
// (GetTradingInfoSS and GetStockSummary), reverse-engineered by correlating
// all ~959 stocks against the listed-company directory:
//
//   - Position 4 is the listing board; it matches PapanPencatatan for >99% of
//     stocks (the stragglers are board moves the slower-moving directory cache
//     hasn't caught up with).
//   - Positions 18-29 are fixed slots for IDX special notations (notasi
//     khusus); each slot holds one letter or '-'. Slot letters observed live
//     match the official legend (e.g. ALTO carries M, L, Y and X while listed
//     on Pemantauan Khusus).
//   - Positions 0-3 and 5-17 encode other attributes (undetermined); they are
//     left undecoded rather than guessed.
const (
	remarksLen        = 30
	remarksBoardPos   = 4
	remarksNotesStart = 18
)

// remarksBoards maps the position-4 digit to the listing board.
var remarksBoards = map[byte]string{
	'1': "Utama",
	'2': "Pengembangan",
	'3': "Akselerasi",
	'4': "Pemantauan Khusus",
	'5': "Ekonomi Baru",
}

// notationMeanings is the official IDX special-notation legend.
var notationMeanings = map[byte]string{
	'B': "bankruptcy petition filed against the issuer",
	'M': "suspension-of-payments (PKPU) petition filed",
	'E': "latest financial statements show negative equity",
	'A': "adverse opinion from the public accountant",
	'D': "disclaimer of opinion from the public accountant",
	'L': "has not filed its financial statements on time",
	'S': "latest financial statements show no operating revenue",
	'C': "material legal proceedings against the issuer",
	'Q': "regulatory restrictions on the issuer's business",
	'Y': "has not held its annual general meeting on time",
	'F': "minor administrative sanction from OJK",
	'G': "moderate administrative sanction from OJK",
	'V': "severe administrative sanction from OJK",
	'N': "does not meet the free-float requirement",
	'I': "does not meet the minimum-shareholder requirement",
	'K': "multiple-voting shares (saham hak suara multipel)",
	'X': "listed on the special monitoring board (pemantauan khusus)",
}

// SpecialNotation is one decoded IDX special-notation letter.
type SpecialNotation struct {
	Code    string `json:"code"`
	Meaning string `json:"meaning"`
}

// decodeRemarks extracts the listing board and special notations from a
// Remarks string. Unknown formats yield ("", nil) rather than guesses.
func decodeRemarks(remarks string) (board string, notations []SpecialNotation) {
	if len(remarks) != remarksLen {
		return "", nil
	}
	board = remarksBoards[remarks[remarksBoardPos]]
	for i := remarksNotesStart; i < remarksLen; i++ {
		ch := remarks[i]
		if ch == '-' {
			continue
		}
		meaning, ok := notationMeanings[ch]
		if !ok {
			meaning = "unrecognized IDX notation letter"
		}
		notations = append(notations, SpecialNotation{Code: string(ch), Meaning: meaning})
	}
	return board, notations
}

// notationCodes reduces decoded notations to their letters.
func notationCodes(notations []SpecialNotation) []string {
	if len(notations) == 0 {
		return nil
	}
	codes := make([]string, len(notations))
	for i, n := range notations {
		codes[i] = n.Code
	}
	return codes
}
