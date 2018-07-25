package builder

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"syscall"

	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"

	"golang.org/x/net/context"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

const (
	pathTempRoot  = "/tmp"
	pathBuildRoot = "/build"
	pathStdout    = "/stdout"
	pathStderr    = "/stderr"
)

type localBuildExecutor struct {
	contentAddressableStorage blobstore.BlobAccess
	inputFileExposer          InputFileExposer
}

func NewLocalBuildExecutor(contentAddressableStorage blobstore.BlobAccess, inputFileExposer InputFileExposer) BuildExecutor {
	return &localBuildExecutor{
		contentAddressableStorage: contentAddressableStorage,
		inputFileExposer:          inputFileExposer,
	}
}

func (be *localBuildExecutor) createInputDirectory(ctx context.Context, instance string, digest *remoteexecution.Digest, base string) error {
	if err := os.Mkdir(base, 0777); err != nil {
		return err
	}

	// TODO(edsch): Translate NOT_FOUND to INVALID_PRECONDITION?
	var directory remoteexecution.Directory
	if err := blobstore.GetMessageFromBlobAccess(be.contentAddressableStorage, ctx, instance, digest, &directory); err != nil {
		return err
	}

	for _, file := range directory.Files {
		// TODO(edsch): Path validation?
		if err := be.inputFileExposer.Expose(ctx, instance, file.Digest, path.Join(base, file.Name), file.IsExecutable); err != nil {
			return err
		}
	}
	for _, directory := range directory.Directories {
		// TODO(edsch): Path validation?
		if err := be.createInputDirectory(ctx, instance, directory.Digest, path.Join(base, directory.Name)); err != nil {
			return err
		}
	}
	return nil
}

func (be *localBuildExecutor) prepareFilesystem(ctx context.Context, request *remoteexecution.ExecuteRequest) error {
	// Copy input files into build environment.
	os.RemoveAll(pathBuildRoot)
	if err := be.createInputDirectory(ctx, request.InstanceName, request.Action.InputRootDigest, pathBuildRoot); err != nil {
		log.Print("Execution.Execute: ", err)
		return err
	}

	// Ensure that directories where output files are stored are present.
	for _, outputFile := range request.Action.OutputFiles {
		// TODO(edsch): Path validation?
		if err := os.MkdirAll(path.Dir(path.Join(pathBuildRoot, outputFile)), 0777); err != nil {
			return err
		}
	}

	// Provide a clean temp directory.
	os.RemoveAll(pathTempRoot)
	return os.Mkdir(pathTempRoot, 0777)
}

func (be *localBuildExecutor) runCommand(ctx context.Context, request *remoteexecution.ExecuteRequest) error {
	// Fetch command.
	// TODO(edsch): Translate NOT_FOUND to INVALID_PRECONDITION?
	var command remoteexecution.Command
	if err := blobstore.GetMessageFromBlobAccess(be.contentAddressableStorage, ctx, request.InstanceName, request.Action.CommandDigest, &command); err != nil {
		log.Print("Execution.Execute: ", err)
		return err
	}
	if len(command.Arguments) < 1 {
		return errors.New("Insufficent number of command arguments")
	}

	// Prepare the command to run.
	cmd := exec.CommandContext(ctx, command.Arguments[0], command.Arguments[1:]...)
	cmd.Dir = pathBuildRoot
	cmd.Env = []string{"HOME=" + pathTempRoot}
	for _, environmentVariable := range command.EnvironmentVariables {
		cmd.Env = append(cmd.Env, environmentVariable.Name+"="+environmentVariable.Value)
	}

	// Output streams.
	stdout, err := os.OpenFile(pathStdout, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0)
	if err != nil {
		return err
	}
	defer stdout.Close()
	cmd.Stdout = stdout
	stderr, err := os.OpenFile(pathStderr, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0)
	if err != nil {
		return err
	}
	defer stderr.Close()
	cmd.Stderr = stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: 1,
			Gid: 1,
		},
	}
	return cmd.Run()
}

func (be *localBuildExecutor) uploadFile(ctx context.Context, instance string, path string) (*remoteexecution.Digest, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, false, err
	}

	// Walk through the file to compute the digest.
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, false, err
	}
	digest := &remoteexecution.Digest{
		Hash:      hex.EncodeToString(hasher.Sum(nil)),
		SizeBytes: info.Size(),
	}
	if _, err := file.Seek(0, 0); err != nil {
		return nil, false, err
	}

	// Store in content addressable storage.
	if err := be.contentAddressableStorage.Put(ctx, instance, digest, file); err != nil {
		return nil, false, err
	}
	return digest, (info.Mode() & 0111) != 0, nil
}

func (be *localBuildExecutor) uploadDirectory(ctx context.Context, instance string, basePath string, permitNonExistent bool, children map[string]*remoteexecution.Directory) (*remoteexecution.Directory, error) {
	files, err := ioutil.ReadDir(basePath)
	if err != nil {
		if permitNonExistent && os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var directory remoteexecution.Directory
	for _, file := range files {
		name := file.Name()
		fullPath := path.Join(basePath, name)
		switch file.Mode() & os.ModeType {
		case 0:
			digest, isExecutable, err := be.uploadFile(ctx, instance, fullPath)
			if err != nil {
				return nil, err
			}
			directory.Files = append(directory.Files, &remoteexecution.FileNode{
				Name:         name,
				Digest:       digest,
				IsExecutable: isExecutable,
			})
		case os.ModeDir:
			child, err := be.uploadDirectory(ctx, instance, path.Join(basePath, name), false, children)
			if err != nil {
				return nil, err
			}
			digest, err := util.DigestFromMessage(child)
			if err != nil {
				return nil, err
			}
			children[digest.Hash] = child
			directory.Directories = append(directory.Directories, &remoteexecution.DirectoryNode{
				Name:   name,
				Digest: digest,
			})
		default:
			return nil, fmt.Errorf("Path %s has an unsupported file type", basePath)
		}
	}
	return &directory, nil
}

func (be *localBuildExecutor) uploadTree(ctx context.Context, instance string, path string) (*remoteexecution.Digest, error) {
	// Gather all individual directory objects and turn them into a tree.
	children := map[string]*remoteexecution.Directory{}
	root, err := be.uploadDirectory(ctx, instance, path, true, children)
	if root == nil || err != nil {
		return nil, err
	}
	tree := &remoteexecution.Tree{
		Root: root,
	}
	for _, child := range children {
		tree.Children = append(tree.Children, child)
	}

	// Upload the tree.
	digest, err := util.DigestFromMessage(tree)
	if err != nil {
		return nil, err
	}
	if err := blobstore.PutMessageToBlobAccess(be.contentAddressableStorage, ctx, instance, digest, tree); err != nil {
		return nil, err
	}
	return digest, nil
}

func (be *localBuildExecutor) Execute(ctx context.Context, request *remoteexecution.ExecuteRequest) (*remoteexecution.ExecuteResponse, error) {
	// Set up inputs.
	if err := be.prepareFilesystem(ctx, request); err != nil {
		return nil, err
	}

	// Invoke command.
	exitCode := 0
	if err := be.runCommand(ctx, request); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus := exitError.Sys().(syscall.WaitStatus)
			exitCode = waitStatus.ExitStatus()
		} else {
			return nil, err
		}
	}

	// Upload command output.
	stdoutDigest, _, err := be.uploadFile(ctx, request.InstanceName, pathStdout)
	if err != nil {
		return nil, err
	}
	stderrDigest, _, err := be.uploadFile(ctx, request.InstanceName, pathStderr)
	if err != nil {
		return nil, err
	}

	response := &remoteexecution.ExecuteResponse{
		Result: &remoteexecution.ActionResult{
			ExitCode:     int32(exitCode),
			StdoutDigest: stdoutDigest,
			StderrDigest: stderrDigest,
		},
	}

	// Upload output files.
	for _, outputFile := range request.Action.OutputFiles {
		// TODO(edsch): Sanitize paths?
		digest, isExecutable, err := be.uploadFile(ctx, request.InstanceName, path.Join(pathBuildRoot, outputFile))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		response.Result.OutputFiles = append(response.Result.OutputFiles, &remoteexecution.OutputFile{
			Path:         outputFile,
			Digest:       digest,
			IsExecutable: isExecutable,
		})
	}

	// Upload output directories.
	for _, outputDirectory := range request.Action.OutputDirectories {
		// TODO(edsch): Sanitize paths?
		digest, err := be.uploadTree(ctx, request.InstanceName, path.Join(pathBuildRoot, outputDirectory))
		if err != nil {
			return nil, err
		}
		if digest != nil {
			response.Result.OutputDirectories = append(response.Result.OutputDirectories, &remoteexecution.OutputDirectory{
				Path:       outputDirectory,
				TreeDigest: digest,
			})
		}
	}
	return response, nil
}
