package p2p

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
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
	dht, err := dht.New(ctx, h)
	if err != nil {
		return nil, err
	}

	if err = dht.Bootstrap(ctx); err != nil {
		return nil, err
	}

	routingDiscovery := routing.NewRoutingDiscovery(dht)
	dutil.Advertise(ctx, routingDiscovery, "p2p-storage-new")

	fmt.Println("Successfully bootstrapped DHT!!")

	return dht, nil
}
