package ant

import (
	"log"
	"time"
)

const (
	// gatewayConnectabilityInitialSleepTime is initial sleep time for gateway
	// connectability
	gatewayConnectabilityInitialSleepTime = time.Minute

	// gatewayConnectabilitySleepTime is sleep time for gateway connectability
	gatewayConnectabilitySleepTime = time.Second * 30
)

// gatewayConnectability will print an error to the log if the node has zero
// peers at any time.
func (j *JobRunner) gatewayConnectability() {
	j.StaticTG.Add()
	defer j.StaticTG.Done()

	// Wait for ants to be synced if the wait group was set
	AntSyncWG.Wait()

	// Initially wait a while to give the other ants some time to spin up.
	select {
	case <-j.StaticTG.StopChan():
		return
	case <-time.After(gatewayConnectabilityInitialSleepTime):
	}

	for {
		// Wait 30 seconds between iterations.
		select {
		case <-j.StaticTG.StopChan():
			return
		case <-time.After(gatewayConnectabilitySleepTime):
		}

		// Count the number of peers that the gateway has. An error is reported
		// for less than two peers because the gateway is likely connected to
		// itself.
		gatewayInfo, err := j.staticClient.GatewayGet()
		if err != nil {
			log.Printf("[ERROR] [gateway] [%v] error when calling /gateway: %v\n", j.staticSiaDirectory, err)
		}
		if len(gatewayInfo.Peers) < 2 {
			log.Printf("[ERROR] [gateway] [%v] ant has less than two peers: %v\n", j.staticSiaDirectory, gatewayInfo.Peers)
		}
	}
}
