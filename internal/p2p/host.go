package p2p

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
)

func NewHost(ctx context.Context) (host.Host, error) {
	host, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
	)
	if err != nil {
		return nil, err
	}

	fmt.Println("Host Created with ID: %s\n", host.ID())
	fmt.Println("Listen Addresses:", host.Addrs())

	return host, nil
}

func InitDHT(ctx context.Context, h host.Host) (*dht.IpfsDHT, error) {
	dht, err := dht.New(ctx, h, dht.Mode(dht.ModeServer))
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err = dht.Bootstrap(ctx); err != nil {
		return nil, err
	}

	// routingDiscovery := routing.NewRoutingDiscovery(dht)
	// dutil.Advertise(ctx, routingDiscovery, "p2p-storage-network")

	fmt.Println("Successfully bootstrapped DHT!!")

	return dht, nil
}
