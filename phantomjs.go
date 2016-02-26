// Copyright 2013 Federico Sogaro. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webdriver

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"time"
)

type PhantomJsSwitches map[string]interface{}

type PhantomJsDriver struct {
	WebDriverCore
	//The port that PhantomJsDriver listens on. Default: 9515
	Port int
	//The URL path prefix to use for all incoming WebDriver REST requests. Default: ""
	BaseUrl string
	//The number of threads to use for handling HTTP requests. Default: 4
	Threads int
	//The path to use for the PhantomJsDriver server log. Default: ./phantomJsdriver.log
	LogPath string
	// Log file to dump phantomJsdriver stdout/stderr. If "" send to terminal. Default: ""
	LogFile string
	// Start method fails if PhantomJsdriver doesn't start in less than StartTimeout. Default 20s.
	StartTimeout time.Duration

	path    string
	cmd     *exec.Cmd
	logFile *os.File
}

//create a new service using phantomJsdriver.
//function returns an error if not supported switches are passed. Actual content
//of valid-named switches is not validate and is passed as it is.
//switch silent is removed (output is needed to check if phantomJsdriver started correctly)
func NewPhantomJsDriver(path string) *PhantomJsDriver {
	d := &PhantomJsDriver{}
	d.path = path
	d.Port = 9515
	d.BaseUrl = ""
	d.Threads = 4
	d.LogPath = "phantomJsdriver.log"
	d.StartTimeout = 20 * time.Second
	return d
}

func (d *PhantomJsDriver) Start() error {
	csferr := "phantomJsdriver start failed: "
	if d.cmd != nil {
		return errors.New(csferr + "phantomJsdriver already running")
	}

	if d.LogPath != "" {
		//check if log-path is writable
		file, err := os.OpenFile(d.LogPath, os.O_WRONLY|os.O_CREATE, 0664)
		if err != nil {
			return errors.New(csferr + "unable to write in log path: " + err.Error())
		}
		file.Close()
	}

	d.url = fmt.Sprintf("http://127.0.0.1:%d%s", d.Port, d.BaseUrl)
	var switches []string
	switches = append(switches, "--webdriver="+strconv.Itoa(d.Port))
	switches = append(switches, "--webdriver-logfile="+d.LogPath)
	switches = append(switches, "--webdriver-loglevel=DEBUG")

	d.cmd = exec.Command(d.path, switches...)
	stdout, err := d.cmd.StdoutPipe()
	if err != nil {
		return errors.New(csferr + err.Error())
	}
	stderr, err := d.cmd.StderrPipe()
	if err != nil {
		return errors.New(csferr + err.Error())
	}
	if err := d.cmd.Start(); err != nil {
		return errors.New(csferr + err.Error())
	}
	if d.LogFile != "" {
		flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		d.logFile, err = os.OpenFile(d.LogFile, flags, 0640)
		if err != nil {
			return err
		}
		go io.Copy(d.logFile, stdout)
		go io.Copy(d.logFile, stderr)
	} else {
		go io.Copy(os.Stdout, stdout)
		go io.Copy(os.Stderr, stderr)
	}
	if err = probePort(d.Port, d.StartTimeout); err != nil {
		return err
	}
	return nil
}

func (d *PhantomJsDriver) Stop() error {
	if d.cmd == nil {
		return errors.New("stop failed: phantomJsdriver not running")
	}
	defer func() {
		d.cmd = nil
	}()
	d.cmd.Process.Signal(os.Interrupt)
	if d.logFile != nil {
		d.logFile.Close()
	}
	return nil
}

func (d *PhantomJsDriver) NewSession(desired, required Capabilities) (*Session, error) {
	//id, capabs, err := d.newSession(desired, required)
	//return &Session{id, capabs, d}, err
	session, err := d.newSession(desired, required)
	if err != nil {
		return nil, err
	}
	session.wd = d
	return session, nil
}

func (d *PhantomJsDriver) Sessions() ([]Session, error) {
	sessions, err := d.sessions()
	if err != nil {
		return nil, err
	}
	for i := range sessions {
		sessions[i].wd = d
	}
	return sessions, nil
}
