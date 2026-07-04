package idx

import (
	"reflect"
	"testing"
)

func TestDecodeRemarks(t *testing.T) {
	// Real BBCA remarks: main board, no notations.
	board, notes := decodeRemarks("XDMO1UQNCNU600G111------------")
	if board != "Utama" || notes != nil {
		t.Errorf("BBCA = %q %v, want Utama with no notations", board, notes)
	}

	// Real ALTO remarks: special monitoring board with M, L, Y, X.
	board, notes = decodeRemarks("--U-4100000000D212M--L---Y---X")
	if board != "Pemantauan Khusus" {
		t.Errorf("board = %q, want Pemantauan Khusus", board)
	}
	if got := notationCodes(notes); !reflect.DeepEqual(got, []string{"M", "L", "Y", "X"}) {
		t.Errorf("notations = %v, want [M L Y X]", got)
	}
	for _, n := range notes {
		if n.Meaning == "" || n.Meaning == "unrecognized IDX notation letter" {
			t.Errorf("notation %s has no legend meaning", n.Code)
		}
	}

	// Unknown formats decode to nothing rather than guesses.
	if b, n := decodeRemarks(""); b != "" || n != nil {
		t.Errorf("empty remarks = %q %v, want none", b, n)
	}
	if b, n := decodeRemarks("short"); b != "" || n != nil {
		t.Errorf("short remarks = %q %v, want none", b, n)
	}

	// Unknown notation letters are surfaced, not dropped.
	_, notes = decodeRemarks("--U-2100000000A111------Z-----")
	if len(notes) != 1 || notes[0].Code != "Z" || notes[0].Meaning != "unrecognized IDX notation letter" {
		t.Errorf("unknown letter = %+v", notes)
	}
}
