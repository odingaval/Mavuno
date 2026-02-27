package storage

import (
	"fmt"
	"time"

	"mavuno/internal/models"

	"github.com/google/uuid"
)

// ============================================================
// HELPER FUNCTIONS
// ============================================================

// generateID creates a new unique ID for any record.
func generateID() string {
	return uuid.New().String()
}

// ============================================================
// SERVICE-FACING HELPERS  (used by ProduceService & ListingService)
// ============================================================

// GetAllProduce returns every non-deleted produce row from SQLite.
func GetAllProduce() ([]models.Produce, error) {
	rows, err := DB.Queryx(`SELECT * FROM produce WHERE deleted = 0`)
	if err != nil {
		return nil, fmt.Errorf("GetAllProduce: %w", err)
	}
	defer rows.Close()

	var out []models.Produce
	for rows.Next() {
		var row struct {
			ID                string    `db:"id"`
			FarmerID          string    `db:"farmer_id"`
			Category          string    `db:"category"`
			ProduceName       string    `db:"produce_name"`
			Quantity          float64   `db:"quantity"`
			QuantitySold      float64   `db:"quantity_sold"`
			QuantityRejected  float64   `db:"quantity_rejected"`
			QuantityRemaining float64   `db:"quantity_remaining"`
			PricePerUnit      float64   `db:"price_per_unit"`
			TotalReceived     float64   `db:"total_received"`
			Unit              string    `db:"unit"`
			Notes             string    `db:"notes"`
			Location          string    `db:"location"`
			Version           int       `db:"version"`
			CreatedAt         time.Time `db:"created_at"`
			UpdatedAt         time.Time `db:"updated_at"`
			Deleted           bool      `db:"deleted"`
		}
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("GetAllProduce scan: %w", err)
		}
		p := models.Produce{
			BaseModel: models.BaseModel{
				ID: row.ID, Version: row.Version,
				CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, Deleted: row.Deleted,
			},
			FarmerID: row.FarmerID, Category: models.ProduceCategory(row.Category),
			ProduceName: row.ProduceName, Quantity: row.Quantity,
			QuantitySold: row.QuantitySold, QuantityRejected: row.QuantityRejected,
			QuantityRemaining: row.QuantityRemaining, PricePerUnit: row.PricePerUnit,
			TotalReceived: row.TotalReceived, Unit: row.Unit,
			Notes: row.Notes, Location: row.Location,
		}
		out = append(out, p)
	}
	return out, nil
}

// SaveProduce upserts a produce record into SQLite.
func SaveProduce(p models.Produce) error {
	_, err := DB.NamedExec(`
		INSERT INTO produce (
			id, farmer_id, category, produce_name, quantity, quantity_sold,
			quantity_rejected, quantity_remaining, price_per_unit, total_received,
			unit, notes, location, version, created_at, updated_at, deleted
		) VALUES (
			:id, :farmer_id, :category, :produce_name, :quantity, :quantity_sold,
			:quantity_rejected, :quantity_remaining, :price_per_unit, :total_received,
			:unit, :notes, :location, :version, :created_at, :updated_at, :deleted
		) ON CONFLICT(id) DO UPDATE SET
			farmer_id=excluded.farmer_id, category=excluded.category,
			produce_name=excluded.produce_name, quantity=excluded.quantity,
			quantity_sold=excluded.quantity_sold, quantity_rejected=excluded.quantity_rejected,
			quantity_remaining=excluded.quantity_remaining, price_per_unit=excluded.price_per_unit,
			total_received=excluded.total_received, unit=excluded.unit,
			notes=excluded.notes, location=excluded.location,
			version=excluded.version, updated_at=excluded.updated_at, deleted=excluded.deleted`,
		map[string]interface{}{
			"id": p.ID, "farmer_id": p.FarmerID, "category": string(p.Category),
			"produce_name": p.ProduceName, "quantity": p.Quantity,
			"quantity_sold": p.QuantitySold, "quantity_rejected": p.QuantityRejected,
			"quantity_remaining": p.QuantityRemaining, "price_per_unit": p.PricePerUnit,
			"total_received": p.TotalReceived, "unit": p.Unit,
			"notes": p.Notes, "location": p.Location,
			"version": p.Version, "created_at": p.CreatedAt, "updated_at": p.UpdatedAt,
			"deleted": p.Deleted,
		})
	return err
}

// GetAllListingRows returns every non-deleted listing row from SQLite.
func GetAllListingRows() ([]models.Listing, error) {
	rows, err := DB.Queryx(`SELECT * FROM listings WHERE deleted = 0`)
	if err != nil {
		return nil, fmt.Errorf("GetAllListingRows: %w", err)
	}
	defer rows.Close()

	var out []models.Listing
	for rows.Next() {
		var row struct {
			ID             string    `db:"id"`
			ProduceID      string    `db:"produce_id"`
			ProduceName    string    `db:"produce_name"`
			FarmerID       string    `db:"farmer_id"`
			QuantityListed float64   `db:"quantity_listed"`
			AskingPrice    float64   `db:"asking_price"`
			Location       string    `db:"location"`
			Contact        string    `db:"contact"`
			Status         string    `db:"status"`
			BuyerName      string    `db:"buyer_name"`
			BuyerContact   string    `db:"buyer_contact"`
			BuyerLocation  string    `db:"buyer_location"`
			Notes          string    `db:"notes"`
			Version        int       `db:"version"`
			CreatedAt      time.Time `db:"created_at"`
			UpdatedAt      time.Time `db:"updated_at"`
			Deleted        bool      `db:"deleted"`
		}
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("GetAllListingRows scan: %w", err)
		}
		l := models.Listing{
			BaseModel: models.BaseModel{
				ID: row.ID, Version: row.Version,
				CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, Deleted: row.Deleted,
			},
			ProduceID: row.ProduceID, ProduceName: row.ProduceName,
			FarmerID: row.FarmerID, QuantityListed: row.QuantityListed,
			AskingPrice: row.AskingPrice, Location: row.Location,
			Contact: row.Contact, Status: models.ListingStatus(row.Status),
			BuyerName: row.BuyerName, BuyerContact: row.BuyerContact,
			BuyerLocation: row.BuyerLocation, Notes: row.Notes,
		}
		out = append(out, l)
	}
	return out, nil
}

// SaveListing upserts a listing row into SQLite.
func SaveListing(l models.Listing) error {
	_, err := DB.NamedExec(`
		INSERT INTO listings (
			id, produce_id, produce_name, farmer_id, quantity_listed, asking_price,
			location, contact, status, buyer_name, buyer_contact, buyer_location,
			notes, version, created_at, updated_at, deleted
		) VALUES (
			:id, :produce_id, :produce_name, :farmer_id, :quantity_listed, :asking_price,
			:location, :contact, :status, :buyer_name, :buyer_contact, :buyer_location,
			:notes, :version, :created_at, :updated_at, :deleted
		) ON CONFLICT(id) DO UPDATE SET
			produce_id=excluded.produce_id, produce_name=excluded.produce_name,
			farmer_id=excluded.farmer_id, quantity_listed=excluded.quantity_listed,
			asking_price=excluded.asking_price, location=excluded.location,
			contact=excluded.contact, status=excluded.status,
			buyer_name=excluded.buyer_name, buyer_contact=excluded.buyer_contact,
			buyer_location=excluded.buyer_location, notes=excluded.notes,
			version=excluded.version, updated_at=excluded.updated_at, deleted=excluded.deleted`,
		map[string]interface{}{
			"id": l.ID, "produce_id": l.ProduceID, "produce_name": l.ProduceName,
			"farmer_id": l.FarmerID, "quantity_listed": l.QuantityListed,
			"asking_price": l.AskingPrice, "location": l.Location,
			"contact": l.Contact, "status": string(l.Status),
			"buyer_name": l.BuyerName, "buyer_contact": l.BuyerContact,
			"buyer_location": l.BuyerLocation, "notes": l.Notes,
			"version": l.Version, "created_at": l.CreatedAt,
			"updated_at": l.UpdatedAt, "deleted": l.Deleted,
		})
	return err
}

// ============================================================
// FARMER REPOSITORY
// ============================================================

// CreateFarmer saves a new farmer profile to the database.
func CreateFarmer(farmer *models.Farmer) error {
	farmer.ID = generateID()
	farmer.Version = 1
	farmer.CreatedAt = time.Now()
	farmer.UpdatedAt = time.Now()
	farmer.Deleted = false

	query := `
		INSERT INTO farmers (
			id, full_name, phone, location,
			version, created_at, updated_at, deleted
		) VALUES (
			:id, :full_name, :phone, :location,
			:version, :created_at, :updated_at, :deleted
		)`

	_, err := DB.NamedExec(query, map[string]interface{}{
		"id":         farmer.ID,
		"full_name":  farmer.FullName,
		"phone":      farmer.Phone,
		"location":   farmer.Location,
		"version":    farmer.Version,
		"created_at": farmer.CreatedAt,
		"updated_at": farmer.UpdatedAt,
		"deleted":    farmer.Deleted,
	})
	if err != nil {
		return fmt.Errorf("error creating farmer: %w", err)
	}

	return nil
}

// GetFarmerByID retrieves a single farmer record by their ID.
func GetFarmerByID(id string) (*models.Farmer, error) {
	farmer := &models.Farmer{}

	query := `
		SELECT 
			id, full_name, phone, location,
			version, created_at, updated_at, deleted
		FROM farmers
		WHERE id = ? AND deleted = 0`

	err := DB.QueryRowx(query, id).Scan(
		&farmer.ID,
		&farmer.FullName,
		&farmer.Phone,
		&farmer.Location,
		&farmer.Version,
		&farmer.CreatedAt,
		&farmer.UpdatedAt,
		&farmer.Deleted,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting farmer: %w", err)
	}

	return farmer, nil
}

// GetAllFarmers retrieves all active farmer records.
func GetAllFarmers() ([]models.Farmer, error) {
	farmers := []models.Farmer{}

	query := `
		SELECT 
			id, full_name, phone, location,
			version, created_at, updated_at, deleted
		FROM farmers
		WHERE deleted = 0
		ORDER BY created_at DESC`

	rows, err := DB.Queryx(query)
	if err != nil {
		return nil, fmt.Errorf("error getting farmers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		farmer := models.Farmer{}
		err := rows.Scan(
			&farmer.ID,
			&farmer.FullName,
			&farmer.Phone,
			&farmer.Location,
			&farmer.Version,
			&farmer.CreatedAt,
			&farmer.UpdatedAt,
			&farmer.Deleted,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning farmer: %w", err)
		}
		farmers = append(farmers, farmer)
	}

	return farmers, nil
}

// UpdateFarmer updates an existing farmer record.
// It increments the version number to track changes for conflict detection.
func UpdateFarmer(farmer *models.Farmer) error {
	farmer.Version = farmer.Version + 1
	farmer.UpdatedAt = time.Now()

	query := `
		UPDATE farmers SET
			full_name  = :full_name,
			phone      = :phone,
			location   = :location,
			version    = :version,
			updated_at = :updated_at
		WHERE id = :id AND deleted = 0`

	_, err := DB.NamedExec(query, map[string]interface{}{
		"id":         farmer.ID,
		"full_name":  farmer.FullName,
		"phone":      farmer.Phone,
		"location":   farmer.Location,
		"version":    farmer.Version,
		"updated_at": farmer.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("error updating farmer: %w", err)
	}

	return nil
}

// ============================================================
// PRODUCE REPOSITORY
// ============================================================

// CreateProduce saves a new produce record to the database.
func CreateProduce(produce *models.Produce) error {
	produce.ID = generateID()
	produce.Version = 1
	produce.CreatedAt = time.Now()
	produce.UpdatedAt = time.Now()
	produce.Deleted = false

	// Calculate remaining quantity and total received automatically
	produce.QuantityRemaining = produce.Quantity - produce.QuantitySold - produce.QuantityRejected
	produce.TotalReceived = produce.QuantitySold * produce.PricePerUnit

	query := `
		INSERT INTO produce (
			id, farmer_id, category, produce_name,
			quantity, quantity_sold, quantity_rejected,
			quantity_remaining, price_per_unit, total_received,
			unit, notes, version, created_at, updated_at, deleted
		) VALUES (
			:id, :farmer_id, :category, :produce_name,
			:quantity, :quantity_sold, :quantity_rejected,
			:quantity_remaining, :price_per_unit, :total_received,
			:unit, :notes, :version, :created_at, :updated_at, :deleted
		)`

	_, err := DB.NamedExec(query, map[string]interface{}{
		"id":                 produce.ID,
		"farmer_id":          produce.FarmerID,
		"category":           produce.Category,
		"produce_name":       produce.ProduceName,
		"quantity":           produce.Quantity,
		"quantity_sold":      produce.QuantitySold,
		"quantity_rejected":  produce.QuantityRejected,
		"quantity_remaining": produce.QuantityRemaining,
		"price_per_unit":     produce.PricePerUnit,
		"total_received":     produce.TotalReceived,
		"unit":               produce.Unit,
		"notes":              produce.Notes,
		"version":            produce.Version,
		"created_at":         produce.CreatedAt,
		"updated_at":         produce.UpdatedAt,
		"deleted":            produce.Deleted,
	})
	if err != nil {
		return fmt.Errorf("error creating produce: %w", err)
	}

	return nil
}

// GetProduceByID retrieves a single produce record by its ID.
func GetProduceByID(id string) (*models.Produce, error) {
	produce := &models.Produce{}

	query := `
		SELECT
			id, farmer_id, category, produce_name,
			quantity, quantity_sold, quantity_rejected,
			quantity_remaining, price_per_unit, total_received,
			unit, notes, version, created_at, updated_at, deleted
		FROM produce
		WHERE id = ? AND deleted = 0`

	err := DB.QueryRowx(query, id).Scan(
		&produce.ID,
		&produce.FarmerID,
		&produce.Category,
		&produce.ProduceName,
		&produce.Quantity,
		&produce.QuantitySold,
		&produce.QuantityRejected,
		&produce.QuantityRemaining,
		&produce.PricePerUnit,
		&produce.TotalReceived,
		&produce.Unit,
		&produce.Notes,
		&produce.Version,
		&produce.CreatedAt,
		&produce.UpdatedAt,
		&produce.Deleted,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting produce: %w", err)
	}

	return produce, nil
}

// GetAllProduceByFarmer retrieves all active produce records for a specific farmer.
func GetAllProduceByFarmer(farmerID string) ([]models.Produce, error) {
	produceList := []models.Produce{}

	query := `
		SELECT
			id, farmer_id, category, produce_name,
			quantity, quantity_sold, quantity_rejected,
			quantity_remaining, price_per_unit, total_received,
			unit, notes, version, created_at, updated_at, deleted
		FROM produce
		WHERE farmer_id = ? AND deleted = 0
		ORDER BY created_at DESC`

	rows, err := DB.Queryx(query, farmerID)
	if err != nil {
		return nil, fmt.Errorf("error getting produce list: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		produce := models.Produce{}
		err := rows.Scan(
			&produce.ID,
			&produce.FarmerID,
			&produce.Category,
			&produce.ProduceName,
			&produce.Quantity,
			&produce.QuantitySold,
			&produce.QuantityRejected,
			&produce.QuantityRemaining,
			&produce.PricePerUnit,
			&produce.TotalReceived,
			&produce.Unit,
			&produce.Notes,
			&produce.Version,
			&produce.CreatedAt,
			&produce.UpdatedAt,
			&produce.Deleted,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning produce: %w", err)
		}
		produceList = append(produceList, produce)
	}

	return produceList, nil
}

// UpdateProduce updates an existing produce record.
// It recalculates derived fields and increments the version number.
func UpdateProduce(produce *models.Produce) error {
	produce.Version = produce.Version + 1
	produce.UpdatedAt = time.Now()

	// Recalculate derived fields on every update
	produce.QuantityRemaining = produce.Quantity - produce.QuantitySold - produce.QuantityRejected
	produce.TotalReceived = produce.QuantitySold * produce.PricePerUnit

	query := `
		UPDATE produce SET
			category           = :category,
			produce_name       = :produce_name,
			quantity           = :quantity,
			quantity_sold      = :quantity_sold,
			quantity_rejected  = :quantity_rejected,
			quantity_remaining = :quantity_remaining,
			price_per_unit     = :price_per_unit,
			total_received     = :total_received,
			unit               = :unit,
			notes              = :notes,
			version            = :version,
			updated_at         = :updated_at
		WHERE id = :id AND deleted = 0`

	_, err := DB.NamedExec(query, map[string]interface{}{
		"id":                 produce.ID,
		"category":           produce.Category,
		"produce_name":       produce.ProduceName,
		"quantity":           produce.Quantity,
		"quantity_sold":      produce.QuantitySold,
		"quantity_rejected":  produce.QuantityRejected,
		"quantity_remaining": produce.QuantityRemaining,
		"price_per_unit":     produce.PricePerUnit,
		"total_received":     produce.TotalReceived,
		"unit":               produce.Unit,
		"notes":              produce.Notes,
		"version":            produce.Version,
		"updated_at":         produce.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("error updating produce: %w", err)
	}

	return nil
}

// DeleteProduce soft deletes a produce record by marking it as deleted.
// The record remains in the database but is never shown to the user.
func DeleteProduce(id string) error {
	query := `
		UPDATE produce SET
			deleted    = 1,
			updated_at = ?
		WHERE id = ? AND deleted = 0`

	_, err := DB.Exec(query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("error deleting produce: %w", err)
	}

	return nil
}

// ============================================================
// LISTINGS REPOSITORY
// ============================================================

// CreateListing saves a new market listing to the database.
func CreateListing(listing *models.Listing) error {
	listing.ID = generateID()
	listing.Version = 1
	listing.CreatedAt = time.Now()
	listing.UpdatedAt = time.Now()
	listing.Deleted = false
	listing.Status = models.StatusAvailable

	query := `
		INSERT INTO listings (
			id, produce_id, farmer_id, quantity_listed,
			asking_price, location, status,
			buyer_name, buyer_contact, buyer_location,
			notes, version, created_at, updated_at, deleted
		) VALUES (
			:id, :produce_id, :farmer_id, :quantity_listed,
			:asking_price, :location, :status,
			:buyer_name, :buyer_contact, :buyer_location,
			:notes, :version, :created_at, :updated_at, :deleted
		)`

	_, err := DB.NamedExec(query, map[string]interface{}{
		"id":              listing.ID,
		"produce_id":      listing.ProduceID,
		"farmer_id":       listing.FarmerID,
		"quantity_listed": listing.QuantityListed,
		"asking_price":    listing.AskingPrice,
		"location":        listing.Location,
		"status":          listing.Status,
		"buyer_name":      listing.BuyerName,
		"buyer_contact":   listing.BuyerContact,
		"buyer_location":  listing.BuyerLocation,
		"notes":           listing.Notes,
		"version":         listing.Version,
		"created_at":      listing.CreatedAt,
		"updated_at":      listing.UpdatedAt,
		"deleted":         listing.Deleted,
	})
	if err != nil {
		return fmt.Errorf("error creating listing: %w", err)
	}

	return nil
}

// GetListingByID retrieves a single listing record by its ID.
func GetListingByID(id string) (*models.Listing, error) {
	listing := &models.Listing{}

	query := `
		SELECT
			id, produce_id, farmer_id, quantity_listed,
			asking_price, location, status,
			buyer_name, buyer_contact, buyer_location,
			notes, version, created_at, updated_at, deleted
		FROM listings
		WHERE id = ? AND deleted = 0`

	err := DB.QueryRowx(query, id).Scan(
		&listing.ID,
		&listing.ProduceID,
		&listing.FarmerID,
		&listing.QuantityListed,
		&listing.AskingPrice,
		&listing.Location,
		&listing.Status,
		&listing.BuyerName,
		&listing.BuyerContact,
		&listing.BuyerLocation,
		&listing.Notes,
		&listing.Version,
		&listing.CreatedAt,
		&listing.UpdatedAt,
		&listing.Deleted,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting listing: %w", err)
	}

	return listing, nil
}

// GetAllListings retrieves all active listings from the database.
// This is what buyers see when browsing the market.
func GetAllListings() ([]models.Listing, error) {
	listings := []models.Listing{}

	query := `
		SELECT
			id, produce_id, farmer_id, quantity_listed,
			asking_price, location, status,
			buyer_name, buyer_contact, buyer_location,
			notes, version, created_at, updated_at, deleted
		FROM listings
		WHERE deleted = 0
		ORDER BY created_at DESC`

	rows, err := DB.Queryx(query)
	if err != nil {
		return nil, fmt.Errorf("error getting listings: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		listing := models.Listing{}
		err := rows.Scan(
			&listing.ID,
			&listing.ProduceID,
			&listing.FarmerID,
			&listing.QuantityListed,
			&listing.AskingPrice,
			&listing.Location,
			&listing.Status,
			&listing.BuyerName,
			&listing.BuyerContact,
			&listing.BuyerLocation,
			&listing.Notes,
			&listing.Version,
			&listing.CreatedAt,
			&listing.UpdatedAt,
			&listing.Deleted,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning listing: %w", err)
		}
		listings = append(listings, listing)
	}

	return listings, nil
}

// GetListingsByFarmer retrieves all active listings for a specific farmer.
func GetListingsByFarmer(farmerID string) ([]models.Listing, error) {
	listings := []models.Listing{}

	query := `
		SELECT
			id, produce_id, farmer_id, quantity_listed,
			asking_price, location, status,
			buyer_name, buyer_contact, buyer_location,
			notes, version, created_at, updated_at, deleted
		FROM listings
		WHERE farmer_id = ? AND deleted = 0
		ORDER BY created_at DESC`

	rows, err := DB.Queryx(query, farmerID)
	if err != nil {
		return nil, fmt.Errorf("error getting farmer listings: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		listing := models.Listing{}
		err := rows.Scan(
			&listing.ID,
			&listing.ProduceID,
			&listing.FarmerID,
			&listing.QuantityListed,
			&listing.AskingPrice,
			&listing.Location,
			&listing.Status,
			&listing.BuyerName,
			&listing.BuyerContact,
			&listing.BuyerLocation,
			&listing.Notes,
			&listing.Version,
			&listing.CreatedAt,
			&listing.UpdatedAt,
			&listing.Deleted,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning listing: %w", err)
		}
		listings = append(listings, listing)
	}

	return listings, nil
}

// UpdateListing updates an existing listing record.
// Used when a buyer shows interest or a deal is completed.
func UpdateListing(listing *models.Listing) error {
	listing.Version = listing.Version + 1
	listing.UpdatedAt = time.Now()

	query := `
		UPDATE listings SET
			quantity_listed  = :quantity_listed,
			asking_price     = :asking_price,
			location         = :location,
			status           = :status,
			buyer_name       = :buyer_name,
			buyer_contact    = :buyer_contact,
			buyer_location   = :buyer_location,
			notes            = :notes,
			version          = :version,
			updated_at       = :updated_at
		WHERE id = :id AND deleted = 0`

	_, err := DB.NamedExec(query, map[string]interface{}{
		"id":              listing.ID,
		"quantity_listed": listing.QuantityListed,
		"asking_price":    listing.AskingPrice,
		"location":        listing.Location,
		"status":          listing.Status,
		"buyer_name":      listing.BuyerName,
		"buyer_contact":   listing.BuyerContact,
		"buyer_location":  listing.BuyerLocation,
		"notes":           listing.Notes,
		"version":         listing.Version,
		"updated_at":      listing.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("error updating listing: %w", err)
	}

	return nil
}

// DeleteListing soft deletes a listing by marking it as deleted.
func DeleteListing(id string) error {
	query := `
		UPDATE listings SET
			deleted    = 1,
			updated_at = ?
		WHERE id = ? AND deleted = 0`

	_, err := DB.Exec(query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("error deleting listing: %w", err)
	}

	return nil
}

// ============================================================
// SYNC QUEUE REPOSITORY
// ============================================================

// AddToSyncQueue adds a new operation to the sync queue.
// This is called every time a farmer makes a change offline.
// Think of it like dropping a letter into a postbox -
// it will be delivered when the postman (sync engine) comes.
func AddToSyncQueue(entityType models.SyncEntity, operation models.SyncOperation, payload string) error {
	item := models.SyncQueue{
		ID:         generateID(),
		EntityType: entityType,
		Operation:  operation,
		Payload:    payload,
		Status:     models.StatusPending,
		RetryCount: 0,
		CreatedAt:  time.Now(),
	}

	query := `
		INSERT INTO sync_queue (
			id, entity_type, operation, payload,
			status, retry_count, created_at
		) VALUES (
			:id, :entity_type, :operation, :payload,
			:status, :retry_count, :created_at
		)`

	_, err := DB.NamedExec(query, map[string]interface{}{
		"id":          item.ID,
		"entity_type": item.EntityType,
		"operation":   item.Operation,
		"payload":     item.Payload,
		"status":      item.Status,
		"retry_count": item.RetryCount,
		"created_at":  item.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("error adding to sync queue: %w", err)
	}

	return nil
}

// GetPendingItems retrieves all items in the sync queue that
// have not been successfully synced yet.
// The sync engine calls this to know what needs to be sent to the server.
func GetPendingItems() ([]models.SyncQueue, error) {
	items := []models.SyncQueue{}

	query := `
		SELECT
			id, entity_type, operation, payload,
			status, retry_count, last_attempt, created_at
		FROM sync_queue
		WHERE status = ?
		ORDER BY created_at ASC`

	rows, err := DB.Queryx(query, models.StatusPending)
	if err != nil {
		return nil, fmt.Errorf("error getting pending items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		item := models.SyncQueue{}
		err := rows.Scan(
			&item.ID,
			&item.EntityType,
			&item.Operation,
			&item.Payload,
			&item.Status,
			&item.RetryCount,
			&item.LastAttempt,
			&item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning sync queue item: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}

// GetFailedItems retrieves all items that have failed to sync.
// These are items that have been retried but keep failing.
func GetFailedItems() ([]models.SyncQueue, error) {
	items := []models.SyncQueue{}

	query := `
		SELECT
			id, entity_type, operation, payload,
			status, retry_count, last_attempt, created_at
		FROM sync_queue
		WHERE status = ?
		ORDER BY created_at ASC`

	rows, err := DB.Queryx(query, models.StatusFailed)
	if err != nil {
		return nil, fmt.Errorf("error getting failed items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		item := models.SyncQueue{}
		err := rows.Scan(
			&item.ID,
			&item.EntityType,
			&item.Operation,
			&item.Payload,
			&item.Status,
			&item.RetryCount,
			&item.LastAttempt,
			&item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning failed item: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}

// UpdateSyncStatus updates the status of a sync queue item.
// Called by the sync engine after attempting to send data to the server.
func UpdateSyncStatus(id string, status models.SyncStatus, retryCount int) error {
	query := `
		UPDATE sync_queue SET
			status       = ?,
			retry_count  = ?,
			last_attempt = ?
		WHERE id = ?`

	_, err := DB.Exec(query, status, retryCount, time.Now(), id)
	if err != nil {
		return fmt.Errorf("error updating sync status: %w", err)
	}

	return nil
}

// DeleteSyncedItems removes all successfully synced items from the queue.
// This keeps the queue clean and prevents it from growing too large.
// Think of it like emptying the postbox after all letters have been delivered.
func DeleteSyncedItems() error {
	query := `
		DELETE FROM sync_queue
		WHERE status = ?`

	_, err := DB.Exec(query, models.StatusSynced)
	if err != nil {
		return fmt.Errorf("error deleting synced items: %w", err)
	}

	return nil
}

// IncrementRetryCount increases the retry count for a failed sync item
// and marks it for retry. The sync engine uses this with exponential
// backoff to avoid hammering a server that is down.
func IncrementRetryCount(id string) error {
	query := `
		UPDATE sync_queue SET
			retry_count  = retry_count + 1,
			last_attempt = ?,
			status       = ?
		WHERE id = ?`

	_, err := DB.Exec(query, time.Now(), models.StatusPending, id)
	if err != nil {
		return fmt.Errorf("error incrementing retry count: %w", err)
	}

	return nil
}
