package nodelist

import (
	"bufio"
	"encoding/gob"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"uncleeugene.kz/momail/ftn"
)

// Node represents a node entry in the nodelist.
type Node struct {
	Address  ftn.FidoAddress
	Name     string
	Location string
	Sysop    string
	Phone    string
	Baud     int
	Flags    []string
}

// List represents a parsed nodelist.
type List struct {
	Nodes map[string]Node
}

// Merge adds nodes from another list into this one.
func (l *List) Merge(other *List) {
	if l.Nodes == nil {
		l.Nodes = make(map[string]Node)
	}
	for k, v := range other.Nodes {
		l.Nodes[k] = v
	}
}

// Load parses a nodelist file and returns a List.
func Load(path string) (*List, error) {
	cachePath := path + ".cache"

	// 1. Try to load from cache if it exists and is newer than the source
	if stat, err := os.Stat(cachePath); err == nil {
		if srcStat, err := os.Stat(path); err == nil {
			if stat.ModTime().After(srcStat.ModTime()) {
				f, err := os.Open(cachePath)
				if err == nil {
					defer f.Close()
					var list List
					if err := gob.NewDecoder(f).Decode(&list); err == nil {
						return &list, nil
					}
				}
			}
		}
	}

	// 2. Parse raw file
	nodes, err := parseFile(path)
	if err != nil {
		return nil, err
	}

	// 3. Save to cache
	list := &List{Nodes: nodes}
	if f, err := os.Create(cachePath); err == nil {
		defer f.Close()
		gob.NewEncoder(f).Encode(list)
	}

	return list, nil
}

func parseFile(path string) (map[string]Node, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	nodes := make(map[string]Node)
	scanner := bufio.NewScanner(f)

	var currentZone uint16 = 1
	var currentNet uint16
	var currentRegion uint16

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}

		// Handle EOF marker if present (some lists end with SUB/0x1A)
		if line == "\x1a" {
			break
		}

		fields := strings.Split(line, ",")
		if len(fields) < 7 {
			continue
		}

		keyword := strings.ToLower(fields[0])
		numberStr := fields[1]
		number, err := strconv.ParseUint(numberStr, 10, 16)
		if err != nil {
			continue
		}
		val := uint16(number)

		var addr ftn.FidoAddress
		isNode := false

		switch keyword {
		case "zone":
			currentZone = val
			currentNet = val // Zone coordinator is usually at Net=Zone
			currentRegion = 0
			addr = ftn.FidoAddress{Zone: currentZone, Net: currentZone, Node: 0}
			isNode = true
		case "region":
			currentRegion = val
			// Region coordinator is usually at Net=Region
			addr = ftn.FidoAddress{Zone: currentZone, Net: currentRegion, Node: 0}
			isNode = true
		case "host", "hub":
			currentNet = val
			addr = ftn.FidoAddress{Zone: currentZone, Net: currentNet, Node: 0}
			isNode = true
		case "pvt", "hold", "down", "":
			// Standard node
			addr = ftn.FidoAddress{Zone: currentZone, Net: currentNet, Node: val}
			isNode = true
		}

		if isNode {
			node := Node{
				Address:  addr,
				Name:     strings.ReplaceAll(fields[2], "_", " "),
				Location: strings.ReplaceAll(fields[3], "_", " "),
				Sysop:    strings.ReplaceAll(fields[4], "_", " "),
				Phone:    fields[5],
			}
			if baud, err := strconv.Atoi(fields[6]); err == nil {
				node.Baud = baud
			}
			if len(fields) > 7 {
				node.Flags = fields[7:]
			}
			nodes[addr.String()] = node
		}
	}

	return nodes, scanner.Err()
}

// GetBinkpAddress extracts the BinkP hostname and port from the node's flags.
// Returns hostname, port, and a boolean indicating if BinkP is supported.
func (n *Node) GetBinkpAddress() (string, int, bool) {
	var host string
	port := 24554 // Default BinkP port
	supportsBinkp := false

	for _, flag := range n.Flags {
		upper := strings.ToUpper(flag)
		if strings.HasPrefix(upper, "IBN") {
			supportsBinkp = true
			// Parse IBN:port or IBN:host:port logic could go here if needed,
			// but usually IBN is just a flag or port, and INA provides the host.
		} else if strings.HasPrefix(upper, "INA:") {
			host = flag[4:]
		}
	}
	return host, port, supportsBinkp
}

// FindLatest scans the directory for files matching the base name with numeric extensions
// and returns the path to the one with the highest number.
func FindLatest(dir, base string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	var bestName string
	var maxExt int = -1

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// If a base name is provided, ensure the file starts with "base."
		if base != "" {
			if !strings.HasPrefix(strings.ToLower(entry.Name()), strings.ToLower(base)+".") {
				continue
			}
		}

		name := entry.Name()
		extStr := filepath.Ext(name)
		if len(extStr) < 2 {
			continue
		}

		// extStr includes dot, e.g. ".007"
		if extVal, err := strconv.Atoi(extStr[1:]); err == nil {
			if extVal > maxExt {
				maxExt = extVal
				bestName = name
			}
		}
	}

	if bestName == "" {
		return "", os.ErrNotExist
	}

	return filepath.Join(dir, bestName), nil
}
