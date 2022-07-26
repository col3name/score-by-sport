package grpc

import (
	"github.com/col3name/lines/pkg/common/domain"
	pb "github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/transport/grpc/proto"
)

type ResponseSenderGrpc struct {
	Stream pb.KiddyLineProcessor_SubscribeOnSportsLinesServer
}

func (s *ResponseSenderGrpc) Send(sports []*domain.SportLine) error {
	var list []*pb.Sport
	for _, sport := range sports {
		list = append(list, &pb.Sport{
			Type: sport.Type.String(),
			Line: sport.Score,
		})
	}
	response := &pb.SubscribeResponse{Sports: list}
	return s.Stream.Send(response)
}
