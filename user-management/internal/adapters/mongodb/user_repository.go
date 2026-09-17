package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/user"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const usersCollection = "users"

type userDocument struct {
	ID           bson.ObjectID `bson:"_id,omitempty"`
	Name         string        `bson:"name"`
	Email        string        `bson:"email"`
	PasswordHash string        `bson:"password_hash"`
	CreatedAt    time.Time     `bson:"created_at"`
}

type UserRepository struct {
	collection *mongo.Collection
}

func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{
		collection: db.Collection(usersCollection),
	}
}

func (r *UserRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateOne(
		ctx,
		mongo.IndexModel{
			Keys: bson.D{
				{Key: "email", Value: 1},
			},
			Options: options.Index().
				SetName("uniq_users_email").
				SetUnique(true),
		},
	)
	if err != nil {
		return fmt.Errorf("create user email index: %w", err)
	}

	return nil
}

func (r *UserRepository) Create(
	ctx context.Context,
	u *user.User,
) error {
	doc := userDocument{
		Name:         u.Name,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		CreatedAt:    u.CreatedAt,
	}

	result, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return user.ErrEmailAlreadyExists
		}

		return fmt.Errorf("insert user: %w", err)
	}

	id, ok := result.InsertedID.(bson.ObjectID)
	if !ok {
		return fmt.Errorf(
			"unexpected inserted user id type: %T",
			result.InsertedID,
		)
	}

	u.ID = id.Hex()

	return nil
}

func (r *UserRepository) FindByEmail(
	ctx context.Context,
	email string,
) (*user.User, error) {
	var doc userDocument

	err := r.collection.FindOne(
		ctx,
		bson.D{
			{Key: "email", Value: email},
		},
	).Decode(&doc)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, user.ErrNotFound
		}

		return nil, fmt.Errorf("find user by email: %w", err)
	}

	return toUser(doc), nil
}

func (r *UserRepository) FindByID(
	ctx context.Context,
	id string,
) (*user.User, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, user.ErrNotFound
	}

	var doc userDocument

	err = r.collection.FindOne(
		ctx,
		bson.D{
			{Key: "_id", Value: objectID},
		},
	).Decode(&doc)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, user.ErrNotFound
		}

		return nil, fmt.Errorf("find user by id: %w", err)
	}

	return toUser(doc), nil
}

func (r *UserRepository) List(
	ctx context.Context,
) ([]*user.User, error) {
	cursor, err := r.collection.Find(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []userDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode users: %w", err)
	}

	users := make([]*user.User, 0, len(docs))

	for _, doc := range docs {
		users = append(users, toUser(doc))
	}

	return users, nil
}

func (r *UserRepository) UpdateByID(
	ctx context.Context,
	id string,
	update user.Update,
) error {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return user.ErrNotFound
	}

	set := bson.D{}

	if update.Name != nil {
		set = append(set, bson.E{
			Key:   "name",
			Value: *update.Name,
		})
	}

	if update.Email != nil {
		set = append(set, bson.E{
			Key:   "email",
			Value: *update.Email,
		})
	}

	if len(set) == 0 {
		return nil
	}

	result, err := r.collection.UpdateOne(
		ctx,
		bson.D{
			{Key: "_id", Value: objectID},
		},
		bson.D{
			{Key: "$set", Value: set},
		},
	)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return user.ErrEmailAlreadyExists
		}

		return fmt.Errorf("update user by id: %w", err)
	}

	if result.MatchedCount == 0 {
		return user.ErrNotFound
	}

	return nil
}

func (r *UserRepository) DeleteByID(
	ctx context.Context,
	id string,
) error {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return user.ErrNotFound
	}

	result, err := r.collection.DeleteOne(
		ctx,
		bson.D{
			{Key: "_id", Value: objectID},
		},
	)
	if err != nil {
		return fmt.Errorf("delete user by id: %w", err)
	}

	if result.DeletedCount == 0 {
		return user.ErrNotFound
	}

	return nil
}

func (r *UserRepository) Count(
	ctx context.Context,
) (int64, error) {
	count, err := r.collection.CountDocuments(ctx, bson.D{})
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}

	return count, nil
}

func toUser(doc userDocument) *user.User {
	return &user.User{
		ID:           doc.ID.Hex(),
		Name:         doc.Name,
		Email:        doc.Email,
		PasswordHash: doc.PasswordHash,
		CreatedAt:    doc.CreatedAt,
	}
}
