package petshop

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Category represents the type of pet shop.
type Category string

const (
	CategoryGrooming Category = "grooming"
	CategoryVet      Category = "vet"
	CategoryBoarding Category = "boarding"
	CategoryPetStore Category = "pet_store"
)

// ValidCategory returns true if c is a valid category.
func ValidCategory(c string) bool {
	switch Category(c) {
	case CategoryGrooming, CategoryVet, CategoryBoarding, CategoryPetStore:
		return true
	}
	return false
}

// PetShop is the aggregate root for a pet service shop.
type PetShop struct {
	id           uuid.UUID
	ownerID      uuid.UUID
	name         string
	address      string
	latitude     float64
	longitude    float64
	phone        string
	email        string
	category     Category
	services     []string
	rating       float64
	imageURL     string
	openingHours string
	description  string
	createdAt    time.Time
	updatedAt    time.Time
}

// NewPetShop creates a new unowned PetShop with validated fields.
func NewPetShop(
	name, address string,
	lat, lng float64,
	phone, email string,
	category Category,
	services []string,
	imageURL, openingHours, description string,
) (*PetShop, error) {
	if name == "" {
		return nil, fmt.Errorf("pet shop name is required")
	}
	if address == "" {
		return nil, fmt.Errorf("pet shop address is required")
	}
	if !ValidCategory(string(category)) {
		return nil, fmt.Errorf("invalid category: %s", category)
	}

	now := time.Now().UTC()
	return &PetShop{
		id:           uuid.New(),
		name:         name,
		address:      address,
		latitude:     lat,
		longitude:    lng,
		phone:        phone,
		email:        email,
		category:     category,
		services:     services,
		rating:       0.0,
		imageURL:     imageURL,
		openingHours: openingHours,
		description:  description,
		createdAt:    now,
		updatedAt:    now,
	}, nil
}

// NewOwnedPetShop creates a new PetShop linked to an owner.
func NewOwnedPetShop(
	ownerID uuid.UUID,
	name, address string,
	lat, lng float64,
	phone, email string,
	category Category,
	services []string,
	imageURL, openingHours, description string,
) (*PetShop, error) {
	if ownerID == uuid.Nil {
		return nil, fmt.Errorf("owner ID is required")
	}
	shop, err := NewPetShop(name, address, lat, lng, phone, email, category, services, imageURL, openingHours, description)
	if err != nil {
		return nil, err
	}
	shop.ownerID = ownerID
	return shop, nil
}

// Reconstruct rebuilds a PetShop from persistence data (no validation).
func Reconstruct(
	id, ownerID uuid.UUID,
	name, address string,
	lat, lng float64,
	phone, email string,
	category Category,
	services []string,
	rating float64,
	imageURL, openingHours, description string,
	createdAt, updatedAt time.Time,
) *PetShop {
	return &PetShop{
		id:           id,
		ownerID:      ownerID,
		name:         name,
		address:      address,
		latitude:     lat,
		longitude:    lng,
		phone:        phone,
		email:        email,
		category:     category,
		services:     services,
		rating:       rating,
		imageURL:     imageURL,
		openingHours: openingHours,
		description:  description,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
	}
}

// --- Getters ---

func (s *PetShop) ID() uuid.UUID        { return s.id }
func (s *PetShop) OwnerID() uuid.UUID   { return s.ownerID }
func (s *PetShop) Name() string         { return s.name }
func (s *PetShop) Address() string      { return s.address }
func (s *PetShop) Latitude() float64    { return s.latitude }
func (s *PetShop) Longitude() float64   { return s.longitude }
func (s *PetShop) Phone() string        { return s.phone }
func (s *PetShop) Email() string        { return s.email }
func (s *PetShop) Category() Category   { return s.category }
func (s *PetShop) Services() []string   { return s.services }
func (s *PetShop) Rating() float64      { return s.rating }
func (s *PetShop) ImageURL() string     { return s.imageURL }
func (s *PetShop) OpeningHours() string { return s.openingHours }
func (s *PetShop) Description() string  { return s.description }
func (s *PetShop) CreatedAt() time.Time { return s.createdAt }
func (s *PetShop) UpdatedAt() time.Time { return s.updatedAt }

// --- Behavior ---

// IsOwnedBy checks if the shop belongs to the given owner.
func (s *PetShop) IsOwnedBy(ownerID uuid.UUID) bool {
	return s.ownerID == ownerID
}

// Update applies partial updates to the pet shop. Only non-zero values are applied.
func (s *PetShop) Update(
	name, address string,
	lat, lng float64,
	phone, email string,
	category string,
	services []string,
	imageURL, openingHours, description string,
) {
	if name != "" {
		s.name = name
	}
	if address != "" {
		s.address = address
	}
	if lat != 0 {
		s.latitude = lat
	}
	if lng != 0 {
		s.longitude = lng
	}
	if phone != "" {
		s.phone = phone
	}
	if email != "" {
		s.email = email
	}
	if category != "" && ValidCategory(category) {
		s.category = Category(category)
	}
	if services != nil {
		s.services = services
	}
	if imageURL != "" {
		s.imageURL = imageURL
	}
	if openingHours != "" {
		s.openingHours = openingHours
	}
	if description != "" {
		s.description = description
	}
	s.updatedAt = time.Now().UTC()
}

// SetRating updates the shop's rating (used by seed and admin).
func (s *PetShop) SetRating(rating float64) {
	if rating < 0 {
		rating = 0
	}
	if rating > 5 {
		rating = 5
	}
	s.rating = rating
	s.updatedAt = time.Now().UTC()
}

// PetShopRepository defines persistence operations for pet shops.
type PetShopRepository interface {
	FindAll(category string, limit int) ([]*PetShop, error)
	FindByID(id uuid.UUID) (*PetShop, error)
	FindByOwnerID(ownerID uuid.UUID) ([]*PetShop, error)
	FindNearby(lat, lng, radiusKm float64, limit int) ([]*PetShop, error)
	Save(shop *PetShop) error
	Update(shop *PetShop) error
	Delete(id uuid.UUID) error
}
