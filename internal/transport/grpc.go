package transport

import (
	"fmt"
	"io"
	"log"

	"kr1sal.com/build-n-destroy/internal/game"
	"kr1sal.com/build-n-destroy/internal/pb"
)

type GameServer struct {
	pb.UnimplementedGameServiceServer
	room *game.Room
}

func NewGameServer(room *game.Room) *GameServer {
	return &GameServer{room: room}
}

func (s *GameServer) Play(stream pb.GameService_PlayServer) error {
	// Receive the join message first
	in, err := stream.Recv();
	if err == io.EOF {
		return nil
	}
	if _, ok := in.Event.(*pb.ClientEvent_Join); !ok {
		return fmt.Errorf("First message must be Join")
	}

	client := s.room.AddClient()
	defer s.room.RemoveClient(client.ID)
	// send Join Acknowlege
	stream.Send(&pb.ServerEvent{
		Event: &pb.ServerEvent_JoinAck{
			JoinAck: &pb.JoinAck{PlayerId: client.ID},
		},
	})

	done := make(chan error, 2)

	// Sender
	go func() {
		for {
			select {
			case <-stream.Context().Done():
				done <- stream.Context().Err()
				return
			case e, ok := <-client.Outbound:
				if !ok {
					done <- nil
					return
				}
				if err := stream.Send(e); err != nil {
					log.Printf("Send to client %v: %v", client.ID, err)
					done <- err
					return
				}
			}
		}
	}()

	// Receiver
	go func() {
		for {
			in, err := stream.Recv();
			if err != nil {
				log.Printf("Receive from client %v: %v", client.ID, err)
            	done <- err
				return
			}
        
			select {
			case <-stream.Context().Done():
				done <- stream.Context().Err()
				return
			case client.Inbound <- in:
			}
		}
	}()

	e := <-done
	if e == io.EOF {
		return nil
	}
	return e
}
