package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/shdmitri/booking-service/internal/domain"
	apperrors "github.com/shdmitri/booking-service/pkg/errors"
)

var _ domain.BookingRepository = (*BookingCacheRepository)(nil)

type BookingCacheRepository struct {
	repo domain.BookingRepository
	redis *redis.Client
	ttl time.Duration
	logger *slog.Logger
}

func NewBookingCacheRepository(repo domain.BookingRepository, redis *redis.Client, ttl time.Duration, logger *slog.Logger) *BookingCacheRepository {
	return &BookingCacheRepository{
		repo: repo,
		redis: redis,
		ttl: ttl,
		logger: logger,
	}
}

func (br *BookingCacheRepository) Create(ctx context.Context, booking *domain.Booking) (string, error) {
	return br.repo.Create(ctx, booking)
}

func (br *BookingCacheRepository) GetByUserId(ctx context.Context, userId string) ([]*domain.Booking, error) {
	if !isValidId(userId) {
		return nil, &apperrors.Errors{
			Code: apperrors.ValidationError,
			Message: "invalid id format",
		}
	}
	cacheKey := fmt.Sprintf("bookings:userId:%s", userId)

	cached, err := br.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var data []*domain.Booking 

		if err := json.Unmarshal([]byte(cached), &data); err != nil {
			br.logger.Warn("error on unmarshaling user's booking cache data", "Error", err.Error())
		} else {
			br.logger.Debug("booking data loaded from redis!")
			return data, nil
		}
	} else if err != redis.Nil {
		br.logger.Error("redis error", "Error", err.Error())
	} else {
		br.logger.Debug("cache not found, falling to main db")
	}

	data, err := br.repo.GetByUserId(ctx, userId)

	if err != nil {
		return nil, err 
	}

	parsedData, err := json.Marshal(data)
	if err != nil {
		br.logger.Warn("error on marshaling user's booking cache data", "Error", err.Error())
	} else {
		if err := br.redis.Set(context.Background(), cacheKey, parsedData, br.ttl).Err(); err != nil {
			br.logger.Error("error on setting user's booking cache data", "Error", err.Error())
		} else {
			br.logger.Debug("user's booking cache data set successfully")
		}
	}

	return data, nil
}

func isValidId(id string) bool {
	if len(id) == 0 || len(id) > 36 {
		return false
	}
	_, err := uuid.Parse(id)

	if err != nil {
		return false
	}

	return true
}