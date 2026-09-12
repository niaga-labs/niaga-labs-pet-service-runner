package application

import (
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/niaga-labs/niaga-labs-pet-lib-common/domain"
	"github.com/niaga-labs/niaga-labs-pet-lib-proto/dto"
	petshopDomain "github.com/niaga-labs/niaga-labs-pet-service-runner/internal/domain/petshop"
)

// CreatePetShopRequest represents a request to create a pet shop.
type CreatePetShopRequest struct {
	Name         string   `json:"name" binding:"required"`
	Address      string   `json:"address" binding:"required"`
	Latitude     float64  `json:"latitude" binding:"required"`
	Longitude    float64  `json:"longitude" binding:"required"`
	Phone        string   `json:"phone"`
	Email        string   `json:"email"`
	Category     string   `json:"category" binding:"required,oneof=grooming vet boarding pet_store"`
	Services     []string `json:"services"`
	ImageURL     string   `json:"image_url"`
	OpeningHours string   `json:"opening_hours"`
	Description  string   `json:"description"`
}

// UpdatePetShopRequest represents a request to update a pet shop.
type UpdatePetShopRequest struct {
	Name         string   `json:"name"`
	Address      string   `json:"address"`
	Latitude     float64  `json:"latitude"`
	Longitude    float64  `json:"longitude"`
	Phone        string   `json:"phone"`
	Email        string   `json:"email"`
	Category     string   `json:"category" binding:"omitempty,oneof=grooming vet boarding pet_store"`
	Services     []string `json:"services"`
	ImageURL     string   `json:"image_url"`
	OpeningHours string   `json:"opening_hours"`
	Description  string   `json:"description"`
}

// PetShopService implements use cases for pet shops.
type PetShopService struct {
	repo   petshopDomain.PetShopRepository
	logger *zap.Logger
}

// NewPetShopService creates a new PetShopService.
func NewPetShopService(repo petshopDomain.PetShopRepository, logger *zap.Logger) *PetShopService {
	return &PetShopService{repo: repo, logger: logger}
}

// ListPetShops returns all pet shops, optionally filtered by category.
func (s *PetShopService) ListPetShops(category string) ([]dto.PetShopDTO, error) {
	shops, err := s.repo.FindAll(category, 50)
	if err != nil {
		return nil, fmt.Errorf("failed to list pet shops: %w", err)
	}
	return toPetShopDTOs(shops), nil
}

// GetPetShop returns a single pet shop by ID.
func (s *PetShopService) GetPetShop(id uuid.UUID) (*dto.PetShopDTO, error) {
	shop, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	result := toPetShopDTO(shop)
	return &result, nil
}

// FindNearbyPetShops returns pet shops within the given radius.
func (s *PetShopService) FindNearbyPetShops(lat, lng, radiusKm float64) ([]dto.PetShopDTO, error) {
	shops, err := s.repo.FindNearby(lat, lng, radiusKm, 20)
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby pet shops: %w", err)
	}
	return toPetShopDTOs(shops), nil
}

// CreatePetShop creates a new pet shop owned by the given user.
func (s *PetShopService) CreatePetShop(ownerID uuid.UUID, req CreatePetShopRequest) (*dto.PetShopDTO, error) {
	shop, err := petshopDomain.NewOwnedPetShop(
		ownerID,
		req.Name, req.Address,
		req.Latitude, req.Longitude,
		req.Phone, req.Email,
		petshopDomain.Category(req.Category),
		req.Services,
		req.ImageURL, req.OpeningHours, req.Description,
	)
	if err != nil {
		return nil, fmt.Errorf("invalid pet shop data: %w", err)
	}

	if err := s.repo.Save(shop); err != nil {
		s.logger.Error("failed to create pet shop", zap.Error(err))
		return nil, fmt.Errorf("failed to create pet shop: %w", err)
	}

	s.logger.Info("pet shop created", zap.String("shop_id", shop.ID().String()), zap.String("owner_id", ownerID.String()))
	result := toPetShopDTO(shop)
	return &result, nil
}

// UpdatePetShop updates a pet shop, verifying ownership.
func (s *PetShopService) UpdatePetShop(ownerID uuid.UUID, shopID uuid.UUID, req UpdatePetShopRequest) (*dto.PetShopDTO, error) {
	shop, err := s.repo.FindByID(shopID)
	if err != nil {
		return nil, domain.NewNotFoundError("PetShop", shopID.String())
	}

	if !shop.IsOwnedBy(ownerID) {
		return nil, domain.NewForbiddenError("you do not own this pet shop")
	}

	// Delegate update logic to the domain aggregate.
	shop.Update(
		req.Name, req.Address,
		req.Latitude, req.Longitude,
		req.Phone, req.Email,
		req.Category,
		req.Services,
		req.ImageURL, req.OpeningHours, req.Description,
	)

	if err := s.repo.Update(shop); err != nil {
		s.logger.Error("failed to update pet shop", zap.Error(err))
		return nil, fmt.Errorf("failed to update pet shop: %w", err)
	}

	s.logger.Info("pet shop updated", zap.String("shop_id", shopID.String()))
	result := toPetShopDTO(shop)
	return &result, nil
}

// DeletePetShop deletes a pet shop, verifying ownership.
func (s *PetShopService) DeletePetShop(ownerID uuid.UUID, shopID uuid.UUID) error {
	shop, err := s.repo.FindByID(shopID)
	if err != nil {
		return domain.NewNotFoundError("PetShop", shopID.String())
	}

	if !shop.IsOwnedBy(ownerID) {
		return domain.NewForbiddenError("you do not own this pet shop")
	}

	if err := s.repo.Delete(shopID); err != nil {
		s.logger.Error("failed to delete pet shop", zap.Error(err))
		return fmt.Errorf("failed to delete pet shop: %w", err)
	}

	s.logger.Info("pet shop deleted", zap.String("shop_id", shopID.String()))
	return nil
}

// GetMyShops returns all pet shops owned by the given user.
func (s *PetShopService) GetMyShops(ownerID uuid.UUID) ([]dto.PetShopDTO, error) {
	shops, err := s.repo.FindByOwnerID(ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get my shops: %w", err)
	}
	return toPetShopDTOs(shops), nil
}

func toPetShopDTO(s *petshopDomain.PetShop) dto.PetShopDTO {
	d := dto.PetShopDTO{
		ID:           s.ID(),
		Name:         s.Name(),
		Address:      s.Address(),
		Latitude:     s.Latitude(),
		Longitude:    s.Longitude(),
		Phone:        s.Phone(),
		Email:        s.Email(),
		Category:     string(s.Category()),
		Services:     s.Services(),
		Rating:       s.Rating(),
		ImageURL:     s.ImageURL(),
		OpeningHours: s.OpeningHours(),
		Description:  s.Description(),
		CreatedAt:    s.CreatedAt(),
	}
	if s.OwnerID() != uuid.Nil {
		ownerID := s.OwnerID()
		d.OwnerID = &ownerID
	}
	return d
}

func toPetShopDTOs(shops []*petshopDomain.PetShop) []dto.PetShopDTO {
	result := make([]dto.PetShopDTO, 0, len(shops))
	for _, s := range shops {
		result = append(result, toPetShopDTO(s))
	}
	return result
}
