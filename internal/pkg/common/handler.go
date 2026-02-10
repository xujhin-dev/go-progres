package handler

import (
	"mime/multipart"
	"net/http"
	"sync"
	"user_crud_jwt/internal/pkg/uploader"
	"user_crud_jwt/pkg/response"

	"github.com/gin-gonic/gin"
)

// UploadFile 上传文件 (支持批量)
// @Summary 上传文件到 OSS (支持批量)
// @Tags Common
// @Accept multipart/form-data
// @Produce json
// @Param files formData file true "Files"
// @Success 200 {object} response.Response{data=[]string} "URLs"
// @Router /upload [post]
func UploadFile(c *gin.Context) {
	// 解析 multipart form
	form, err := c.MultipartForm()
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidParam, "Invalid form data")
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidParam, "No files uploaded")
		return
	}

	if uploader.GlobalUploader == nil {
		response.Error(c, http.StatusInternalServerError, response.ErrServerInternal, "Uploader not initialized")
		return
	}

	// 结果数组，预分配大小
	urls := make([]string, len(files))
	
	// 使用 WaitGroup 和 Mutex 控制并发并保证顺序
	var wg sync.WaitGroup
	var errOnce sync.Once
	var uploadErr error

	// 限制并发数为 5，避免过多协程
	sem := make(chan struct{}, 5)

	for i, file := range files {
		wg.Add(1)
		go func(index int, f *multipart.FileHeader) {
			defer wg.Done()
			
			// 获取信号量
			sem <- struct{}{}
			defer func() { <-sem }()

			// 如果已经有错误发生，直接返回
			if uploadErr != nil {
				return
			}

			url, err := uploader.GlobalUploader.UploadFile(f)
			if err != nil {
				errOnce.Do(func() {
					uploadErr = err
				})
				return
			}

			// 直接按索引赋值，保证顺序
			urls[index] = url
		}(i, file)
	}

	wg.Wait()

	if uploadErr != nil {
		response.Error(c, http.StatusInternalServerError, response.ErrServerInternal, "Upload failed: "+uploadErr.Error())
		return
	}

	response.Success(c, urls)
}
