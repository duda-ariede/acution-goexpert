package auction

import (
	"auction_go/configuration/logger"
	"auction_go/internal/entity/auction_entity"
	"auction_go/internal/internal_error"
	"context"
	"os"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

type AuctionEntityMongo struct {
	Id          string                          `bson:"_id"`
	ProductName string                          `bson:"product_name"`
	Category    string                          `bson:"category"`
	Description string                          `bson:"description"`
	Condition   auction_entity.ProductCondition `bson:"condition"`
	Status      auction_entity.AuctionStatus    `bson:"status"`
	Timestamp   int64                           `bson:"timestamp"`
}

type AuctionRepository struct {
	Collection       *mongo.Collection
	auctionInterval  time.Duration
	activeAuctions   map[string]time.Time
	auctionsMutex    *sync.RWMutex
	auctionCloserCtx context.Context
	cancelCloser     context.CancelFunc
}

func NewAuctionRepository(database *mongo.Database) *AuctionRepository {
	ctx, cancel := context.WithCancel(context.Background())

	repo := &AuctionRepository{
		Collection:       database.Collection("auctions"),
		auctionInterval:  getAuctionInterval(),
		activeAuctions:   make(map[string]time.Time),
		auctionsMutex:    &sync.RWMutex{},
		auctionCloserCtx: ctx,
		cancelCloser:     cancel,
	}

	// Start the auction closer goroutine
	go repo.startAuctionCloser()

	return repo
}

func (ar *AuctionRepository) CreateAuction(
	ctx context.Context,
	auctionEntity *auction_entity.Auction) *internal_error.InternalError {
	auctionEntityMongo := &AuctionEntityMongo{
		Id:          auctionEntity.Id,
		ProductName: auctionEntity.ProductName,
		Category:    auctionEntity.Category,
		Description: auctionEntity.Description,
		Condition:   auctionEntity.Condition,
		Status:      auctionEntity.Status,
		Timestamp:   auctionEntity.Timestamp.Unix(),
	}
	_, err := ar.Collection.InsertOne(ctx, auctionEntityMongo)
	if err != nil {
		logger.Error("Error trying to insert auction", err)
		return internal_error.NewInternalServerError("Error trying to insert auction")
	}

	// Add the auction to the active auctions map with its end time
	endTime := auctionEntity.Timestamp.Add(ar.auctionInterval)
	ar.auctionsMutex.Lock()
	ar.activeAuctions[auctionEntity.Id] = endTime
	ar.auctionsMutex.Unlock()

	return nil
}

// Close auction repository and stop the auction closer goroutine
func (ar *AuctionRepository) Close() {
	ar.cancelCloser()
}

// Start a goroutine to check for expired auctions and close them
func (ar *AuctionRepository) startAuctionCloser() {
	ticker := time.NewTicker(10 * time.Second) // Check every 10 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ar.closeExpiredAuctions()
		case <-ar.auctionCloserCtx.Done():
			logger.Info("Auction closer goroutine stopped")
			return
		}
	}
}

// Check for expired auctions and close them
func (ar *AuctionRepository) closeExpiredAuctions() {
	now := time.Now()
	expiredAuctions := make([]string, 0)

	// Find expired auctions
	ar.auctionsMutex.RLock()
	for auctionID, endTime := range ar.activeAuctions {
		if now.After(endTime) {
			expiredAuctions = append(expiredAuctions, auctionID)
		}
	}
	ar.auctionsMutex.RUnlock()

	// Close expired auctions
	for _, auctionID := range expiredAuctions {
		if err := ar.closeAuction(auctionID); err != nil {
			logger.Error("Failed to close expired auction", err)
		} else {
			// Remove from active auctions map
			ar.auctionsMutex.Lock()
			delete(ar.activeAuctions, auctionID)
			ar.auctionsMutex.Unlock()
		}
	}
}

// Close a specific auction by updating its status
func (ar *AuctionRepository) closeAuction(auctionID string) *internal_error.InternalError {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": auctionID, "status": auction_entity.Active}
	update := bson.M{"$set": bson.M{"status": auction_entity.Completed}}

	result, err := ar.Collection.UpdateOne(ctx, filter, update)
	if err != nil {
		logger.Error("Error closing auction", err)
		return internal_error.NewInternalServerError("Error closing auction")
	}

	if result.ModifiedCount > 0 {
		logger.Info("Auction closed successfully", zap.String("auctionID", auctionID))
	}

	return nil
}

// Load existing active auctions from database
func (ar *AuctionRepository) LoadActiveAuctions(ctx context.Context) *internal_error.InternalError {
	filter := bson.M{"status": auction_entity.Active}

	cursor, err := ar.Collection.Find(ctx, filter)
	if err != nil {
		logger.Error("Error loading active auctions", err)
		return internal_error.NewInternalServerError("Error loading active auctions")
	}
	defer cursor.Close(ctx)

	var auctions []AuctionEntityMongo
	if err := cursor.All(ctx, &auctions); err != nil {
		logger.Error("Error decoding auctions", err)
		return internal_error.NewInternalServerError("Error decoding auctions")
	}

	ar.auctionsMutex.Lock()
	defer ar.auctionsMutex.Unlock()

	for _, auction := range auctions {
		auctionTime := time.Unix(auction.Timestamp, 0)
		endTime := auctionTime.Add(ar.auctionInterval)

		// Only add if not already expired
		if time.Now().Before(endTime) {
			ar.activeAuctions[auction.Id] = endTime
		} else {
			// Close already expired auctions
			go ar.closeAuction(auction.Id)
		}
	}

	return nil
}

func getAuctionInterval() time.Duration {
	auctionInterval := os.Getenv("AUCTION_INTERVAL")
	duration, err := time.ParseDuration(auctionInterval)
	if err != nil {
		return time.Minute * 5 // Default to 5 minutes if not specified
	}

	return duration
}