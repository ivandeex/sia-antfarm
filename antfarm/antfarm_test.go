package antfarm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"go.sia.tech/sia-antfarm/ant"
	"go.sia.tech/sia-antfarm/test"
	"go.sia.tech/siad/modules"
	"go.sia.tech/siad/node/api/client"
)

// verify that createAntfarm() creates a new antfarm correctly.
func TestNewAntfarm(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	addrs, err := ant.GetAddrs(2)
	if err != nil {
		t.Fatal(err)
	}
	ip := "127.0.0.1"
	antFarmAddr := ip + addrs[0]
	antAddr := ip + addrs[1]
	dataDir := test.TestDir(t.Name())
	antFarmDir := filepath.Join(dataDir, "antfarm-data")
	antDirs, err := test.AntDirs(dataDir, 1)
	if err != nil {
		t.Fatal(err)
	}

	logger, err := NewAntfarmLogger(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := logger.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	config := AntfarmConfig{
		ListenAddress: antFarmAddr,
		DataDir:       antFarmDir,
		AntConfigs: []ant.AntConfig{
			{
				SiadConfig: ant.SiadConfig{
					AllowHostLocalNetAddress: true,
					DataDir:                  antDirs[0],
					RPCAddr:                  antAddr,
					SiadPath:                 test.TestSiadFilename,
				},
				Jobs: []string{
					"gateway",
				},
			},
		},
	}

	antfarm, err := New(logger, config)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := antfarm.Close(); err != nil {
			logger.Errorf("can't close antfarm: %v", err)
		}
	}()
	go func() {
		if err := antfarm.ServeAPI(); err != nil {
			logger.Errorf("can't serve antfarm http API: %v", err)
		}
	}()

	res, err := http.DefaultClient.Get("http://" + antFarmAddr + "/ants")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			logger.Errorf("can't close antfarm response body: %v", err)
		}
	}()

	var ants []*ant.Ant
	err = json.NewDecoder(res.Body).Decode(&ants)
	if err != nil {
		t.Fatal(err)
	}
	if len(ants) != len(config.AntConfigs) {
		t.Fatal("expected /ants to return the correct number of ants")
	}
	if ants[0].RPCAddr != config.AntConfigs[0].RPCAddr {
		t.Fatal("expected /ants to return the correct rpc address")
	}
}

// verify that connectExternalAntfarm connects antfarms to eachother correctly
func TestConnectExternalAntfarm(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	datadir := test.TestDir(t.Name())
	antFarmDataDirs := []string{filepath.Join(datadir, "antfarm-data1"), filepath.Join(datadir, "antfarm-data2")}

	logger1, err := NewAntfarmLogger(antFarmDataDirs[0])
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := logger1.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	addrs, err := ant.GetAddrs(2)
	if err != nil {
		t.Fatal(err)
	}

	antConfig := ant.AntConfig{
		SiadConfig: ant.SiadConfig{
			AllowHostLocalNetAddress: true,
			SiadPath:                 test.TestSiadFilename,
		},
		Jobs: []string{
			"gateway",
		},
	}

	antDataDirs1, err := test.AntDirs(antFarmDataDirs[0], 1)
	if err != nil {
		t.Fatal(err)
	}
	antConfig.SiadConfig.DataDir = antDataDirs1[0]
	config1 := AntfarmConfig{
		ListenAddress: addrs[0],
		DataDir:       antFarmDataDirs[0],
		AntConfigs:    []ant.AntConfig{antConfig},
	}

	logger2, err := NewAntfarmLogger(antFarmDataDirs[1])
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := logger2.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	antDataDirs2, err := test.AntDirs(antFarmDataDirs[1], 1)
	if err != nil {
		t.Fatal(err)
	}
	antConfig.SiadConfig.DataDir = antDataDirs2[0]
	config2 := AntfarmConfig{
		ListenAddress: addrs[1],
		DataDir:       antFarmDataDirs[1],
		AntConfigs:    []ant.AntConfig{antConfig},
	}

	farm1, err := New(logger1, config1)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := farm1.Close(); err != nil {
			logger1.Errorf("can't close antfarm: %v", err)
		}
	}()
	go func() {
		if err := farm1.ServeAPI(); err != nil {
			logger1.Errorf("can't serve antfarm http API: %v", err)
		}
	}()

	farm2, err := New(logger2, config2)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := farm2.Close(); err != nil {
			logger2.Errorf("can't close antfarm: %v", err)
		}
	}()
	go func() {
		if err := farm2.ServeAPI(); err != nil {
			logger2.Errorf("can't serve antfarm http API: %v", err)
		}
	}()

	err = farm1.ConnectExternalAntfarm(config2.ListenAddress)
	if err != nil {
		t.Fatal(err)
	}

	// give a bit of time for the connection to succeed
	time.Sleep(time.Second * 3)

	// verify that farm2 has farm1 as its peer
	opts, err := client.DefaultOptions()
	if err != nil {
		t.Fatal(err)
	}
	opts.Address = farm1.Ants[0].APIAddr
	c := client.New(opts)
	gatewayInfo, err := c.GatewayGet()
	if err != nil {
		t.Fatal(err)
	}

	for _, ant := range farm2.Ants {
		hasAddr := false
		for _, peer := range gatewayInfo.Peers {
			if fmt.Sprint(peer.NetAddress) == ant.RPCAddr {
				hasAddr = true
				break
			}
		}
		if !hasAddr {
			t.Fatalf("farm1 is missing %v", ant.RPCAddr)
		}
	}
}

// TestUploadDownloadFileData uploads and downloads a file and checks that
// their content is identical by comparing their merkle root hashes
func TestUploadDownloadFileData(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	// Prepare test logger
	dataDir := test.TestDir(t.Name())
	testLogger := test.NewTestLogger(t, dataDir)
	defer func() {
		if err := testLogger.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	// Start Antfarm
	antfarmLogger, err := NewAntfarmLogger(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := antfarmLogger.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	config, err := NewDefaultRenterAntfarmTestingConfig(dataDir, true)
	if err != nil {
		t.Fatal(err)
	}
	farm, err := New(antfarmLogger, config)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := farm.Close(); err != nil {
			antfarmLogger.Errorf("can't close antfarm: %v", err)
		}
	}()

	// Timeout the test if the renter doesn't becomes upload ready
	renterAnt, err := farm.GetAntByName(test.RenterAntName)
	if err != nil {
		t.Fatal(err)
	}
	err = renterAnt.Jr.WaitForRenterUploadReady()
	if err != nil {
		t.Fatal(err)
	}

	// Upload a file
	renterJob := renterAnt.Jr.NewRenterJob()
	_, err = renterJob.Upload(modules.SectorSize)
	if err != nil {
		t.Fatal(err)
	}

	// DownloadAndVerifyFiles
	err = DownloadAndVerifyFiles(testLogger, renterAnt, renterJob.Files)
	if err != nil {
		t.Fatal(err)
	}
}

// TestUpdateRenter verifies that renter ant's siad can be upgraded using given
// path to siad binary
func TestUpdateRenter(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	// Start Antfarm
	dataDir := test.TestDir(t.Name())

	logger, err := NewAntfarmLogger(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := logger.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	config, err := NewDefaultRenterAntfarmTestingConfig(dataDir, true)
	if err != nil {
		t.Fatal(err)
	}
	farm, err := New(logger, config)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := farm.Close(); err != nil {
			logger.Errorf("can't close antfarm: %v", err)
		}
	}()

	// Timeout the test if the renter doesn't become upload ready
	renterAnt, err := farm.GetAntByName(test.RenterAntName)
	if err != nil {
		t.Fatal(err)
	}
	err = renterAnt.Jr.WaitForRenterUploadReady()
	if err != nil {
		t.Fatal(err)
	}

	// Restart the renter with given siad path (simulates an ant update
	// process)
	err = renterAnt.UpdateSiad(test.RelativeSiadPath())
	if err != nil {
		t.Fatal(err)
	}

	// Timeout the test if the renter after update doesn't become upload ready
	renterAnt, err = farm.GetAntByName(test.RenterAntName)
	if err != nil {
		t.Fatal(err)
	}
	err = renterAnt.Jr.WaitForRenterUploadReady()
	if err != nil {
		t.Fatal(err)
	}

	// Verify that renter is working correctly by uploading and downloading a
	// file

	// Upload a file
	renterJob := renterAnt.Jr.NewRenterJob()
	siaPath, err := renterJob.Upload(modules.SectorSize)
	if err != nil {
		t.Fatal(err)
	}

	// Download the file
	destPath := filepath.Join(renterAnt.Config.DataDir, "downloadedFiles", "downloadedFile")
	err = renterJob.Download(siaPath, destPath)
	if err != nil {
		t.Fatal(err)
	}
}
