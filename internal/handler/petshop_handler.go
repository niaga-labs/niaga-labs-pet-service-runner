package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/niaga-labs/niaga-labs-pet-lib-common/auth"
	"github.com/niaga-labs/niaga-labs-pet-lib-common/middleware"
	"github.com/niaga-labs/niaga-labs-pet-lib-common/response"
	"github.com/niaga-labs/niaga-labs-pet-service-runner/internal/application"
)

// PetShopHandler handles HTTP requests for pet shop operations.
type PetShopHandler struct {
	service *application.PetShopService
}

// NewPetShopHandler creates a new PetShopHandler.
func NewPetShopHandler(service *application.PetShopService) *PetShopHandler {
	return &PetShopHandler{service: service}
}

// RegisterRoutes registers all pet shop routes.
func (h *PetShopHandler) RegisterRoutes(r *gin.RouterGroup, jwtManager *auth.JWTManager) {
	authMw := middleware.AuthMiddleware(jwtManager)
	shopRole := middleware.RequireRole(auth.RoleShop)

	petshops := r.Group("/petshops")
	{
		// Read endpoints — any authenticated user
		petshops.GET("", authMw, h.ListPetShops)
		petshops.GET("/nearby", authMw, h.FindNearbyPetShops)
		petshops.GET("/mine", authMw, shopRole, h.GetMyShops)
		petshops.GET("/:id", authMw, h.GetPetShop)

		// Write endpoints — shop role only
		petshops.POST("", authMw, shopRole, h.CreatePetShop)
		petshops.PUT("/:id", authMw, shopRole, h.UpdatePetShop)
		petshops.DELETE("/:id", authMw, shopRole, h.DeletePetShop)
	}
}

// ListPetShops returns all pet shops, optionally filtered by category.
func (h *PetShopHandler) ListPetShops(c *gin.Context) {
	category := c.Query("category")

	results, err := h.service.ListPetShops(category)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, results)
}

// GetPetShop returns a single pet shop by ID.
func (h *PetShopHandler) GetPetShop(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "invalid pet shop ID")
		return
	}

	result, err := h.service.GetPetShop(id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// FindNearbyPetShops returns pet shops near the given coordinates.
func (h *PetShopHandler) FindNearbyPetShops(c *gin.Context) {
	latStr := c.Query("latitude")
	lngStr := c.Query("longitude")
	radiusStr := c.DefaultQuery("radius_km", "10")

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		response.BadRequest(c, "invalid latitude")
		return
	}

	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		response.BadRequest(c, "invalid longitude")
		return
	}

	radiusKm, err := strconv.ParseFloat(radiusStr, 64)
	if err != nil {
		response.BadRequest(c, "invalid radius")
		return
	}

	results, err := h.service.FindNearbyPetShops(lat, lng, radiusKm)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, results)
}

// CreatePetShop creates a new pet shop.
func (h *PetShopHandler) CreatePetShop(c *gin.Context) {
	ownerID, ok := middleware.GetUserID(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	var req application.CreatePetShopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.service.CreatePetShop(ownerID, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": result})
}

// UpdatePetShop updates an existing pet shop.
func (h *PetShopHandler) UpdatePetShop(c *gin.Context) {
	ownerID, ok := middleware.GetUserID(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	shopID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid pet shop ID")
		return
	}

	var req application.UpdatePetShopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.service.UpdatePetShop(ownerID, shopID, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// DeletePetShop deletes a pet shop.
func (h *PetShopHandler) DeletePetShop(c *gin.Context) {
	ownerID, ok := middleware.GetUserID(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	shopID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid pet shop ID")
		return
	}

	if err := h.service.DeletePetShop(ownerID, shopID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "pet shop deleted"})
}

// GetMyShops returns all shops owned by the current user.
func (h *PetShopHandler) GetMyShops(c *gin.Context) {
	ownerID, ok := middleware.GetUserID(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	results, err := h.service.GetMyShops(ownerID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, results)
}
