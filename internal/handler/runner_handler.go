package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/niaga-labs/niaga-labs-pet-lib-common/auth"
	"github.com/niaga-labs/niaga-labs-pet-lib-common/middleware"
	"github.com/niaga-labs/niaga-labs-pet-lib-common/response"
	"github.com/niaga-labs/niaga-labs-pet-service-runner/internal/application"
)

// RunnerHandler handles HTTP requests for runner operations.
type RunnerHandler struct {
	service *application.RunnerService
}

// NewRunnerHandler creates a new RunnerHandler.
func NewRunnerHandler(service *application.RunnerService) *RunnerHandler {
	return &RunnerHandler{service: service}
}

// RegisterRoutes registers all runner-related routes.
func (h *RunnerHandler) RegisterRoutes(r *gin.RouterGroup, jwtManager *auth.JWTManager) {
	authMw := middleware.AuthMiddleware(jwtManager)
	runnerRole := middleware.RequireRole(auth.RoleRunner)

	runners := r.Group("/runners")
	{
		// Runner-only routes
		runners.POST("", authMw, runnerRole, h.Register)
		runners.GET("/me", authMw, runnerRole, h.GetMyProfile)
		runners.POST("/me/online", authMw, runnerRole, h.GoOnline)
		runners.POST("/me/offline", authMw, runnerRole, h.GoOffline)
		runners.POST("/me/location", authMw, runnerRole, h.UpdateLocation)
		runners.POST("/me/crates", authMw, runnerRole, h.AddCrateSpec)

		// Any authenticated role can search nearby runners
		runners.GET("/nearby", authMw, h.FindNearbyRunners)
	}
}

// Register handles runner registration.
func (h *RunnerHandler) Register(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.BadRequest(c, "invalid user context")
		return
	}

	var req application.RegisterRunnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.service.RegisterRunner(c.Request.Context(), userID, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, result)
}

// GetMyProfile handles fetching the current runner's profile.
func (h *RunnerHandler) GetMyProfile(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.BadRequest(c, "invalid user context")
		return
	}

	result, err := h.service.GetMyProfile(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// GoOnline handles setting the runner to active status.
func (h *RunnerHandler) GoOnline(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.BadRequest(c, "invalid user context")
		return
	}

	var req application.GoOnlineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.service.GoOnline(c.Request.Context(), userID, req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "runner is now online"})
}

// GoOffline handles setting the runner to inactive status.
func (h *RunnerHandler) GoOffline(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.BadRequest(c, "invalid user context")
		return
	}

	if err := h.service.GoOffline(c.Request.Context(), userID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "runner is now offline"})
}

// UpdateLocation handles GPS location updates from the runner.
func (h *RunnerHandler) UpdateLocation(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.BadRequest(c, "invalid user context")
		return
	}

	var req application.UpdateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.service.UpdateLocation(c.Request.Context(), userID, req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "location updated"})
}

// AddCrateSpec handles adding a crate specification to the runner's profile.
func (h *RunnerHandler) AddCrateSpec(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.BadRequest(c, "invalid user context")
		return
	}

	var req application.AddCrateSpecRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.service.AddCrateSpec(c.Request.Context(), userID, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, result)
}

// FindNearbyRunners handles searching for active runners near a location.
func (h *RunnerHandler) FindNearbyRunners(c *gin.Context) {
	latStr := c.Query("latitude")
	lngStr := c.Query("longitude")
	radiusStr := c.DefaultQuery("radius_km", "5")

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

	results, err := h.service.FindNearbyRunners(c.Request.Context(), lat, lng, radiusKm)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, results)
}
