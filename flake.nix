{
  description = "Cartomancer: the GPX track library with a touch of magic";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.11";
  };

  outputs =
    { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
      ];

      forAllSystems =
        f:
        nixpkgs.lib.genAttrs systems (
          system:
          f (
            import nixpkgs {
              inherit system;
            }
          )
        );

      version = self.shortRev or self.dirtyShortRev or "devel";

      # Frontend: React + Vite bundle, output ends up in ../static/ at
      # frontend build time (see frontend/vite.config.ts).
      # Source includes the openapi.yaml from the backend because the npm
      # `gen` script reads ../internal/pkg/api/openapi.yaml during build.
      mkFrontend =
        pkgs:
        pkgs.buildNpmPackage {
          pname = "cartomancer-frontend";
          inherit version;

          src = pkgs.lib.fileset.toSource {
            root = ./.;
            fileset = pkgs.lib.fileset.unions [
              ./frontend
              ./internal/pkg/api/openapi.yaml
            ];
          };
          sourceRoot = "source/frontend";

          # To update: replace with pkgs.lib.fakeHash, run `nix build .#frontend`,
          # copy the expected hash from the error message.
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
            license = pkgs.lib.licenses.mit;
          };
        };

      # Backend: Go binary with embedded frontend assets.
      mkCartomancer =
        pkgs:
        let
          frontend = mkFrontend pkgs;
        in
        pkgs.buildGoModule {
          pname = "cartomancer";
          inherit version;

          src = ./.;

          # To update: replace with pkgs.lib.fakeHash, run `nix build`, copy
          # the expected hash from the error message.
          vendorHash = "sha256-NGtUTC5hzZnxKElQG1t0g1ZH/b0ijPU52Ww0TiXJRJM=";

          # proxyVendor is required because `go generate` uses `go tool sqlc`
          # which needs access to the full module graph, not just imported
          # packages.
          proxyVendor = true;

          subPackages = [ "." ];

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

          # Tests require a populated data directory and network for some
          # fixtures, so they are skipped here. Use `make test` for tests.
          doCheck = false;

          meta = {
            description = "Cartomancer: the GPX track library with a touch of magic";
            homepage = "https://github.com/jo-m/cartomancer";
            license = pkgs.lib.licenses.mit;
            mainProgram = "cartomancer";
          };
        };

      # Minimal container image with only the binary and CA certificates.
      mkDockerImage =
        pkgs:
        let
          cartomancer = mkCartomancer pkgs;
        in
        pkgs.dockerTools.buildImage {
          name = "cartomancer";
          tag = version;

          copyToRoot = pkgs.buildEnv {
            name = "cartomancer-root";
            paths = [
              cartomancer
              pkgs.cacert
              pkgs.dockerTools.caCertificates
              pkgs.dockerTools.fakeNss
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
            Entrypoint = [ "/bin/cartomancer" ];
            Cmd = [ "serve" ];
            Env = [
              "LISTEN_ADDR=0.0.0.0:8080"
              "SSL_CERT_FILE=/etc/ssl/certs/ca-bundle.crt"
            ];
            ExposedPorts = {
              "8080/tcp" = { };
            };
            WorkingDir = "/";
          };
        };
    in
    {
      packages = forAllSystems (pkgs: {
        default = mkCartomancer pkgs;
        cartomancer = mkCartomancer pkgs;
        frontend = mkFrontend pkgs;
        dockerImage = mkDockerImage pkgs;
      });

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            go
            nodejs_22
            gcc
            git
            gnumake
            sqlite
            goreleaser
          ];

          shellHook = ''
            echo "cartomancer dev shell: $(go version | cut -d' ' -f1-3), node $(node --version)"
          '';
        };
      });

      formatter = forAllSystems (pkgs: pkgs.nixfmt);
    };
}
