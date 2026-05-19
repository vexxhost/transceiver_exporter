{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";

    treefmt-nix.url = "github:numtide/treefmt-nix";
    treefmt-nix.inputs.nixpkgs.follows = "nixpkgs";

    devshell.url = "github:numtide/devshell";
    devshell.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs =
    inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [
        inputs.treefmt-nix.flakeModule
        inputs.devshell.flakeModule
      ];

      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
        "x86_64-darwin"
      ];
      perSystem =
        {
          config,
          self',
          inputs',
          pkgs,
          system,
          ...
        }:
        {
          treefmt = {
            programs.gofmt = {
              enable = true;
            };
          };

          devshells.default = {
            env = [
              {
                name = "CGO_ENABLED";
                value = "0";
              }
            ];

            packages =
              with pkgs;
              [
                actionlint
                go
                goreleaser
                golangci-lint
                jsonnet
                kubernetes-helm
                prometheus.cli
                stdenv.cc
                yq-go
              ]
              ++ (builtins.attrValues config.treefmt.build.programs);
          };
        };

      flake = {
      };
    };
}
