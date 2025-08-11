package handlers

import (
	"cisdi-test-cms/helper"
	"cisdi-test-cms/models"
	"cisdi-test-cms/services"
	"strconv"

	"github.com/gin-gonic/gin"
)

type TagHandler struct {
	tagService services.TagService
	Helper     *helper.HTTPHelper
}

func NewTagHandler(tagService services.TagService) *TagHandler {
	return &TagHandler{tagService: tagService}
}

func (h *TagHandler) CreateTag(c *gin.Context) {
	role, _ := c.Get("role")
	if role != "admin" {
		h.Helper.SendUnauthorizedError(c, "Only admin can create tag", h.Helper.EmptyJsonMap())
		return
	}
	var req models.CreateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.Helper.SendBadRequest(c, "Error ", err.Error())
		return
	}

	tag, err := h.tagService.CreateTag(req)
	if err != nil {
		h.Helper.SendBadRequest(c, "Error ", err.Error())
		return
	}

	h.Helper.SendSuccess(c, "Tag created successfully", tag)
}

func (h *TagHandler) GetTags(c *gin.Context) {
	tags, err := h.tagService.GetTags()
	if err != nil {
		h.Helper.SendBadRequest(c, "Error ", err.Error())
		return
	}

	h.Helper.SendSuccess(c, "Success", tags)
}

func (h *TagHandler) GetTag(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.Helper.SendBadRequest(c, "Invalid tag ID", h.Helper.EmptyJsonMap())
		return
	}

	tag, err := h.tagService.GetTag(uint(id))
	if err != nil {
		h.Helper.SendNotFoundError(c, err.Error(), h.Helper.EmptyJsonMap())
		return
	}

	h.Helper.SendSuccess(c, "Success", tag)
}
