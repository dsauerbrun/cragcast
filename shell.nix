{ sources ? import ./nix/sources.nix,
  pkgs ? import sources.nixpkgs {},
}:

pkgs.mkShell {
  nativeBuildInputs = [ 
    pkgs.pulumi-bin
    pkgs.go
    pkgs.gopls
    pkgs.awscli2
    pkgs.golangci-lint
  ];

}
