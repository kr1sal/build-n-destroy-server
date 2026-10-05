package game

import (
	"context"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/golang/geo/r3"
	"github.com/google/uuid"
	"kr1sal.com/build-n-destroy/internal/pb"
)

const (
	TPS = 60
	STPS = 20
	MoveSpeed = 5.0
)

type Client struct {
	ID string
	Inbound chan *pb.ClientEvent
	Outbound chan *pb.ServerEvent
}

type Player struct {
	ClientID string
	Position r3.Vector
	MoveDir r3.Vector
	Yaw float32
	Pitch float32
	Health int
}

type Room struct {
	mu sync.Mutex
	players map[string]*Player
	clients map[string]*Client
}

func NewRoom() *Room {
	return &Room{
		players: make(map[string]*Player),
		clients: make(map[string]*Client),
	}
}

func (r *Room) addPlayerLocked(clientID string) *Player {
	r.players[clientID] = &Player{
		ClientID: clientID,
		Position: r3.Vector{
			X: rand.Float64() * 100,
			Y: rand.Float64() * 5,
			Z: rand.Float64() * 100},
		Health:   100,
	}
	return r.players[clientID]
}

func (r *Room) removePlayerLocked(clientID string) {
	delete(r.players, clientID)
}

func (r *Room) AddClient() *Client {
	r.mu.Lock()
	defer r.mu.Unlock()
	clientID := uuid.NewString()
	r.clients[clientID] = &Client{
		ID:       clientID,
		Inbound:  make(chan *pb.ClientEvent, 200),
		Outbound: make(chan *pb.ServerEvent, 200),
	}
	r.addPlayerLocked(clientID)
	return r.clients[clientID]
}

func (r *Room) RemoveClient(clientID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.removePlayerLocked(clientID)
	delete(r.clients, clientID)
}


func (r *Room) Run(ctx context.Context) {
	ticks := uint32(0)

	dtf := time.Second / 60
	accumulator := time.Duration(0)
	last := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		now := time.Now()
		accumulator += now.Sub(last)
		last = now

		for accumulator >= dtf {
			r.processClientEvents()
			if (ticks % (TPS / STPS)) == 0 {
				gameState := r.buildGameState(ticks)
				r.BroadcastGameState(&gameState)
			}
			r.simulateTick()
			ticks++
			accumulator -= dtf

		}

		time.Sleep(time.Millisecond)
	}
}

func (r *Room) simulateTick() {
	r.mu.Lock()
	defer r.mu.Unlock()
	dt := 1.0 / float64(TPS)
	for _, p := range r.players {
		d := p.MoveDir
		if n := d.Norm(); n > 1 {
			d = d.Mul(1 / n)
		}
		p.Position = p.Position.Add(d.Mul(MoveSpeed * dt))
	}
}

func (r *Room) processClientEvents() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for clientID, client := range r.clients {
		player := r.players[clientID]
	processing:
		for {
			select {
			case event := <-client.Inbound:
				switch e := event.Event.(type) {
				case *pb.ClientEvent_Join:
					log.Printf("PLayer (%v) joined!", clientID)
				case *pb.ClientEvent_PlayerInput:
					if player == nil {
						continue
					}
					if d := e.PlayerInput.MovementDirection; d != nil {
						player.MoveDir = r3.Vector{
							X: float64(d.X),
							Y: float64(d.Y),
							Z: float64(d.Z),
						}
					}
					if o := e.PlayerInput.CameraOrientation; o != nil {
						player.Yaw = o.Yaw
						player.Pitch = o.Pitch
					}
					log.Printf("PLayer (%v) moved!", clientID)
				case *pb.ClientEvent_ChatMessage:
				default:
				}
			default:
				break processing // ahuet, eto goto
			}
		}
	}
}

func (r *Room) buildGameState(ticks uint32) pb.GameState {
	r.mu.Lock()
	defer r.mu.Unlock()

	players := make([]*pb.PlayerState, 0, len(r.players))
	for _, player := range r.players {
		players = append(players, &pb.PlayerState{
			PlayerId: player.ClientID,
			Position: &pb.Vector3{
				X: float32(player.Position.X),
				Y: float32(player.Position.Y),
				Z: float32(player.Position.Z),
			},
			CameraOrientation: &pb.Orientation{Yaw: player.Yaw, Pitch: player.Pitch},
		})
	}

	return pb.GameState{
		Tick: ticks,
		Players: players,
	}
}

func (r *Room) BroadcastGameState(state *pb.GameState) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, client := range r.clients {
		select {
		case client.Outbound <- &pb.ServerEvent{Event: &pb.ServerEvent_GameState{GameState: state}}:
		default:
		}
	}
}
