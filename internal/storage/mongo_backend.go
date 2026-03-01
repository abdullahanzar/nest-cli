package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/platanist/nest-cli/internal/api"
	"github.com/platanist/nest-cli/internal/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const defaultMongoDatabase = "nest_cli"

type mongoBackend struct {
	originName string
	db         *mongo.Database
}

type keyDoc struct {
	KeyID       string     `bson:"keyId"`
	Profile     string     `bson:"profile"`
	Fingerprint string     `bson:"fingerprint"`
	PublicKey   string     `bson:"publicKey"`
	CreatedAt   time.Time  `bson:"createdAt"`
	UpdatedAt   time.Time  `bson:"updatedAt"`
	LastUsedAt  *time.Time `bson:"lastUsedAt,omitempty"`
	RevokedAt   *time.Time `bson:"revokedAt,omitempty"`
}

type secretDoc struct {
	Origin       string    `bson:"origin"`
	Application  string    `bson:"application"`
	Version      int       `bson:"version"`
	Envelope     string    `bson:"envelope"`
	Profile      string    `bson:"profile"`
	KeyID        string    `bson:"keyId"`
	Fingerprint  string    `bson:"fingerprint"`
	ChecksumSHA  string    `bson:"checksumSha256"`
	ContentBytes int       `bson:"contentBytes"`
	CreatedAt    time.Time `bson:"createdAt"`
}

func newMongoBackend(originName string, origin config.Origin) (Backend, error) {
	mongoURI := config.ResolveMongoURI(originName, origin)
	if mongoURI == "" {
		return nil, fmt.Errorf("origin %q mongo mode requires mongo_uri", originName)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, fmt.Errorf("connect mongo for origin %q: %w", originName, err)
	}

	dbName := origin.MongoDatabase
	if dbName == "" {
		dbName = defaultMongoDatabase
	}

	backend := &mongoBackend{
		originName: originName,
		db:         client.Database(dbName),
	}

	if err := backend.ensureIndexes(ctx); err != nil {
		return nil, err
	}

	return backend, nil
}

func (b *mongoBackend) ensureIndexes(ctx context.Context) error {
	secrets := b.db.Collection("nest_cli_secrets")
	keys := b.db.Collection("nest_cli_keys")
	versions := b.db.Collection("nest_cli_versions")

	if _, err := secrets.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "origin", Value: 1}, {Key: "application", Value: 1}, {Key: "version", Value: -1}}}); err != nil {
		return fmt.Errorf("create secrets index: %w", err)
	}
	if _, err := keys.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "origin", Value: 1}, {Key: "keyId", Value: 1}}, Options: options.Index().SetUnique(true)}); err != nil {
		return fmt.Errorf("create keys index: %w", err)
	}
	if _, err := versions.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "_id", Value: 1}}}); err != nil {
		return fmt.Errorf("create versions index: %w", err)
	}

	return nil
}

func (b *mongoBackend) PushSecret(ctx context.Context, req api.PushSecretRequest) (api.PushSecretResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	version, err := b.nextVersion(ctx, req.Origin, req.Application)
	if err != nil {
		return api.PushSecretResponse{}, err
	}

	now := time.Now().UTC()
	if _, err := b.db.Collection("nest_cli_keys").UpdateOne(
		ctx,
		bson.M{"origin": b.originName, "keyId": req.KeyID},
		bson.M{
			"$setOnInsert": bson.M{"createdAt": now},
			"$set": bson.M{
				"origin":      b.originName,
				"profile":     req.Profile,
				"fingerprint": req.Fingerprint,
				"updatedAt":   now,
				"lastUsedAt":  now,
			},
		},
		options.Update().SetUpsert(true),
	); err != nil {
		return api.PushSecretResponse{}, fmt.Errorf("upsert key metadata: %w", err)
	}

	doc := secretDoc{
		Origin:       req.Origin,
		Application:  req.Application,
		Version:      version,
		Envelope:     req.Envelope,
		Profile:      req.Profile,
		KeyID:        req.KeyID,
		Fingerprint:  req.Fingerprint,
		ChecksumSHA:  req.ChecksumSHA,
		ContentBytes: req.ContentBytes,
		CreatedAt:    now,
	}
	if _, err := b.db.Collection("nest_cli_secrets").InsertOne(ctx, doc); err != nil {
		return api.PushSecretResponse{}, fmt.Errorf("insert encrypted secret: %w", err)
	}

	return api.PushSecretResponse{Version: version}, nil
}

func (b *mongoBackend) PullSecret(ctx context.Context, req api.PullSecretRequest) (api.PullSecretResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var latest secretDoc
	err := b.db.Collection("nest_cli_secrets").FindOne(
		ctx,
		bson.M{"origin": req.Origin, "application": req.Application},
		options.FindOne().SetSort(bson.D{{Key: "version", Value: -1}}),
	).Decode(&latest)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return api.PullSecretResponse{}, fmt.Errorf("no secret found for origin/application")
		}
		return api.PullSecretResponse{}, fmt.Errorf("query encrypted secret: %w", err)
	}

	if latest.KeyID != req.KeyID {
		return api.PullSecretResponse{}, fmt.Errorf("key mismatch for origin/application")
	}
	if latest.Fingerprint != req.Fingerprint {
		return api.PullSecretResponse{}, fmt.Errorf("fingerprint mismatch for origin/application")
	}

	var key keyDoc
	err = b.db.Collection("nest_cli_keys").FindOne(
		ctx,
		bson.M{"origin": b.originName, "keyId": req.KeyID},
	).Decode(&key)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return api.PullSecretResponse{}, fmt.Errorf("key %q is not registered for origin %q", req.KeyID, b.originName)
		}
		return api.PullSecretResponse{}, fmt.Errorf("query key metadata: %w", err)
	}
	if key.RevokedAt != nil {
		return api.PullSecretResponse{}, fmt.Errorf("key %q has been revoked and cannot be used for pull", req.KeyID)
	}

	return api.PullSecretResponse{
		Envelope:    latest.Envelope,
		Version:     latest.Version,
		KeyID:       latest.KeyID,
		Profile:     latest.Profile,
		Fingerprint: latest.Fingerprint,
	}, nil
}

func (b *mongoBackend) RegisterKey(ctx context.Context, req api.RegisterKeyRequest) (api.RegisterKeyResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	now := time.Now().UTC()
	if _, err := b.db.Collection("nest_cli_keys").UpdateOne(
		ctx,
		bson.M{"origin": b.originName, "keyId": req.KeyID},
		bson.M{
			"$setOnInsert": bson.M{"createdAt": now},
			"$set": bson.M{
				"origin":      b.originName,
				"profile":     req.Profile,
				"fingerprint": req.Fingerprint,
				"publicKey":   req.PublicKey,
				"updatedAt":   now,
				"revokedAt":   nil,
			},
		},
		options.Update().SetUpsert(true),
	); err != nil {
		return api.RegisterKeyResponse{}, fmt.Errorf("register key: %w", err)
	}

	return api.RegisterKeyResponse{Status: true, Reason: "key registered"}, nil
}

func (b *mongoBackend) RevokeKey(ctx context.Context, keyID string) (api.RegisterKeyResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	now := time.Now().UTC()
	result, err := b.db.Collection("nest_cli_keys").UpdateOne(
		ctx,
		bson.M{"origin": b.originName, "keyId": keyID, "revokedAt": bson.M{"$eq": nil}},
		bson.M{
			"$set": bson.M{"revokedAt": now, "updatedAt": now},
		},
	)
	if err != nil {
		return api.RegisterKeyResponse{}, fmt.Errorf("revoke key: %w", err)
	}
	if result.MatchedCount == 0 {
		return api.RegisterKeyResponse{}, fmt.Errorf("key %q not found or already revoked", keyID)
	}

	return api.RegisterKeyResponse{Status: true, Reason: "key revoked"}, nil
}

func (b *mongoBackend) ListRemoteKeys(ctx context.Context) (api.ListKeysResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cursor, err := b.db.Collection("nest_cli_keys").Find(
		ctx,
		bson.M{"origin": b.originName},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		return api.ListKeysResponse{}, fmt.Errorf("list keys: %w", err)
	}
	defer cursor.Close(ctx)

	keys := make([]api.RemoteKey, 0)
	for cursor.Next(ctx) {
		var doc keyDoc
		if err := cursor.Decode(&doc); err != nil {
			return api.ListKeysResponse{}, fmt.Errorf("decode key: %w", err)
		}
		entry := api.RemoteKey{
			KeyID:       doc.KeyID,
			Profile:     doc.Profile,
			Fingerprint: doc.Fingerprint,
			PublicKey:   doc.PublicKey,
			CreatedAt:   doc.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   doc.UpdatedAt.Format(time.RFC3339),
		}
		if doc.LastUsedAt != nil {
			entry.LastUsedAt = doc.LastUsedAt.Format(time.RFC3339)
		}
		if doc.RevokedAt != nil {
			entry.RevokedAt = doc.RevokedAt.Format(time.RFC3339)
		}
		keys = append(keys, entry)
	}
	if err := cursor.Err(); err != nil {
		return api.ListKeysResponse{}, fmt.Errorf("iterate keys: %w", err)
	}

	return api.ListKeysResponse{Status: true, Keys: keys}, nil
}

func (b *mongoBackend) HealthCheck(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := b.db.Client().Ping(ctx, nil); err != nil {
		return "", fmt.Errorf("mongo ping failed: %w", err)
	}

	return fmt.Sprintf("mongo: %s/%s (ok)", b.originName, b.db.Name()), nil
}

func (b *mongoBackend) nextVersion(ctx context.Context, origin string, application string) (int, error) {
	var out struct {
		Version int `bson:"version"`
	}

	err := b.db.Collection("nest_cli_versions").FindOneAndUpdate(
		ctx,
		bson.M{"_id": origin + "::" + application},
		bson.M{
			"$setOnInsert": bson.M{"createdAt": time.Now().UTC()},
			"$inc":         bson.M{"version": 1},
			"$set":         bson.M{"updatedAt": time.Now().UTC()},
		},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	).Decode(&out)
	if err != nil {
		return 0, fmt.Errorf("increment secret version: %w", err)
	}

	if out.Version <= 0 {
		return 0, fmt.Errorf("invalid computed version")
	}
	return out.Version, nil
}
