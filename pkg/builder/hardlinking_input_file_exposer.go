package builder

import (
	"math/rand"
	"os"
	"path"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

type hardlinkingInputFileExposer struct {
	base        InputFileExposer
	digestKeyer util.DigestKeyer
	path        string
	maxFiles    int
	maxSize     int64

	filesPresentList      []string
	filesPresentSize      map[string]int64
	filesPresentTotalSize int64
}

func NewHardlinkingInputFileExposer(base InputFileExposer, digestKeyer util.DigestKeyer, path string, maxFiles int, maxSize int64) InputFileExposer {
	return &hardlinkingInputFileExposer{
		base:        base,
		digestKeyer: digestKeyer,
		path:        path,
		maxFiles:    maxFiles,
		maxSize:     maxSize,

		filesPresentSize: map[string]int64{},
	}
}

func (fe *hardlinkingInputFileExposer) makeSpace(size int64) error {
	for len(fe.filesPresentList) > 0 && (len(fe.filesPresentList) >= fe.maxFiles || fe.filesPresentTotalSize+size > fe.maxSize) {
		// Remove random file from disk.
		idx := rand.Intn(len(fe.filesPresentList))
		key := fe.filesPresentList[idx]
		if err := os.Remove(path.Join(fe.path, key)); err != nil {
			return err
		}

		// Remove file from bookkeeping.
		fe.filesPresentTotalSize -= fe.filesPresentSize[key]
		delete(fe.filesPresentSize, key)
		last := len(fe.filesPresentList) - 1
		fe.filesPresentList[idx] = fe.filesPresentList[last]
		fe.filesPresentList = fe.filesPresentList[:last]
	}
	return nil
}

func (fe *hardlinkingInputFileExposer) Expose(instance string, digest *remoteexecution.Digest, outputPath string, isExecutable bool) error {
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
	if _, ok := fe.filesPresentSize[key]; !ok {
		if err := fe.makeSpace(digest.SizeBytes); err != nil {
			return err
		}
		if err := fe.base.Expose(instance, digest, cachePath, isExecutable); err != nil {
			return err
		}
		fe.filesPresentList = append(fe.filesPresentList, key)
		fe.filesPresentSize[key] = digest.SizeBytes
		fe.filesPresentTotalSize += digest.SizeBytes
	}
	return os.Link(cachePath, outputPath)
}
