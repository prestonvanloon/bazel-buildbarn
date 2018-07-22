load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "6dede2c65ce86289969b907f343a1382d33c14fbce5e30dd17bb59bb55bb6593",
    strip_prefix = "rules_docker-0.4.0",
    urls = ["https://github.com/bazelbuild/rules_docker/archive/v0.4.0.tar.gz"],
)

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "ba79c532ac400cefd1859cbc8a9829346aa69e3b99482cd5a54432092cbc3933",
    urls = ["https://github.com/bazelbuild/rules_go/releases/download/0.13.0/rules_go-0.13.0.tar.gz"],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "bc653d3e058964a5a26dcad02b6c72d7d63e6bb88d94704990b908a1445b8758",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.13.0/bazel-gazelle-0.13.0.tar.gz"],
)

http_file(
    name = "debian_deb_bash",
    sha256 = "ec375aa432a743ec44781cb91802f5752021e7e107b48ee2e9aa2f7b9fc23f86",
    url = "http://ftp.nl.debian.org/debian/pool/main/b/bash/bash_4.4-5_amd64.deb",
)

http_file(
    name = "debian_deb_dash",
    sha256 = "5084b7e30fde9c51c4312f4da45d4fdfb861ab91c1d514a164dcb8afd8612f65",
    url = "http://ftp.nl.debian.org/debian/pool/main/d/dash/dash_0.5.8-2.4_amd64.deb",
)

http_file(
    name = "debian_deb_libc6",
    sha256 = "e57b3e24ea79fcdb46549d4ed2b95bb9657f21bba60ed5d9136d5b7112500084",
    url = "http://ftp.nl.debian.org/debian/pool/main/g/glibc/libc6_2.24-11+deb9u3_amd64.deb",
)

http_file(
    name = "debian_deb_libgcc1",
    sha256 = "423a6541ee7ade69967c99492e267e724fd4675de53310861af5d1a1d249c4bf",
    url = "http://ftp.nl.debian.org/debian/pool/main/g/gcc-6/libgcc1_6.3.0-18+deb9u1_amd64.deb",
)

http_file(
    name = "debian_deb_libtinfo5",
    sha256 = "1d249a3193568b5ef785ad8993b9ba6d6fdca0eb359204c2355532b82d25e9f5",
    url = "http://ftp.nl.debian.org/debian/pool/main/n/ncurses/libtinfo5_6.0+20161126-1+deb9u2_amd64.deb",
)

load("@io_bazel_rules_docker//container:container.bzl", "container_pull", container_repositories = "repositories")

container_repositories()

load("@io_bazel_rules_go//go:def.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

gazelle_dependencies()

go_repository(
    name = "com_github_golang_protobuf",
    commit = "b4deda0973fb4c70b50d226b1af49f3da59f5265",
    importpath = "github.com/golang/protobuf",
)

go_repository(
    name = "com_github_satori_go_uuid",
    commit = "36e9d2ebbde5e3f13ab2e25625fd453271d6522e",
    importpath = "github.com/satori/go.uuid",
)

go_repository(
    name = "org_golang_google_genproto",
    commit = "e92b116572682a5b432ddd840aeaba2a559eeff1",
    importpath = "google.golang.org/genproto",
)

go_repository(
    name = "org_golang_google_grpc",
    commit = "168a6198bcb0ef175f7dacec0b8691fc141dc9b8",
    importpath = "google.golang.org/grpc",
)

go_repository(
    name = "org_golang_x_net",
    commit = "039a4258aec0ad3c79b905677cceeab13b296a77",
    importpath = "golang.org/x/net",
)

go_repository(
    name = "org_golang_x_text",
    commit = "f21a4dfb5e38f5895301dc265a8def02365cc3d0",
    importpath = "golang.org/x/text",
)
