package hybrid

import (
	"testing"

	"github.com/xavskye/gorag/internal/model"
)

func TestPlanTextSearchUsesBoth(t *testing.T) {
	p := PlanSearch(model.SearchRequest{Query: "cat"}, false)
	if !p.UseVector || !p.UseKeyword || p.CrossModal {
		t.Fatalf("%+v", p)
	}
}

func TestPlanTextToImageWithoutCLIP(t *testing.T) {
	p := PlanSearch(model.SearchRequest{Query: "cat", Modality: model.ModalityImage}, false)
	if p.UseVector || p.CrossModal || p.Note == "" {
		t.Fatalf("should degrade: %+v", p)
	}
}

func TestPlanTextToImageWithCLIP(t *testing.T) {
	p := PlanSearch(model.SearchRequest{Query: "cat", Modality: model.ModalityImage}, true)
	if !p.UseVector || !p.CrossModal {
		t.Fatalf("%+v", p)
	}
}
