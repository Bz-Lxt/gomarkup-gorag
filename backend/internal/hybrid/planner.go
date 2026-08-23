package hybrid

import "github.com/xavskye/gorag/internal/model"

// Plan 描述一次混合查询将启用的通道。
type Plan struct {
	UseVector  bool
	UseKeyword bool
	CrossModal bool
	Note       string
}

func PlanSearch(req model.SearchRequest, clipEnabled bool) Plan {
	p := Plan{UseKeyword: req.Query != ""}
	if req.Modality == model.ModalityImage && req.Query != "" {
		if clipEnabled {
			p.UseVector = true
			p.CrossModal = true
		} else {
			p.UseVector = false
			p.Note = "以文搜图未启用 CLIP，仅走 caption/tag 标量通道"
		}
		return p
	}
	p.UseVector = true
	return p
}
