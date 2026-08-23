package invert

import "testing"

func TestBM25RanksRelevantFirst(t *testing.T) {
	idx := New()
	idx.Add(1, "向量检索与倒排索引的混合查询")
	idx.Add(2, "今天天气很好适合散步")
	idx.Add(3, "多模态向量检索系统支持以图搜图")
	hits := idx.Search("向量检索", 3)
	if len(hits) == 0 {
		t.Fatal("no hits")
	}
	if hits[0].ID != 1 && hits[0].ID != 3 {
		t.Fatalf("unexpected top=%d score=%v all=%+v", hits[0].ID, hits[0].Score, hits)
	}
}

func TestDeleteRemovesDoc(t *testing.T) {
	idx := New()
	idx.Add(1, "cat sat on the mat")
	idx.Delete(1)
	if hits := idx.Search("cat", 5); len(hits) != 0 {
		t.Fatalf("deleted still hit: %+v", hits)
	}
}
