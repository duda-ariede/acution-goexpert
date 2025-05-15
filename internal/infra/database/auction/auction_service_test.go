package auction

import (
	"auction_go/internal/entity/auction_entity"
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AuctionRepositorySuite struct {
	suite.Suite
	repo       *AuctionRepository
	client     *mongo.Client
	database   *mongo.Database
	collection *mongo.Collection
}

func (suite *AuctionRepositorySuite) SetupSuite() {
	// Set auction interval to a very short duration for testing
	os.Setenv("AUCTION_INTERVAL", "2s")

	// Setup MongoDB connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		suite.T().Fatal(err)
	}
	suite.client = client
	suite.database = client.Database("auction_test")
	suite.collection = suite.database.Collection("auctions")

	// Create a new auction repository
	suite.repo = NewAuctionRepository(suite.database)
}

func (suite *AuctionRepositorySuite) TearDownSuite() {
	// Clean up and close the MongoDB connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	suite.repo.Close()
	suite.collection.Drop(ctx)
	suite.client.Disconnect(ctx)
}

func (suite *AuctionRepositorySuite) SetupTest() {
	// Clean the collection before each test
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	suite.collection.Drop(ctx)
}

func (suite *AuctionRepositorySuite) TestCreateAuction() {
	// Create a test auction
	auction, err := auction_entity.CreateAuction(
		"Test Product",
		"Electronics",
		"This is a test product description for testing purposes",
		auction_entity.New,
	)
	assert.Nil(suite.T(), err)

	// Save the auction
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err = suite.repo.CreateAuction(ctx, auction)
	assert.Nil(suite.T(), err)

	// Verify auction was created with active status
	savedAuction, err := suite.repo.FindAuctionById(ctx, auction.Id)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), auction_entity.Active, savedAuction.Status)
}

func (suite *AuctionRepositorySuite) TestAuctionAutoClose() {
	// Create a test auction with timestamp set to expire VERY soon
	auction := &auction_entity.Auction{
		Id:          "test-auction-123",
		ProductName: "Test Product",
		Category:    "Electronics",
		Description: "This is a test product description for testing purposes",
		Condition:   auction_entity.New,
		Status:      auction_entity.Active,
		// Set timestamp to 3 seconds in the past, which is beyond our 2 second interval
		Timestamp:   time.Now().Add(-3 * time.Second),
	}

	// Save the auction
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err := suite.repo.CreateAuction(ctx, auction)
	assert.Nil(suite.T(), err)

	// Force close expired auctions directly instead of waiting
	// This helps ensure the test is deterministic
	suite.repo.closeExpiredAuctions()
	
	// Additionally wait for a moment to ensure processing completes
	time.Sleep(3 * time.Second)

	// Verify the auction was closed
	ctxCheck, cancelCheck := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelCheck()
	
	savedAuction, err := suite.repo.FindAuctionById(ctxCheck, auction.Id)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), auction_entity.Completed, savedAuction.Status)
}

func (suite *AuctionRepositorySuite) TestLoadActiveAuctions() {
	// Create two test auctions
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Auction 1: Active and not expired - ensure it's VERY recent
	auction1 := &auction_entity.Auction{
		Id:          "test-auction-active",
		ProductName: "Active Product",
		Category:    "Electronics",
		Description: "This is an active product that should remain active",
		Condition:   auction_entity.New,
		Status:      auction_entity.Active,
		// Set timestamp to current time to ensure it won't expire during test
		Timestamp:   time.Now(),
	}
	
	err := suite.repo.CreateAuction(ctx, auction1)
	assert.Nil(suite.T(), err)

	// Auction 2: Active but should be expired (timestamp in the past)
	auction2 := &auction_entity.Auction{
		Id:          "test-auction-expired",
		ProductName: "Expired Product",
		Category:    "Electronics",
		Description: "This is a product that should be closed when loading",
		Condition:   auction_entity.New,
		Status:      auction_entity.Active,
		// Set to 5 seconds in the past, well beyond our interval
		Timestamp:   time.Now().Add(-5 * time.Second), 
	}
	
	err = suite.repo.CreateAuction(ctx, auction2)
	assert.Nil(suite.T(), err)

	// Clear the active auctions map to test reloading
	suite.repo.auctionsMutex.Lock()
	suite.repo.activeAuctions = make(map[string]time.Time)
	suite.repo.auctionsMutex.Unlock()

	// Load active auctions from database
	ctxLoad, cancelLoad := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelLoad()
	
	err = suite.repo.LoadActiveAuctions(ctxLoad)
	assert.Nil(suite.T(), err)

	// Force close expired auctions directly
	suite.repo.closeExpiredAuctions()
	
	// Also wait for a moment to ensure processing
	time.Sleep(3 * time.Second)

	// Verify auction1 is still active
	ctxCheck, cancelCheck := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelCheck()
	
	savedAuction1, err := suite.repo.FindAuctionById(ctxCheck, auction1.Id)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), auction_entity.Active, savedAuction1.Status)

	// Verify auction2 was closed
	savedAuction2, err := suite.repo.FindAuctionById(ctxCheck, auction2.Id)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), auction_entity.Completed, savedAuction2.Status)
}

func TestAuctionRepositorySuite(t *testing.T) {
	suite.Run(t, new(AuctionRepositorySuite))
}
