package main

import (
	"fmt"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

const (
	CtxStreamingBodyInvoked = "streaming_body_invoked"
	CtxResponseEncoding     = "response_content_encoding"
	CtxRequestAccept        = "request_accept_encoding"
	CtxResponseStatus       = "response_status"
)

type PluginConfig struct {
	DebugMode bool
}

func main() {}

func init() {
	wrapper.SetCtx(
		"content-encoding-repro-higress-plugin",
		wrapper.ParseConfig(parseConfig),
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		wrapper.ProcessResponseHeaders(onHttpResponseHeaders),
		wrapper.ProcessStreamingResponseBody(onHttpStreamingResponseBody),
		wrapper.ProcessStreamDone(onHttpStreamDone),
	)
}

func parseConfig(json gjson.Result, config *PluginConfig) error {
	config.DebugMode = json.Get("debugMode").Bool()
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config PluginConfig) types.Action {
	acceptEncoding, _ := proxywasm.GetHttpRequestHeader("Accept-Encoding")
	ctx.SetContext(CtxRequestAccept, acceptEncoding)
	ctx.SetContext(CtxStreamingBodyInvoked, false)

	if config.DebugMode {
		log.Warnf("[content-encoding-repro] request Accept-Encoding=%q", acceptEncoding)
	}

	return types.ActionContinue
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config PluginConfig) types.Action {
	responseEncoding, _ := proxywasm.GetHttpResponseHeader("content-encoding")
	status, _ := proxywasm.GetHttpResponseHeader(":status")
	ctx.SetContext(CtxResponseEncoding, responseEncoding)
	ctx.SetContext(CtxResponseStatus, status)

	if config.DebugMode {
		log.Warnf("[content-encoding-repro] response :status=%q content-encoding=%q", status, responseEncoding)
	}

	return types.ActionContinue
}

func onHttpStreamingResponseBody(ctx wrapper.HttpContext, config PluginConfig, chunk []byte, endOfStream bool) []byte {
	ctx.SetContext(CtxStreamingBodyInvoked, true)

	if config.DebugMode {
		log.Warnf("[content-encoding-repro] streaming callback invoked: chunk=%d endOfStream=%v", len(chunk), endOfStream)
	}

	return chunk
}

func onHttpStreamDone(ctx wrapper.HttpContext, config PluginConfig) {
	acceptEncoding, _ := ctx.GetContext(CtxRequestAccept).(string)
	responseEncoding, _ := ctx.GetContext(CtxResponseEncoding).(string)
	status, _ := ctx.GetContext(CtxResponseStatus).(string)
	invoked, _ := ctx.GetContext(CtxStreamingBodyInvoked).(bool)

	state := "NOT_EXECUTED"
	if invoked {
		state = "EXECUTED"
	}

	log.Warnf(
		"[content-encoding-repro] stream done: callback=%s request.accept-encoding=%q response.content-encoding=%q response.status=%q",
		state,
		acceptEncoding,
		responseEncoding,
		status,
	)

	if config.DebugMode {
		log.Warnf("[content-encoding-repro] summary=%s", fmt.Sprintf("callback=%s", state))
	}
}
