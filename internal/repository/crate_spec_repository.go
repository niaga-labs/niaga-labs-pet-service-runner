package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	runnerDomain "github.com/niaga-labs/niaga-labs-pet-service-runner/internal/domain/runner"
)

// CrateSpecModel is the GORM model for the crate_specs table.
type CrateSpecModel struct {
	ID                    uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	RunnerID              uuid.UUID `gorm:"type:uuid;not null;index"`
	Size                  string    `gorm:"type:varchar(20);not null"`
	PetTypes              string    `gorm:"type:jsonb;not null;default:'[]'"`
	MaxWeightKg           float64   `gorm:"type:decimal(6,2);not null"`
	WidthCm               float64   `gorm:"type:decimal(6,2)"`
	HeightCm              float64   `gorm:"type:decimal(6,2)"`
	DepthCm               float64   `gorm:"type:decimal(6,2)"`
	Ventilated            bool      `gorm:"type:boolean;default:false"`
	TemperatureControlled bool      `gorm:"type:boolean;default:false"`
	CreatedAt             time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

// TableName overrides the default GORM table name.
func (CrateSpecModel) TableName() string {
	return "crate_specs"
}

// GormCrateSpecRepository implements CrateSpecRepository using GORM.
type GormCrateSpecRepository struct {
	db *gorm.DB
}

// NewGormCrateSpecRepository creates a new GORM-based crate spec repository.
func NewGormCrateSpecRepository(db *gorm.DB) *GormCrateSpecRepository {
	return &GormCrateSpecRepository{db: db}
}

// FindByRunnerID retrieves all crate specs for a given runner.
func (r *GormCrateSpecRepository) FindByRunnerID(ctx context.Context, runnerID uuid.UUID) ([]*runnerDomain.CrateSpec, error) {
	var models []CrateSpecModel
	if err := r.db.WithContext(ctx).Where("runner_id = ?", runnerID).Find(&models).Error; err != nil {
		return nil, err
	}

	specs := make([]*runnerDomain.CrateSpec, 0, len(models))
	for _, m := range models {
		specs = append(specs, toDomainCrateSpec(&m))
	}

	return specs, nil
}

// Save persists a new crate spec to the database.
func (r *GormCrateSpecRepository) Save(ctx context.Context, spec *runnerDomain.CrateSpec) error {
	model := toCrateSpecModel(spec)
	return r.db.WithContext(ctx).Create(&model).Error
}

// Delete removes a crate spec by ID.
func (r *GormCrateSpecRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&CrateSpecModel{}, "id = ?", id).Error
}

// --- CrateSpec Model / Domain conversions ---

func toCrateSpecModel(cs *runnerDomain.CrateSpec) *CrateSpecModel {
	petTypesJSON, _ := json.Marshal(cs.PetTypes)
	return &CrateSpecModel{
		ID:                    cs.ID,
		RunnerID:              cs.RunnerID,
		Size:                  string(cs.Size),
		PetTypes:              string(petTypesJSON),
		MaxWeightKg:           cs.MaxWeightKg,
		WidthCm:               cs.WidthCm,
		HeightCm:              cs.HeightCm,
		DepthCm:               cs.DepthCm,
		Ventilated:            cs.Ventilated,
		TemperatureControlled: cs.TemperatureControlled,
		CreatedAt:             cs.CreatedAt,
	}
}

func toDomainCrateSpec(m *CrateSpecModel) *runnerDomain.CrateSpec {
	var petTypes []runnerDomain.PetType
	_ = json.Unmarshal([]byte(m.PetTypes), &petTypes)

	return &runnerDomain.CrateSpec{
		ID:                    m.ID,
		RunnerID:              m.RunnerID,
		Size:                  runnerDomain.CrateSize(m.Size),
		PetTypes:              petTypes,
		MaxWeightKg:           m.MaxWeightKg,
		WidthCm:               m.WidthCm,
		HeightCm:              m.HeightCm,
		DepthCm:               m.DepthCm,
		Ventilated:            m.Ventilated,
		TemperatureControlled: m.TemperatureControlled,
		CreatedAt:             m.CreatedAt,
	}
}
