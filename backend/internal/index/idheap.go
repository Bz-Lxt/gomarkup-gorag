package index

// Cand 是带距离的候选。Dist 越小越近。
type Cand struct {
	ID   uint64
	Dist float64
}

type minHeap []Cand

func (h minHeap) Len() int            { return len(h) }
func (h minHeap) Less(i, j int) bool  { return h[i].Dist < h[j].Dist }
func (h minHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *minHeap) Push(x interface{}) { *h = append(*h, x.(Cand)) }
func (h *minHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

type maxHeap []Cand

func (h maxHeap) Len() int            { return len(h) }
func (h maxHeap) Less(i, j int) bool  { return h[i].Dist > h[j].Dist }
func (h maxHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *maxHeap) Push(x interface{}) { *h = append(*h, x.(Cand)) }
func (h *maxHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

func SortedByDist(in []Cand) []Cand {
	out := append([]Cand(nil), in...)
	for i := 1; i < len(out); i++ {
		j := i
		for j > 0 && out[j].Dist < out[j-1].Dist {
			out[j], out[j-1] = out[j-1], out[j]
			j--
		}
	}
	return out
}
