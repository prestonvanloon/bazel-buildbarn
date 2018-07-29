package blobstore

import (
	"io"

	"github.com/prometheus/client_golang/prometheus"

	"golang.org/x/net/context"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

var (
	blobAccessOperationsStartedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "buildbarn",
			Subsystem: "blobstore",
			Name:      "blob_access_operations_started_total",
			Help:      "Total number of operations started on blob access objects.",
		},
		[]string{"name", "operation"},
	)
	blobAccessOperationsCompletedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "buildbarn",
			Subsystem: "blobstore",
			Name:      "blob_access_operations_completed_total",
			Help:      "Total number of operations completed on blob access objects.",
		},
		[]string{"name", "operation"},
	)
)

func init() {
	prometheus.MustRegister(blobAccessOperationsStartedTotal)
	prometheus.MustRegister(blobAccessOperationsCompletedTotal)
}

type metricsBlobAccess struct {
	blobAccess BlobAccess
	name       string
}

func NewMetricsBlobAccess(blobAccess BlobAccess, name string) BlobAccess {
	return &metricsBlobAccess{
		blobAccess: blobAccess,
		name:       name,
	}
}

func (ba *metricsBlobAccess) Get(ctx context.Context, instance string, digest *remoteexecution.Digest) io.ReadCloser {
	blobAccessOperationsStartedTotal.WithLabelValues(ba.name, "Get").Inc()
	defer blobAccessOperationsCompletedTotal.WithLabelValues(ba.name, "Get").Inc()
	return ba.blobAccess.Get(ctx, instance, digest)
}

func (ba *metricsBlobAccess) Put(ctx context.Context, instance string, digest *remoteexecution.Digest, r io.ReadCloser) error {
	blobAccessOperationsStartedTotal.WithLabelValues(ba.name, "Put").Inc()
	defer blobAccessOperationsCompletedTotal.WithLabelValues(ba.name, "Put").Inc()
	return ba.blobAccess.Put(ctx, instance, digest, r)
}

func (ba *metricsBlobAccess) FindMissing(ctx context.Context, instance string, digests []*remoteexecution.Digest) ([]*remoteexecution.Digest, error) {
	blobAccessOperationsStartedTotal.WithLabelValues(ba.name, "FindMissing").Inc()
	defer blobAccessOperationsCompletedTotal.WithLabelValues(ba.name, "FindMissing").Inc()
	return ba.blobAccess.FindMissing(ctx, instance, digests)
}
