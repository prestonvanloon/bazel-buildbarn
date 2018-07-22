package builder

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"syscall"

	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

const (
	pathBuildRoot = "/build"
)

type localBuildExecutor struct {
	contentAddressableStorage blobstore.BlobAccess
}

func NewLocalBuildExecutor(contentAddressableStorage blobstore.BlobAccess) BuildExecutor {
	return &localBuildExecutor{
		contentAddressableStorage: contentAddressableStorage,
	}
}

func (be *localBuildExecutor) createInputFile(instance string, digest *remoteexecution.Digest, base string, isExecutable bool) error {
	var mode os.FileMode = 0444
	if isExecutable {
		mode = 0555
	}
	f, err := os.OpenFile(base, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	defer f.Close()

	r, err := be.contentAddressableStorage.Get(instance, digest)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, r)
	return err
}

func (be *localBuildExecutor) createInputDirectory(instance string, digest *remoteexecution.Digest, base string) error {
	if err := os.Mkdir(base, 0555); err != nil {
		return err
	}

	var directory remoteexecution.Directory
	if err := blobstore.GetMessageFromBlobAccess(be.contentAddressableStorage, instance, digest, &directory); err != nil {
		return err
	}

	for _, file := range directory.Files {
		// TODO(edsch): Path validation?
		if err := be.createInputFile(instance, file.Digest, path.Join(base, file.Name), file.IsExecutable); err != nil {
			return err
		}
	}
	for _, directory := range directory.Directories {
		// TODO(edsch): Path validation?
		if err := be.createInputDirectory(instance, directory.Digest, path.Join(base, directory.Name)); err != nil {
			return err
		}
	}
	return nil
}

func (be *localBuildExecutor) prepareFilesystem(request *remoteexecution.ExecuteRequest) error {
	// Copy input files into build environment.
	os.RemoveAll(pathBuildRoot)
	if err := be.createInputDirectory(request.InstanceName, request.Action.InputRootDigest, pathBuildRoot); err != nil {
		log.Print("Execution.Execute: ", err)
		return err
	}

	// Create writable directories for all output files.
	for _, outputFile := range request.Action.OutputFiles {
		// TODO(edsch): Path validation?
		if err := os.MkdirAll(path.Dir(path.Join(pathBuildRoot, outputFile)), 0555); err != nil {
			return err
		}
	}
	for _, outputFile := range request.Action.OutputFiles {
		// TODO(edsch): Path validation?
		if err := os.Chmod(path.Dir(path.Join(pathBuildRoot, outputFile)), 0777); err != nil {
			return err
		}
	}
	if len(request.Action.OutputDirectories) != 0 {
		return errors.New("Output directories not yet supported!")
	}

	// Provide a clean temp directory.
	os.RemoveAll("/tmp")
	return os.Mkdir("/tmp", 0777)
}

func (be *localBuildExecutor) Execute(request *remoteexecution.ExecuteRequest) (*remoteexecution.ExecuteResponse, error) {
	if err := be.prepareFilesystem(request); err != nil {
		return nil, err
	}

	// Get command to run.
	var command remoteexecution.Command
	if err := blobstore.GetMessageFromBlobAccess(be.contentAddressableStorage, request.InstanceName, request.Action.CommandDigest, &command); err != nil {
		log.Print("Execution.Execute: ", err)
		return nil, err
	}
	if len(command.Arguments) < 1 {
		return nil, errors.New("Insufficent number of command arguments")
	}

	// Prepare the command to run.
	// TODO(edsch): Use CommandContext(), so we have a proper timeout.
	cmd := exec.Command(command.Arguments[0], command.Arguments[1:]...)
	cmd.Dir = pathBuildRoot
	for _, environmentVariable := range command.EnvironmentVariables {
		cmd.Env = append(cmd.Env, environmentVariable.Name+"="+environmentVariable.Value)
	}
	stdout := bytes.NewBuffer(nil)
	cmd.Stdout = stdout
	stderr := bytes.NewBuffer(nil)
	cmd.Stderr = stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: 1,
			Gid: 1,
		},
	}

	// Run the command.
	exitCode := 0
	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus := exitError.Sys().(syscall.WaitStatus)
			exitCode = waitStatus.ExitStatus()
		} else {
			return nil, err
		}
	}
	response := &remoteexecution.ExecuteResponse{
		Result: &remoteexecution.ActionResult{
			ExitCode:  int32(exitCode),
			StdoutRaw: stdout.Bytes(),
			StderrRaw: stderr.Bytes(),
		},
	}

	// Collect output files.
	for _, outputFile := range request.Action.OutputFiles {
		// TODO(edsch): Sanitize paths?
		file, err := os.Open(path.Join(pathBuildRoot, outputFile))
		if err != nil {
			// TODO(edsch): Bail out of we see something other than ENOENT.
			continue
		}
		defer file.Close()

		info, err := file.Stat()
		if err != nil {
			return nil, err
		}

		// TODO(edsch): Store large files in external blobs.
		content, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, err
		}

		response.Result.OutputFiles = append(response.Result.OutputFiles, &remoteexecution.OutputFile{
			Path:         outputFile,
			Content:      content,
			IsExecutable: (info.Mode() & 0111) != 0,
		})
	}
	// TODO(edsch): Collect output directories.
	return response, nil
}
