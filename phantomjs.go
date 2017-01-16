// Copyright 2013 Federico Sogaro. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webdriver

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
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
	// Host. Default 127.0.0.1
	Host string
	// LogLevel. Default DEBUG
	LogLevel string

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
	d.Port = 0
	d.Host = "127.0.0.1"
	d.BaseUrl = ""
	d.Threads = 4
	d.LogPath = "phantomJsdriver.log"
	d.LogFile = "phantomJsOutput.log"
	d.LogLevel = "DEBUG"
	d.StartTimeout = 20 * time.Second
	return d
}

func (d *PhantomJsDriver) Start() error {
	if d.Port == 0 {
		var err error
		d.Port, err = GetFreePort()
		if err != nil {
			return err
		}
	}

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

	d.url = fmt.Sprintf("http://%s:%d%s", d.Host, d.Port, d.BaseUrl)
	var switches []string
	switches = append(switches, fmt.Sprintf("--webdriver=%s:%d", d.Host, d.Port))
	switches = append(switches, fmt.Sprintf("--webdriver-logfile=%s", d.LogPath))
	switches = append(switches, fmt.Sprintf("--webdriver-loglevel=%s", d.LogLevel))

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
		go func() {
			if _, err := io.Copy(d.logFile, stdout); err != nil {
				log.Println(err)
			}
		}()
		go func() {
			if _, err := io.Copy(d.logFile, stderr); err != nil {
				log.Println(err)
			}
		}()
	} else {
		go func() {
			if _, err := io.Copy(os.Stdout, stdout); err != nil {
				log.Println(err)
			}
		}()
		go func() {
			if _, err := io.Copy(os.Stderr, stderr); err != nil {
				log.Println(err)
			}
		}()
	}
	if err = probePort(d.Port, d.StartTimeout); err != nil {
		return err
	}
	return nil
}

func (d *PhantomJsDriver) Stop() error {
	defer func() {
		d.cmd = nil
	}()
	cmd := d.cmd
	if cmd == nil {
		return errors.New("stop failed: phantomJsdriver not running")
	}
	if cmd.Process == nil {
		return errors.New("stop failed: process nil")
	}
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		return err
	}
	if d.logFile != nil {
		if err := d.logFile.Close(); err != nil {
			return err
		}
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
