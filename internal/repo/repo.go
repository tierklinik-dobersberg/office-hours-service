package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	commonv1 "github.com/tierklinik-dobersberg/apis/gen/go/tkd/common/v1"
	office_hoursv1 "github.com/tierklinik-dobersberg/apis/gen/go/tkd/office_hours/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrNotFound = errors.New("office-hour not found")

type Repo struct {
	col *mongo.Collection
}

func NewRepo(ctx context.Context, url string, db string) (*Repo, error) {
	clientOptions := options.Client().ApplyURI(url)

	cli, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	r := &Repo{
		col: cli.Database(db).Collection("office-hours"),
	}

	return r, nil
}

func (r *Repo) UpsertOfficeHours(ctx context.Context, pb *office_hoursv1.OfficeHour) (*office_hoursv1.OfficeHour, error) {
	model, err := ModelFromProto(pb)
	if err != nil {
		return nil, err
	}

	if model.ID.IsZero() {
		model.ID = primitive.NewObjectIDFromTimestamp(time.Now())
	}

	replaceOptions := options.FindOneAndReplace().
		SetUpsert(true).
		SetReturnDocument(options.After)

	res := r.col.FindOneAndReplace(ctx, bson.M{
		"_id": model.ID,
	}, model, replaceOptions)

	if res.Err() != nil {
		return nil, fmt.Errorf("failed to perform findAndReplace operation: %w", err)
	}

	var newModel OfficeHourModel

	if err := res.Decode(&newModel); err != nil {
		return nil, fmt.Errorf("failed to decode new office-hour document: %w", err)
	}

	return newModel.ToProto(), nil
}

func (r *Repo) ListOfficeHours(ctx context.Context) ([]*office_hoursv1.OfficeHour, error) {
	return r.find(ctx, bson.M{})
}

func (r *Repo) DeleteOfficeHour(ctx context.Context, name string) error {
	oid, err := primitive.ObjectIDFromHex(name)
	if err != nil {
		return fmt.Errorf("invalid office-hour name: %w", err)
	}

	res, err := r.col.DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ErrNotFound
		}

		return err
	}

	if res.DeletedCount == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *Repo) FindByDate(ctx context.Context, date *commonv1.Date) ([]*office_hoursv1.OfficeHour, error) {
	return r.find(ctx, bson.M{
		"date": bson.M{
			"$in": bson.A{
				date.AsTime().Format("01-02"),
				date.AsTime().Format("2006-01-02"),
			},
		},
	})
}

func (r *Repo) FindByWeekday(ctx context.Context, weekday time.Weekday) ([]*office_hoursv1.OfficeHour, error) {
	return r.find(ctx, bson.M{
		"dayOfWeek": weekday,
	})
}

func (r *Repo) find(ctx context.Context, filter bson.M) ([]*office_hoursv1.OfficeHour, error) {
	res, err := r.col.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	var models []OfficeHourModel

	if err := res.All(ctx, &models); err != nil {
		return nil, fmt.Errorf("failed to decode office-hour documents: %w", err)
	}

	pbRes := make([]*office_hoursv1.OfficeHour, len(models))
	for idx, m := range models {
		pbRes[idx] = m.ToProto()
	}

	return pbRes, nil
}
