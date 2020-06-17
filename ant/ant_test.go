package ant

import (
	"io/ioutil"
	"os"
	"testing"

	"gitlab.com/NebulousLabs/Sia/node/api/client"
	"gitlab.com/NebulousLabs/Sia/types"
)

func TestNewAnt(t *testing.T) {
	t.Parallel()
	datadir, err := ioutil.TempDir("", "testing-data")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(datadir)
	config := AntConfig{
		SiadConfig: SiadConfig{
			APIAddr:  "localhost:31337",
			RPCAddr:  "localhost:31338",
			HostAddr: "localhost:31339",
			SiadPath: "siad",
		},
		SiaDirectory: datadir,
	}

	ant, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	defer ant.Close()

	opts, err := client.DefaultOptions()
	if err != nil {
		t.Fatal(err)
	}
	opts.Address = "localhost:31337"
	c := client.New(opts)
	if _, err = c.ConsensusGet(); err != nil {
		t.Fatal(err)
	}
}

func TestStartJob(t *testing.T) {
	t.Parallel()
	datadir, err := ioutil.TempDir("", "testing-data")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(datadir)

	config := AntConfig{
		SiadConfig: SiadConfig{
			APIAddr:  "localhost:31337",
			RPCAddr:  "localhost:31338",
			HostAddr: "localhost:31339",
			SiadPath: "siad",
		},
		SiaDirectory: datadir,
	}

	ant, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	defer ant.Close()

	// nonexistent job should throw an error
	err = ant.StartJob("thisjobdoesnotexist")
	if err == nil {
		t.Fatal("StartJob should return an error with a nonexistent job")
	}
}

func TestWalletAddress(t *testing.T) {
	t.Parallel()
	datadir, err := ioutil.TempDir("", "testing-data")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(datadir)

	config := AntConfig{
		SiadConfig: SiadConfig{
			APIAddr:  "localhost:31337",
			RPCAddr:  "localhost:31338",
			HostAddr: "localhost:31339",
			SiadPath: "siad",
		},
		SiaDirectory: datadir,
	}

	ant, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	defer ant.Close()

	addr, err := ant.WalletAddress()
	if err != nil {
		t.Fatal(err)
	}
	blankaddr := types.UnlockHash{}
	if *addr == blankaddr {
		t.Fatal("WalletAddress returned an empty address")
	}
}
