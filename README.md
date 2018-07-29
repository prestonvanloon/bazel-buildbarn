# Bazel Buildbarn

Bazel Buildbarn is an implementation of a Bazel
[buildfarm](https://en.wikipedia.org/wiki/Compile_farm) written in the
Go programming language. The intent behind this implementation is that
it is fast and easy to scale. It consists of the following three
components:

- `bbb_frontend`: A service capable of processing RPCs from Bazel. It
  can store build input and serve cached build output and action results.
- `bbb_scheduler`: A service that receives requests from `bbb_frontend`
  to queue build actions that need to be run.
- `bbb_worker`: A service that runs build actions by fetching them from
  the `bbb_scheduler`.

The `bbb_frontend` and `bbb_worker` services can be replicated easily.
It is also possible to start multiple `bbb_scheduler` processes if
multiple build queues are desired (e.g., supporting multiple build
operating systems). These processes may use S3-like buckets to store
data.

Below is a diagram of what a typical Bazel Buildbarn deployment may look
like. In this diagram, the arrows represent the direction in which
network connections are established.

<p align="center">
  <img src="https://github.com/EdSchouten/bazel-buildbarn/raw/master/doc/diagrams/bbb-overview.png" alt="Overview of a typical Bazel Buildbarn deployment"/>
</p>

One common use case for this implementation is to be run in Docker
containers on Kubernetes. In such environments it is
generally impossible to use [sandboxfs](https://github.com/bazelbuild/sandboxfs/),
meaning `bbb_worker` uses basic UNIX credentials management (privilege
separation) to provide a rudimentary form of sandboxing. The
`bbb_worker` daemon runs as user `root`, whereas the build action is run
as user `build`. Input files are only readable to the latter.

## Setting up Bazel Buildbarn

TODO(edsch): Provide example Kubernetes configuration files.

## Using Bazel Buildbarn

Bazel can make use of Bazel Buildbarn by invoking it as follows:

    bazel build \
        --experimental_strict_action_env --genrule_strategy=remote \
        --remote_executor=...:8980 --spawn_strategy=remote \
        --strategy=Closure=remote --strategy=Javac=remote \
        //...
