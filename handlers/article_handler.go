package handlers

import (
	"cisdi-test-cms/helper"
	"cisdi-test-cms/models"
	"cisdi-test-cms/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ArticleHandler struct {
	articleService services.ArticleService
	Helper         *helper.HTTPHelper
}

func NewArticleHandler(articleService services.ArticleService) *ArticleHandler {
	return &ArticleHandler{articleService: articleService}
}

func (h *ArticleHandler) CreateArticle(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req models.CreateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.Helper.SendBadRequest(c, "Invalid request data", err.Error())
		return
	}

	article, err := h.articleService.CreateArticle(req, userID.(uint))
	if err != nil {
		h.Helper.SendBadRequest(c, "Error :", h.Helper.EmptyJsonMap())
		return
	}

	h.Helper.SendSuccess(c, "Article created successfully", article)
}

func (h *ArticleHandler) GetArticles(c *gin.Context) {
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	// Ambil parameter query
	status := c.DefaultQuery("status", "published")
	authorIDStr := c.Query("author_id")
	tagIDStr := c.Query("tag_id")
	sortBy := c.DefaultQuery("sort_by", "published_at")
	sortOrder := c.DefaultQuery("sort_order", "desc")
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	// Konversi parameter
	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)
	authorID := uint(0)
	if authorIDStr != "" {
		aid, err := strconv.ParseUint(authorIDStr, 10, 32)
		if err == nil {
			authorID = uint(aid)
		}
	}
	tagID := uint(0)
	if tagIDStr != "" {
		tid, err := strconv.ParseUint(tagIDStr, 10, 32)
		if err == nil {
			tagID = uint(tid)
		}
	}

	// Siapkan params
	params := models.ArticleListParams{
		Status:    status,
		AuthorID:  authorID,
		TagID:     tagID,
		Page:      page,
		Limit:     limit,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}

	// Role-based access: jika bukan admin/editor, hanya bisa akses milik sendiri atau yang published
	isAdmin := role == "admin" || role == "editor"
	if !isAdmin {
		// Jika status bukan published, hanya boleh akses milik sendiri
		if status != "published" {
			params.AuthorID = userID.(uint)
		}
	}

	articles, total, err := h.articleService.GetArticles(params, userID.(uint), false)
	if err != nil {
		h.Helper.SendBadRequest(c, "Error : ", err.Error())
		return
	}

	data := map[string]interface{}{
		"articles": articles,
		"total":    total,
		"page":     params.Page,
		"limit":    params.Limit,
	}
	h.Helper.SendSuccess(c, "Success", data)
}

func (h *ArticleHandler) GetPublicArticles(c *gin.Context) {
	var params models.ArticleListParams
	if err := c.ShouldBindQuery(&params); err != nil {
		h.Helper.SendBadRequest(c, "Error : ", err.Error())
		return
	}

	// Set defaults
	if params.Page == 0 {
		params.Page = 1
	}
	if params.Limit == 0 {
		params.Limit = 10
	}

	articles, total, err := h.articleService.GetArticles(params, 0, true)
	if err != nil {
		h.Helper.SendBadRequest(c, "Error : ", err.Error())
		return
	}

	data := map[string]interface{}{
		"articles": articles,
		"total":    total,
		"page":     params.Page,
		"limit":    params.Limit,
	}
	h.Helper.SendSuccess(c, "Success", data)
}

func (h *ArticleHandler) GetArticle(c *gin.Context) {
	userID, _ := c.Get("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.Helper.SendBadRequest(c, "Invalid article ID", h.Helper.EmptyJsonMap())
		return
	}

	article, err := h.articleService.GetArticle(uint(id), userID.(uint), false)
	if err != nil {
		h.Helper.SendNotFoundError(c, err.Error(), h.Helper.EmptyJsonMap())
		return
	}

	h.Helper.SendSuccess(c, "Success", article)
}

func (h *ArticleHandler) GetPublicArticle(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.Helper.SendBadRequest(c, "Invalid article ID", h.Helper.EmptyJsonMap())
		return
	}

	article, err := h.articleService.GetArticle(uint(id), 0, true)
	if err != nil {
		h.Helper.SendNotFoundError(c, err.Error(), h.Helper.EmptyJsonMap())
		return
	}

	h.Helper.SendSuccess(c, "Success", article)
}

func (h *ArticleHandler) DeleteArticle(c *gin.Context) {
	userID, _ := c.Get("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.Helper.SendBadRequest(c, "Invalid article ID", h.Helper.EmptyJsonMap())
		return
	}

	if err := h.articleService.DeleteArticle(uint(id), userID.(uint)); err != nil {
		h.Helper.SendBadRequest(c, "Error : ", err.Error())
		return
	}

	h.Helper.SendSuccess(c, "Article deleted successfully", h.Helper.EmptyJsonMap())
}

func (h *ArticleHandler) CreateArticleVersion(c *gin.Context) {
	userID, _ := c.Get("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.Helper.SendBadRequest(c, "Invalid article ID", h.Helper.EmptyJsonMap())
		return
	}

	var req models.CreateArticleVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.Helper.SendBadRequest(c, "Invalid request data ", err.Error())
		return
	}

	version, err := h.articleService.CreateArticleVersion(uint(id), req, userID.(uint))
	if err != nil {
		h.Helper.SendBadRequest(c, "Error : ", err.Error())
		return
	}

	h.Helper.SendSuccess(c, "Version created successfully", version)
}

func (h *ArticleHandler) UpdateVersionStatus(c *gin.Context) {
	userID, _ := c.Get("user_id")
	articleID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.Helper.SendBadRequest(c, "Invalid article ID", h.Helper.EmptyJsonMap())
		return
	}

	versionID, err := strconv.ParseUint(c.Param("version_id"), 10, 32)
	if err != nil {
		h.Helper.SendBadRequest(c, "Invalid version ID", h.Helper.EmptyJsonMap())
		return
	}

	var req models.UpdateVersionStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.Helper.SendBadRequest(c, "Error ", err.Error())
		return
	}

	if err := h.articleService.UpdateVersionStatus(uint(articleID), uint(versionID), req.Status, userID.(uint)); err != nil {
		h.Helper.SendBadRequest(c, err.Error(), h.Helper.EmptyJsonMap())
		return
	}

	h.Helper.SendResponse(h.Helper.SetResponse(c, "success", "Version status updated successfully", h.Helper.EmptyJsonMap(), http.StatusOK, "success"))
}

func (h *ArticleHandler) GetArticleVersions(c *gin.Context) {
	userID, _ := c.Get("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.Helper.SendBadRequest(c, "Invalid article ID", h.Helper.EmptyJsonMap())
		return
	}

	versions, err := h.articleService.GetArticleVersions(uint(id), userID.(uint))
	if err != nil {
		h.Helper.SendBadRequest(c, "Error : ", err.Error())
		return
	}

	h.Helper.SendSuccess(c, "Success", versions)
}

func (h *ArticleHandler) GetArticleVersion(c *gin.Context) {
	userID, _ := c.Get("user_id")
	articleID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.Helper.SendBadRequest(c, "Invalid article ID", h.Helper.EmptyJsonMap())
		return
	}

	versionID, err := strconv.ParseUint(c.Param("version_id"), 10, 32)
	if err != nil {
		h.Helper.SendBadRequest(c, "Invalid version ID", h.Helper.EmptyJsonMap())
		return
	}

	version, err := h.articleService.GetArticleVersion(uint(articleID), uint(versionID), userID.(uint))
	if err != nil {
		h.Helper.SendNotFoundError(c, err.Error(), h.Helper.EmptyJsonMap())
		return
	}

	h.Helper.SendSuccess(c, "Success", version)
}
