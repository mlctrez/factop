package example

import (
	"fmt"
	"github.com/mlctrez/factop/api"
	"github.com/nats-io/nats.go"
	"os"
)

/*
This golang code provides an easy way to get at the rcon api to execute lua scripts.
*/

type Example struct {
	NatsConn        *nats.Conn
	DontExitOnError bool
}

func New() *Example {
	e := &Example{}
	conn, err := nats.Connect("nats://factorio")
	e.ExitOnErr(err)
	e.NatsConn = conn
	return e
}

func (e *Example) RconSc(payload string) {
	e.Rcon("/sc " + payload)
}

func (e *Example) Rcon(payload string) {
	client := api.NewRconClient(e.NatsConn)
	result, err := client.Execute(&api.RconCommand{Payload: payload})
	e.ExitOnErr(err)
	if result != nil && result.Payload != "" {
		fmt.Println(result.Payload)
	}
}

func (e *Example) Close() {
	if e != nil && e.NatsConn != nil {
		e.NatsConn.Close()
	}
}

func (e *Example) ExitOnErr(err error) {
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error : %v\n", err)
		if !e.DontExitOnError {
			os.Exit(1)
		}
	}
}
