package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/niaga-labs/niaga-labs-pet-lib-common/domain"
	runnerDomain "github.com/niaga-labs/niaga-labs-pet-service-runner/internal/domain/runner"
)

// RunnerModel is the GORM model for the runners table.
type RunnerModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	UserID         uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	FullName       string    `gorm:"type:varchar(255);not null"`
	Phone          string    `gorm:"type:varchar(20)"`
	VehicleType    string    `gorm:"type:varchar(20);not null"`
	VehiclePlate   string    `gorm:"type:varchar(20)"`
	VehicleModel   string    `gorm:"type:varchar(100)"`
	VehicleYear    int       `gorm:"type:int"`
	AirConditioned bool      `gorm:"type:boolean;default:false"`
	SessionStatus  string    `gorm:"type:varchar(20);not null;default:'inactive'"`
	CurrentLat     *float64  `gorm:"type:double precision"`
	CurrentLng     *float64  `gorm:"type:double precision"`
	Rating         float64   `gorm:"type:decimal(3,2);default:0.0"`
	TotalTrips     int       `gorm:"type:int;default:0"`
	CompletionRate float64   `gorm:"type:decimal(3,2);default:1.0"`
	Version        int64     `gorm:"type:bigint;not null;default:1"`
	CreatedAt      time.Time `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt      time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

// TableName overrides the default GORM table name.
func (RunnerModel) TableName() string {
	return "runners"
}

// GormRunnerRepository implements RunnerRepository using GORM.
type GormRunnerRepository struct {
	db *gorm.DB
}

// NewGormRunnerRepository creates a new GORM-based runner repository.
func NewGormRunnerRepository(db *gorm.DB) *GormRunnerRepository {
	return &GormRunnerRepository{db: db}
}

// FindByID retrieves a runner by its unique ID.
func (r *GormRunnerRepository) FindByID(ctx context.Context, id uuid.UUID) (*runnerDomain.Runner, error) {
	var model RunnerModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return toDomainRunner(&model), nil
}

// FindByUserID retrieves a runner by the associated user ID.
func (r *GormRunnerRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*runnerDomain.Runner, error) {
	var model RunnerModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return toDomainRunner(&model), nil
}

// Save persists a new runner to the database.
func (r *GormRunnerRepository) Save(ctx context.Context, runner *runnerDomain.Runner) error {
	model := toRunnerModel(runner)
	return r.db.WithContext(ctx).Create(&model).Error
}

// Update persists changes to an existing runner.
func (r *GormRunnerRepository) Update(ctx context.Context, runner *runnerDomain.Runner) error {
	model := toRunnerModel(runner)
	return r.db.WithContext(ctx).Save(&model).Error
}

// UpdateLocation updates only the lat/lng fields for a runner.
func (r *GormRunnerRepository) UpdateLocation(ctx context.Context, runnerID uuid.UUID, lat, lng float64) error {
	return r.db.WithContext(ctx).
		Model(&RunnerModel{}).
		Where("id = ?", runnerID).
		Updates(map[string]interface{}{
			"current_lat": lat,
			"current_lng": lng,
			"updated_at":  time.Now().UTC(),
		}).Error
}

// FindNearbyActive finds active runners within the given radius using PostGIS.
func (r *GormRunnerRepository) FindNearbyActive(ctx context.Context, lat, lng float64, radiusKm float64, limit int) ([]*runnerDomain.Runner, error) {
	var models []RunnerModel

	err := r.db.WithContext(ctx).Raw(`
		SELECT *, ST_Distance(
			ST_MakePoint(current_lng, current_lat)::geography,
			ST_MakePoint(?, ?)::geography
		) / 1000 as distance_km
		FROM runners
		WHERE session_status = 'active'
		AND current_lat IS NOT NULL
		AND ST_DWithin(
			ST_MakePoint(current_lng, current_lat)::geography,
			ST_MakePoint(?, ?)::geography,
			?
		)
		ORDER BY distance_km
		LIMIT ?`,
		lng, lat, lng, lat, radiusKm*1000, limit,
	).Scan(&models).Error

	if err != nil {
		return nil, err
	}

	runners := make([]*runnerDomain.Runner, 0, len(models))
	for _, m := range models {
		runners = append(runners, toDomainRunner(&m))
	}

	return runners, nil
}

// --- Model / Domain conversions ---

func toRunnerModel(r *runnerDomain.Runner) *RunnerModel {
	return &RunnerModel{
		ID:             r.ID(),
		UserID:         r.UserID(),
		FullName:       r.FullName(),
		Phone:          r.Phone(),
		VehicleType:    string(r.VehicleType()),
		VehiclePlate:   r.VehiclePlate(),
		VehicleModel:   r.VehicleModel(),
		VehicleYear:    r.VehicleYear(),
		AirConditioned: r.AirConditioned(),
		SessionStatus:  string(r.SessionStatus()),
		CurrentLat:     r.CurrentLat(),
		CurrentLng:     r.CurrentLng(),
		Rating:         r.Rating(),
		TotalTrips:     r.TotalTrips(),
		CompletionRate: r.CompletionRate(),
		Version:        r.Version(),
		CreatedAt:      r.CreatedAt(),
		UpdatedAt:      r.UpdatedAt(),
	}
}

func toDomainRunner(m *RunnerModel) *runnerDomain.Runner {
	return runnerDomain.Reconstruct(
		m.ID,
		m.UserID,
		m.FullName,
		m.Phone,
		runnerDomain.VehicleType(m.VehicleType),
		m.VehiclePlate,
		m.VehicleModel,
		m.VehicleYear,
		m.AirConditioned,
		runnerDomain.SessionStatus(m.SessionStatus),
		m.CurrentLat,
		m.CurrentLng,
		m.Rating,
		m.TotalTrips,
		m.CompletionRate,
		m.Version,
		m.CreatedAt,
		m.UpdatedAt,
	)
}
