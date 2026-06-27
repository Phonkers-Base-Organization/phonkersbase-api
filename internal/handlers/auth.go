package handlers

import (
	"net/http"
	"time"

	"github.com/PhonkersBase/base-api2/internal/middlewares"
	"github.com/PhonkersBase/base-api2/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

func (h *Handler) Login(c *gin.Context) {
	var body struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	user, hash, err := h.users.GetByUsername(c.Request.Context(), body.Username)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(body.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Invalid credentials"})
		return
	}

	claims := &middlewares.Claims{
		Role:     user.Role,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		// JWT signing is a local operation — context cancellation is irrelevant here.
		log.Error().Err(err).Msg("failed to sign token")
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "token": signed})
}

func (h *Handler) Logout(c *gin.Context) {
	c.Status(http.StatusOK)
}

func (h *Handler) GetMe(c *gin.Context) {
	userID, _ := c.Get("userID")
	username, _ := c.Get("username")
	role, _ := c.Get("userRole")
	c.JSON(http.StatusOK, gin.H{"user": gin.H{
		"id":       userID,
		"username": username,
		"role":     role,
	}})
}

func (h *Handler) GetUsers(c *gin.Context) {
	users, err := h.users.GetAll(c.Request.Context())
	if err != nil {
		internalErr(c, err, "failed to get users")
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (h *Handler) RegisterUser(c *gin.Context) {
	var body struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}

	role := body.Role
	if role == "" {
		role = "EDITOR"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), 12)
	if err != nil {
		log.Error().Err(err).Msg("failed to hash password")
		c.Status(http.StatusInternalServerError)
		return
	}

	user, err := h.users.Create(c.Request.Context(), body.Username, string(hash), role)
	if err != nil {
		// Check for duplicate username (unique constraint)
		log.Error().Err(err).Msg("failed to create user")
		c.JSON(http.StatusBadRequest, gin.H{"message": "username already exists"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"user": user})
}

func (h *Handler) DeleteUser(c *gin.Context) {
	targetID := c.Param("id")
	currentUserID, _ := c.Get("userID")

	if currentUserID.(string) == targetID {
		c.JSON(http.StatusBadRequest, gin.H{"message": "cannot delete yourself"})
		return
	}

	target, err := h.users.GetByID(c.Request.Context(), targetID)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"message": "user not found"})
			return
		}
		internalErr(c, err, "failed to get user")
		return
	}

	if target.Role == "ADMIN" {
		c.JSON(http.StatusForbidden, gin.H{"message": "cannot delete another admin"})
		return
	}

	if err := h.users.Delete(c.Request.Context(), targetID); err != nil {
		internalErr(c, err, "failed to delete user")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) UpdateUserRole(c *gin.Context) {
	targetID := c.Param("id")
	var body struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}

	currentRole, _ := c.Get("userRole")
	if body.Role == "ADMIN" && currentRole.(string) != "ADMIN" {
		c.JSON(http.StatusForbidden, gin.H{"message": "only admins can promote to ADMIN"})
		return
	}

	if err := h.users.UpdateRole(c.Request.Context(), targetID, body.Role); err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"message": "user not found"})
			return
		}
		internalErr(c, err, "failed to update user role")
		return
	}
	c.Status(http.StatusNoContent)
}
