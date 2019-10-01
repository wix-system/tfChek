package launcher

import (
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"os"
	"os/exec"
	"time"
)

type TaskManager interface {
	Launch(conn *websocket.Conn, env, layer string, cmd []string) (int, error)
	Close() error
	Start() error
	TaskOfRunSh(rcs RunShCmd, ctx context.Context) (BackgroundTask, error)
}

type BackgroundTask interface {
	Run() error
	GetStdOut() io.Reader
	GetStdErr() io.Reader
	GetStdIn() io.Writer
}

type BackgroundTaskImpl struct {
	Name    string
	Id      int
	Command string
	Args    []string
	Context context.Context
	Status  TaskStatus
	Socket  chan *websocket.Conn
	out     io.Reader
	err     io.Reader
	in      io.Writer
}

func (bti *BackgroundTaskImpl) GetStdOut() io.Reader {
	return bti.out
}

func (bti *BackgroundTaskImpl) GetStdErr() io.Reader {
	return bti.err
}

func (bti *BackgroundTaskImpl) GetStdIn() io.Writer {
	return bti.in
}

func (bti *BackgroundTaskImpl) Run() error {
	outPipeReader, outPipeWriter := io.Pipe()
	errPipeReader, errPipeWriter := io.Pipe()
	inPipeReader, inPipeWriter := io.Pipe()
	bti.out = outPipeReader
	bti.err = errPipeReader
	bti.in = inPipeWriter
	//Get working directory
	var cwd string
	if d, ok := bti.Context.Value(WD).(string); ok {
		cwd = d
	} else {
		d, err := os.Getwd()
		if err != nil {
			return err
		}
		cwd = d
	}
	log.Printf("Task id: %d working directory: %s", bti.Id, cwd)
	//Get environment
	sysenv := make([]string, 0)
	if d, ok := bti.Context.Value(ENVVARS).(map[string]string); ok {
		for k, v := range d {
			sysenv = append(sysenv, fmt.Sprintf("%s=%s", k, v))
		}
	} else {
		sysenv = os.Environ()
	}
	log.Printf("Task id: %d environment: %s", bti.Id, sysenv)

	command := exec.CommandContext(bti.Context, bti.Command, bti.Args...)
	command.Dir = cwd
	command.Env = sysenv
	log.Printf("Running command and waiting for it to finish...")
	command.Stdout = outPipeWriter
	command.Stderr = errPipeWriter
	command.Stdin = inPipeReader
	err := command.Run()
	log.Printf("Command finished with error: %v", err)
	return err
}

type TaskManagerImpl struct {
	sequence int
	started  bool
	stop     chan bool
	threads  map[string]chan BackgroundTask
	//lock     sync.Mutex
	defaultWorkDir string
}

func NewTaskManager() TaskManager {
	return &TaskManagerImpl{started: false, stop: make(chan bool), sequence: 0, threads: make(map[string]chan BackgroundTask), defaultWorkDir: "/tmp/production_42"}
}

func (tm *TaskManagerImpl) TaskOfRunSh(rcs RunShCmd, ctx context.Context) (BackgroundTask, error) {
	command, args, err := rcs.CommandArgs()
	if err != nil {
		return nil, err
	}
	tm.sequence++
	t := BackgroundTaskImpl{Id: tm.sequence, Command: command, Args: args, Context: ctx, Status: OPEN, Socket: make(chan *websocket.Conn)}
	return &t, nil
}

func (tm *TaskManagerImpl) Launch(conn *websocket.Conn, env, layer string, cmd []string) (int, error) {
	panic("implement me")
}

func (tm *TaskManagerImpl) Close() error {
	close(tm.stop)
	return nil
}

func (tm *TaskManagerImpl) Start() error {
	if tm.started {
		return errors.New("dispatcher already has been started")
	}
	started := make(map[string]bool)
	for {
		for s, tasks := range tm.threads {
			if !started[s] {
				go runTasks(tasks)
				started[s] = true
			}
		}
		//Event sourcing
		select {
		case <-tm.stop:
			for _, tasks := range tm.threads {
				close(tasks)
			}
			break
		default:
			time.Sleep(time.Second)
		}
	}
}

func runTasks(tasks <-chan BackgroundTask) {
	for t := range tasks {
		err := t.Run()
		if err != nil {
			log.Printf("Task failed: %s", err)
		}
	}
}
