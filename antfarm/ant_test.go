package antfarm

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"go.sia.tech/sia-antfarm/ant"
	"go.sia.tech/sia-antfarm/test"
	"go.sia.tech/siad/node/api/client"
)

// TestStartAnts verifies that startAnts successfully starts ants given some
// configs.
func TestStartAnts(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	// Create minimum configs
	dataDir := test.TestDir(t.Name())
	antDirs, err := test.AntDirs(dataDir, 3)
	if err != nil {
		t.Fatal(err)
	}
	configs := []ant.AntConfig{
		{
			SiadConfig: ant.SiadConfig{
				AllowHostLocalNetAddress: true,
				DataDir:                  antDirs[0],
				SiadPath:                 test.TestSiadFilename,
			},
		},
		{
			SiadConfig: ant.SiadConfig{
				AllowHostLocalNetAddress: true,
				DataDir:                  antDirs[1],
				SiadPath:                 test.TestSiadFilename,
			},
		},
		{
			SiadConfig: ant.SiadConfig{
				AllowHostLocalNetAddress: true,
				DataDir:                  antDirs[2],
				SiadPath:                 test.TestSiadFilename,
			},
		},
	}

	// Create logger
	logger := test.NewTestLogger(t, dataDir)
	defer func() {
		if err := logger.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	// Start ants
	ants, err := startAnts(&sync.WaitGroup{}, logger, configs...)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		for _, ant := range ants {
			err := ant.Close()
			if err != nil {
				t.Error(err)
			}
		}
	}()

	// verify each ant has a reachable api
	for _, ant := range ants {
		opts, err := client.DefaultOptions()
		if err != nil {
			t.Fatal(err)
		}
		opts.Address = ant.APIAddr
		c := client.New(opts)
		if _, err := c.ConsensusGet(); err != nil {
			t.Fatal(err)
		}
	}
}

// TestStartAntWithSiadPath verifies that startAnts successfully starts ant
// given relative or absolute path to siad binary that is not in PATH
func TestStartAntWithSiadPath(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	// Get test paths to siad binaries
	relativeSiadPath := test.RelativeSiadPath()
	absoluteSiadPath, err := test.AbsoluteSiadPath()
	if err != nil {
		t.Fatal(err)
	}

	var tests = []struct {
		name     string
		siadPath string
	}{
		{name: "TestRelativePath", siadPath: relativeSiadPath},
		{name: "TestAbsolutePath", siadPath: absoluteSiadPath},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create minimum configs
			dataDir := test.TestDir(tt.name)
			antDirs, err := test.AntDirs(dataDir, 1)
			if err != nil {
				t.Fatal(err)
			}
			configs := []ant.AntConfig{
				{
					SiadConfig: ant.SiadConfig{
						AllowHostLocalNetAddress: true,
						DataDir:                  antDirs[0],
						SiadPath:                 tt.siadPath,
					},
				},
			}

			// Create logger
			logger := test.NewTestLogger(t, dataDir)
			defer func() {
				if err := logger.Close(); err != nil {
					t.Fatal(err)
				}
			}()

			// Start ants
			ants, err := startAnts(&sync.WaitGroup{}, logger, configs...)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				for _, ant := range ants {
					err := ant.Close()
					if err != nil {
						t.Error(err)
					}
				}
			}()

			// Verify the ant has a reachable api
			for _, ant := range ants {
				opts, err := client.DefaultOptions()
				if err != nil {
					t.Fatal(err)
				}
				opts.Address = ant.APIAddr
				c := client.New(opts)
				if _, err := c.ConsensusGet(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

// TestRenterDisableIPViolationCheck verifies that IPViolationCheck can be set
// via renter ant config
func TestRenterDisableIPViolationCheck(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	// Define test cases data
	testCases := []struct {
		name                          string
		renterDisableIPViolationCheck bool
	}{
		{"TestDefaultIPViolationCheck", false},
		{"TestDisabledIPViolationCheck", true},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create minimum configs
			dataDir := test.TestDir(t.Name())
			antDirs, err := test.AntDirs(dataDir, 1)
			if err != nil {
				t.Fatal(err)
			}
			configs := []ant.AntConfig{
				{
					SiadConfig: ant.SiadConfig{
						AllowHostLocalNetAddress: true,
						DataDir:                  antDirs[0],
						SiadPath:                 test.TestSiadFilename,
					},
					Jobs: []string{"renter"},
				},
			}

			// Update config if testing disabled IP violation check
			if tc.renterDisableIPViolationCheck {
				configs[0].RenterDisableIPViolationCheck = true
			}

			// Create logger
			logger := test.NewTestLogger(t, dataDir)
			defer func() {
				if err := logger.Close(); err != nil {
					t.Fatal(err)
				}
			}()

			// Start ants
			ants, err := startAnts(&sync.WaitGroup{}, logger, configs...)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				for _, ant := range ants {
					err := ant.Close()
					if err != nil {
						t.Error(err)
					}
				}
			}()
			renterAnt := ants[0]

			// Get http client
			c, err := getClient(renterAnt.APIAddr, "")
			if err != nil {
				t.Fatal(err)
			}

			// Get renter settings
			renterInfo, err := c.RenterGet()
			if err != nil {
				t.Fatal(err)
			}
			// Check that IP violation check was not set by default and was set
			// correctly if configured so
			if !tc.renterDisableIPViolationCheck && !renterInfo.Settings.IPViolationCheck {
				t.Fatal("Setting IPViolationCheck is supposed to be true by default")
			} else if tc.renterDisableIPViolationCheck && renterInfo.Settings.IPViolationCheck {
				t.Fatal("Setting IPViolationCheck is supposed to be set false by the ant config")
			}
		})
	}
}

// TestConnectAnts verifies that ants will connect
func TestConnectAnts(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	// connectAnts should throw an error if only one ant is provided
	if err := ConnectAnts(&ant.Ant{}); err == nil {
		t.Fatal("connectAnts didnt throw an error with only one ant")
	}

	// Create minimum configs
	dataDir := test.TestDir(t.Name())
	antDirs, err := test.AntDirs(dataDir, 5)
	if err != nil {
		t.Fatal(err)
	}
	configs := []ant.AntConfig{
		{
			SiadConfig: ant.SiadConfig{
				AllowHostLocalNetAddress: true,
				DataDir:                  antDirs[0],
				SiadPath:                 test.TestSiadFilename,
			},
		},
		{
			SiadConfig: ant.SiadConfig{
				AllowHostLocalNetAddress: true,
				DataDir:                  antDirs[1],
				SiadPath:                 test.TestSiadFilename,
			},
		},
		{
			SiadConfig: ant.SiadConfig{
				AllowHostLocalNetAddress: true,
				DataDir:                  antDirs[2],
				SiadPath:                 test.TestSiadFilename,
			},
		},
		{
			SiadConfig: ant.SiadConfig{
				AllowHostLocalNetAddress: true,
				DataDir:                  antDirs[3],
				SiadPath:                 test.TestSiadFilename,
			},
		},
		{
			SiadConfig: ant.SiadConfig{
				AllowHostLocalNetAddress: true,
				DataDir:                  antDirs[4],
				SiadPath:                 test.TestSiadFilename,
			},
		},
	}

	// Create logger
	logger := test.NewTestLogger(t, dataDir)
	defer func() {
		if err := logger.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	// Start ants
	ants, err := startAnts(&sync.WaitGroup{}, logger, configs...)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		for _, ant := range ants {
			err := ant.Close()
			if err != nil {
				t.Error(err)
			}
		}
	}()

	// Connect the ants
	err = ConnectAnts(ants...)
	if err != nil {
		t.Fatal(err)
	}

	// Get the Gateway info from on of the ants
	opts, err := client.DefaultOptions()
	if err != nil {
		t.Fatal(err)
	}
	opts.Address = ants[0].APIAddr
	c := client.New(opts)
	gatewayInfo, err := c.GatewayGet()
	if err != nil {
		t.Fatal(err)
	}
	// Verify the ants are peers
	for _, ant := range ants[1:] {
		hasAddr := false
		for _, peer := range gatewayInfo.Peers {
			if fmt.Sprint(peer.NetAddress) == ant.RPCAddr {
				hasAddr = true
				break
			}
		}
		if !hasAddr {
			t.Fatalf("the central ant is missing %v", ant.RPCAddr)
		}
	}
}

// TestAntConsensusGroups probes the antConsensusGroup function
func TestAntConsensusGroups(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	// Create minimum configs
	dataDir := test.TestDir(t.Name())
	antDirs, err := test.AntDirs(dataDir, 4)
	if err != nil {
		t.Fatal(err)
	}
	configs := []ant.AntConfig{
		{
			SiadConfig: ant.SiadConfig{
				AllowHostLocalNetAddress: true,
				DataDir:                  antDirs[0],
				SiadPath:                 test.TestSiadFilename,
			},
		},
		{
			SiadConfig: ant.SiadConfig{
				AllowHostLocalNetAddress: true,
				DataDir:                  antDirs[1],
				SiadPath:                 test.TestSiadFilename,
			},
		},
		{
			SiadConfig: ant.SiadConfig{
				AllowHostLocalNetAddress: true,
				DataDir:                  antDirs[2],
				SiadPath:                 test.TestSiadFilename,
			},
		},
	}

	// Create logger
	logger := test.NewTestLogger(t, dataDir)
	defer func() {
		if err := logger.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	// Start ants
	ants, err := startAnts(&sync.WaitGroup{}, logger, configs...)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		for _, ant := range ants {
			err := ant.Close()
			if err != nil {
				t.Error(err)
			}
		}
	}()

	// Get the consensus groups
	groups, err := antConsensusGroups(ants...)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 {
		t.Fatal("expected 1 consensus group initially")
	}
	if len(groups[0]) != len(ants) {
		t.Fatal("expected the consensus group to have all the ants")
	}

	// Start an ant that is desynced from the rest of the network
	cfg, err := parseConfig(logger, ant.AntConfig{
		Jobs: []string{"miner"},
		SiadConfig: ant.SiadConfig{
			AllowHostLocalNetAddress: true,
			DataDir:                  antDirs[3],
			SiadPath:                 test.TestSiadFilename,
		},
	},
	)
	if err != nil {
		t.Fatal(err)
	}
	otherAnt, err := ant.New(&sync.WaitGroup{}, logger, cfg)
	if err != nil {
		t.Fatal(err)
	}
	ants = append(ants, otherAnt)

	// Wait for the other ant to mine a few blocks
	time.Sleep(time.Second * 30)

	// Verify the ants are synced
	groups, err = antConsensusGroups(ants...)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 {
		t.Fatal("expected 2 consensus groups")
	}
	if len(groups[0]) != len(ants)-1 {
		t.Fatal("expected the first consensus group to have 3 ants")
	}
	if len(groups[1]) != 1 {
		t.Fatal("expected the second consensus group to have 1 ant")
	}
	if !reflect.DeepEqual(groups[1][0], otherAnt) {
		t.Fatal("expected the miner ant to be in the second consensus group")
	}
}
