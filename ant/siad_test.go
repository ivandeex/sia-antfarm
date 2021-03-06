package ant

import (
	"testing"

	"go.sia.tech/sia-antfarm/test"
	"go.sia.tech/siad/node/api/client"
	"go.sia.tech/siad/persist"
	"gitlab.com/NebulousLabs/errors"
)

// newTestingSiadConfig creates a generic SiadConfig for the provided datadir.
func newTestingSiadConfig(datadir string) (SiadConfig, error) {
	addrs, err := GetAddrs(NumPorts)
	if err != nil {
		return SiadConfig{}, errors.AddContext(err, "can't get free local addresses")
	}
	ip := "127.0.0.1"
	sc := SiadConfig{
		AllowHostLocalNetAddress: true,
		APIAddr:                  ip + addrs[0],
		APIPassword:              persist.RandomSuffix(),
		DataDir:                  datadir,
		HostAddr:                 ip + addrs[1],
		RPCAddr:                  ip + addrs[2],
		SiadPath:                 test.TestSiadFilename,
		SiaMuxAddr:               ip + addrs[3],
		SiaMuxWsAddr:             ip + addrs[4],
	}
	return sc, nil
}

// TestNewSiad tests that NewSiad creates a reachable Sia API
func TestNewSiad(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	// Create testing config
	dataDir := test.TestDir(t.Name())
	config, err := newTestingSiadConfig(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create logger
	logger := test.NewTestLogger(t, dataDir)
	defer func() {
		if err := logger.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	// Create the siad process
	siad, err := newSiad(logger, config)
	if err != nil {
		t.Fatal(err)
	}

	// Create Sia Client
	opts, err := client.DefaultOptions()
	if err != nil {
		t.Fatal(err)
	}
	opts.Address = config.APIAddr
	c := client.New(opts)

	// Test Client by pinging the ConsensusGet endpoint
	if _, err := c.ConsensusGet(); err != nil {
		t.Error(err)
	}

	// Stop siad process
	stopSiad(logger, config.DataDir, config.APIAddr, config.APIPassword, siad.Process)

	// Test Creating siad with a blank config
	_, err = newSiad(logger, SiadConfig{})
	if err == nil {
		t.Fatal("Shouldn't be able to create siad process with empty config")
	}

	// verify that NewSiad returns an error given invalid args
	config.APIAddr = "this_is_an_invalid_address:1000000"
	_, err = newSiad(logger, config)
	if err == nil {
		t.Fatal("expected newsiad to return an error with invalid args")
	}
}
