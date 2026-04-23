{
  description = "Cartomancer: the GPX track library with a touch of magic";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.11";
    unstable.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = {
    self,
    nixpkgs,
    unstable,
    ...
  } @ inputs: let
    inherit (nixpkgs) lib;

    # The platform we build on.
    buildSystem = "x86_64-linux";

    # Systems we can produce packages for (via cross-compilation when different from buildSystem).
    eachSupportedSystem = lib.genAttrs [
      "x86_64-linux"
      "aarch64-linux"
    ];

    # Package sets for the build platform. Used for tools that run during the build (e.g. Go compiler, npm) regardless of target architecture.
    buildPkgs = import nixpkgs {localSystem = buildSystem;};
    buildPkgsUnstable = import unstable {localSystem = buildSystem;};

    version = self.shortRev or self.dirtyShortRev or "devel";

    # Frontend: React + Vite bundle, output ends up in ../static/ at frontend build time (see frontend/vite.config.ts).
    # Source includes the openapi.yaml from the backend because the npm `gen` script reads ../internal/pkg/api/openapi.yaml during build.
    # This is always built with native build-platform tooling.
    frontend = buildPkgs.buildNpmPackage {
      pname = "cartomancer-frontend";
      inherit version;

      src = lib.fileset.toSource {
        root = ./.;
        fileset = lib.fileset.unions [
          ./frontend
          ./internal/pkg/api/openapi.yaml
        ];
      };
      sourceRoot = "source/frontend";

      # To update: replace with pkgs.lib.fakeHash, run `nix build .#frontend`, copy the expected hash from the error message.
      npmDepsHash = "sha256-Yt1bLqm9SdGz6YPPuRcQi0FHV6eok0hmFmFJx6A8eks=";

      # npm run build invokes `npm run gen` which reads
      # ../internal/pkg/api/openapi.yaml. vite writes output to
      # ../static/, which is outside sourceRoot and read-only by default.
      npmBuildScript = "build";

      preBuild = ''
        chmod -R u+w ..
      '';

      installPhase = ''
        runHook preInstall
        mkdir -p $out
        cp -r ../static/. $out/
        runHook postInstall
      '';

      meta = {
        description = "Cartomancer frontend static assets";
        license = lib.licenses.mit;
      };
    };

    # Backend: Go binary with embedded frontend assets. Takes a (potentially cross-compiled) package set
    # and produces a binary for that set's host platform.
    # The Go toolchain itself is always pulled from the build-platform unstable set.
    mkCartomancer = crossPkgs: let
      buildGoModule = crossPkgs.buildGoModule.override {
        go = buildPkgsUnstable.go_1_26;
      };
    in
      buildGoModule {
        pname = "cartomancer";
        inherit version;

        src = ./.;

        # To update: replace with pkgs.lib.fakeHash, run `nix build`, copy the expected hash from the error message.
        vendorHash = "sha256-NGtUTC5hzZnxKElQG1t0g1ZH/b0ijPU52Ww0TiXJRJM=";

        # proxyVendor is required because `go generate` uses `go tool sqlc` which needs access to the full module graph, not just imported packages.
        proxyVendor = true;

        subPackages = ["."];

        env.CGO_ENABLED = "1";

        preBuild = ''
          mkdir -p static
          cp -r ${frontend}/. static/
          go generate ./...
        '';

        ldflags = [
          "-s"
          "-w"
          "-X=jo-m.ch/go/cartomancer/internal/pkg/api.buildVersion=${version}"
        ];

        # TODO:
        # Tests require a populated data directory and network for some fixtures, so they are skipped here. Use `make test` for tests.
        doCheck = false;

        meta = {
          description = "Cartomancer: the GPX track library with a touch of magic";
          homepage = "https://github.com/jo-m/cartomancer";
          license = lib.licenses.mit;
          mainProgram = "cartomancer";
        };
      };

    # Minimal container image with only the binary and CA certificates.
    mkDockerImage = crossPkgs: let
      cartomancer = mkCartomancer crossPkgs;
    in
      crossPkgs.dockerTools.buildImage {
        name = "cartomancer";
        tag = version;

        copyToRoot = crossPkgs.buildEnv {
          name = "cartomancer-root";
          paths = [
            cartomancer
            crossPkgs.cacert
            crossPkgs.dockerTools.caCertificates
            crossPkgs.dockerTools.fakeNss
          ];
          pathsToLink = [
            "/bin"
            "/etc"
          ];
        };

        extraCommands = ''
          mkdir -p data tmp
        '';

        config = {
          Entrypoint = ["/bin/cartomancer"];
          Cmd = ["serve"];
          Env = [
            "LISTEN_ADDR=0.0.0.0:8080"
            "SSL_CERT_FILE=/etc/ssl/certs/ca-bundle.crt"
          ];
          ExposedPorts = {
            "8080/tcp" = {};
          };
          WorkingDir = "/";
        };
      };
  in {
    packages = eachSupportedSystem (system: let
      # We treat everything as cross-compilation without a special case for the build platform.
      # Nixpkgs will do the right thing.
      crossPkgs = import nixpkgs {
        localSystem = buildSystem;
        crossSystem = system;
      };
    in {
      default = mkCartomancer crossPkgs;
      cartomancer = mkCartomancer crossPkgs;
      inherit frontend;
      dockerImage = mkDockerImage crossPkgs;
    });

    devShells.${buildSystem}.default = buildPkgs.mkShell {
      packages = [
        buildPkgsUnstable.go_1_26
        buildPkgs.nodejs_22
        buildPkgs.gcc
        buildPkgs.git
        buildPkgs.gnumake
        buildPkgs.sqlite
        buildPkgs.goreleaser
      ];

      shellHook = ''
        echo "cartomancer dev shell: $(go version | cut -d' ' -f1-3), node $(node --version)"
      '';
    };

    formatter.${buildSystem} = buildPkgs.alejandra;
  };
}
