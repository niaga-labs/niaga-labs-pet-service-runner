package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/niaga-labs/niaga-labs-pet-lib-common/domain"
	"github.com/niaga-labs/niaga-labs-pet-lib-common/kafka"
	"github.com/niaga-labs/niaga-labs-pet-lib-proto/dto"
	"github.com/niaga-labs/niaga-labs-pet-lib-proto/events"
	runnerDomain "github.com/niaga-labs/niaga-labs-pet-service-runner/internal/domain/runner"
)

// RegisterRunnerRequest holds data for registering a new runner.
type RegisterRunnerRequest struct {
	FullName       string `json:"full_name" binding:"required"`
	Phone          string `json:"phone" binding:"required"`
	VehicleType    string `json:"vehicle_type" binding:"required"`
	VehiclePlate   string `json:"vehicle_plate" binding:"required"`
	VehicleModel   string `json:"vehicle_model" binding:"required"`
	VehicleYear    int    `json:"vehicle_year" binding:"required"`
	AirConditioned bool   `json:"air_conditioned"`
}

// AddCrateSpecRequest holds data for adding a crate specification.
type AddCrateSpecRequest struct {
	Size                  string   `json:"size" binding:"required"`
	PetTypes              []string `json:"pet_types" binding:"required"`
	MaxWeightKg           float64  `json:"max_weight_kg" binding:"required"`
	WidthCm               float64  `json:"width_cm"`
	HeightCm              float64  `json:"height_cm"`
	DepthCm               float64  `json:"depth_cm"`
	Ventilated            bool     `json:"ventilated"`
	TemperatureControlled bool     `json:"temperature_controlled"`
}

// GoOnlineRequest holds coordinates for going online.
type GoOnlineRequest struct {
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
}

// UpdateLocationRequest holds GPS telemetry data.
type UpdateLocationRequest struct {
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
	Speed     float64 `json:"speed_kmh"`
	Heading   float64 `json:"heading_degrees"`
}

// RunnerService implements the application use cases for the runner domain.
type RunnerService struct {
	runnerRepo    runnerDomain.RunnerRepository
	crateSpecRepo runnerDomain.CrateSpecRepository
	qualityPolicy runnerDomain.QualityPolicy
	producer      *kafka.Producer
	logger        *zap.Logger
}

// NewRunnerService creates a new RunnerService.
func NewRunnerService(
	runnerRepo runnerDomain.RunnerRepository,
	crateSpecRepo runnerDomain.CrateSpecRepository,
	qualityPolicy runnerDomain.QualityPolicy,
	producer *kafka.Producer,
	logger *zap.Logger,
) *RunnerService {
	return &RunnerService{
		runnerRepo:    runnerRepo,
		crateSpecRepo: crateSpecRepo,
		qualityPolicy: qualityPolicy,
		producer:      producer,
		logger:        logger,
	}
}

// RegisterRunner registers a new runner profile for the given user.
func (s *RunnerService) RegisterRunner(ctx context.Context, userID uuid.UUID, req RegisterRunnerRequest) (*dto.RunnerDTO, error) {
	// Check if runner already exists for this user.
	existing, _ := s.runnerRepo.FindByUserID(ctx, userID)
	if existing != nil {
		return nil, domain.NewAlreadyExistsError("runner", "user_id", userID.String())
	}

	r := runnerDomain.NewRunner(
		userID,
		req.FullName,
		req.Phone,
		runnerDomain.VehicleType(req.VehicleType),
		req.VehiclePlate,
		req.VehicleModel,
		req.VehicleYear,
		req.AirConditioned,
	)

	if err := s.runnerRepo.Save(ctx, r); err != nil {
		s.logger.Error("failed to save runner", zap.Error(err))
		return nil, fmt.Errorf("failed to register runner: %w", err)
	}

	result := s.toRunnerDTO(r, nil)
	return &result, nil
}

// GetMyProfile returns the runner profile for the given user.
func (s *RunnerService) GetMyProfile(ctx context.Context, userID uuid.UUID) (*dto.RunnerDTO, error) {
	r, err := s.runnerRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, domain.NewNotFoundError("runner", userID.String())
	}

	crateSpecs, err := s.crateSpecRepo.FindByRunnerID(ctx, r.ID())
	if err != nil {
		s.logger.Warn("failed to load crate specs", zap.Error(err))
		crateSpecs = nil
	}

	result := s.toRunnerDTO(r, crateSpecs)
	return &result, nil
}

// GoOnline sets the runner's session to active and publishes a RunnerOnlineEvent.
func (s *RunnerService) GoOnline(ctx context.Context, userID uuid.UUID, req GoOnlineRequest) error {
	r, err := s.runnerRepo.FindByUserID(ctx, userID)
	if err != nil {
		return domain.NewNotFoundError("runner", userID.String())
	}

	if err := r.GoOnline(req.Latitude, req.Longitude); err != nil {
		return err
	}

	if err := s.runnerRepo.Update(ctx, r); err != nil {
		return fmt.Errorf("failed to update runner: %w", err)
	}

	// Publish RunnerOnlineEvent.
	evt := events.RunnerOnlineEvent{
		RunnerID:   r.ID(),
		Latitude:   req.Latitude,
		Longitude:  req.Longitude,
		OccurredAt: time.Now().UTC(),
	}
	cloudEvt, err := kafka.NewCloudEvent("service-runner", events.RunnerOnline, evt)
	if err != nil {
		s.logger.Error("failed to create cloud event", zap.Error(err))
	} else if err := s.producer.PublishEvent(ctx, events.TopicRunnerEvents, cloudEvt); err != nil {
		s.logger.Error("failed to publish runner online event", zap.Error(err))
	}

	s.logger.Info("runner went online",
		zap.String("runner_id", r.ID().String()),
		zap.Float64("lat", req.Latitude),
		zap.Float64("lng", req.Longitude),
	)
	return nil
}

// GoOffline sets the runner's session to inactive and publishes a RunnerOfflineEvent.
func (s *RunnerService) GoOffline(ctx context.Context, userID uuid.UUID) error {
	r, err := s.runnerRepo.FindByUserID(ctx, userID)
	if err != nil {
		return domain.NewNotFoundError("runner", userID.String())
	}

	r.GoOffline()

	if err := s.runnerRepo.Update(ctx, r); err != nil {
		return fmt.Errorf("failed to update runner: %w", err)
	}

	// Publish RunnerOfflineEvent.
	evt := events.RunnerOfflineEvent{
		RunnerID:   r.ID(),
		OccurredAt: time.Now().UTC(),
	}
	cloudEvt, err := kafka.NewCloudEvent("service-runner", events.RunnerOffline, evt)
	if err != nil {
		s.logger.Error("failed to create cloud event", zap.Error(err))
	} else if err := s.producer.PublishEvent(ctx, events.TopicRunnerEvents, cloudEvt); err != nil {
		s.logger.Error("failed to publish runner offline event", zap.Error(err))
	}

	s.logger.Info("runner went offline", zap.String("runner_id", r.ID().String()))
	return nil
}

// UpdateLocation updates the runner's GPS position and publishes a RunnerLocationUpdateEvent.
func (s *RunnerService) UpdateLocation(ctx context.Context, userID uuid.UUID, req UpdateLocationRequest) error {
	r, err := s.runnerRepo.FindByUserID(ctx, userID)
	if err != nil {
		return domain.NewNotFoundError("runner", userID.String())
	}

	if !r.IsActive() {
		return domain.NewValidationError("runner must be online to update location")
	}

	if err := s.runnerRepo.UpdateLocation(ctx, r.ID(), req.Latitude, req.Longitude); err != nil {
		return fmt.Errorf("failed to update location: %w", err)
	}

	// Publish RunnerLocationUpdateEvent.
	now := time.Now().UTC()
	evt := events.RunnerLocationUpdateEvent{
		RunnerID:   r.ID(),
		Latitude:   req.Latitude,
		Longitude:  req.Longitude,
		Speed:      req.Speed,
		Heading:    req.Heading,
		Timestamp:  now,
		OccurredAt: now,
	}
	cloudEvt, err := kafka.NewCloudEvent("service-runner", events.RunnerLocationUpdate, evt)
	if err != nil {
		s.logger.Error("failed to create cloud event", zap.Error(err))
	} else if err := s.producer.PublishEvent(ctx, events.TopicRunnerEvents, cloudEvt); err != nil {
		s.logger.Error("failed to publish location update event", zap.Error(err))
	}

	return nil
}

// AddCrateSpec adds a crate specification to the runner's profile.
func (s *RunnerService) AddCrateSpec(ctx context.Context, userID uuid.UUID, req AddCrateSpecRequest) (*dto.CrateSpecDTO, error) {
	r, err := s.runnerRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, domain.NewNotFoundError("runner", userID.String())
	}

	petTypes := make([]runnerDomain.PetType, len(req.PetTypes))
	for i, pt := range req.PetTypes {
		petTypes[i] = runnerDomain.PetType(pt)
	}

	spec := runnerDomain.NewCrateSpec(
		r.ID(),
		runnerDomain.CrateSize(req.Size),
		petTypes,
		req.MaxWeightKg,
		req.WidthCm,
		req.HeightCm,
		req.DepthCm,
		req.Ventilated,
		req.TemperatureControlled,
	)

	if err := s.crateSpecRepo.Save(ctx, spec); err != nil {
		return nil, fmt.Errorf("failed to save crate spec: %w", err)
	}

	result := toCrateSpecDTO(spec)
	return &result, nil
}

// FindNearbyRunners finds active runners within the given radius.
func (s *RunnerService) FindNearbyRunners(ctx context.Context, lat, lng, radiusKm float64) ([]dto.RunnerDTO, error) {
	runners, err := s.runnerRepo.FindNearbyActive(ctx, lat, lng, radiusKm, 20)
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby runners: %w", err)
	}

	results := make([]dto.RunnerDTO, 0, len(runners))
	for _, r := range runners {
		crateSpecs, _ := s.crateSpecRepo.FindByRunnerID(ctx, r.ID())
		d := s.toRunnerDTO(r, crateSpecs)
		results = append(results, d)
	}

	return results, nil
}

// toRunnerDTO converts a Runner domain object and its crate specs into a RunnerDTO.
func (s *RunnerService) toRunnerDTO(r *runnerDomain.Runner, crateSpecs []*runnerDomain.CrateSpec) dto.RunnerDTO {
	crateSpecDTOs := make([]dto.CrateSpecDTO, 0, len(crateSpecs))
	for _, cs := range crateSpecs {
		crateSpecDTOs = append(crateSpecDTOs, toCrateSpecDTO(cs))
	}

	return dto.RunnerDTO{
		ID:             r.ID(),
		UserID:         r.UserID(),
		FullName:       r.FullName(),
		Phone:          r.Phone(),
		VehicleType:    string(r.VehicleType()),
		VehiclePlate:   r.VehiclePlate(),
		VehicleModel:   r.VehicleModel(),
		AirConditioned: r.AirConditioned(),
		SessionStatus:  string(r.SessionStatus()),
		Rating:         r.Rating(),
		TotalTrips:     r.TotalTrips(),
		CrateSpecs:     crateSpecDTOs,
		CreatedAt:      r.CreatedAt(),
	}
}

// toCrateSpecDTO converts a CrateSpec domain object into a CrateSpecDTO.
func toCrateSpecDTO(cs *runnerDomain.CrateSpec) dto.CrateSpecDTO {
	petTypes := make([]string, len(cs.PetTypes))
	for i, pt := range cs.PetTypes {
		petTypes[i] = string(pt)
	}

	return dto.CrateSpecDTO{
		ID:                    cs.ID,
		Size:                  string(cs.Size),
		PetTypes:              petTypes,
		MaxWeightKg:           cs.MaxWeightKg,
		WidthCm:               cs.WidthCm,
		HeightCm:              cs.HeightCm,
		DepthCm:               cs.DepthCm,
		Ventilated:            cs.Ventilated,
		TemperatureControlled: cs.TemperatureControlled,
	}
}
