package trackerinfogrpc

import (
	"context"

	trackerinfov1 "github.com/MRibalko/smogtracker/protos/gen/go"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TrackerInfo interface {
	InfoList(ctx context.Context) ([]models.Tracker, error)
}

type serverAPI struct {
	trackerinfov1.UnimplementedTrackerInfoServer
	infoService TrackerInfo
}

func Register(gRPCServer *grpc.Server, infoService TrackerInfo) {
	trackerinfov1.RegisterTrackerInfoServer(gRPCServer, &serverAPI{infoService: infoService})
}

func (s *serverAPI) Full(
	ctx context.Context,
	in *trackerinfov1.TrackerInfoRequest,
) (*trackerinfov1.TrackerFullInfoResponse, error) {

	list, err := s.infoService.InfoList(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "no data")
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
	return &trackerinfov1.TrackerFullInfoResponse{Result: result}, nil
}

func (s *serverAPI) Short(
	ctx context.Context,
	in *trackerinfov1.TrackerInfoRequest,
) (*trackerinfov1.TrackerShortInfoResponse, error) {
	list, err := s.infoService.InfoList(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "no data")
	}

	var result []*trackerinfov1.TrackerShortInfo
	for _, v := range list {
		info := trackerinfov1.TrackerShortInfo{
			OrigId: v.OrigId,
			Source: v.Source,
		}
		result = append(result, &info)
	}
	return &trackerinfov1.TrackerShortInfoResponse{Result: result}, nil
}
