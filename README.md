# Bazel Buildbarn

Bazel Buildbarn is an implementation of a Bazel buildfarm written in the
Go programming language. The intent is that this implementation is going
to be run in Docker containers on Kubernetes. In such environments it is
generally impossible to use [sandboxfs]{https://github.com/bazelbuild/sandboxfs/),
meaning we'll need to use basic UNIX credentials management (privilege
separation) to provide a rudimentary form of sandboxing.

Right now this codebase only provides a monolithic builder. A
distributed version will still need to be implemented. It can be
built and launched as follows:

    bazel run //cmd/bbb_monolithic:bbb_monolithic_container
    docker run -p 8980:8980 bazel/cmd/bbb_monolithic:bbb_monolithic_container

Bazel can make use of it by invoking it as follows:

    bazel build \
        --spawn_strategy=remote --genrule_strategy=remote \
        --strategy=Javac=remote --strategy=Closure=remote \
        --remote_executor=localhost:8980 --experimental_strict_action_env //...
