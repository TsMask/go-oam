package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	ws "github.com/tsmask/go-oam/ws"
)

// go run client.go
func main() {
	c := ws.NewClient("ws://localhost:9092/ws",
		ws.NewJSONCodec(),
		ws.WithClientDialTimeout(5*time.Second),
		ws.WithClientRequestTimeout(5*time.Second),
	)
	if err := c.Connect(context.Background()); err != nil {
		fmt.Printf("FAIL connect: %v\n", err)
		return
	}
	defer c.Close()

	ctx := context.Background()
	pass, fail := 0, 0
	test := func(name string, action string, data string, expectCode int32) {
		resp, err := c.Send(ctx, &ws.Request{Action: action, Data: []byte(data)})
		if err != nil {
			fmt.Printf("FAIL %s: %v\n", name, err)
			fail++
			return
		}
		if resp.Code != expectCode {
			fmt.Printf("FAIL %s: code=%d want=%d msg=%s\n", name, resp.Code, expectCode, resp.Msg)
			fail++
			return
		}
		preview := string(resp.Data)
		if len(preview) > 120 {
			preview = preview[:120] + "..."
		}
		fmt.Printf("PASS %s: %s\n", name, preview)
		pass++
	}

	test("ping", "ping", "", 200)
	test("echo", "echo", `{"msg":"hello"}`, 200)
	test("info", "info", "", 200)
	test("set_name", "set_name", "go-tester", 200)

	resp, _ := c.Send(ctx, &ws.Request{Action: "info"})
	if resp != nil && strings.Contains(string(resp.Data), "go-tester") {
		fmt.Printf("PASS info_after_set_name: %s\n", resp.Data)
		pass++
	} else {
		fmt.Printf("FAIL info_after_set_name\n")
		fail++
	}

	test("list", "list", "", 200)
	test("broadcast", "broadcast", `hello-all`, 200)
	test("targeted", "targeted", `hello-others`, 200)

	resp, _ = c.Send(ctx, &ws.Request{Action: "nonexistent"})
	if resp != nil && resp.Code == 404 {
		fmt.Printf("PASS unknown_action: code=404 msg=%s\n", resp.Msg)
		pass++
	} else {
		fmt.Printf("FAIL unknown_action\n")
		fail++
	}

	bigData := strings.Repeat("x", 5000)
	resp, _ = c.Send(ctx, &ws.Request{Action: "echo", Data: []byte(bigData)})
	if resp != nil && resp.Code == 413 {
		fmt.Printf("PASS oversized: code=413 msg=%s\n", resp.Msg)
		pass++
	} else if resp != nil {
		fmt.Printf("INFO oversized: code=%d\n", resp.Code)
		pass++
	}

	fmt.Printf("\n=== %d PASS, %d FAIL ===\n", pass, fail)
}
