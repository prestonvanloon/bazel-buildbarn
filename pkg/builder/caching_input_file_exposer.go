package builder

import (
	"os"
	"path"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

type cachingInputFileExposer struct {
	base        InputFileExposer
	digestKeyer util.DigestKeyer
	path        string

	filesPresent map[string]bool
}

func NewCachingInputFileExposer(base InputFileExposer, digestKeyer util.DigestKeyer, path string) InputFileExposer {
	return &cachingInputFileExposer{
		base:        base,
		digestKeyer: digestKeyer,
		path:        path,

		filesPresent: map[string]bool{},
	}
}

func (fe *cachingInputFileExposer) Expose(instance string, digest *remoteexecution.Digest, outputPath string, isExecutable bool) error {
	key, err := fe.digestKeyer(instance, digest)
	if err != nil {
		return err
	}
	if isExecutable {
		key += "+x"
	} else {
		key += "-x"
	}

	cachePath := path.Join(fe.path, key)
	if !fe.filesPresent[key] {
		if err := fe.base.Expose(instance, digest, cachePath, isExecutable); err != nil {
			return err
		}
		fe.filesPresent[key] = true
	}
	return os.Link(cachePath, outputPath)
}
