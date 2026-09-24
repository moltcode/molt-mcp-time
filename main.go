// molt-mcp-time is a tiny, dependency-free MCP server speaking JSON-RPC 2.0
// over stdio. It exposes a handful of time/timezone tools and exists mainly
// as the reference "self-contained plugin" for the Molt plugin system: one
// static binary per platform, no post-install steps, configured only through
// environment variables declared in package.json.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	serverName      = "molt-mcp-time"
	serverVersion   = "0.1.0"
	protocolVersion = "2024-11-05"
	// MOLT_TIME_DEFAULT_TZ is the one declared parameter: the IANA timezone
	// used when a tool call does not pass one.
	defaultTzEnv = "MOLT_TIME_DEFAULT_TZ"
)

type request struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

func main() {
	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	out := json.NewEncoder(os.Stdout)

	for in.Scan() {
		line := in.Bytes()
		if len(line) == 0 {
			continue
		}
		var req request
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}
		// Notifications carry no id and expect no reply.
		if req.ID == nil {
			continue
		}
		resp := response{Jsonrpc: "2.0", ID: req.ID}
		switch req.Method {
		case "initialize":
			resp.Result = map[string]any{
				"protocolVersion": protocolVersion,
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": serverName, "version": serverVersion},
			}
		case "ping":
			resp.Result = map[string]any{}
		case "tools/list":
			resp.Result = map[string]any{"tools": tools()}
		case "tools/call":
			resp.Result, resp.Error = callTool(req.Params)
		default:
			resp.Error = &rpcError{Code: -32601, Message: "method not found: " + req.Method}
		}
		if err := out.Encode(resp); err != nil {
			os.Exit(1)
		}
	}
}

func tools() []toolDef {
	return []toolDef{
		{
			Name:        "current_time",
			Description: "Current date and time. Optionally in a specific IANA timezone; defaults to $" + defaultTzEnv + " or the system timezone.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"timezone": map[string]any{
						"type":        "string",
						"description": "IANA timezone name, e.g. Asia/Kolkata or Europe/Berlin",
					},
				},
			},
		},
		{
			Name:        "unix_timestamp",
			Description: "Current Unix timestamp in seconds and milliseconds.",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			Name:        "add_duration",
			Description: "Add a Go-style duration (e.g. 2h45m, -30m, 72h) to now or to a given RFC3339 timestamp.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"duration": map[string]any{
						"type":        "string",
						"description": "Duration such as 90m, 2h45m or -1h30m",
					},
					"start": map[string]any{
						"type":        "string",
						"description": "Optional RFC3339 start timestamp; defaults to now",
					},
				},
				"required": []string{"duration"},
			},
		},
	}
}

func callTool(params json.RawMessage) (any, *rpcError) {
	var call struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(params, &call); err != nil {
		return nil, &rpcError{Code: -32602, Message: "invalid tools/call params"}
	}
	text, err := runTool(call.Name, call.Arguments)
	if err != nil {
		// Tool-level failures are results with isError, not protocol errors.
		return map[string]any{
			"content": []map[string]any{{"type": "text", "text": err.Error()}},
			"isError": true,
		}, nil
	}
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": text}},
	}, nil
}

func runTool(name string, args map[string]any) (string, error) {
	switch name {
	case "current_time":
		loc, err := resolveLocation(stringArg(args, "timezone"))
		if err != nil {
			return "", err
		}
		now := time.Now().In(loc)
		return fmt.Sprintf("%s (%s)", now.Format(time.RFC3339), loc.String()), nil
	case "unix_timestamp":
		now := time.Now()
		return fmt.Sprintf("seconds=%d milliseconds=%d", now.Unix(), now.UnixMilli()), nil
	case "add_duration":
		d, err := time.ParseDuration(stringArg(args, "duration"))
		if err != nil {
			return "", fmt.Errorf("invalid duration: %v", err)
		}
		start := time.Now()
		if s := stringArg(args, "start"); s != "" {
			start, err = time.Parse(time.RFC3339, s)
			if err != nil {
				return "", fmt.Errorf("invalid start timestamp: %v", err)
			}
		}
		return start.Add(d).Format(time.RFC3339), nil
	default:
		return "", fmt.Errorf("unknown tool: %s", strconv.Quote(name))
	}
}

func resolveLocation(tz string) (*time.Location, error) {
	if tz == "" {
		tz = os.Getenv(defaultTzEnv)
	}
	if tz == "" {
		return time.Local, nil
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("unknown timezone %q", tz)
	}
	return loc, nil
}

func stringArg(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	s, _ := args[key].(string)
	return s
}
