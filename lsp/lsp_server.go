package main

import (
	"encoding/json"
	"github.com/kr/pretty"
	"github.com/ybbus/jsonrpc/v2"
	"lsp/logger"
)

const address = "http://127.0.0.1:9877"

var log = logger.Get()

type Workspace struct {
	Name      string `json:"Name"`
	ModuleDir string `json:"ModuleDir"`
}

type WorkspaceResponse struct {
	Workspaces []Workspace `json:"Workspaces"`
}

type LogTraceParams struct {
	Message string `json:"message"`
	Verbose string `json:"verbose,omitempty"`
}

type CreateFilesParams struct {
	/**
	 * An array of all files/folders created in this operation.
	 */
	Files []FileCreate `json:"files"`
}

type FileCreate struct {
	/**
	 * A file:// URI for the location of the file/folder being created.
	 */
	URI string `json:"uri"`
}

type ExecuteCommandParams struct {
	/**
	 * The identifier of the actual command handler.
	 */
	Command string `json:"command"`
	/**
	 * Arguments that the command should be invoked with.
	 */
	Arguments []json.RawMessage `json:"arguments,omitempty"`
	WorkDoneProgressParams
}

type WorkDoneProgressParams struct {
	/**
	 * An optional token that a server can use to report work done progress.
	 */
	WorkDoneToken ProgressToken `json:"workDoneToken,omitempty"`
}

type ProgressToken = interface{} /*number | string*/

func main() {
	log.Infof("connect to lsp server start")
	client := jsonrpc.NewClient(address)
	log.Infof("connect to lsp server success")

	param := ExecuteCommandParams{}
	param.Command = "gopls.workspace_metadata"
	call, err := client.Call("workspace/executeCommand", &param)
	if err != nil {
		log.Errorf("call json rpc method failed. err: %s", err)
	}
	log.Infof("call json rpc response. response:%v", pretty.Sprint(call))
}
