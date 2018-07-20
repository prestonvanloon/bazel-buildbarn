package builder

import (
	"errors"
	"io/ioutil"
	"log"

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
	log.Print("Got ExecuteRequest:", request)

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
	log.Print("Got command: ", command)

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

	return nil, errors.New("Fail!")
}
