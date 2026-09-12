package repository

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/niaga-labs/niaga-labs-pet-lib-common/domain"
	petshopDomain "github.com/niaga-labs/niaga-labs-pet-service-runner/internal/domain/petshop"
)

// PetShopModel is the GORM model for the pet_shops table.
type PetShopModel struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	OwnerID      *uuid.UUID `gorm:"type:uuid;index"`
	Name         string     `gorm:"type:varchar(255);not null"`
	Address      string     `gorm:"type:text;not null"`
	Latitude     float64    `gorm:"type:double precision;not null"`
	Longitude    float64    `gorm:"type:double precision;not null"`
	Phone        string     `gorm:"type:varchar(20)"`
	Email        string     `gorm:"type:varchar(255)"`
	Category     string     `gorm:"type:varchar(20);not null;index"`
	Services     string     `gorm:"type:jsonb;default:'[]'"`
	Rating       float64    `gorm:"type:decimal(3,2);default:0.0"`
	ImageURL     string     `gorm:"type:text"`
	OpeningHours string     `gorm:"type:varchar(100)"`
	Description  string     `gorm:"type:text"`
	CreatedAt    time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt    time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

func (PetShopModel) TableName() string {
	return "pet_shops"
}

// GormPetShopRepository implements PetShopRepository using GORM.
type GormPetShopRepository struct {
	db *gorm.DB
}

func NewGormPetShopRepository(db *gorm.DB) *GormPetShopRepository {
	return &GormPetShopRepository{db: db}
}

func (r *GormPetShopRepository) FindAll(category string, limit int) ([]*petshopDomain.PetShop, error) {
	var models []PetShopModel
	query := r.db
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if limit <= 0 {
		limit = 50
	}
	if err := query.Order("rating DESC, name ASC").Limit(limit).Find(&models).Error; err != nil {
		return nil, err
	}
	return toDomainPetShops(models), nil
}

func (r *GormPetShopRepository) FindByID(id uuid.UUID) (*petshopDomain.PetShop, error) {
	var model PetShopModel
	if err := r.db.Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return toDomainPetShop(&model), nil
}

func (r *GormPetShopRepository) FindNearby(lat, lng, radiusKm float64, limit int) ([]*petshopDomain.PetShop, error) {
	var models []PetShopModel
	if limit <= 0 {
		limit = 20
	}

	err := r.db.Raw(`
		SELECT *, ST_Distance(
			ST_MakePoint(longitude, latitude)::geography,
			ST_MakePoint(?, ?)::geography
		) / 1000 as distance_km
		FROM pet_shops
		WHERE ST_DWithin(
			ST_MakePoint(longitude, latitude)::geography,
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
	return toDomainPetShops(models), nil
}

func (r *GormPetShopRepository) Save(shop *petshopDomain.PetShop) error {
	model := toPetShopModel(shop)
	return r.db.Create(&model).Error
}

func (r *GormPetShopRepository) FindByOwnerID(ownerID uuid.UUID) ([]*petshopDomain.PetShop, error) {
	var models []PetShopModel
	if err := r.db.Where("owner_id = ?", ownerID).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, err
	}
	return toDomainPetShops(models), nil
}

func (r *GormPetShopRepository) Update(shop *petshopDomain.PetShop) error {
	model := toPetShopModel(shop)
	return r.db.Save(model).Error
}

func (r *GormPetShopRepository) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&PetShopModel{}).Error
}

// --- Conversions ---

func toPetShopModel(s *petshopDomain.PetShop) *PetShopModel {
	servicesJSON, _ := json.Marshal(s.Services())
	model := &PetShopModel{
		ID:           s.ID(),
		Name:         s.Name(),
		Address:      s.Address(),
		Latitude:     s.Latitude(),
		Longitude:    s.Longitude(),
		Phone:        s.Phone(),
		Email:        s.Email(),
		Category:     string(s.Category()),
		Services:     string(servicesJSON),
		Rating:       s.Rating(),
		ImageURL:     s.ImageURL(),
		OpeningHours: s.OpeningHours(),
		Description:  s.Description(),
		CreatedAt:    s.CreatedAt(),
		UpdatedAt:    s.UpdatedAt(),
	}
	if s.OwnerID() != uuid.Nil {
		ownerID := s.OwnerID()
		model.OwnerID = &ownerID
	}
	return model
}

func toDomainPetShop(m *PetShopModel) *petshopDomain.PetShop {
	var services []string
	_ = json.Unmarshal([]byte(m.Services), &services)

	var ownerID uuid.UUID
	if m.OwnerID != nil {
		ownerID = *m.OwnerID
	}

	return petshopDomain.Reconstruct(
		m.ID, ownerID,
		m.Name, m.Address,
		m.Latitude, m.Longitude,
		m.Phone, m.Email,
		petshopDomain.Category(m.Category),
		services,
		m.Rating,
		m.ImageURL, m.OpeningHours, m.Description,
		m.CreatedAt, m.UpdatedAt,
	)
}

func toDomainPetShops(models []PetShopModel) []*petshopDomain.PetShop {
	shops := make([]*petshopDomain.PetShop, 0, len(models))
	for _, m := range models {
		shops = append(shops, toDomainPetShop(&m))
	}
	return shops
}
