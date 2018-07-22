package builder

import (
	"bytes"
	"io/ioutil"
	"log"
	"os/exec"

	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"
	"github.com/golang/protobuf/proto"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

type localBuildExecutor struct {
	contentAddressableStorage blobstore.BlobAccess
}

func NewLocalBuildExecutor(contentAddressableStorage blobstore.BlobAccess) BuildExecutor {
	return &localBuildExecutor{
		contentAddressableStorage: contentAddressableStorage,
	}
}

func (be *localBuildExecutor) Execute(request *remoteexecution.ExecuteRequest) (*remoteexecution.ExecuteResponse, error) {
	r, err := be.contentAddressableStorage.Get(request.InstanceName, request.Action.CommandDigest)
	if err != nil {
		log.Print("Execution.Execute: ", err)
		return nil, err
	}
	commandData, err := ioutil.ReadAll(r)
	if err != nil {
		log.Print("Execution.Execute: ", err)
		return nil, err
	}
	var command remoteexecution.Command
	if err := proto.Unmarshal(commandData, &command); err != nil {
		log.Print("Execution.Execute: ", err)
		return nil, err
	}

	// TODO(edsch): Set up file system.
	/*
		r, err = be.contentAddressableStorage.Get(request.InstanceName, request.Action.InputRootDigest)
		if err != nil {
			log.Print("Execution.Execute: ", err)
			return nil, err
		}
		inputRootData, err := ioutil.ReadAll(r)
		if err != nil {
			log.Print("Execution.Execute: ", err)
			return nil, err
		}
		var inputRoot remoteexecution.Directory
		if err := proto.Unmarshal(inputRootData, &inputRoot); err != nil {
			log.Print("Execution.Execute: ", err)
			return nil, err
		}
		log.Print("Got input root: ", inputRoot)
	*/

	// TODO(edsch): Use CommandContext(), so we have a proper timeout.
	// TODO(edsch): Test len(command.Arguments) properly!
	cmd := exec.Command(command.Arguments[0], command.Arguments[1:]...)
	for _, environmentVariable := range command.EnvironmentVariables {
		cmd.Env = append(cmd.Env, environmentVariable.Name+"="+environmentVariable.Value)
	}
	stdout := bytes.NewBuffer(nil)
	cmd.Stdout = stdout
	stderr := bytes.NewBuffer(nil)
	cmd.Stderr = stderr
	cmd.Run()
	// TODO(edsch): Set error code properly!
	return &remoteexecution.ExecuteResponse{
		Result: &remoteexecution.ActionResult{
			ExitCode:  123,
			StdoutRaw: stdout.Bytes(),
			StderrRaw: stderr.Bytes(),
		},
	}, nil
}
