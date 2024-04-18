package trackerinfogrpc

import (
	"context"
	"time"

	"github.com/MRibalko/smogtracker/protos/gen/trackerinfov1"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TrackerInfo interface {
	Sources(ctx context.Context) ([]string, error)
	IdsBySource(ctx context.Context, source string) ([]string, error)
	List(ctx context.Context) ([]models.Tracker, error)
	ListSince(ctx context.Context, modifiedFrom time.Time) ([]models.Tracker, error)
}

type serverAPI struct {
	trackerinfov1.UnimplementedTrackerInfoServer
	infoService TrackerInfo
}

func Register(gRPCServer *grpc.Server, infoService TrackerInfo) {
	trackerinfov1.RegisterTrackerInfoServer(gRPCServer, &serverAPI{infoService: infoService})
}

func (s *serverAPI) Sources(
	ctx context.Context,
	in *trackerinfov1.EmptyRequest,
) (*trackerinfov1.SourcesResponse, error) {
	sources, err := s.infoService.Sources(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "storage error")
	}
	if len(sources) == 0 {
		return nil, status.Error(codes.NotFound, "no data")
	}
	return &trackerinfov1.SourcesResponse{Result: sources}, nil
}

func (s *serverAPI) IdsBySource(
	ctx context.Context,
	in *trackerinfov1.SourceRequest,
) (*trackerinfov1.IdsBySourceResponse, error) {
	if len(in.Source) == 0 {
		return nil, status.Error(codes.InvalidArgument, "source is empty")
	}
	ids, err := s.infoService.IdsBySource(ctx, in.Source)
	if err != nil {
		return nil, status.Error(codes.Internal, "storage error")
	}

	if len(ids) == 0 {
		return nil, status.Error(codes.NotFound, "no source")
	}
	return &trackerinfov1.IdsBySourceResponse{Result: ids}, nil
}

func (s *serverAPI) List(
	ctx context.Context,
	in *trackerinfov1.ModifiedFromRequest,
) (*trackerinfov1.FullInfoResponse, error) {

	var (
		list []models.Tracker
		err  error
	)

	if err = in.From.CheckValid(); err != nil {
		list, err = s.infoService.List(ctx)
	} else {
		list, err = s.infoService.ListSince(ctx, in.From.AsTime())
	}

	if err != nil {
		return nil, status.Error(codes.Internal, "storage error")
	}

	if len(list) == 0 {
		return nil, status.Error(codes.NotFound, "no data")
	}

	var result []*trackerinfov1.TrackerFullInfo
	for _, v := range list {
		info := trackerinfov1.TrackerFullInfo{
			OrigId:      v.OrigId,
			Source:      v.Source,
			Description: v.Description,
			Latitude:    v.Latitude,
			Longitude:   v.Longitude,
		}
		result = append(result, &info)
	}
	return &trackerinfov1.FullInfoResponse{Result: result}, nil
}
