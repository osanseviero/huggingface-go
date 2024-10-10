# huggingface-go

A minimal, unofficial, community-contributed PoC client library for interacting with the Hugging Face Hub with Go. Currently, the focus will be on downloading/uploading files from model repositories. PRs are welcome for features outside v0.

## Roadmap

- [x] Repository creation
- [x] Upload file to the Hub
- [x] Upload LFS file to the Hub
- [x] Download single-file from the Hub
- [x] Get model info (including list of model files)
- [ ] Multipart uploads
- [ ] Pull Requests
- [ ] Multifile upload
- [ ] Advanced repository management
- [ ] Support for datasets and Spaces repositories. Upload assumes it's a model repository.

The library intends to be minimal. For feature-complete options, check the official `huggingface_hub` Python or TypeScript libraries or the CLI.

## Minimal usage

In your Go project, run

```bash
go get github.com/osanseviero/huggingface-go
```

Then, you can perform different actions

```go
// Create a new Hugging Face Hub client
client, err := huggingface.NewHubClient()

// Create a repo
repoName := "osanseviero/test-in-go8" 
createRepoOptions := &huggingface.CreateRepoOptions{
  ExistsOK: true,
}
_, err = client.CreateRepo(repoName, "model", createRepoOptions)

// Upload normal file
err = client.UploadFile(repoName, "model", "test.txt")

// Upload LFS
lfsFilePath := "test/tokenizer.json"
err = client.UploadFile(repoName, "model", lfsFilePath)

// Download file
err = client.DownloadFile(repoName, "model", "tokenizer.json", "path/tokenizer.json")

// Iterate over all siblings
for _, sibling := range modelInfo.Siblings {
  fmt.Println("Sibling:", sibling.Rfilename)
}

// Download all files in a repo
for _, sibling := range modelInfo.Siblings {
  // Download the file
  err = client.DownloadFile(repoName, "model", sibling.Rfilename, "path/" + sibling.Rfilename)
  if err != nil {
    fmt.Println("Error downloading file:", err)
  }
}
```
