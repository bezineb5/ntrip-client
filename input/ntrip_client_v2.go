package input

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	"github.com/go-gnss/ntrip"
)

type ntripClient struct {
	hostUrl string
}

type ntripMountPointClient struct {
	url string

	// Mutable.
	mu   sync.Mutex
	stop chan struct{}
}

func NewNtripV2Client(hostUrl string) SourceTableInput {
	return &ntripClient{
		hostUrl: hostUrl,
	}
}

func (c *ntripClient) SourceTable() (ntrip.Sourcetable, error) {
	req, err := ntrip.NewClientRequest(c.hostUrl)
	if err != nil {
		return ntrip.Sourcetable{}, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ntrip.Sourcetable{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return ntrip.Sourcetable{}, fmt.Errorf("received non-200 response code: %d", resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ntrip.Sourcetable{}, err
	}

	sourcetable, _ := ntrip.ParseSourcetable(string(data))
	return sourcetable, nil
}

func (c *ntripClient) Url() string {
	return c.hostUrl
}

func NewNtripV2MountPointClient(url string) RtcmInput {
	return &ntripMountPointClient{
		url: url,
	}
}

func (c *ntripMountPointClient) Stream() (<-chan []byte, error) {
	// We need to lock if there are multiple Stream
	// calls simultaneously.
	c.mu.Lock()
	defer c.mu.Unlock()

	req, err := ntrip.NewClientRequest(c.url)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response code: %d", resp.StatusCode)
	}

	// First release the current continuous reading if there is one
	if c.stop != nil {
		c.stop <- struct{}{}
		c.stop = nil
	}
	c.stop = make(chan struct{})
	ch := make(chan []byte, 4)

	go func(s <-chan struct{}) {
		defer close(ch)
		defer resp.Body.Close()
		buf := bufio.NewReader(resp.Body)

		for {
			select {
			case <-s:
				return
			default:
				line, err := buf.ReadBytes('\n')
				if err != nil {
					log.Println("Error in reading mountpoint", err)
					return
				}

				if len(line) > 0 {
					ch <- line
				}
			}
		}
	}(c.stop)

	return ch, nil
}

func (c *ntripMountPointClient) Close() error {
	// We need to lock if there are multiple Halt or ReadContinuous
	// calls simultaneously.
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stop != nil {
		c.stop <- struct{}{}
		c.stop = nil
	}
	return nil
}
