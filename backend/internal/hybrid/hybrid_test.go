package hybrid

import "testing"

func TestRRFMergesLists(t *testing.T) {
	v := []Ranked{{ID: 1, Rank: 1}, {ID: 2, Rank: 2}}
	k := []Ranked{{ID: 2, Rank: 1}, {ID: 3, Rank: 2}}
	out := Fuse(60, 1, 1, v, k)
	if len(out) != 3 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0].ID != 2 {
		t.Fatalf("expected id 2 first, got %d rrf=%v", out[0].ID, out[0].RRF)
	}
}

func TestRecallAtK(t *testing.T) {
	truth := []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	pred := []uint64{1, 2, 3, 4, 5, 99, 98, 97, 96, 95}
	r := RecallAtK(truth, pred, 10)
	if r < 0.49 || r > 0.51 {
		t.Fatalf("recall=%v", r)
	}
}

func TestDefaultK(t *testing.T) {
	out := Fuse(0, 0, 0, []Ranked{{ID: 1, Rank: 1}}, nil)
	if len(out) != 1 || out[0].RRF <= 0 {
		t.Fatalf("%+v", out)
	}
}
