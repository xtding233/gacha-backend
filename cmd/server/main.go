package main

import (
	"log"
	"net"

	"google.golang.org/grpc"

	// generated stubs
	gachav1 "github.com/xtding233/gacha-backend/gen/gacha/v1"
	gamev1 "github.com/xtding233/gacha-backend/gen/game/v1"
)

// ---- Minimal server implementations ----

// GachaServer implements gachav1.GachaServiceServer
type GachaServer struct {
	gachav1.UnimplementedGachaServiceServer
	// add fields: loader, resolver, engine, etc.
}

// GameServer implements gamev1.GameServiceServer
type GameServer struct {
	gamev1.UnimplementedGameServiceServer
	// add fields: loader, resolver, etc.
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	// Register services
	gachav1.RegisterGachaServiceServer(grpcServer, &GachaServer{})
	gamev1.RegisterGameServiceServer(grpcServer, &GameServer{})

	log.Println("gRPC server listening on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
