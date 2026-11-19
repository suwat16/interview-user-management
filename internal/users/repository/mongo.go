package repository

import (
	"context"
	"errors"
	"fmt"

	"example.com/internal/users"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type UserRepository struct {
	collection *mongo.Collection
}

func NewUserRepository(collection *mongo.Collection) *UserRepository {
	return &UserRepository{collection: collection}
}

func EnsureEmailIndex(ctx context.Context, collection *mongo.Collection) error {
	_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("create unique user email index: %w", err)
	}
	return nil
}

func (r *UserRepository) Find(ctx context.Context) ([]users.User, error) {
	cursor, err := r.collection.Find(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("find users: %w", err)
	}

	result := []users.User{}
	if err := cursor.All(ctx, &result); err != nil {
		return nil, fmt.Errorf("decode users: %w", err)
	}
	return result, nil
}

func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	count, err := r.collection.CountDocuments(ctx, bson.D{}, options.Count().SetHint("_id_"))
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*users.User, error) {
	return r.findOne(ctx, bson.M{"_id": id})
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*users.User, error) {
	return r.findOne(ctx, bson.M{"email": email})
}

func (r *UserRepository) findOne(ctx context.Context, filter bson.M) (*users.User, error) {
	var user users.User
	err := r.collection.FindOne(ctx, filter).Decode(&user)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, users.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) Save(ctx context.Context, user *users.User) error {
	if user == nil {
		return errors.New("save user: user is nil")
	}

	if user.ID == "" {
		user.ID = bson.NewObjectID().Hex()
	}
	filter := bson.M{"_id": user.ID}
	if _, err := r.collection.ReplaceOne(ctx, filter, user, options.Replace().SetUpsert(true)); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return users.ErrEmailTaken
		}
		return fmt.Errorf("save user: %w", err)
	}
	return nil
}

func (r *UserRepository) Update(ctx context.Context, id string, fields users.UpdateFields) (*users.User, error) {
	set := bson.D{}
	if fields.Name != nil {
		set = append(set, bson.E{Key: "name", Value: *fields.Name})
	}
	if fields.Email != nil {
		set = append(set, bson.E{Key: "email", Value: *fields.Email})
	}

	var user users.User
	err := r.collection.FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{{Key: "$set", Value: set}},
		options.FindOneAndUpdate().SetReturnDocument(options.After).SetUpsert(false),
	).Decode(&user)
	switch {
	case errors.Is(err, mongo.ErrNoDocuments):
		return nil, users.ErrUserNotFound
	case mongo.IsDuplicateKeyError(err):
		return nil, users.ErrEmailTaken
	case err != nil:
		return nil, fmt.Errorf("update user: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) Delete(ctx context.Context, user *users.User) error {
	if user == nil {
		return errors.New("delete user: user is nil")
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": user.ID})
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if result.DeletedCount == 0 {
		return users.ErrUserNotFound
	}
	return nil
}
