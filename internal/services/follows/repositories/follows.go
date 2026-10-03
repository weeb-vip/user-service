package repositories

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/weeb-vip/user-service/internal/db"
	followmodels "github.com/weeb-vip/user-service/internal/services/follows/models"
	usermodels "github.com/weeb-vip/user-service/internal/services/users/models"
	"github.com/weeb-vip/user-service/metrics"
	"github.com/weeb-vip/user-service/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const table = "user_follows"

// FollowsRepository is the follow graph's storage.
//
// The write methods take the transaction to run in, because the service
// writes an outbox event in the same transaction as the edge change and the
// two must commit together. Transaction opens one; the reads use the pool.
type FollowsRepository interface {
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error

	Get(ctx context.Context, followerID, followeeID string) (*followmodels.UserFollow, error)
	Create(ctx context.Context, tx *gorm.DB, follow *followmodels.UserFollow) error
	Accept(ctx context.Context, tx *gorm.DB, followerID, followeeID string, at time.Time) (int64, error)
	Delete(ctx context.Context, tx *gorm.DB, followerID, followeeID string, status followmodels.Status) (int64, error)

	ListFollowers(ctx context.Context, followeeID string, status followmodels.Status, page, limit int) ([]*usermodels.User, int64, error)
	ListFollowing(ctx context.Context, followerID string, status followmodels.Status, page, limit int) ([]*usermodels.User, int64, error)
	Counts(ctx context.Context, userID string) (followmodels.Counts, error)
	// FollowerIDs pages accepted follower ids in id order, starting after the
	// given id. Keyset rather than offset because fan-out reads the whole set
	// and an offset scan would re-read every earlier page.
	FollowerIDs(ctx context.Context, followeeID string, after string, limit int) ([]string, error)
}

type followsRepository struct {
	DBService db.DB
}

// NewFollowsRepository returns the Postgres implementation.
func NewFollowsRepository() FollowsRepository {
	return &followsRepository{DBService: db.GetDBService()}
}

func (r *followsRepository) span(ctx context.Context, op string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	attrs = append(attrs, attribute.String("table", table), attribute.String("operation", op))

	return tracing.GetTracer(ctx).Start(ctx, "repository.follows."+op,
		trace.WithAttributes(attrs...), tracing.GetEnvironmentAttribute())
}

func record(start time.Time, method string, err error) {
	result := metrics.Success
	if err != nil {
		result = metrics.Error
	}
	metrics.GetAppMetrics().DatabaseMetric(float64(time.Since(start).Microseconds())/1000, table, method, result)
}

func (r *followsRepository) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.DBService.GetDB().WithContext(ctx).Transaction(fn)
}

func (r *followsRepository) Get(ctx context.Context, followerID, followeeID string) (*followmodels.UserFollow, error) {
	ctx, span := r.span(ctx, "select", attribute.String("follower.id", followerID), attribute.String("followee.id", followeeID))
	defer span.End()
	start := time.Now()

	var follow followmodels.UserFollow
	err := r.DBService.GetDB().WithContext(ctx).
		Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
		First(&follow).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		record(start, "select", nil)

		return nil, nil
	}
	record(start, "select", err)
	if err != nil {
		span.RecordError(err)

		return nil, err
	}

	return &follow, nil
}

func (r *followsRepository) Create(ctx context.Context, tx *gorm.DB, follow *followmodels.UserFollow) error {
	ctx, span := r.span(ctx, "insert", attribute.String("follower.id", follow.FollowerID), attribute.String("followee.id", follow.FolloweeID))
	defer span.End()
	start := time.Now()

	err := tx.WithContext(ctx).Create(follow).Error
	record(start, "insert", err)
	if err != nil {
		span.RecordError(err)
	}

	return err
}

func (r *followsRepository) Accept(ctx context.Context, tx *gorm.DB, followerID, followeeID string, at time.Time) (int64, error) {
	ctx, span := r.span(ctx, "update", attribute.String("follower.id", followerID), attribute.String("followee.id", followeeID))
	defer span.End()
	start := time.Now()

	result := tx.WithContext(ctx).Model(&followmodels.UserFollow{}).
		Where("follower_id = ? AND followee_id = ? AND status = ?", followerID, followeeID, followmodels.StatusPending).
		Updates(map[string]any{"status": followmodels.StatusAccepted, "accepted_at": at})
	record(start, "update", result.Error)
	if result.Error != nil {
		span.RecordError(result.Error)

		return 0, result.Error
	}

	return result.RowsAffected, nil
}

// Delete removes the edge if it has the given status; an empty status removes
// it whatever its status. The row count tells the caller whether anything was
// there, which is what decides whether an event is written.
func (r *followsRepository) Delete(ctx context.Context, tx *gorm.DB, followerID, followeeID string, status followmodels.Status) (int64, error) {
	ctx, span := r.span(ctx, "delete", attribute.String("follower.id", followerID), attribute.String("followee.id", followeeID))
	defer span.End()
	start := time.Now()

	query := tx.WithContext(ctx).Where("follower_id = ? AND followee_id = ?", followerID, followeeID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	result := query.Delete(&followmodels.UserFollow{})
	record(start, "delete", result.Error)
	if result.Error != nil {
		span.RecordError(result.Error)

		return 0, result.Error
	}

	return result.RowsAffected, nil
}

// listUsers runs one paginated join from user_follows to users. Both list
// methods are the same query with the two id columns swapped, so they share it.
func (r *followsRepository) listUsers(ctx context.Context, byColumn, selectColumn, id string, status followmodels.Status, page, limit int) ([]*usermodels.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	database := r.DBService.GetDB().WithContext(ctx)

	base := database.Table(table).Where(byColumn+" = ? AND status = ?", id, status)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var users []*usermodels.User
	err := database.Table("users").
		Select("users.*").
		Joins("JOIN "+table+" f ON f."+selectColumn+" = users.id").
		Where("f."+byColumn+" = ? AND f.status = ?", id, status).
		Order("f.created_at DESC, users.id ASC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&users).Error
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *followsRepository) ListFollowers(ctx context.Context, followeeID string, status followmodels.Status, page, limit int) ([]*usermodels.User, int64, error) {
	ctx, span := r.span(ctx, "select", attribute.String("followee.id", followeeID))
	defer span.End()
	start := time.Now()

	users, total, err := r.listUsers(ctx, "followee_id", "follower_id", followeeID, status, page, limit)
	record(start, "select", err)
	if err != nil {
		span.RecordError(err)
	}

	return users, total, err
}

func (r *followsRepository) ListFollowing(ctx context.Context, followerID string, status followmodels.Status, page, limit int) ([]*usermodels.User, int64, error) {
	ctx, span := r.span(ctx, "select", attribute.String("follower.id", followerID))
	defer span.End()
	start := time.Now()

	users, total, err := r.listUsers(ctx, "follower_id", "followee_id", followerID, status, page, limit)
	record(start, "select", err)
	if err != nil {
		span.RecordError(err)
	}

	return users, total, err
}

func (r *followsRepository) Counts(ctx context.Context, userID string) (followmodels.Counts, error) {
	ctx, span := r.span(ctx, "select", attribute.String("user.id", userID))
	defer span.End()
	start := time.Now()

	// One query for both directions: a profile header asks for both counts at
	// once, and each is an index-only scan on its own index.
	var counts followmodels.Counts
	err := r.DBService.GetDB().WithContext(ctx).Raw(`
		SELECT
		  (SELECT count(*) FROM `+table+` WHERE followee_id = @id AND status = @accepted) AS followers,
		  (SELECT count(*) FROM `+table+` WHERE follower_id = @id AND status = @accepted) AS following`,
		map[string]any{"id": userID, "accepted": followmodels.StatusAccepted},
	).Scan(&counts).Error
	record(start, "select", err)
	if err != nil {
		span.RecordError(err)

		return followmodels.Counts{}, err
	}

	return counts, nil
}

func (r *followsRepository) FollowerIDs(ctx context.Context, followeeID string, after string, limit int) ([]string, error) {
	ctx, span := r.span(ctx, "select", attribute.String("followee.id", followeeID))
	defer span.End()
	start := time.Now()

	if limit < 1 {
		limit = 500
	}
	var ids []string
	err := r.DBService.GetDB().WithContext(ctx).Model(&followmodels.UserFollow{}).
		Where("followee_id = ? AND status = ? AND follower_id > ?", followeeID, followmodels.StatusAccepted, after).
		Order("follower_id ASC").
		Limit(limit).
		Pluck("follower_id", &ids).Error
	record(start, "select", err)
	if err != nil {
		span.RecordError(err)

		return nil, err
	}

	return ids, nil
}
