package handlers

import (
	"cisdi-test-cms/helper"
	"cisdi-test-cms/models"
	"cisdi-test-cms/services"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService services.AuthService
	Helper      *helper.HTTPHelper
}

func NewAuthHandler(authService services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.Helper.SendBadRequest(c, "Error ", err.Error())
		return
	}

	response, err := h.authService.Register(req)
	if err != nil {
		h.Helper.SendBadRequest(c, "Error ", err.Error())
		return
	}

	h.Helper.SendSuccess(c, "Register success", response)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.Helper.SendBadRequest(c, "Error ", err.Error())
		return
	}

	response, err := h.authService.Login(req)
	if err != nil {
		h.Helper.SendUnauthorizedError(c, err.Error(), h.Helper.EmptyJsonMap())
		return
	}

	h.Helper.SendSuccess(c, "Login success", response)
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		h.Helper.SendUnauthorizedError(c, "User not found in context", h.Helper.EmptyJsonMap())
		return
	}

	user, err := h.authService.GetUserByID(userID.(uint))
	if err != nil {
		h.Helper.SendNotFoundError(c, "User not found", h.Helper.EmptyJsonMap())
		return
	}

	h.Helper.SendSuccess(c, "Profile loaded", user)
}
