package repository

import (
	"go.uber.org/zap"
	"gorm.io/gorm"

	petshopDomain "github.com/Kilat-Pet-Delivery/service-runner/internal/domain/petshop"
)

// SeedPetShops inserts sample pet shops if the table is empty.
func SeedPetShops(db *gorm.DB, logger *zap.Logger) {
	var count int64
	db.Model(&PetShopModel{}).Count(&count)
	if count > 0 {
		logger.Info("pet_shops table already seeded", zap.Int64("count", count))
		return
	}

	type seedShop struct {
		name, address                       string
		lat, lng                            float64
		phone, email                        string
		category                            petshopDomain.Category
		services                            []string
		imageURL, openingHours, description string
		rating                              float64
	}

	seeds := []seedShop{
		{
			"Paws & Claws Grooming",
			"12, Jalan Bukit Bintang, 55100 Kuala Lumpur",
			3.1466, 101.7108,
			"+60123456789", "pawsclaws@example.com",
			petshopDomain.CategoryGrooming,
			[]string{"Bath & Blow Dry", "Full Grooming", "Nail Trimming", "Ear Cleaning"},
			"", "9:00 AM - 7:00 PM",
			"Professional pet grooming services in the heart of KL. We pamper your pets with love and care.",
			4.5,
		},
		{
			"KL Pet Clinic",
			"45, Jalan Ampang, 50450 Kuala Lumpur",
			3.1590, 101.7200,
			"+60198765432", "klpetclinic@example.com",
			petshopDomain.CategoryVet,
			[]string{"Vaccination", "Health Checkup", "Surgery", "Dental Care", "Emergency"},
			"", "8:00 AM - 10:00 PM",
			"Full-service veterinary clinic with experienced vets. Emergency services available.",
			4.3,
		},
		{
			"Happy Tails Boarding",
			"88, Jalan PJU 1A/46, 47301 Petaling Jaya",
			3.1209, 101.6270,
			"+60112233445", "happytails@example.com",
			petshopDomain.CategoryBoarding,
			[]string{"Day Care", "Overnight Boarding", "Long Stay", "Play Area", "Webcam Monitoring"},
			"", "Open 24 Hours",
			"Premium pet boarding with spacious rooms, play areas, and 24/7 monitoring via webcam.",
			4.7,
		},
		{
			"Pet Paradise Store",
			"Lot G-15, Mid Valley Megamall, 59200 Kuala Lumpur",
			3.1178, 101.6773,
			"+60133445566", "petparadise@example.com",
			petshopDomain.CategoryPetStore,
			[]string{"Pet Food", "Accessories", "Toys", "Crates & Carriers", "Health Supplements"},
			"", "10:00 AM - 10:00 PM",
			"One-stop pet store with premium brands. From food to fashion, we have everything for your pet.",
			4.2,
		},
		{
			"Furry Friends Vet",
			"23, Jalan SS 2/55, 47300 Petaling Jaya",
			3.1172, 101.6250,
			"+60177889900", "furryvet@example.com",
			petshopDomain.CategoryVet,
			[]string{"Vaccination", "Microchipping", "Spay/Neuter", "X-Ray", "Blood Test"},
			"", "9:00 AM - 6:00 PM",
			"Caring veterinary practice with state-of-the-art equipment and gentle handling.",
			4.6,
		},
		{
			"Bark & Bath Spa",
			"15, Jalan Telawi 3, Bangsar, 59100 Kuala Lumpur",
			3.1300, 101.6710,
			"+60144556677", "barkbath@example.com",
			petshopDomain.CategoryGrooming,
			[]string{"Spa Treatment", "De-shedding", "Flea Treatment", "Teeth Brushing", "Creative Styling"},
			"", "10:00 AM - 8:00 PM",
			"Luxury pet spa experience. Your pet deserves the royal treatment!",
			4.4,
		},
		{
			"Meow Manor Cat Hotel",
			"Unit 3-1, Jalan Desa Kiara, 60000 Kuala Lumpur",
			3.1650, 101.6550,
			"+60155667788", "meowmanor@example.com",
			petshopDomain.CategoryBoarding,
			[]string{"Cat Boarding", "Private Suites", "Play Sessions", "Grooming", "Special Diet"},
			"", "Open 24 Hours",
			"Exclusive cat boarding hotel with private suites and enrichment activities.",
			4.8,
		},
	}

	repo := NewGormPetShopRepository(db)
	for _, sd := range seeds {
		shop, err := petshopDomain.NewPetShop(
			sd.name, sd.address,
			sd.lat, sd.lng,
			sd.phone, sd.email,
			sd.category,
			sd.services,
			sd.imageURL, sd.openingHours, sd.description,
		)
		if err != nil {
			logger.Error("failed to create seed pet shop", zap.String("name", sd.name), zap.Error(err))
			continue
		}
		shop.SetRating(sd.rating)

		if err := repo.Save(shop); err != nil {
			logger.Error("failed to seed pet shop", zap.String("name", sd.name), zap.Error(err))
		}
	}

	logger.Info("seeded pet shops", zap.Int("count", len(seeds)))
}
