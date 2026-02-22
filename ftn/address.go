package ftn

import (
	"fmt"
	"strconv"
	"strings"
)

// FidoAddress represents a FidoNet address in the format Zone:Net/Node.Point.
type FidoAddress struct {
	Zone  uint16
	Net   uint16
	Node  uint16
	Point uint16
}

// String returns the canonical string representation of a FidoAddress.
// It omits the .0 point if it's zero.
func (a *FidoAddress) String() string {
	if a.Point == 0 {
		return fmt.Sprintf("%d:%d/%d", a.Zone, a.Net, a.Node)
	}
	return fmt.Sprintf("%d:%d/%d.%d", a.Zone, a.Net, a.Node, a.Point)
}

// ParseFidoAddress parses a string into a FidoAddress struct.
// It handles formats like "Z:N/F.P", "N/F.P", and "N/F", using a default
// zone if the zone is not specified in the address string.
func ParseFidoAddress(addr string, defaultZone uint16) (*FidoAddress, error) {
	res := &FidoAddress{
		Zone: defaultZone,
	}

	// Split Zone from the rest
	parts := strings.SplitN(addr, ":", 2)
	rest := addr
	if len(parts) == 2 {
		zone, err := strconv.ParseUint(parts[0], 10, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid zone '%s': %w", parts[0], err)
		}
		res.Zone = uint16(zone)
		rest = parts[1]
	}

	// Split Net/Node from Point
	parts = strings.SplitN(rest, ".", 2)
	if len(parts) == 2 {
		point, err := strconv.ParseUint(parts[1], 10, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid point '%s': %w", parts[1], err)
		}
		res.Point = uint16(point)
	}
	rest = parts[0]

	// Split Net from Node
	parts = strings.SplitN(rest, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid net/node format in '%s'", rest)
	}

	net, err := strconv.ParseUint(parts[0], 10, 16)
	if err != nil {
		return nil, fmt.Errorf("invalid net '%s': %w", parts[0], err)
	}
	res.Net = uint16(net)

	node, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		return nil, fmt.Errorf("invalid node '%s': %w", parts[1], err)
	}
	res.Node = uint16(node)

	return res, nil
}
