## 使用说明

### LSP和源代码

LSP需要和源代码在一台服务器上，LSP初始化工作空间，工作空间就是LSP服务器上的一个目录（源代码目录）。

可以理解LSP协议就是在源代码目录上执行代码分析等工作。

**注：LSP的工作空间必须是源代码目录。**

### gopls

gopls是Go官方提供的语言服务器，可以完成代码自动补全、错误提示等功能。

#### 执行 go.tidy 初始化 go.sum 文件。执行完成之后会更新 go.sum 文件。

```go
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
//
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
```

### 代码自动补全

```go
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
```