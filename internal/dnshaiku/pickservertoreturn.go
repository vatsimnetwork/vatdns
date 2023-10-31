package dnshaiku

import (
	"github.com/jftuga/geodist"
	"github.com/spf13/viper"
	"github.com/vatsimnetwork/vatdns/internal/logger"
	"github.com/vatsimnetwork/vatdns/pkg/common"
	"sort"
)

func PickServerToReturn(sourceIpLatLng geodist.Coord) *common.FSDServer {
	// Slices are easier for sorting
	initialServers := make([]common.FSDServer, 0)
	finalServers := make([]common.FSDServer, 0)

	// Get servers into a slice, skipping those that are not accepting connections
	fsdServers.Range(func(k, v interface{}) bool {
		fsdServerStruct := v.(*common.FSDServer)
		if fsdServerStruct.AcceptingConnections() == 0 {
			return true
		}
		miles, _, _ := geodist.VincentyDistance(sourceIpLatLng, geodist.Coord{Lat: fsdServerStruct.Latitude, Lon: fsdServerStruct.Longitude})
		initialServers = append(initialServers, common.FSDServer{
			Name:           fsdServerStruct.Name,
			Distance:       miles,
			RemainingSlots: fsdServerStruct.RemainingSlots,
			IpAddress:      fsdServerStruct.IpAddress,
			Country:        fsdServerStruct.Country,
		})
		return true
	})

	if len(initialServers) == 0 {
		logger.Error("No servers possible for a request, using default FSD server")
		fsdServer, _ := fsdServers.Load(viper.GetString("DEFAULT_FSD_SERVER"))
		fsdServerStruct := fsdServer.(*common.FSDServer)
		return fsdServerStruct

	}

	// Sort slice of servers by distance from request
	sort.Slice(initialServers, func(i, j int) bool {
		return initialServers[i].Distance < initialServers[j].Distance
	})

	// Get country for first server to be returned based upon distance
	// and populate a new slice with other servers in that country
	firstServer := initialServers[0].Country
	for _, server := range initialServers {
		if server.Country == firstServer {
			finalServers = append(finalServers, server)
		}
	}

	// Sort slice by remaining slots
	sort.Slice(finalServers, func(i, j int) bool {
		return finalServers[i].RemainingSlots > finalServers[j].RemainingSlots
	})

	// If no servers in the final slice return random otherwise return first element of finalServers
	// Value returned from finalServers should be the closest server to a user with the most available slots
	if len(finalServers) == 0 {
		logger.Error("No server found for request returning random")
		fsdServer, _ := fsdServers.Load(initialServers[0].Name)
		fsdServerStruct := fsdServer.(*common.FSDServer)
		fsdServerStruct.RemainingSlots -= 1
		return fsdServerStruct
	} else {
		fsdServer, _ := fsdServers.Load(finalServers[0].Name)
		fsdServerStruct := fsdServer.(*common.FSDServer)
		fsdServerStruct.RemainingSlots -= 1
		return fsdServerStruct
	}
}
