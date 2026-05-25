{
  description = "GoDumb - line-count-hyperoptimized Go";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            gotools
            golangci-lint
            delve
            go-task
            nil
            nixfmt
          ];

          shellHook = ''
            export CGO_ENABLED=1
            echo "GoDumb dev shell ready 🚀 (try: task test)"
          '';
        };
      });
}
