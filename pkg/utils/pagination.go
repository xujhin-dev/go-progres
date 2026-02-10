package utils

// Pagination 分页请求参数
type Pagination struct {
	Page  int `json:"page" form:"page"`
	Limit int `json:"limit" form:"limit"`
}

// PageResult 分页响应结果
type PageResult struct {
	List  interface{} `json:"list"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Limit int         `json:"limit"`
}

// GetPageOffset 计算分页偏移量
func (p *Pagination) GetPageOffset() (int, int) {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.Limit <= 0 {
		p.Limit = 10
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
	return (p.Page - 1) * p.Limit, p.Limit
}
