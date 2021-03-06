package main

import (
	"context"
	"encoding/json"
	"github.com/kr/pretty"
	"github.com/sourcegraph/jsonrpc2"
	"io/ioutil"
	"lsp/logger"
	"lsp/protocol"
	"net"
	"sync"
)

const (
	workSpaceName = "test"
	workSpaceURI  = "file:///home/xiaotian/test/"
	//workSpaceURI  = "file:///d:\\test\\"
	codeTemplate  = "./data/template.go"
	modTemplate   = "./data/go.mod"
	sumTemplate   = "./data/go.sum"
)

var log = logger.Get()

type ServerConfig struct {
	NetWork string
	Address string
}

type LanguageServer struct {
	initialized bool
	started     bool

	mutex        sync.Mutex
	ctx          context.Context
	conn         net.Conn
	rpcConn      *jsonrpc2.Conn
	serverConfig ServerConfig
}

func InitLanguageServer(ctx context.Context, config ServerConfig) *LanguageServer {
	server := LanguageServer{}
	server.ctx = ctx
	server.serverConfig.NetWork = config.NetWork
	server.serverConfig.Address = config.Address
	server.initialized = true
	return &server
}

func (lsp *LanguageServer) Start() {
	lsp.mutex.Lock()
	defer lsp.mutex.Unlock()
	if lsp.started {
		log.Infof("LanguageServer start server, server have started")
		return
	}

	lsp.fatalfIfNotInit()

	log.Infof("LanguageServer start server")
	conn, err := net.Dial(lsp.serverConfig.NetWork, lsp.serverConfig.Address)
	if err != nil {
		log.Fatalf("LanguageServer net dial failed. network:%s, address:%s", lsp.serverConfig.NetWork, lsp.serverConfig.Address)
	}
	lsp.conn = conn

	stream := jsonrpc2.NewBufferedStream(conn, jsonrpc2.VSCodeObjectCodec{})
	client := jsonrpc2.NewConn(lsp.ctx, stream, &LSPHandler{}, jsonrpc2.LogMessages(&Logger{}))
	lsp.rpcConn = client

	go lsp.serverListenerLoop()

	lsp.started = true

	log.Infof("LanguageServer start success")
}

func (lsp *LanguageServer) serverListenerLoop() {
	for {
		select {
		case <-lsp.ctx.Done():
			log.Infof("lsp context done")
			lsp.Shutdown()
		}
	}
}

func (lsp *LanguageServer) Shutdown() {
	err := lsp.rpcConn.Close()
	if err != nil {
		log.Warnf("LanguageServer Shutdown close rpc connection failed. err:%s", err)
	}
}

func (lsp *LanguageServer) InitWorkSpace(name, uri string) {
	log.Infof("LanguageServer InitWorkSpace start. name:%s, uri:%s", name, uri)

	initializeParams := protocol.InitializeParams{}
	initializeParams.ClientInfo.Name = "test"
	initializeParams.ClientInfo.Version = "v1.0.2"
	initializeParams.WorkspaceFolders = []protocol.WorkspaceFolder{{Name: name, URI: uri}}

	initializeResult := protocol.InitializeResult{}
	err := lsp.rpcConn.Call(lsp.ctx, "initialize", initializeParams, &initializeResult, nil)
	if err != nil {
		log.Errorf("InitWorkSpace call json rpc method `initialize` failed. err: %s", err)
	}

	log.Infof("InitWorkSpace initialize response: %s", pretty.Sprint(initializeResult))

	err = lsp.rpcConn.Call(lsp.ctx, "initialized", protocol.InitializedParams{}, nil, nil)
	if err != nil {
		log.Errorf("InitWorkSpace call json rpc method `initialized` failed. err: %s", err)
	}
	log.Infof("LanguageServer InitWorkSpace success")
}

func (lsp *LanguageServer) DidOpenTextDocument(url, text, languageId string) {
	log.Infof("DidOpenTextDocument start")
	didOpenParam := protocol.DidOpenTextDocumentParams{}
	didOpenParam.TextDocument.URI = protocol.DocumentURI(url)
	didOpenParam.TextDocument.Version = 0
	didOpenParam.TextDocument.LanguageID = languageId
	didOpenParam.TextDocument.Text = text
	err := lsp.rpcConn.Call(lsp.ctx, "textDocument/didOpen", didOpenParam, nil, nil)
	if err != nil {
		log.Errorf("DidOpenTextDocument call json rpc method [textDocument/didOpen] failed. err: %s", err)
	}
}

func (lsp *LanguageServer) DidSaveTextDocument(url, data string) {
	log.Infof("DidSaveTextDocument start")
	didSaveParam := protocol.DidSaveTextDocumentParams{}
	didSaveParam.TextDocument.URI = protocol.DocumentURI(url)
	didSaveParam.Text = &data
	err := lsp.rpcConn.Call(lsp.ctx, "textDocument/didSave", didSaveParam, nil, nil)
	if err != nil {
		log.Errorf("DidSaveTextDocument call json rpc method [textDocument/didSave] failed. err: %s", err)
	}

	log.Infof("DidSaveTextDocument success.")
}

func (lsp *LanguageServer) ExecuteGoModTidy(uri string) {
	log.Infof("ExecuteGoModTidy start")
	type uris struct {
		URIs []string `json:"URIs"`
	}
	u := uris{
		URIs: []string{uri},
	}

	marshal, err := json.Marshal(u)
	if err != nil {
		log.Errorf("[ExecuteGoModTidy] marshal data faild. err: %s", err)
	}

	executeParams := protocol.ExecuteCommandParams{}
	executeParams.Command = "gopls.tidy"
	executeParams.Arguments = []json.RawMessage{marshal}
	executeParams.WorkDoneToken = "11111111111111111"
	response := make(map[string]interface{})
	err = lsp.rpcConn.Call(lsp.ctx, "workspace/executeCommand", &executeParams, &response, nil)
	if err != nil {
		log.Errorf("call json rpc method [workspace/executeCommand gopls.tidy] failed. err: %s", err)
	}

	log.Infof("ExecuteGoModTidy success: %s", pretty.Sprint(response))
}

func (lsp *LanguageServer) ExecuteGoModGenerate(uri string) {
	log.Infof("ExecuteGoModGenerate start")
	type uriS struct {
		URI string `json:"URI"`
	}
	u := uriS{
		URI: uri,
	}

	marshal, err := json.Marshal(u)
	if err != nil {
		log.Errorf("[ExecuteGoModGenerate] marshal data faild. err: %s", err)
	}

	executeParams := protocol.ExecuteCommandParams{}
	executeParams.Command = "gopls.generate_gopls_mod"
	executeParams.Arguments = []json.RawMessage{marshal}
	executeParams.WorkDoneToken = "11111111111111111"
	response := make(map[string]interface{})
	err = lsp.rpcConn.Call(lsp.ctx, "workspace/executeCommand", &executeParams, &response, nil)
	if err != nil {
		log.Errorf("call json rpc method [workspace/executeCommand gopls.generate_gopls_mod] failed. err: %s", err)
	}

	log.Infof("ExecuteGoModGenerate success: %s", pretty.Sprint(response))
}

func (lsp *LanguageServer) Completion(uri string, line uint32, character uint32) {
	////????????????

	completionParams := protocol.CompletionParams{}
	completionParams.WorkDoneToken = "333333333333333333333"
	completionParams.TextDocument.URI = protocol.DocumentURI(uri)
	completionParams.Position.Line = line
	completionParams.Position.Character = character
	log.Infof("textDocument/completion request: %s", pretty.Sprint(completionParams))
	completionList := protocol.CompletionList{}
	err := lsp.rpcConn.Call(context.Background(), "textDocument/completion", &completionParams, &completionList, nil)
	if err != nil {
		log.Errorf("call json rpc method failed. err: %s", err)
	}
	log.Infof("textDocument/completion: %s", pretty.Sprint(completionList))

}

func (lsp *LanguageServer) fatalfIfNotInit() {
	if !lsp.initialized {
		log.Fatalf("start language server failed. please init first")
	}
}

func main() {
	ctx := context.Background()
	languageServer := InitLanguageServer(ctx, ServerConfig{NetWork: "tcp", Address: "192.168.88.201:9877"})
	languageServer.Start()
	languageServer.InitWorkSpace(workSpaceName, workSpaceURI)

	//languageServer.DidOpenTextDocument(workSpaceURI+"/hello/", "")

	helloURI := workSpaceURI + "hello.go"
	codeData := readData(codeTemplate)
	languageServer.DidOpenTextDocument(helloURI, codeData, "go")
	languageServer.DidSaveTextDocument(helloURI, codeData)

	modURI := workSpaceURI + "go.mod"
	modData := readData(modTemplate)
	languageServer.DidOpenTextDocument(modURI, modData, "go.mod")
	languageServer.DidSaveTextDocument(modURI, modData)

	//sumURI := workSpaceURI + "go.sum"
	//sumData := readData(sumTemplate)
	//languageServer.DidOpenTextDocument(sumURI, sumData, "go.sum")
	//languageServer.DidSaveTextDocument(sumURI, sumData)

	languageServer.ExecuteGoModTidy(modURI)

	//fmt.
	languageServer.Completion(helloURI, 7, 5)
	//io_tool.
	languageServer.Completion(helloURI, 8, 20)
	//Per
	languageServer.Completion(helloURI, 9, 21)

	languageServer.Completion(helloURI, 10, 20)

	languageServer.Shutdown()

	////???????????????????????????
	//executeParams1 := ExecuteCommandParams{}
	//executeParams1.Command = "gopls.tidy"
	//executeParams1.Arguments = []json.RawMessage{[]byte(`{"URI":"file://test"}`)}
	//executeParams1.WorkDoneToken = "11111111111111111"
	//response = make(map[string]interface{})
	////executeParams.Arguments = []json.RawMessage{[]byte("[version]")}
	//err = client.Call(ctx, "workspace/executeCommand", &executeParams1, &response, nil)
	//if err != nil {
	//	log.Errorf("gopls.tidy call json rpc method failed. err: %s", err)
	//}
	//log.Infof("workspace/executeCommand gopls.tidy: %s", pretty.Sprint(response))
	//
	////???????????????????????????
	//executeParams2 := ExecuteCommandParams{}
	//executeParams2.Command = "gopls.vendor"
	//executeParams2.Arguments = []json.RawMessage{[]byte(`{"URI":"file://test/"}`)}
	//executeParams2.WorkDoneToken = "xxxxxxxxxxxxxxxx"
	//response = make(map[string]interface{})
	////executeParams.Arguments = []json.RawMessage{[]byte("[version]")}
	//err = client.Call(ctx, "workspace/executeCommand", &executeParams2, &response, nil)
	//if err != nil {
	//	log.Errorf("gopls.vendor call json rpc method failed. err: %s", err)
	//}
	//log.Infof("workspace/executeCommand gopls.vendor: %s", pretty.Sprint(response))
	//
	//filesParams := DidChangeWatchedFilesParams{}
	//filesParams.Changes = []FileEvent{{URI: "file://test//go.mod", Type: 6}, {URI: "file://test//hello.go", Type: 6}}
	//response = make(map[string]interface{})
	//err = client.Call(ctx, "workspace/didChangeWatchedFiles", filesParams, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("workspace/didChangeWatchedFiles response: %s", pretty.Sprint(response))
	//
	//didOpenParam := DidOpenTextDocumentParams{}
	//didOpenParam.TextDocument.URI = "file://test//hello.go"
	//didOpenParam.TextDocument.Version = 0
	//didOpenParam.TextDocument.LanguageID = "go"
	//didOpenParam.TextDocument.Text = "package main\n\nimport \"fmt\"\nimport \"github.com/sourcegraph/jsonrpc2\"\n\nfunc main() {\n\tresult := jsonrpc2.NewBufferedStream(nil, nil)\n\tfmt.Println(result)\n\tfmt.Println()\n}\n"
	//response = make(map[string]interface{})
	//err = client.Call(ctx, "textDocument/didOpen", didOpenParam, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/didOpen response: %s", pretty.Sprint(response))
	//
	//didSaveTextDocumentParams := DidSaveTextDocumentParams{}
	//didSaveTextDocumentParams.TextDocument.URI = "file://test//hello.go"
	//data := "package main\n\nimport \"fmt\"\nimport \"github.com/sourcegraph/jsonrpc2\"\n\nfunc main() {\n\tresult := jsonrpc2.NewBufferedStream(nil, nil)\n\tfmt.Println(result)\n\tfmt.Println()\n}\n"
	//didSaveTextDocumentParams.Text = &data
	//response = make(map[string]interface{})
	//err = client.Call(ctx, "textDocument/didSave", &didSaveTextDocumentParams, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/didSave: %s", pretty.Sprint(response))
	//
	//didOpenParam = DidOpenTextDocumentParams{}
	//didOpenParam.TextDocument.URI = "file://test//go.mod"
	//didOpenParam.TextDocument.Version = 0
	//didOpenParam.TextDocument.LanguageID = "go"
	//didOpenParam.TextDocument.Text = "module hello\n\ngo 1.16\n"
	//response = make(map[string]interface{})
	//err = client.Call(ctx, "textDocument/didOpen", didOpenParam, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/didOpen response: %s", pretty.Sprint(response))
	//
	//didSaveTextDocumentParams = DidSaveTextDocumentParams{}
	//didSaveTextDocumentParams.TextDocument.URI = "file://test//go.mod"
	//data = "module hello\n\ngo 1.16\n"
	//didSaveTextDocumentParams.Text = &data
	//response = make(map[string]interface{})
	//err = client.Call(ctx, "textDocument/didSave", &didSaveTextDocumentParams, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/didSave: %s", pretty.Sprint(response))
	//
	//completionParams := CompletionParams{}
	//completionParams.WorkDoneToken = "333333333333333333333"
	//completionParams.TextDocument.URI = "file://test//hello.go"
	//completionParams.Position.Line = 1
	//completionParams.Position.Character = 4
	//
	////????????????
	//log.Infof("textDocument/completion request: %s", pretty.Sprint(completionParams))
	//completionList := CompletionList{}
	//err = client.Call(ctx, "textDocument/completion", &completionParams, &completionList, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/completion: %s", pretty.Sprint(response))
	//
	////????????????
	//hoverParams := HoverParams{}
	//hoverParams.TextDocument.URI = "file://test//hello.go"
	//hoverParams.Position.Line = 8
	//hoverParams.Position.Character = 10
	//hoverParams.WorkDoneToken = "44444444444444"
	//response = make(map[string]interface{})
	//log.Infof("textDocument/hover request: %s", pretty.Sprint(hoverParams))
	//err = client.Call(ctx, "textDocument/hover", &hoverParams, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/hover: %s", pretty.Sprint(response))
	//
	////????????????
	//signatureHelpParams := SignatureHelpParams{}
	//signatureHelpParams.TextDocument.URI = "file://test//hello.go"
	//signatureHelpParams.Position.Line = 9
	//signatureHelpParams.Position.Character = 17
	//signatureHelpParams.WorkDoneToken = "5555555555555"
	//response = make(map[string]interface{})
	//log.Infof("textDocument/signatureHelp request: %s", pretty.Sprint(signatureHelpParams))
	//err = client.Call(ctx, "textDocument/signatureHelp", &signatureHelpParams, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/signatureHelp: %s", pretty.Sprint(response))
	//
	//////????????????
	////definitionParams := DefinitionParams{}
	////definitionParams.TextDocument.URI = "file://test//hello.go"
	////definitionParams.Position.Line = 9
	////definitionParams.Position.Character = 12
	////definitionParams.WorkDoneToken = "6666666666666666666"
	////response = make(map[string]interface{})
	////log.Infof("textDocument/definition request: %s", pretty.Sprint(definitionParams))
	////err = client.Call(ctx, "textDocument/definition", &definitionParams, &response, nil)
	////if err != nil {
	////	log.Errorf("call json rpc method failed. err: %s", err)
	////}
	////log.Infof("textDocument/definition: %s", pretty.Sprint(response))
	//
	////????????????
	//referenceParams := ReferenceParams{}
	//referenceParams.TextDocument.URI = "file://test//hello.go"
	//referenceParams.Position.Line = 7
	//referenceParams.Position.Character = 5
	//referenceParams.WorkDoneToken = "77777777777777777777"
	//responseLocation := make([]Location, 0)
	//log.Info("textDocument/references request: %s", pretty.Sprint(referenceParams))
	//err = client.Call(ctx, "textDocument/references", &referenceParams, &responseLocation, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/references: %s", pretty.Sprint(responseLocation))
	//
	////????????????
	//documentHighlightParams := DocumentHighlightParams{}
	//documentHighlightParams.TextDocument.URI = "file://test//hello.go"
	//documentHighlightParams.Position.Line = 7
	//documentHighlightParams.Position.Character = 5
	//documentHighlightParams.WorkDoneToken = "888888888888888"
	//documentHighlightResponse := make([]DocumentHighlight, 0)
	//log.Info("textDocument/documentHighlight request: %s", pretty.Sprint(documentHighlightParams))
	//err = client.Call(ctx, "textDocument/documentHighlight", &referenceParams, &documentHighlightResponse, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/documentHighlight: %s", pretty.Sprint(documentHighlightResponse))
	//
	////????????????
	//documentSymbolParams := DocumentSymbolParams{}
	//documentSymbolParams.TextDocument.URI = "file://test//hello.go"
	//documentSymbolParams.WorkDoneToken = "99999999999999999999"
	//documentSymbolResponse := make([]DocumentSymbol, 0)
	//log.Info("textDocument/documentSymbol request: %s", pretty.Sprint(documentSymbolParams))
	//err = client.Call(ctx, "textDocument/documentSymbol", &documentSymbolParams, &documentSymbolResponse, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/documentSymbol: %s", pretty.Sprint(documentSymbolResponse))
	//
	////???????????????????????????
	//codeActionParams := CodeActionParams{}
	//codeActionParams.TextDocument.URI = "file://test//hello.go"
	//codeActionParams.WorkDoneToken = "1010101010101010"
	//codeActionParams.Range.Start.Line = 7
	//codeActionParams.Range.Start.Character = 0
	//codeActionParams.Range.End.Line = 8
	//codeActionParams.Range.End.Line = 0
	//codeActionResponse := make([]CodeAction, 0)
	//log.Info("textDocument/codeAction request: %s", pretty.Sprint(codeActionParams))
	//err = client.Call(ctx, "textDocument/codeAction", &codeActionParams, &codeActionResponse, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/codeAction: %s", pretty.Sprint(codeActionResponse))
	//
	////??????????????????
	//codeLensParams := CodeLensParams{}
	//codeLensParams.TextDocument.URI = "file://test//hello.go"
	//codeLensParams.WorkDoneToken = "1212121212121212"
	//codeLensResponse := CodeLens{}
	//log.Info("textDocument/codeLens: %s", pretty.Sprint(codeLensParams))
	//err = client.Call(ctx, "textDocument/codeLens", &codeLensParams, &codeLensResponse, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/codeLens: %s", pretty.Sprint(codeLensResponse))
	//
	////????????????
	//documentColorParams := DocumentColorParams{}
	//documentColorParams.TextDocument.URI = "file://test//hello.go"
	//documentColorParams.WorkDoneToken = "1313131313131313131313"
	//colorInformation := ColorInformation{}
	//log.Info("textDocument/documentColor: %s", pretty.Sprint(documentColorParams))
	//err = client.Call(ctx, "textDocument/documentColor", &documentColorParams, &colorInformation, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/documentColor: %s", pretty.Sprint(colorInformation))
	//
	////?????????
	//renameParams := RenameParams{}
	//renameParams.TextDocument.URI = "file://test//hello.go"
	//renameParams.Position.Line = 6
	//renameParams.Position.Character = 3
	//renameParams.NewName = "aa"
	//renameParams.WorkDoneToken = "1414141414141441411441"
	//renameResponse := WorkspaceEdit{}
	//log.Info("textDocument/rename: %s", pretty.Sprint(renameParams))
	//err = client.Call(ctx, "textDocument/rename", &renameParams, &renameResponse, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/rename: %s", pretty.Sprint(renameResponse))
	//
	////???????????????????????????
	//executeParams := ExecuteCommandParams{}
	//executeParams.Command = "gopls.list_known_packages"
	//executeParams.WorkDoneToken = "2222222222222222"
	//executeParams.Arguments = []json.RawMessage{[]byte(`{"URI":"file://test/hello1.go"}`)}
	//response = make(map[string]interface{})
	////executeParams.Arguments = []json.RawMessage{[]byte("[version]")}
	//err = client.Call(ctx, "workspace/executeCommand", &executeParams, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("workspace/executeCommand: %s", pretty.Sprint(response))
	//
	////???????????????????????????
	//params := WorkspaceSymbolParams{}
	//params.Query = "math"
	//symbolResponse := make([]interface{}, 0)
	//err = client.Call(ctx, "workspace/symbol", &params, &symbolResponse, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("workspace/symbol: %s", pretty.Sprint(symbolResponse))
	//
	////todo not implement
	////createFilesParams := CreateFilesParams{}
	////createFilesParams.Files = []FileCreate{{URI: "file://hello1.go"}}
	////err = client.Call(ctx, "workspace/didCreateFiles", &createFilesParams, &response, nil)
	////if err != nil {
	////	log.Errorf("call json rpc method failed. err: %s", err)
	////}
	////log.Infof("workspace/didCreateFiles: %s", pretty.Sprint(response))
	//
	//didOpenParam = DidOpenTextDocumentParams{}
	//didOpenParam.TextDocument.URI = "file://test//hello1.go"
	//didOpenParam.TextDocument.Version = 0
	//didOpenParam.TextDocument.LanguageID = "GO"
	//didOpenParam.TextDocument.Text = "package main\n\nimport \"fmt\"\n\nfunc main(){\n    fmt.Println(\"Hello\")\n}"
	//response = make(map[string]interface{})
	//err = client.Call(ctx, "textDocument/didOpen", didOpenParam, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/didOpen response: %s", pretty.Sprint(response))
	//
	//didChangeTextDocumentParams := DidChangeTextDocumentParams{}
	//didChangeTextDocumentParams.TextDocument.URI = "file://test//hello1.go"
	//didChangeTextDocumentParams.TextDocument.Version = 0
	//didChangeTextDocumentParams.ContentChanges = make([]TextDocumentContentChangeEvent, 1)
	//didChangeTextDocumentParams.ContentChanges[0].Text = "package main\n\nimport \"fmt\"\n\nfunc main(){\n    fmt.Println(\"Hello\")\n}"
	//response = make(map[string]interface{})
	//err = client.Call(ctx, "textDocument/didChange", didChangeTextDocumentParams, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/didChange response: %s", pretty.Sprint(response))
	//
	//didSaveTextDocumentParams = DidSaveTextDocumentParams{}
	//didSaveTextDocumentParams.TextDocument.URI = "file://test//hello1.go"
	//data = "package main\n\nimport \"fmt\"\n\nfunc main(){\n    fmt.Println(\"Hello\")\n}"
	//didSaveTextDocumentParams.Text = &data
	//response = make(map[string]interface{})
	//err = client.Call(ctx, "textDocument/didSave", didSaveTextDocumentParams, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/didSave: %s", pretty.Sprint(response))
	//
	//didCloseTextDocumentParams := DidCloseTextDocumentParams{}
	//didCloseTextDocumentParams.TextDocument.URI = "file://test//hello1.go"
	//response = make(map[string]interface{})
	//err = client.Call(ctx, "textDocument/didClose", didCloseTextDocumentParams, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/didClose: %s", pretty.Sprint(response))
	//
	//response = make(map[string]interface{})
	//err = client.Call(ctx, "shutdown", nil, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("shutdown")
	//
	//response = make(map[string]interface{})
	//err = client.Call(ctx, "exit", nil, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//client.Close()
	//log.Infof("exit")
	//
	////referenceParams := ReferenceParams{}
	////referenceParams.TextDocument.URI = "file://test//test.go"
	////referenceParams.Position.Line = 6
	////referenceParams.Position.Character = 6
	////referenceParams.Context = ReferenceContext{IncludeDeclaration: true}
	////response = make(map[string]interface{})
	////err = client.Call(ctx, "textDocument/references", referenceParams, &response, nil)
	////if err != nil {
	////	log.Errorf("call json rpc method failed. err: %s", err)
	////}
	////log.Infof("initialized: %s", pretty.Sprint(response))

}

func readData(filePath string) string {
	fileData, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic(err)
	}
	return string(fileData)
}

type LSPHandler struct {
}

func (l *LSPHandler) Handle(context context.Context, conn *jsonrpc2.Conn, request *jsonrpc2.Request) {
	result := make(map[string]interface{})
	bytes := []byte(*request.Params)
	json.Unmarshal(bytes, &result)
	log.Infof("method:%s, message:%s", request.Method, pretty.Sprint(result["message"]))
}

type Logger struct {
}

func (l *Logger) Printf(format string, v ...interface{}) () {
	log.Infof(format, v...)
}
