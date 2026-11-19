package repository_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"example.com/internal/users"
	"example.com/internal/users/repository"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/drivertest"
)

const collectionName = "users"

var storedUser = users.User{ID: "user-id", Name: "Ada", Email: "ada@example.com", Password: "stored-hash"}

func TestEnsureEmailIndex(t *testing.T) {
	t.Parallel()
	collection, command := mockCollection(t, "createIndexes", bson.D{{Key: "ok", Value: 1}})
	if err := repository.EnsureEmailIndex(t.Context(), collection); err != nil {
		t.Fatal(err)
	}
	assertCommandValue(t, *command, int32(1), "indexes", "0", "key", "email")
	assertCommandValue(t, *command, true, "indexes", "0", "unique")
}

func TestFindOne(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		find  func(*repository.UserRepository, context.Context, string) (*users.User, error)
		field string
		value string
	}{
		{"by ID", (*repository.UserRepository).FindByID, "_id", storedUser.ID},
		{"by email", (*repository.UserRepository).FindByEmail, "email", storedUser.Email},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			t.Run("found", func(t *testing.T) {
				collection, command := mockCollection(t, "find", cursorResponse(storedUser))
				got, err := tc.find(repository.NewUserRepository(collection), t.Context(), tc.value)
				if err != nil || got == nil || *got != storedUser {
					t.Fatalf("find returned user %v and error %v, want stored user", got, err)
				}
				assertCommandValue(t, *command, tc.value, "filter", tc.field)
			})
			t.Run("missing", func(t *testing.T) {
				collection, _ := mockCollection(t, "find", cursorResponse())
				got, err := tc.find(repository.NewUserRepository(collection), t.Context(), tc.value)
				if got != nil || !errors.Is(err, users.ErrUserNotFound) {
					t.Fatalf("find returned user %v and error %v, want ErrUserNotFound", got, err)
				}
			})
		})
	}
}

func TestFind(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		docs []any
	}{
		{"lists users", []any{storedUser}},
		{"returns empty list", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			collection, _ := mockCollection(t, "find", cursorResponse(tc.docs...))
			got, err := repository.NewUserRepository(collection).Find(t.Context())
			if err != nil || got == nil || len(got) != len(tc.docs) {
				t.Fatalf("Find returned %d users and error %v, want %d users", len(got), err, len(tc.docs))
			}
			if len(got) == 1 && got[0] != storedUser {
				t.Fatal("Find did not decode the stored user")
			}
		})
	}
}

func TestCount(t *testing.T) {
	t.Parallel()
	reply := cursorResponse(bson.D{{Key: "n", Value: int64(3)}})
	collection, command := mockCollection(t, "aggregate", reply)
	got, err := repository.NewUserRepository(collection).Count(t.Context())
	if err != nil || got != 3 {
		t.Fatalf("Count = %d, err = %v, want 3", got, err)
	}
	assertCommandValue(t, *command, "_id_", "hint")
}

func TestSave(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		reply bson.D
		want  error
	}{
		{"assigns ID and upserts", bson.D{{Key: "ok", Value: 1}, {Key: "n", Value: 1}}, nil},
		{"maps duplicate key", bson.D{{Key: "ok", Value: 1}, {Key: "writeErrors", Value: bson.A{
			bson.D{{Key: "index", Value: 0}, {Key: "code", Value: 11000}, {Key: "errmsg", Value: "duplicate email"}},
		}}}, users.ErrEmailTaken},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			collection, command := mockCollection(t, "update", tc.reply)
			user := storedUser
			user.ID = ""
			err := repository.NewUserRepository(collection).Save(t.Context(), &user)
			if !errors.Is(err, tc.want) {
				t.Fatalf("Save error = %v, want %v", err, tc.want)
			}
			if _, err := bson.ObjectIDFromHex(user.ID); err != nil {
				t.Fatalf("Save generated invalid ObjectID: %v", err)
			}
			assertCommandValue(t, *command, user.ID, "updates", "0", "q", "_id")
			assertCommandValue(t, *command, user.ID, "updates", "0", "u", "_id")
			assertCommandValue(t, *command, user.Email, "updates", "0", "u", "email")
			assertCommandValue(t, *command, true, "updates", "0", "upsert")
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	updated := storedUser
	updated.Email = "grace@example.com"
	for _, tc := range []struct {
		name  string
		reply bson.D
		want  error
	}{
		{"updates email", bson.D{{Key: "ok", Value: 1}, {Key: "value", Value: updated}}, nil},
		{"missing user", bson.D{{Key: "ok", Value: 1}, {Key: "value", Value: nil}}, users.ErrUserNotFound},
		{"duplicate email", bson.D{{Key: "ok", Value: 0}, {Key: "code", Value: 11000}, {Key: "errmsg", Value: "duplicate email"}}, users.ErrEmailTaken},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			collection, command := mockCollection(t, "findAndModify", tc.reply)
			got, err := repository.NewUserRepository(collection).Update(t.Context(), storedUser.ID, users.UpdateFields{Email: &updated.Email})
			if !errors.Is(err, tc.want) {
				t.Fatalf("Update error = %v, want %v", err, tc.want)
			}
			if (tc.want == nil && (got == nil || *got != updated)) || (tc.want != nil && got != nil) {
				t.Fatal("Update returned unexpected user")
			}
			assertCommandValue(t, *command, storedUser.ID, "query", "_id")
			assertCommandValue(t, *command, updated.Email, "update", "$set", "email")
			if _, err := command.LookupErr("update", "$set", "name"); err == nil {
				t.Error("email-only update also changed name")
			}
			assertCommandValue(t, *command, true, "new")
			assertCommandValue(t, *command, false, "upsert")
		})
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		deleted int
		want    error
	}{
		{"deletes user", 1, nil},
		{"missing user", 0, users.ErrUserNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			reply := bson.D{{Key: "ok", Value: 1}, {Key: "n", Value: tc.deleted}}
			collection, command := mockCollection(t, "delete", reply)
			err := repository.NewUserRepository(collection).Delete(t.Context(), &users.User{ID: storedUser.ID})
			if !errors.Is(err, tc.want) {
				t.Fatalf("Delete error = %v, want %v", err, tc.want)
			}
			assertCommandValue(t, *command, storedUser.ID, "deletes", "0", "q", "_id")
		})
	}
}

func mockCollection(t *testing.T, commandName string, reply bson.D) (*mongo.Collection, *bson.Raw) {
	t.Helper()
	command := new(bson.Raw)
	opts := options.Client().SetMonitor(&event.CommandMonitor{
		Started: func(_ context.Context, event *event.CommandStartedEvent) {
			if event.CommandName == commandName {
				*command = append(bson.Raw(nil), event.Command...)
			}
		},
	})
	opts.Deployment = drivertest.NewMockDeployment(reply)
	client, err := mongo.Connect(opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Disconnect(context.Background()); err != nil {
			t.Error(err)
		}
	})
	return client.Database("test").Collection(collectionName), command
}

func cursorResponse(docs ...any) bson.D {
	return bson.D{{Key: "ok", Value: 1}, {Key: "cursor", Value: bson.D{
		{Key: "id", Value: int64(0)},
		{Key: "ns", Value: "test." + collectionName},
		{Key: "firstBatch", Value: append(bson.A{}, docs...)},
	}}}
}

func assertCommandValue(t *testing.T, command bson.Raw, want any, path ...string) {
	t.Helper()
	got, err := command.LookupErr(path...)
	if err != nil {
		t.Fatalf("command field %v missing: %v", path, err)
	}
	kind, value, err := bson.MarshalValue(want)
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != kind || !bytes.Equal(got.Value, value) {
		t.Errorf("command field %v = %v, want %v", path, got, want)
	}
}
